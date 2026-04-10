package service

import (
	"context"
	"eth-sweeper/model"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func (s *CandidateService) StartScheduler(ctx context.Context) {
	s.mu.Lock()
	s.runtimeCtx = ctx
	s.mu.Unlock()

	if strings.EqualFold(os.Getenv("ENABLE_CANDIDATE_JOBS"), "false") {
		log.Println("[candidates] incremental scheduler disabled; set ENABLE_CANDIDATE_JOBS=true to enable it")
		return
	}

	interval := candidateIncrementalInterval()
	log.Printf("[candidates] incremental scheduler enabled; interval=%s", interval)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !s.hasFullSnapshot() {
					continue
				}
				if !s.startBuild("incremental") {
					continue
				}
			}
		}
	}()
}

func (s *CandidateService) startBuild(mode string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if buildRunning(s.build.Status) {
		return false
	}

	s.build = model.CandidateBuildState{
		Status:    "queued",
		Mode:      mode,
		Message:   buildQueuedMessage(mode),
		StartedAt: nowISO(),
	}
	generation := s.snapshotGeneration
	ctx := s.backgroundContextLocked()

	go s.runBuild(ctx, mode, generation)
	return true
}

func (s *CandidateService) runBuild(ctx context.Context, mode string, generation uint64) {
	s.updateBuildState(func(state *model.CandidateBuildState) {
		state.Status = "running"
		state.Mode = mode
		state.Message = buildRunningMessage(mode)
		if state.StartedAt == "" {
			state.StartedAt = nowISO()
		}
	})

	result, err := s.buildSnapshotWithOptions(ctx, candidateBuildOptions{
		mode: mode,
		progress: func(processed, total int, address string) {
			s.updateBuildState(func(state *model.CandidateBuildState) {
				state.Status = "running"
				state.Mode = mode
				state.Processed = processed
				state.Total = total
				state.Message = buildProgressMessage(mode, processed, total, address)
			})
		},
	})
	if err != nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if generation != s.snapshotGeneration {
			return
		}
		s.build = model.CandidateBuildState{
			Status:     "failed",
			Mode:       mode,
			Message:    buildFailedMessage(mode),
			Processed:  s.build.Processed,
			Total:      s.build.Total,
			StartedAt:  s.build.StartedAt,
			FinishedAt: nowISO(),
			Error:      err.Error(),
		}
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if generation != s.snapshotGeneration {
		return
	}

	s.items = result.items
	s.summary = result.summary
	s.refreshedAt = result.summary.RefreshedAt
	s.cursors = result.cursors
	s.activityHistory = result.activityHistory
	if mode == "full" {
		s.fullSnapshotReady = true
		s.lastFullBuildAt = result.summary.RefreshedAt
	}
	if mode == "incremental" {
		s.fullSnapshotReady = true
		s.lastIncrementalAt = result.summary.RefreshedAt
	}
	s.build = model.CandidateBuildState{
		Status:     "completed",
		Mode:       mode,
		Message:    buildCompletedMessage(mode, result.summary.ReviewTotal, result.summary.WatchTotal),
		Processed:  result.summary.ActivityEnrichedCount,
		Total:      result.summary.ScanLimit,
		StartedAt:  s.build.StartedAt,
		FinishedAt: nowISO(),
	}
}

func (s *CandidateService) updateBuildState(update func(*model.CandidateBuildState)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	update(&s.build)
}

func (s *CandidateService) summarySnapshotLocked() model.CandidateSummaryResponse {
	summary := s.summary
	summary.FullSnapshotReady = s.fullSnapshotReady
	summary.LastFullBuildAt = s.lastFullBuildAt
	summary.LastIncrementalAt = s.lastIncrementalAt
	summary.Build = s.build
	return summary
}

func (s *CandidateService) hasFullSnapshot() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.fullSnapshotReady
}

func (s *CandidateService) backgroundContextLocked() context.Context {
	if s.runtimeCtx != nil {
		return s.runtimeCtx
	}
	return context.Background()
}

func buildRunning(status string) bool {
	return status == "queued" || status == "running"
}

func buildQueuedMessage(mode string) string {
	switch mode {
	case "incremental":
		return "queued incremental refresh"
	default:
		return "queued full scan"
	}
}

func buildRunningMessage(mode string) string {
	switch mode {
	case "incremental":
		return "incremental refresh started"
	default:
		return "full candidate scan started"
	}
}

func buildProgressMessage(mode string, processed, total int, address string) string {
	if total <= 0 {
		return buildRunningMessage(mode)
	}
	label := "full scan"
	if mode == "incremental" {
		label = "incremental refresh"
	}
	if address == "" {
		return label + " prepared " + strconv.Itoa(total) + " candidates"
	}
	return label + " " + strconv.Itoa(processed) + "/" + strconv.Itoa(total) + " " + ShortAddress(address)
}

func buildFailedMessage(mode string) string {
	if mode == "incremental" {
		return "incremental refresh failed"
	}
	return "full candidate scan failed"
}

func buildCompletedMessage(mode string, reviewTotal, watchTotal int) string {
	if mode == "incremental" {
		return "incremental refresh completed with " + strconv.Itoa(reviewTotal) + " review and " + strconv.Itoa(watchTotal) + " watch addresses"
	}
	return "full candidate scan completed with " + strconv.Itoa(reviewTotal) + " review and " + strconv.Itoa(watchTotal) + " watch addresses"
}

func candidateIncrementalInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv("CANDIDATE_INCREMENTAL_INTERVAL"))
	if raw == "" {
		return 15 * time.Minute
	}
	interval, err := time.ParseDuration(raw)
	if err != nil || interval < time.Minute {
		log.Printf("[candidates] invalid CANDIDATE_INCREMENTAL_INTERVAL=%q; using 15m", raw)
		return 15 * time.Minute
	}
	return interval
}
