package model

type User struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url,omitempty"`
	CreatedAt string `json:"created_at"`
}

type GoogleLoginRequest struct {
	IDToken           string `json:"id_token"`
	Email             string `json:"email"`
	Name              string `json:"name"`
	AvatarURL         string `json:"avatar_url"`
	GmailAccessToken  string `json:"gmail_access_token"`
	GmailRefreshToken string `json:"gmail_refresh_token"`
	GmailTokenExpiry  string `json:"gmail_token_expiry"`
}

type GmailToken struct {
	AccessToken  string `json:"-"`
	RefreshToken string `json:"-"`
	Expiry       string `json:"-"`
}

type NotificationPreference struct {
	UserID       string `json:"user_id"`
	Email        string `json:"email"`
	GmailEnabled bool   `json:"gmail_enabled"`
	MinSeverity  string `json:"min_severity"`
	UpdatedAt    string `json:"updated_at"`
}

type AddressLabelResult struct {
	Category      string  `json:"category"`
	Name          string  `json:"name"`
	Source        string  `json:"source"`
	Confidence    float64 `json:"confidence"`
	LastCheckedAt string  `json:"last_checked_at"`
	EvidenceRef   string  `json:"evidence_ref,omitempty"`
	Heuristic     bool    `json:"heuristic"`
}

type WhaleAccount struct {
	Rank        int                  `json:"rank"`
	Address     string               `json:"address"`
	NameTag     string               `json:"name_tag,omitempty"`
	BalanceETH  string               `json:"balance_eth"`
	BalanceWei  string               `json:"balance_wei,omitempty"`
	Percentage  string               `json:"percentage,omitempty"`
	TxnCount    int                  `json:"txn_count"`
	Labels      []AddressLabelResult `json:"labels"`
	IsTracked   bool                 `json:"is_tracked"`
	UpdatedAt   string               `json:"updated_at"`
	Source      string               `json:"source"`
	Confidence  float64              `json:"confidence"`
	EvidenceRef string               `json:"evidence_ref,omitempty"`
}

type WhaleListResponse struct {
	Items       []WhaleAccount `json:"items"`
	Page        int            `json:"page"`
	PageSize    int            `json:"page_size"`
	Total       int            `json:"total"`
	HasNext     bool           `json:"has_next"`
	SnapshotAt  string         `json:"snapshot_at,omitempty"`
	Source      string         `json:"source"`
	LimitNotice string         `json:"limit_notice"`
}

type CandidateScoreBreakdown struct {
	Balance    int `json:"balance"`
	Historical int `json:"historical"`
	Flow       int `json:"flow"`
	Activity   int `json:"activity"`
	Protocol   int `json:"protocol"`
	Anomaly    int `json:"anomaly"`
	Total      int `json:"total"`
}

type CandidateActivityStats struct {
	ActivityLoaded         bool   `json:"activity_loaded"`
	ActivitySource         string `json:"activity_source"`
	LastActivityAt         string `json:"last_activity_at,omitempty"`
	LastEnrichedAt         string `json:"last_enriched_at,omitempty"`
	TxCount24h             int    `json:"tx_count_24h"`
	TxCount7d              int    `json:"tx_count_7d"`
	TxCount30d             int    `json:"tx_count_30d"`
	InflowETH24h           string `json:"inflow_eth_24h"`
	OutflowETH24h          string `json:"outflow_eth_24h"`
	NetflowETH24h          string `json:"netflow_eth_24h"`
	InflowETH7d            string `json:"inflow_eth_7d"`
	OutflowETH7d           string `json:"outflow_eth_7d"`
	NetflowETH7d           string `json:"netflow_eth_7d"`
	LargestInTxETH7d       string `json:"largest_in_tx_eth_7d"`
	LargestOutTxETH7d      string `json:"largest_out_tx_eth_7d"`
	LargestTxETH7d         string `json:"largest_tx_eth_7d"`
	ActiveDays7d           int    `json:"active_days_7d"`
	DormancyDays           int    `json:"dormancy_days"`
	IsReactivated          bool   `json:"is_reactivated"`
	ProtocolInteractions7d int    `json:"protocol_interactions_7d"`
	ProtocolTypes7d        int    `json:"protocol_types_7d"`
}

type CandidateScanCursor struct {
	Address          string `json:"address"`
	LastScannedBlock uint64 `json:"last_scanned_block"`
	LastScannedAt    string `json:"last_scanned_at,omitempty"`
	LastActivityAt   string `json:"last_activity_at,omitempty"`
}

type CandidateBuildState struct {
	Status     string `json:"status"`
	Mode       string `json:"mode"`
	Message    string `json:"message,omitempty"`
	Processed  int    `json:"processed"`
	Total      int    `json:"total"`
	StartedAt  string `json:"started_at,omitempty"`
	FinishedAt string `json:"finished_at,omitempty"`
	Error      string `json:"error,omitempty"`
}

type CandidateAddress struct {
	Address           string                  `json:"address"`
	NameTag           string                  `json:"name_tag,omitempty"`
	Rank              int                     `json:"rank"`
	BalanceETH        string                  `json:"balance_eth"`
	TxnCount          int                     `json:"txn_count"`
	Labels            []AddressLabelResult    `json:"labels"`
	BasePass          bool                    `json:"base_pass"`
	EventPass         bool                    `json:"event_pass"`
	Score             int                     `json:"score"`
	PriorityTier      string                  `json:"priority_tier"`
	SelectedForReview bool                    `json:"selected_for_review"`
	Reasons           []string                `json:"reasons"`
	ScoreBreakdown    CandidateScoreBreakdown `json:"score_breakdown"`
	Activity          CandidateActivityStats  `json:"activity"`
	UpdatedAt         string                  `json:"updated_at"`
}

type CandidateListResponse struct {
	Items                 []CandidateAddress `json:"items"`
	Total                 int                `json:"total"`
	AvailableTotal        int                `json:"available_total"`
	ReviewTotal           int                `json:"review_total"`
	WatchTotal            int                `json:"watch_total"`
	RefreshedAt           string             `json:"refreshed_at,omitempty"`
	LastBuildMode         string             `json:"last_build_mode"`
	ActivityEnrichedCount int                `json:"activity_enriched_count"`
	ScanLimit             int                `json:"scan_limit"`
	LimitNotice           string             `json:"limit_notice"`
}

type CandidateSummaryResponse struct {
	AvailableTotal        int                 `json:"available_total"`
	ReviewTotal           int                 `json:"review_total"`
	WatchTotal            int                 `json:"watch_total"`
	RefreshedAt           string              `json:"refreshed_at,omitempty"`
	LastBuildMode         string              `json:"last_build_mode"`
	ActivityEnrichedCount int                 `json:"activity_enriched_count"`
	ScanLimit             int                 `json:"scan_limit"`
	FullSnapshotReady     bool                `json:"full_snapshot_ready"`
	LastFullBuildAt       string              `json:"last_full_build_at,omitempty"`
	LastIncrementalAt     string              `json:"last_incremental_at,omitempty"`
	Build                 CandidateBuildState `json:"build"`
}

type CandidateRebuildResponse struct {
	OK      bool                     `json:"ok"`
	Started bool                     `json:"started"`
	Message string                   `json:"message"`
	Summary CandidateSummaryResponse `json:"summary"`
}

type TokenApprovalItem struct {
	TokenAddress    string `json:"token_address"`
	TokenName       string `json:"token_name,omitempty"`
	TokenSymbol     string `json:"token_symbol,omitempty"`
	TokenDecimals   int    `json:"token_decimals"`
	Spender         string `json:"spender"`
	SpenderLabel    string `json:"spender_label,omitempty"`
	ApprovalValue   string `json:"approval_value"`
	ApprovalDisplay string `json:"approval_display,omitempty"`
	ApprovalType    string `json:"approval_type"`
	RiskLevel       string `json:"risk_level"`
	TxHash          string `json:"tx_hash"`
	Timestamp       string `json:"timestamp"`
}

type TokenApprovalResponse struct {
	Address        string              `json:"address"`
	Items          []TokenApprovalItem `json:"items"`
	ScannedAt      string              `json:"scanned_at"`
	Source         string              `json:"source"`
	CandidateOnly  bool                `json:"candidate_only"`
	CandidateTier  string              `json:"candidate_tier,omitempty"`
	LimitApplied   int                 `json:"limit_applied"`
	ReviewRequired bool                `json:"review_required"`
}

type WhaleImportResponse struct {
	ImportID   string `json:"import_id"`
	Imported   int    `json:"imported"`
	Skipped    int    `json:"skipped"`
	Source     string `json:"source"`
	ImportedAt string `json:"imported_at"`
}

type WhaleImportURLRequest struct {
	URL string `json:"url"`
}

type WatchlistItem struct {
	ID                string               `json:"id"`
	UserID            string               `json:"user_id"`
	Address           string               `json:"address"`
	Alias             string               `json:"alias,omitempty"`
	MinInteractionETH string               `json:"min_interaction_eth"`
	NotificationOn    bool                 `json:"notification_on"`
	Labels            []AddressLabelResult `json:"labels"`
	CreatedAt         string               `json:"created_at"`
	UpdatedAt         string               `json:"updated_at"`
}

type UpsertWatchlistRequest struct {
	Address           string `json:"address" binding:"required"`
	Alias             string `json:"alias"`
	MinInteractionETH string `json:"min_interaction_eth"`
	NotificationOn    *bool  `json:"notification_on"`
}

type AlertEvent struct {
	ID           string               `json:"id"`
	UserID       string               `json:"user_id"`
	Address      string               `json:"address"`
	Type         string               `json:"type"`
	Severity     string               `json:"severity"`
	ThresholdETH string               `json:"threshold_eth"`
	Title        string               `json:"title"`
	Description  string               `json:"description"`
	Evidence     []Evidence           `json:"evidence"`
	Labels       []AddressLabelResult `json:"labels"`
	Confidence   float64              `json:"confidence"`
	Heuristic    bool                 `json:"heuristic"`
	Status       string               `json:"status"`
	DedupeKey    string               `json:"dedupe_key"`
	CreatedAt    string               `json:"created_at"`
}

type Evidence struct {
	TxHash       string `json:"tx_hash,omitempty"`
	From         string `json:"from,omitempty"`
	To           string `json:"to,omitempty"`
	ValueETH     string `json:"value_eth,omitempty"`
	Asset        string `json:"asset,omitempty"`
	Timestamp    string `json:"timestamp,omitempty"`
	Counterparty string `json:"counterparty,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

type NotificationLog struct {
	ID                string `json:"id"`
	AlertID           string `json:"alert_id"`
	UserID            string `json:"user_id"`
	Channel           string `json:"channel"`
	ProviderMessageID string `json:"provider_message_id,omitempty"`
	Status            string `json:"status"`
	Attempts          int    `json:"attempts"`
	NextRetryAt       string `json:"next_retry_at,omitempty"`
	Error             string `json:"error,omitempty"`
	CreatedAt         string `json:"created_at"`
}

type PricePoint struct {
	Timestamp string  `json:"timestamp"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    float64 `json:"volume,omitempty"`
	Source    string  `json:"source"`
}

type PriceSeriesResponse struct {
	Asset    string       `json:"asset"`
	Interval string       `json:"interval"`
	Items    []PricePoint `json:"items"`
	Source   string       `json:"source"`
	CachedAt string       `json:"cached_at"`
}

type NewsItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Source      string `json:"source"`
	PublishedAt string `json:"published_at"`
	Snippet     string `json:"snippet"`
}

type NewsResponse struct {
	Items    []NewsItem `json:"items"`
	Source   string     `json:"source"`
	CachedAt string     `json:"cached_at"`
}

type FigureNewsItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Source      string `json:"source"`
	PublishedAt string `json:"published_at"`
	Snippet     string `json:"snippet"`
	Person      string `json:"person"`
}

type FigureNewsResponse struct {
	Items    []FigureNewsItem `json:"items"`
	Source   string           `json:"source"`
	CachedAt string           `json:"cached_at"`
}

type AddressDetailResponse struct {
	Address       string               `json:"address"`
	Balance       *BalanceResponse     `json:"balance,omitempty"`
	Whale         *WhaleAccount        `json:"whale,omitempty"`
	Candidate     *CandidateAddress    `json:"candidate,omitempty"`
	Labels        []AddressLabelResult `json:"labels"`
	RiskScore     RiskScore            `json:"risk_score"`
	IsTracked     bool                 `json:"is_tracked"`
	LastCheckedAt string               `json:"last_checked_at"`
}

type RiskScore struct {
	Score         int      `json:"score"`
	Level         string   `json:"level"`
	Source        string   `json:"source"`
	Confidence    float64  `json:"confidence"`
	Heuristic     bool     `json:"heuristic"`
	Reasons       []string `json:"reasons"`
	LastCheckedAt string   `json:"last_checked_at"`
}

type AISummaryResponse struct {
	Address    string               `json:"address"`
	Summary    string               `json:"summary"`
	Heuristic  bool                 `json:"heuristic"`
	Confidence float64              `json:"confidence"`
	Evidence   []Evidence           `json:"evidence"`
	CreatedAt  string               `json:"created_at"`
	Labels     []AddressLabelResult `json:"labels"`
}
