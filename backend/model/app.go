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
