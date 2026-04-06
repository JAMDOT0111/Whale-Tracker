package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"eth-sweeper/model"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const anonymousUserID = "anonymous"

type AppStore struct {
	mu            sync.RWMutex
	users         map[string]model.User
	gmailTokens   map[string]model.GmailToken
	preferences   map[string]model.NotificationPreference
	whales        map[string]model.WhaleAccount
	watchlists    map[string]model.WatchlistItem
	alerts        map[string]model.AlertEvent
	notifications map[string]model.NotificationLog
	snapshotAt    string
}

func NewAppStore() *AppStore {
	store := &AppStore{
		users:         map[string]model.User{},
		gmailTokens:   map[string]model.GmailToken{},
		preferences:   map[string]model.NotificationPreference{},
		whales:        map[string]model.WhaleAccount{},
		watchlists:    map[string]model.WatchlistItem{},
		alerts:        map[string]model.AlertEvent{},
		notifications: map[string]model.NotificationLog{},
	}
	if strings.EqualFold(os.Getenv("ENABLE_DEMO_DATA"), "true") {
		store.seedDemoWhales()
	}
	return store
}

func (s *AppStore) UpsertUser(_ context.Context, req model.GoogleLoginRequest) model.User {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" {
		email = "demo@example.com"
	}
	id := stableID("user:" + email)
	now := nowISO()

	s.mu.Lock()
	defer s.mu.Unlock()

	user := s.users[id]
	if user.ID == "" {
		user.CreatedAt = now
	}
	user.ID = id
	user.Email = email
	user.Name = strings.TrimSpace(req.Name)
	if user.Name == "" {
		user.Name = email
	}
	user.AvatarURL = strings.TrimSpace(req.AvatarURL)
	s.users[id] = user
	if strings.TrimSpace(req.GmailAccessToken) != "" {
		existing := s.gmailTokens[id]
		token := model.GmailToken{
			AccessToken:  strings.TrimSpace(req.GmailAccessToken),
			RefreshToken: strings.TrimSpace(req.GmailRefreshToken),
			Expiry:       strings.TrimSpace(req.GmailTokenExpiry),
		}
		if token.RefreshToken == "" {
			token.RefreshToken = existing.RefreshToken
		}
		s.gmailTokens[id] = token
	}

	pref := s.preferences[id]
	if pref.UserID == "" {
		pref = model.NotificationPreference{
			UserID:       id,
			Email:        email,
			GmailEnabled: true,
			MinSeverity:  "info",
			UpdatedAt:    now,
		}
		s.preferences[id] = pref
	}

	return user
}

func (s *AppStore) GetGmailToken(_ context.Context, userID string) model.GmailToken {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.gmailTokens[userID]
}

func (s *AppStore) GetUser(_ context.Context, userID string) (model.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[userID]
	return user, ok
}

func (s *AppStore) UpsertWhales(_ context.Context, whales []model.WhaleAccount) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := nowISO()
	if containsRealWhaleSource(whales) {
		for addr, existing := range s.whales {
			if strings.Contains(existing.Source, "demo") {
				delete(s.whales, addr)
			}
		}
	}
	for _, whale := range whales {
		addr := strings.ToLower(strings.TrimSpace(whale.Address))
		if !IsValidEthAddress(addr) {
			continue
		}
		whale.Address = addr
		if whale.UpdatedAt == "" {
			whale.UpdatedAt = now
		}
		if whale.Source == "" {
			whale.Source = "etherscan_top_accounts_import"
		}
		if whale.Confidence == 0 {
			whale.Confidence = 0.95
		}
		if len(whale.Labels) == 0 {
			whale.Labels = s.labelsForAddressLocked(addr, whale)
		}
		s.whales[addr] = whale
	}
	s.snapshotAt = now
	return len(whales)
}

func (s *AppStore) ListWhales(_ context.Context, minBalance float64, sortKey string, page, pageSize int, userID string) model.WhaleListResponse {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 25
	}
	if pageSize > 100 {
		pageSize = 100
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	tracked := s.trackedAddressesLocked(userID)
	items := make([]model.WhaleAccount, 0, len(s.whales))
	for _, whale := range s.whales {
		bal := parseFloatSafe(whale.BalanceETH)
		if minBalance > 0 && bal < minBalance {
			continue
		}
		whale.IsTracked = tracked[whale.Address]
		items = append(items, whale)
	}

	sort.SliceStable(items, func(i, j int) bool {
		switch sortKey {
		case "balance_asc":
			return parseFloatSafe(items[i].BalanceETH) < parseFloatSafe(items[j].BalanceETH)
		case "rank_desc":
			return items[i].Rank > items[j].Rank
		case "rank_asc":
			return items[i].Rank < items[j].Rank
		default:
			return parseFloatSafe(items[i].BalanceETH) > parseFloatSafe(items[j].BalanceETH)
		}
	})

	total := len(items)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	source := "etherscan_top_accounts_import"
	if len(items) > 0 {
		source = items[0].Source
	}

	return model.WhaleListResponse{
		Items:       items[start:end],
		Page:        page,
		PageSize:    pageSize,
		Total:       total,
		HasNext:     end < total,
		SnapshotAt:  s.snapshotAt,
		Source:      source,
		LimitNotice: "Etherscan Top Accounts data is synced from configured CSV URL or parsed from public account pages.",
	}
}

func (s *AppStore) GetWhale(_ context.Context, address string) (model.WhaleAccount, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	whale, ok := s.whales[strings.ToLower(address)]
	return whale, ok
}

func (s *AppStore) UpsertWatchlist(_ context.Context, userID string, req model.UpsertWatchlistRequest) (model.WatchlistItem, error) {
	addr := strings.ToLower(strings.TrimSpace(req.Address))
	if !IsValidEthAddress(addr) {
		return model.WatchlistItem{}, fmt.Errorf("invalid ethereum address")
	}
	if userID == "" {
		userID = anonymousUserID
	}
	threshold := normalizeETHAmount(req.MinInteractionETH)
	if threshold == "" {
		threshold = "500"
	}
	notificationsOn := true
	if req.NotificationOn != nil {
		notificationsOn = *req.NotificationOn
	}

	now := nowISO()
	id := stableID(userID + ":" + addr)

	s.mu.Lock()
	defer s.mu.Unlock()

	item := s.watchlists[id]
	if item.ID == "" {
		item.CreatedAt = now
	}
	item.ID = id
	item.UserID = userID
	item.Address = addr
	item.Alias = strings.TrimSpace(req.Alias)
	item.MinInteractionETH = threshold
	item.NotificationOn = notificationsOn
	item.Labels = s.labelsForAddressLocked(addr, s.whales[addr])
	item.UpdatedAt = now
	s.watchlists[id] = item
	return item, nil
}

func (s *AppStore) ListWatchlists(_ context.Context, userID string) []model.WatchlistItem {
	if userID == "" {
		userID = anonymousUserID
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]model.WatchlistItem, 0)
	for _, item := range s.watchlists {
		if item.UserID == userID {
			items = append(items, item)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})
	return items
}

func (s *AppStore) DeleteWatchlist(_ context.Context, userID, id string) bool {
	if userID == "" {
		userID = anonymousUserID
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.watchlists[id]
	if !ok || item.UserID != userID {
		return false
	}
	delete(s.watchlists, id)
	return true
}

func (s *AppStore) UpsertPreference(_ context.Context, userID string, pref model.NotificationPreference) model.NotificationPreference {
	if userID == "" {
		userID = anonymousUserID
	}
	pref.UserID = userID
	pref.Email = strings.ToLower(strings.TrimSpace(pref.Email))
	pref.UpdatedAt = nowISO()
	if pref.MinSeverity == "" {
		pref.MinSeverity = "info"
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.preferences[userID] = pref
	return pref
}

func (s *AppStore) GetPreference(_ context.Context, userID string) model.NotificationPreference {
	if userID == "" {
		userID = anonymousUserID
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	pref := s.preferences[userID]
	if pref.UserID == "" {
		pref = model.NotificationPreference{
			UserID:       userID,
			GmailEnabled: false,
			MinSeverity:  "info",
			UpdatedAt:    nowISO(),
		}
	}
	return pref
}

func (s *AppStore) CreateAlert(_ context.Context, alert model.AlertEvent) (model.AlertEvent, bool) {
	if alert.ID == "" {
		alert.ID = stableID(alert.DedupeKey)
	}
	if alert.CreatedAt == "" {
		alert.CreatedAt = nowISO()
	}
	if alert.Status == "" {
		alert.Status = "new"
	}
	if alert.DedupeKey == "" {
		alert.DedupeKey = alert.ID
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.alerts {
		if existing.DedupeKey == alert.DedupeKey {
			return existing, false
		}
	}
	s.alerts[alert.ID] = alert
	return alert, true
}

func (s *AppStore) ListAlerts(_ context.Context, userID string) []model.AlertEvent {
	if userID == "" {
		userID = anonymousUserID
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]model.AlertEvent, 0)
	for _, alert := range s.alerts {
		if alert.UserID == userID {
			items = append(items, alert)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})
	return items
}

func (s *AppStore) MarkAlertRead(_ context.Context, userID, id string) bool {
	if userID == "" {
		userID = anonymousUserID
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	alert, ok := s.alerts[id]
	if !ok || alert.UserID != userID {
		return false
	}
	alert.Status = "read"
	s.alerts[id] = alert
	return true
}

func (s *AppStore) AddNotificationLog(_ context.Context, log model.NotificationLog) model.NotificationLog {
	if log.ID == "" {
		log.ID = stableID(log.AlertID + ":" + log.Channel + ":" + strconv.Itoa(time.Now().Nanosecond()))
	}
	if log.CreatedAt == "" {
		log.CreatedAt = nowISO()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifications[log.ID] = log
	return log
}

func (s *AppStore) AllWatchlists(_ context.Context) []model.WatchlistItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]model.WatchlistItem, 0, len(s.watchlists))
	for _, item := range s.watchlists {
		items = append(items, item)
	}
	return items
}

func (s *AppStore) LabelsForAddress(_ context.Context, address string) []model.AddressLabelResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.labelsForAddressLocked(strings.ToLower(address), s.whales[strings.ToLower(address)])
}

func (s *AppStore) trackedAddressesLocked(userID string) map[string]bool {
	if userID == "" {
		userID = anonymousUserID
	}
	tracked := map[string]bool{}
	for _, item := range s.watchlists {
		if item.UserID == userID {
			tracked[item.Address] = true
		}
	}
	return tracked
}

func (s *AppStore) labelsForAddressLocked(address string, whale model.WhaleAccount) []model.AddressLabelResult {
	now := nowISO()
	labels := make([]model.AddressLabelResult, 0, 3)

	if whale.Address != "" || parseFloatSafe(whale.BalanceETH) >= 1000 {
		labels = append(labels, model.AddressLabelResult{
			Category:      "whale",
			Name:          "Whale",
			Source:        "etherscan_top_accounts_import",
			Confidence:    0.95,
			LastCheckedAt: now,
			EvidenceRef:   "top_accounts_rank",
			Heuristic:     false,
		})
	}

	if known := LookupAddress(address); known != nil {
		labels = append(labels, model.AddressLabelResult{
			Category:      normalizeLabelCategory(known.Tag),
			Name:          known.Name,
			Source:        "curated_local_labels",
			Confidence:    0.9,
			LastCheckedAt: now,
			EvidenceRef:   "backend/service/labels.go",
			Heuristic:     false,
		})
	}

	if len(labels) == 0 {
		labels = append(labels, model.AddressLabelResult{
			Category:      "unknown",
			Name:          "Unknown",
			Source:        "heuristic_default",
			Confidence:    0.2,
			LastCheckedAt: now,
			Heuristic:     true,
		})
	}
	return labels
}

func (s *AppStore) seedDemoWhales() {
	now := nowISO()
	demo := []model.WhaleAccount{
		{Rank: 1, Address: "0xbe0eb53f46cd790cd13851d5eff43d12404d33e8", NameTag: "Binance cold wallet", BalanceETH: "1830000", Percentage: "4.7%", TxnCount: 884, UpdatedAt: now, Source: "demo_seed", Confidence: 0.8},
		{Rank: 2, Address: "0x00000000219ab540356cbb839cbe05303d7705fa", NameTag: "Eth2 deposit contract", BalanceETH: "1140000", Percentage: "2.9%", TxnCount: 1312, UpdatedAt: now, Source: "demo_seed", Confidence: 0.8},
		{Rank: 3, Address: "0x40b38765696e3d5d8d9d834d8aad4bb6e418e489", NameTag: "Robinhood", BalanceETH: "781600", Percentage: "2.0%", TxnCount: 451, UpdatedAt: now, Source: "demo_seed", Confidence: 0.8},
		{Rank: 4, Address: "0xda9dfa130df4de4673b89022ee50ff26f6ea73cf", NameTag: "Kraken", BalanceETH: "523700", Percentage: "1.3%", TxnCount: 265, UpdatedAt: now, Source: "demo_seed", Confidence: 0.8},
		{Rank: 5, Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", NameTag: "WETH contract", BalanceETH: "481600", Percentage: "1.2%", TxnCount: 982422, UpdatedAt: now, Source: "demo_seed", Confidence: 0.8},
		{Rank: 6, Address: "0xf977814e90da44bfa03b6295a0616a897441acec", NameTag: "Binance", BalanceETH: "329700", Percentage: "0.8%", TxnCount: 726, UpdatedAt: now, Source: "demo_seed", Confidence: 0.8},
	}
	for rank := 7; rank <= 10000; rank++ {
		balance := demoWhaleBalance(rank)
		demo = append(demo, model.WhaleAccount{
			Rank:       rank,
			Address:    "0x" + stableID(fmt.Sprintf("demo-expanded-whale-%d", rank)),
			NameTag:    fmt.Sprintf("Whale Candidate #%d", rank),
			BalanceETH: strconv.FormatFloat(balance, 'f', 2, 64),
			Percentage: fmt.Sprintf("%.3f%%", balance/39000000*100),
			TxnCount:   50 + rank%900,
			UpdatedAt:  now,
			Source:     "demo_expanded_seed",
			Confidence: 0.55,
		})
	}
	s.UpsertWhales(context.Background(), demo)
}

func demoWhaleBalance(rank int) float64 {
	switch {
	case rank <= 20:
		return 329700 - float64(rank-6)*14500
	case rank <= 100:
		return 128000 - float64(rank-20)*900
	case rank <= 500:
		return 56000 - float64(rank-100)*90
	case rank <= 1500:
		return 20000 - float64(rank-500)*10
	case rank <= 4000:
		return 10000 - float64(rank-1500)*2.4
	default:
		return 4000 - float64(rank-4000)*0.5
	}
}

func stableID(input string) string {
	sum := sha1.Sum([]byte(input))
	return hex.EncodeToString(sum[:])
}

func containsRealWhaleSource(whales []model.WhaleAccount) bool {
	for _, whale := range whales {
		if whale.Source != "" && !strings.Contains(whale.Source, "demo") {
			return true
		}
	}
	return false
}

func normalizeLabelCategory(tag string) string {
	switch strings.ToLower(strings.TrimSpace(tag)) {
	case "exchange":
		return "exchange"
	case "bridge":
		return "bridge"
	case "defi":
		return "defi_protocol"
	default:
		return tag
	}
}

func normalizeETHAmount(raw string) string {
	cleaned := strings.ToUpper(strings.TrimSpace(raw))
	cleaned = strings.ReplaceAll(cleaned, "TEH", "ETH")
	cleaned = strings.ReplaceAll(cleaned, "ETH", "")
	cleaned = strings.ReplaceAll(cleaned, ">", "")
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return ""
	}
	if _, err := strconv.ParseFloat(cleaned, 64); err != nil {
		return ""
	}
	return cleaned
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}
