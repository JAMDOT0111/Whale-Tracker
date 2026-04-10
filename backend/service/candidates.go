package service

import (
	"context"
	"eth-sweeper/model"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	candidateMinBalanceETH = 1000
	candidateMinTxnCount   = 10
	candidateWatchScore    = 40
)

type candidateBase struct {
	whale           model.WhaleAccount
	labels          []model.AddressLabelResult
	exclusionReason string
	preScore        int
}

type CandidateService struct {
	store              *AppStore
	etherscan          *EtherscanClient
	mu                 sync.RWMutex
	items              []model.CandidateAddress
	summary            model.CandidateSummaryResponse
	refreshedAt        string
	cursors            map[string]model.CandidateScanCursor
	activityHistory    map[string][]model.Transaction
	fullSnapshotReady  bool
	lastFullBuildAt    string
	lastIncrementalAt  string
	build              model.CandidateBuildState
	runtimeCtx         context.Context
	snapshotGeneration uint64
}

func NewCandidateService(store *AppStore, etherscan *EtherscanClient) *CandidateService {
	return &CandidateService{
		store:           store,
		etherscan:       etherscan,
		cursors:         map[string]model.CandidateScanCursor{},
		activityHistory: map[string][]model.Transaction{},
		build: model.CandidateBuildState{
			Status: "idle",
			Mode:   "quick",
		},
	}
}

func (s *CandidateService) Invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = nil
	s.summary = model.CandidateSummaryResponse{}
	s.refreshedAt = ""
	s.cursors = map[string]model.CandidateScanCursor{}
	s.activityHistory = map[string][]model.Transaction{}
	s.fullSnapshotReady = false
	s.lastFullBuildAt = ""
	s.lastIncrementalAt = ""
	s.build = model.CandidateBuildState{
		Status:  "idle",
		Mode:    "quick",
		Message: "candidate cache invalidated; full scan required",
	}
	s.snapshotGeneration++
}

func (s *CandidateService) ListCandidates(ctx context.Context, tier string, limit, minScore int) model.CandidateListResponse {
	s.ensureSnapshot(ctx)

	s.mu.RLock()
	items := copyCandidateItems(s.items)
	summary := s.summarySnapshotLocked()
	refreshedAt := s.refreshedAt
	s.mu.RUnlock()

	filtered := make([]model.CandidateAddress, 0, len(items))
	for _, item := range items {
		if minScore > 0 && item.Score < minScore {
			continue
		}
		if tier == "review" && !item.SelectedForReview {
			continue
		}
		if tier == "watch" && item.PriorityTier != "watch" {
			continue
		}
		filtered = append(filtered, item)
	}

	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return model.CandidateListResponse{
		Items:                 filtered,
		Total:                 len(filtered),
		AvailableTotal:        summary.AvailableTotal,
		ReviewTotal:           summary.ReviewTotal,
		WatchTotal:            summary.WatchTotal,
		RefreshedAt:           refreshedAt,
		LastBuildMode:         summary.LastBuildMode,
		ActivityEnrichedCount: summary.ActivityEnrichedCount,
		ScanLimit:             summary.ScanLimit,
		LimitNotice:           "Candidate pool v2 keeps quick snapshot as backlog only. Review and watch are shown only after a completed full or incremental activity scan.",
	}
}

func (s *CandidateService) Summary(ctx context.Context) model.CandidateSummaryResponse {
	s.ensureSnapshot(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.summarySnapshotLocked()
}

func (s *CandidateService) GetCandidate(ctx context.Context, address string) (model.CandidateAddress, bool) {
	s.ensureSnapshot(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()

	address = strings.ToLower(strings.TrimSpace(address))
	for _, item := range s.items {
		if item.Address == address {
			return item, true
		}
	}
	return model.CandidateAddress{}, false
}

func (s *CandidateService) Rebuild(ctx context.Context) (model.CandidateRebuildResponse, error) {
	started := s.startBuild("full")
	summary := s.Summary(ctx)
	if !started {
		return model.CandidateRebuildResponse{
			OK:      true,
			Started: false,
			Message: "candidate rebuild already running",
			Summary: summary,
		}, nil
	}
	return model.CandidateRebuildResponse{
		OK:      true,
		Started: true,
		Message: "candidate rebuild started in background",
		Summary: summary,
	}, nil
}

func (s *CandidateService) ensureSnapshot(ctx context.Context) {
	s.mu.RLock()
	needsBuild := len(s.items) == 0
	s.mu.RUnlock()
	if !needsBuild {
		return
	}

	snapshot, summary, _ := s.buildSnapshot(ctx, false)
	s.mu.Lock()
	if len(s.items) == 0 {
		s.items = snapshot
		s.summary = summary
		s.refreshedAt = summary.RefreshedAt
	}
	s.mu.Unlock()
}

type candidateBuildOptions struct {
	mode     string
	progress func(processed, total int, address string)
}

type candidateBuildResult struct {
	items           []model.CandidateAddress
	summary         model.CandidateSummaryResponse
	cursors         map[string]model.CandidateScanCursor
	activityHistory map[string][]model.Transaction
}

func (s *CandidateService) buildSnapshot(ctx context.Context, enrichActivity bool) ([]model.CandidateAddress, model.CandidateSummaryResponse, error) {
	result, err := s.buildSnapshotWithOptions(ctx, candidateBuildOptions{
		mode: buildMode(enrichActivity),
	})
	if err != nil {
		return nil, model.CandidateSummaryResponse{}, err
	}
	return result.items, result.summary, nil
}

func (s *CandidateService) buildSnapshotWithOptions(ctx context.Context, opts candidateBuildOptions) (candidateBuildResult, error) {
	now := nowISO()
	whales := s.store.AllWhales(ctx)
	if len(whales) == 0 {
		return candidateBuildResult{
			items: []model.CandidateAddress{},
			summary: model.CandidateSummaryResponse{
				AvailableTotal:        0,
				ReviewTotal:           0,
				WatchTotal:            0,
				RefreshedAt:           now,
				LastBuildMode:         opts.mode,
				ActivityEnrichedCount: 0,
				ScanLimit:             0,
			},
			cursors:         map[string]model.CandidateScanCursor{},
			activityHistory: map[string][]model.Transaction{},
		}, nil
	}

	eligible := make([]candidateBase, 0, len(whales))
	for _, whale := range whales {
		balance := parseFloatSafe(whale.BalanceETH)
		if balance < candidateMinBalanceETH || whale.TxnCount < candidateMinTxnCount {
			continue
		}

		labels := s.store.LabelsForAddress(ctx, whale.Address)
		reason := candidateExclusionReason(whale.Address, whale.NameTag, whale.TxnCount, labels)
		if reason != "" {
			continue
		}

		eligible = append(eligible, candidateBase{
			whale:           whale,
			labels:          labels,
			exclusionReason: reason,
			preScore:        scoreBalance(whale.BalanceETH) + scoreHistoricalActivity(whale.TxnCount),
		})
	}

	sort.SliceStable(eligible, func(i, j int) bool {
		if eligible[i].preScore != eligible[j].preScore {
			return eligible[i].preScore > eligible[j].preScore
		}
		leftBalance := parseFloatSafe(eligible[i].whale.BalanceETH)
		rightBalance := parseFloatSafe(eligible[j].whale.BalanceETH)
		if leftBalance != rightBalance {
			return leftBalance > rightBalance
		}
		if eligible[i].whale.TxnCount != eligible[j].whale.TxnCount {
			return eligible[i].whale.TxnCount > eligible[j].whale.TxnCount
		}
		return eligible[i].whale.Rank < eligible[j].whale.Rank
	})

	availableTotal := len(eligible)
	scanLimit := len(eligible)
	if opts.mode == "quick" {
		scanLimit = min(len(eligible), candidateScanLimit())
		if len(eligible) > scanLimit {
			eligible = eligible[:scanLimit]
		}
	}

	historyByAddress := map[string][]model.Transaction{}
	cursorByAddress := map[string]model.CandidateScanCursor{}
	existingItems := map[string]model.CandidateAddress{}
	existingHistory := map[string][]model.Transaction{}
	existingCursors := map[string]model.CandidateScanCursor{}

	s.mu.RLock()
	for _, item := range s.items {
		existingItems[item.Address] = item
	}
	for address, history := range s.activityHistory {
		existingHistory[address] = append([]model.Transaction(nil), history...)
	}
	for address, cursor := range s.cursors {
		existingCursors[address] = cursor
	}
	s.mu.RUnlock()

	latestBlock := uint64(0)
	fetchEndBlock := uint64(candidateMaxEndBlock())
	if opts.mode != "quick" {
		if s.etherscan == nil || s.etherscan.apiKey == "" {
			return candidateBuildResult{}, fmt.Errorf("etherscan api key not configured")
		}
		var err error
		latestBlock, err = s.etherscan.latestBlockNumber()
		if err != nil {
			if opts.mode == "incremental" {
				return candidateBuildResult{}, err
			}
			latestBlock = 0
		} else {
			fetchEndBlock = latestBlock
		}
	}

	items := make([]model.CandidateAddress, 0, len(eligible))
	activityEnrichedCount := 0
	if opts.progress != nil {
		opts.progress(0, len(eligible), "")
	}
	for index, base := range eligible {
		if opts.progress != nil && opts.mode != "quick" {
			opts.progress(index, len(eligible), base.whale.Address)
		}
		stats := quickActivityStats()
		history := []model.Transaction(nil)
		cursor := model.CandidateScanCursor{}

		if opts.mode != "quick" {
			var (
				enriched model.CandidateActivityStats
				err      error
			)
			switch opts.mode {
			case "incremental":
				enriched, history, cursor, err = s.loadIncrementalActivity(base.whale.Address, latestBlock, existingCursors[base.whale.Address], existingHistory[base.whale.Address])
			default:
				enriched, history, cursor, err = s.loadFullActivity(base.whale.Address, fetchEndBlock, latestBlock)
			}
			if err == nil {
				stats = enriched
				if stats.ActivityLoaded {
					activityEnrichedCount++
				}
				if len(history) > 0 {
					historyByAddress[base.whale.Address] = history
				}
				if cursor.LastScannedBlock > 0 {
					cursorByAddress[base.whale.Address] = cursor
				}
			} else {
				if previous, ok := existingItems[base.whale.Address]; ok && previous.Activity.ActivityLoaded {
					stats = previous.Activity
					stats.ActivitySource = previous.Activity.ActivitySource + "_stale"
					historyByAddress[base.whale.Address] = append([]model.Transaction(nil), existingHistory[base.whale.Address]...)
					cursorByAddress[base.whale.Address] = existingCursors[base.whale.Address]
				} else {
					stats.ActivitySource = "recent_activity_unavailable"
				}
			}
		}

		item := buildCandidateItem(base, stats, now, opts.mode)
		items = append(items, item)
		if opts.progress != nil && opts.mode != "quick" {
			opts.progress(index+1, len(eligible), base.whale.Address)
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		return candidateLess(items[i], items[j])
	})

	applyCandidateSelection(items, candidateReviewMin(), candidateReviewLimit())

	reviewTotal := 0
	watchTotal := 0
	for index := range items {
		if items[index].SelectedForReview {
			reviewTotal++
			items[index].PriorityTier = "review"
			continue
		}
		if items[index].Activity.ActivityLoaded && (items[index].EventPass || items[index].Score >= candidateWatchScore) {
			watchTotal++
			items[index].PriorityTier = "watch"
			continue
		}
		items[index].PriorityTier = "backlog"
	}

	summary := model.CandidateSummaryResponse{
		AvailableTotal:        availableTotal,
		ReviewTotal:           reviewTotal,
		WatchTotal:            watchTotal,
		RefreshedAt:           now,
		LastBuildMode:         opts.mode,
		ActivityEnrichedCount: activityEnrichedCount,
		ScanLimit:             scanLimit,
	}
	return candidateBuildResult{
		items:           items,
		summary:         summary,
		cursors:         cursorByAddress,
		activityHistory: historyByAddress,
	}, nil
}

func buildCandidateItem(base candidateBase, stats model.CandidateActivityStats, now string, buildMode string) model.CandidateAddress {
	breakdown := model.CandidateScoreBreakdown{
		Balance:    scoreBalance(base.whale.BalanceETH),
		Historical: scoreHistoricalActivity(base.whale.TxnCount),
		Flow:       scoreFlow(stats),
		Activity:   scoreRecentActivity(stats),
		Protocol:   scoreProtocol(stats),
		Anomaly:    scoreAnomaly(stats),
	}
	breakdown.Total = breakdown.Balance + breakdown.Historical + breakdown.Flow + breakdown.Activity + breakdown.Protocol + breakdown.Anomaly

	eventPass := hasRecentActivitySignal(stats)

	reasons := []string{
		fmt.Sprintf("balance %s ETH", base.whale.BalanceETH),
		fmt.Sprintf("historical tx count %d", base.whale.TxnCount),
	}
	if stats.ActivityLoaded {
		if stats.TxCount7d > 0 {
			reasons = append(reasons, fmt.Sprintf("7d tx count %d", stats.TxCount7d))
		}
		if math.Abs(parseFloatSafe(stats.NetflowETH7d)) >= 300 {
			reasons = append(reasons, fmt.Sprintf("7d netflow %s ETH", stats.NetflowETH7d))
		}
		if parseFloatSafe(stats.LargestTxETH7d) >= 150 {
			reasons = append(reasons, fmt.Sprintf("largest 7d tx %s ETH", stats.LargestTxETH7d))
		}
		if stats.ProtocolInteractions7d > 0 {
			reasons = append(reasons, fmt.Sprintf("protocol interactions %d in 7d", stats.ProtocolInteractions7d))
		}
		if stats.IsReactivated {
			reasons = append(reasons, fmt.Sprintf("reactivated after %d dormancy days", stats.DormancyDays))
		}
	} else if buildMode != "quick" {
		reasons = append(reasons, "recent ETH activity enrichment unavailable")
	} else {
		reasons = append(reasons, "quick snapshot only; full activity scan pending")
	}

	return model.CandidateAddress{
		Address:        base.whale.Address,
		NameTag:        base.whale.NameTag,
		Rank:           base.whale.Rank,
		BalanceETH:     base.whale.BalanceETH,
		TxnCount:       base.whale.TxnCount,
		Labels:         append([]model.AddressLabelResult(nil), base.labels...),
		BasePass:       true,
		EventPass:      eventPass,
		Score:          breakdown.Total,
		PriorityTier:   "backlog",
		Reasons:        reasons,
		ScoreBreakdown: breakdown,
		Activity:       stats,
		UpdatedAt:      now,
	}
}

func (s *CandidateService) loadFullActivity(address string, endBlock, cursorBlock uint64) (model.CandidateActivityStats, []model.Transaction, model.CandidateScanCursor, error) {
	if s.etherscan == nil || s.etherscan.apiKey == "" {
		return quickActivityStats(), nil, model.CandidateScanCursor{}, fmt.Errorf("etherscan api key not configured")
	}
	history, err := s.fetchRecentETHActivityHistory(address, 0, endBlock)
	if err != nil {
		return quickActivityStats(), nil, model.CandidateScanCursor{}, err
	}
	stats := computeCandidateActivityStats(address, history)
	stats.ActivitySource = "etherscan_full_activity"
	cursor := model.CandidateScanCursor{
		Address:          address,
		LastScannedBlock: cursorBlock,
		LastScannedAt:    nowISO(),
		LastActivityAt:   stats.LastActivityAt,
	}
	return stats, history, cursor, nil
}

func (s *CandidateService) loadIncrementalActivity(address string, latestBlock uint64, cursor model.CandidateScanCursor, existingHistory []model.Transaction) (model.CandidateActivityStats, []model.Transaction, model.CandidateScanCursor, error) {
	if cursor.LastScannedBlock == 0 {
		return s.loadFullActivity(address, uint64(candidateMaxEndBlock()), 0)
	}
	startBlock := cursor.LastScannedBlock + 1
	if latestBlock > 0 && startBlock > latestBlock {
		history := trimActivityHistory(existingHistory)
		stats := computeCandidateActivityStats(address, history)
		stats.ActivitySource = "etherscan_incremental_activity"
		cursor.LastScannedAt = nowISO()
		cursor.LastActivityAt = stats.LastActivityAt
		return stats, history, cursor, nil
	}

	newHistory, err := s.fetchRecentETHActivityHistory(address, startBlock, latestBlock)
	if err != nil {
		return quickActivityStats(), nil, model.CandidateScanCursor{}, err
	}
	merged := mergeActivityHistory(existingHistory, newHistory)
	stats := computeCandidateActivityStats(address, merged)
	stats.ActivitySource = "etherscan_incremental_activity"
	cursor = model.CandidateScanCursor{
		Address:          address,
		LastScannedBlock: latestBlock,
		LastScannedAt:    nowISO(),
		LastActivityAt:   stats.LastActivityAt,
	}
	return stats, merged, cursor, nil
}

func (s *CandidateService) fetchRecentETHActivityHistory(address string, startBlock, endBlock uint64) ([]model.Transaction, error) {
	if s.etherscan == nil || s.etherscan.apiKey == "" {
		return nil, fmt.Errorf("etherscan api key not configured")
	}
	cutoff30d := time.Now().UTC().Add(-30 * 24 * time.Hour)
	pageLimit := candidateActivityPageLimit()
	pageSize := candidateActivityPageSize()
	all := make([]model.Transaction, 0, pageLimit*pageSize*2)
	successes := 0

	for _, fetcher := range []func(string, int, int, uint64, uint64) ([]model.Transaction, error){
		s.etherscan.fetchNormalTxRange,
		s.etherscan.fetchInternalTxRange,
	} {
		stream := make([]model.Transaction, 0, pageLimit*pageSize)
		for page := 1; page <= pageLimit; page++ {
			txs, err := fetcher(address, page, pageSize, startBlock, endBlock)
			if err != nil {
				if page == 1 {
					break
				}
				break
			}
			if len(txs) == 0 {
				break
			}
			successes++
			stream = append(stream, txs...)
			if startBlock == 0 && olderThanCutoff(txs[len(txs)-1], cutoff30d) && len(stream) >= 2 {
				break
			}
			if len(txs) < pageSize {
				break
			}
		}
		all = append(all, stream...)
	}

	if successes == 0 {
		return nil, fmt.Errorf("recent ETH activity unavailable")
	}
	return mergeActivityHistory(nil, all), nil
}

func computeCandidateActivityStats(address string, history []model.Transaction) model.CandidateActivityStats {
	now := time.Now().UTC()
	cutoff24h := now.Add(-24 * time.Hour)
	cutoff7d := now.Add(-7 * 24 * time.Hour)
	cutoff30d := now.Add(-30 * 24 * time.Hour)

	stats := model.CandidateActivityStats{
		ActivityLoaded:    true,
		ActivitySource:    "etherscan_eth_activity",
		InflowETH24h:      "0",
		OutflowETH24h:     "0",
		NetflowETH24h:     "0",
		InflowETH7d:       "0",
		OutflowETH7d:      "0",
		NetflowETH7d:      "0",
		LargestInTxETH7d:  "0",
		LargestOutTxETH7d: "0",
		LargestTxETH7d:    "0",
		LastEnrichedAt:    nowISO(),
	}

	address = strings.ToLower(address)
	activityDays := map[string]bool{}
	protocolCategories := map[string]bool{}
	lastSeenTimes := make([]time.Time, 0, 2)

	for _, tx := range history {
		timestamp, err := time.Parse(time.RFC3339, tx.Timestamp)
		if err != nil {
			continue
		}
		if strings.EqualFold(tx.From, tx.To) {
			continue
		}
		if tx.Asset != "ETH" {
			continue
		}
		value := parseFloatSafe(tx.Value)
		if value <= 0 {
			continue
		}

		if stats.LastActivityAt == "" {
			stats.LastActivityAt = tx.Timestamp
		}
		if len(lastSeenTimes) < 2 {
			lastSeenTimes = append(lastSeenTimes, timestamp)
		}

		if timestamp.After(cutoff30d) {
			stats.TxCount30d++
		}
		if timestamp.After(cutoff24h) {
			stats.TxCount24h++
		}
		if timestamp.After(cutoff7d) {
			stats.TxCount7d++
			activityDays[timestamp.Format("2006-01-02")] = true

			if strings.EqualFold(tx.To, address) {
				stats.InflowETH7d = formatDecimal(parseFloatSafe(stats.InflowETH7d) + value)
				if timestamp.After(cutoff24h) {
					stats.InflowETH24h = formatDecimal(parseFloatSafe(stats.InflowETH24h) + value)
				}
				if value > parseFloatSafe(stats.LargestInTxETH7d) {
					stats.LargestInTxETH7d = formatDecimal(value)
				}
			}
			if strings.EqualFold(tx.From, address) {
				stats.OutflowETH7d = formatDecimal(parseFloatSafe(stats.OutflowETH7d) + value)
				if timestamp.After(cutoff24h) {
					stats.OutflowETH24h = formatDecimal(parseFloatSafe(stats.OutflowETH24h) + value)
				}
				if value > parseFloatSafe(stats.LargestOutTxETH7d) {
					stats.LargestOutTxETH7d = formatDecimal(value)
				}
			}

			if value > parseFloatSafe(stats.LargestTxETH7d) {
				stats.LargestTxETH7d = formatDecimal(value)
			}

			counterparty := strings.ToLower(tx.To)
			if strings.EqualFold(tx.To, address) {
				counterparty = strings.ToLower(tx.From)
			}
			if category := candidateProtocolCategory(counterparty); category != "" {
				stats.ProtocolInteractions7d++
				protocolCategories[category] = true
			}
		}
	}

	stats.ActiveDays7d = len(activityDays)
	stats.ProtocolTypes7d = len(protocolCategories)
	stats.NetflowETH24h = formatDecimal(parseFloatSafe(stats.InflowETH24h) - parseFloatSafe(stats.OutflowETH24h))
	stats.NetflowETH7d = formatDecimal(parseFloatSafe(stats.InflowETH7d) - parseFloatSafe(stats.OutflowETH7d))
	if len(lastSeenTimes) >= 2 {
		stats.DormancyDays = int(lastSeenTimes[0].Sub(lastSeenTimes[1]).Hours() / 24)
	}
	stats.IsReactivated = stats.TxCount7d > 0 && stats.DormancyDays >= 30
	return stats
}

func mergeActivityHistory(existing, incoming []model.Transaction) []model.Transaction {
	merged := make([]model.Transaction, 0, len(existing)+len(incoming))
	seen := map[string]bool{}
	for _, tx := range append(append([]model.Transaction{}, incoming...), existing...) {
		key := tx.Hash + ":" + tx.Category
		if seen[key] {
			continue
		}
		seen[key] = true
		merged = append(merged, tx)
	}
	sort.SliceStable(merged, func(i, j int) bool {
		return merged[i].Timestamp > merged[j].Timestamp
	})
	return trimActivityHistory(merged)
}

func trimActivityHistory(history []model.Transaction) []model.Transaction {
	if len(history) <= 2 {
		return append([]model.Transaction(nil), history...)
	}
	cutoff := time.Now().UTC().Add(-45 * 24 * time.Hour)
	trimmed := make([]model.Transaction, 0, len(history))
	for index, tx := range history {
		if index < 2 {
			trimmed = append(trimmed, tx)
			continue
		}
		timestamp, err := time.Parse(time.RFC3339, tx.Timestamp)
		if err != nil || timestamp.After(cutoff) {
			trimmed = append(trimmed, tx)
		}
	}
	return trimmed
}

func olderThanCutoff(tx model.Transaction, cutoff time.Time) bool {
	timestamp, err := time.Parse(time.RFC3339, tx.Timestamp)
	if err != nil {
		return false
	}
	return timestamp.Before(cutoff)
}

func candidateExclusionReason(address, nameTag string, txnCount int, labels []model.AddressLabelResult) string {
	lowerName := strings.ToLower(strings.TrimSpace(nameTag))
	for _, label := range labels {
		switch label.Category {
		case "exchange":
			return "known_exchange"
		case "bridge":
			return "known_bridge"
		case "defi_protocol":
			return "known_protocol"
		}
	}

	if address == "0x0000000000000000000000000000000000000000" || strings.HasSuffix(strings.ToLower(address), "dead") || containsKeyword(lowerName, []string{" burn", "null", "dead"}) {
		return "burn_null_dead"
	}
	if containsKeyword(lowerName, []string{
		"hacker", "exploiter", "attacker", "drainer",
	}) {
		return "known_incident_actor"
	}
	if containsKeyword(lowerName, []string{
		"binance", "coinbase", "kraken", "okx", "bybit", "bitfinex", "gemini", "kucoin", "gate.io", "gate dep",
		"huobi", "htx", "mexc", "robinhood", "bithumb", "bitstamp", "upbit", "crypto.com", "paxos", "deribit",
		"bitgo", "wintermute", "cumberland", "matrixport", "cobo", "bitbank", "revolut", "coincheck", "quadrigacx",
		"btcturk", "bitpanda", "bitflyer", "bullish", "hashkey", "bitso", "etoro", "coinone", "coinhako", "bitget",
	}) {
		return "exchange_or_custody_like"
	}
	if containsKeyword(lowerName, []string{
		"wormhole", "stargate", "layerzero", "across", "hop protocol", "celer", "synapse", "arbitrum",
		"optimism", "base portal", "base bridge", "polygon bridge", "zksync", "mantle", "bridge",
	}) {
		return "bridge_or_l2_infra"
	}
	if containsKeyword(lowerName, []string{
		"beacon deposit contract", "wrapped ether", "treasury", "custody", "staking", "router", "liquidat", "collector",
		"vault", "pool", "multisig", "settlement", "fee collector", "withdrawal", "withdrawdao", "batcher", "oracle", "depositor",
		"deposit contract", "hot wallet", "cold wallet", "wallet", "dep:", "proxy", "lockbox", "portal", "reserve",
		"rollup", "escrow", "distributor", "safe", "stake.com", "liquidity",
	}) {
		return "service_or_protocol"
	}
	if strings.ContainsAny(nameTag, ":：") {
		return "tagged_service_or_protocol"
	}
	if txnCount >= 100000 && lowerName != "" {
		return "high_frequency_tagged_service"
	}
	return ""
}

func candidateProtocolCategory(address string) string {
	if label := LookupAddress(address); label != nil {
		return normalizeLabelCategory(label.Tag)
	}
	return ""
}

func looksLikePersonalTag(name string) bool {
	trimmed := strings.TrimSpace(name)
	parts := strings.Fields(trimmed)
	if len(parts) == 2 && isCapitalized(parts[0]) && isCapitalized(parts[1]) {
		return true
	}
	return strings.HasPrefix(trimmed, "Vb ")
}

func isCapitalized(part string) bool {
	if part == "" {
		return false
	}
	runes := []rune(part)
	return runes[0] >= 'A' && runes[0] <= 'Z'
}

func scoreBalance(balanceETH string) int {
	balance := parseFloatSafe(balanceETH)
	switch {
	case balance >= 10000:
		return 20
	case balance >= 3000:
		return 14
	case balance >= 1000:
		return 8
	default:
		return 0
	}
}

func scoreHistoricalActivity(txnCount int) int {
	switch {
	case txnCount >= 100000:
		return 10
	case txnCount >= 10000:
		return 8
	case txnCount >= 1000:
		return 6
	case txnCount >= 100:
		return 4
	case txnCount >= 10:
		return 2
	default:
		return 0
	}
}

func scoreFlow(stats model.CandidateActivityStats) int {
	score := 0
	netflow := math.Abs(parseFloatSafe(stats.NetflowETH7d))
	largest := parseFloatSafe(stats.LargestTxETH7d)
	switch {
	case netflow >= 1000:
		score += 20
	case netflow >= 300:
		score += 10
	}
	switch {
	case largest >= 500:
		score += 15
	case largest >= 150:
		score += 10
	}
	if score > 30 {
		return 30
	}
	return score
}

func scoreRecentActivity(stats model.CandidateActivityStats) int {
	score := 0
	switch {
	case stats.TxCount7d >= 5:
		score += 14
	case stats.TxCount7d >= 2:
		score += 8
	}
	if stats.ActiveDays7d >= 3 {
		score += 6
	}
	if score > 20 {
		return 20
	}
	return score
}

func scoreProtocol(stats model.CandidateActivityStats) int {
	switch {
	case stats.ProtocolTypes7d >= 2:
		return 15
	case stats.ProtocolInteractions7d >= 3:
		return 14
	case stats.ProtocolInteractions7d >= 1:
		return 10
	default:
		return 0
	}
}

func scoreAnomaly(stats model.CandidateActivityStats) int {
	if stats.IsReactivated {
		return 10
	}
	return 0
}

func quickActivityStats() model.CandidateActivityStats {
	return model.CandidateActivityStats{
		ActivityLoaded:         false,
		ActivitySource:         "quick_snapshot_only",
		InflowETH24h:           "0",
		OutflowETH24h:          "0",
		NetflowETH24h:          "0",
		InflowETH7d:            "0",
		OutflowETH7d:           "0",
		NetflowETH7d:           "0",
		LargestInTxETH7d:       "0",
		LargestOutTxETH7d:      "0",
		LargestTxETH7d:         "0",
		ProtocolTypes7d:        0,
		ProtocolInteractions7d: 0,
	}
}

func candidateLess(left, right model.CandidateAddress) bool {
	if left.Score != right.Score {
		return left.Score > right.Score
	}
	leftNetflow := math.Abs(parseFloatSafe(left.Activity.NetflowETH7d))
	rightNetflow := math.Abs(parseFloatSafe(right.Activity.NetflowETH7d))
	if leftNetflow != rightNetflow {
		return leftNetflow > rightNetflow
	}
	leftLargest := parseFloatSafe(left.Activity.LargestTxETH7d)
	rightLargest := parseFloatSafe(right.Activity.LargestTxETH7d)
	if leftLargest != rightLargest {
		return leftLargest > rightLargest
	}
	leftBalance := parseFloatSafe(left.BalanceETH)
	rightBalance := parseFloatSafe(right.BalanceETH)
	if leftBalance != rightBalance {
		return leftBalance > rightBalance
	}
	return left.Rank < right.Rank
}

func applyCandidateSelection(items []model.CandidateAddress, minReview, maxReview int) {
	for index := range items {
		items[index].SelectedForReview = false
	}

	thresholds := []int{60, 55, 50}
	selectedCount := 0
	for _, threshold := range thresholds {
		selectedCount = 0
		for index := range items {
			if items[index].BasePass && items[index].Activity.ActivityLoaded && items[index].EventPass && items[index].Score >= threshold {
				items[index].SelectedForReview = true
				selectedCount++
			}
		}
		if selectedCount >= minReview {
			break
		}
		for index := range items {
			items[index].SelectedForReview = false
		}
	}

	if selectedCount < minReview {
		target := minReview
		if target > maxReview {
			target = maxReview
		}
		eligibleCount := 0
		for index := range items {
			if !items[index].BasePass || !items[index].Activity.ActivityLoaded || !items[index].EventPass {
				continue
			}
			if eligibleCount >= target {
				break
			}
			if !items[index].SelectedForReview {
				items[index].SelectedForReview = true
			}
			eligibleCount++
		}
	} else if selectedCount > maxReview {
		count := 0
		for index := range items {
			if !items[index].SelectedForReview {
				continue
			}
			count++
			if count > maxReview {
				items[index].SelectedForReview = false
			}
		}
	}
}

func hasRecentActivitySignal(stats model.CandidateActivityStats) bool {
	if !stats.ActivityLoaded {
		return false
	}
	return stats.TxCount7d >= 2 ||
		math.Abs(parseFloatSafe(stats.NetflowETH7d)) >= 300 ||
		parseFloatSafe(stats.LargestTxETH7d) >= 150 ||
		stats.ProtocolInteractions7d >= 1 ||
		stats.IsReactivated
}

func copyCandidateItems(items []model.CandidateAddress) []model.CandidateAddress {
	copied := make([]model.CandidateAddress, 0, len(items))
	for _, item := range items {
		itemCopy := item
		itemCopy.Labels = append([]model.AddressLabelResult(nil), item.Labels...)
		itemCopy.Reasons = append([]string(nil), item.Reasons...)
		copied = append(copied, itemCopy)
	}
	return copied
}

func candidateScanLimit() int {
	return candidateEnvInt("CANDIDATE_SCAN_LIMIT", 200)
}

func candidateActivityPageLimit() int {
	return candidateEnvInt("CANDIDATE_ACTIVITY_PAGE_LIMIT", 3)
}

func candidateActivityPageSize() int {
	return candidateEnvInt("CANDIDATE_ACTIVITY_PAGE_SIZE", 100)
}

func candidateMaxEndBlock() int {
	return candidateEnvInt("CANDIDATE_MAX_END_BLOCK", 99999999)
}

func candidateReviewLimit() int {
	return candidateEnvInt("CANDIDATE_REVIEW_LIMIT", 200)
}

func candidateReviewMin() int {
	return candidateEnvInt("CANDIDATE_REVIEW_MIN", 100)
}

func candidateEnvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func buildMode(enrichActivity bool) string {
	if enrichActivity {
		return "full"
	}
	return "quick"
}

func containsKeyword(value string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func formatDecimal(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
