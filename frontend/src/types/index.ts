export interface Transaction {
  hash: string;
  from: string;
  to: string;
  value: string;
  asset: string;
  category: string;
  block_number: string;
  timestamp: string;
}

export interface ScanRequest {
  address: string;
  page_size?: number;
  page_key?: string;
}

export interface ScanResponse {
  transactions: Transaction[];
  page_key: string;
  total: number;
}

export interface GraphRequest {
  address: string;
}

export interface GraphNode {
  id: string;
  label: string;
  is_center: boolean;
  is_contract: boolean;
  tag?: string;
  tag_name?: string;
  tx_count: number;
}

export interface GraphEdge {
  source: string;
  target: string;
  value: string;
  tx_count: number;
}

export interface GraphResponse {
  nodes: GraphNode[];
  edges: GraphEdge[];
}

export interface TokenBalance {
  symbol: string;
  name: string;
  balance: string;
}

export interface BalanceResponse {
  eth_balance: string;
  tokens: TokenBalance[];
}

export interface AddressLabelResult {
  category: string;
  name: string;
  source: string;
  confidence: number;
  last_checked_at: string;
  evidence_ref?: string;
  heuristic: boolean;
}

export interface WhaleAccount {
  rank: number;
  address: string;
  name_tag?: string;
  balance_eth: string;
  balance_wei?: string;
  percentage?: string;
  txn_count: number;
  labels: AddressLabelResult[];
  is_tracked: boolean;
  updated_at: string;
  source: string;
  confidence: number;
  evidence_ref?: string;
}

export interface WhaleListResponse {
  items: WhaleAccount[];
  page: number;
  page_size: number;
  total: number;
  has_next: boolean;
  snapshot_at?: string;
  source: string;
  limit_notice: string;
}

export interface CandidateScoreBreakdown {
  balance: number;
  historical: number;
  flow: number;
  activity: number;
  protocol: number;
  anomaly: number;
  total: number;
}

export interface CandidateActivityStats {
  activity_loaded: boolean;
  activity_source: string;
  last_activity_at?: string;
  last_enriched_at?: string;
  tx_count_24h: number;
  tx_count_7d: number;
  tx_count_30d: number;
  inflow_eth_24h: string;
  outflow_eth_24h: string;
  netflow_eth_24h: string;
  inflow_eth_7d: string;
  outflow_eth_7d: string;
  netflow_eth_7d: string;
  largest_in_tx_eth_7d: string;
  largest_out_tx_eth_7d: string;
  largest_tx_eth_7d: string;
  active_days_7d: number;
  dormancy_days: number;
  is_reactivated: boolean;
  protocol_interactions_7d: number;
  protocol_types_7d: number;
}

export interface CandidateScanCursor {
  address: string;
  last_scanned_block: number;
  last_scanned_at?: string;
  last_activity_at?: string;
}

export interface CandidateBuildState {
  status: string;
  mode: string;
  message?: string;
  processed: number;
  total: number;
  started_at?: string;
  finished_at?: string;
  error?: string;
}

export interface CandidateAddress {
  address: string;
  name_tag?: string;
  rank: number;
  balance_eth: string;
  txn_count: number;
  labels: AddressLabelResult[];
  base_pass: boolean;
  event_pass: boolean;
  score: number;
  priority_tier: string;
  selected_for_review: boolean;
  reasons: string[];
  score_breakdown: CandidateScoreBreakdown;
  activity: CandidateActivityStats;
  updated_at: string;
}

export interface CandidateListResponse {
  items: CandidateAddress[];
  total: number;
  available_total: number;
  review_total: number;
  watch_total: number;
  refreshed_at?: string;
  last_build_mode: string;
  activity_enriched_count: number;
  scan_limit: number;
  limit_notice: string;
}

export interface CandidateSummaryResponse {
  available_total: number;
  review_total: number;
  watch_total: number;
  refreshed_at?: string;
  last_build_mode: string;
  activity_enriched_count: number;
  scan_limit: number;
  full_snapshot_ready: boolean;
  last_full_build_at?: string;
  last_incremental_at?: string;
  build: CandidateBuildState;
}

export interface CandidateRebuildResponse {
  ok: boolean;
  started: boolean;
  message: string;
  summary: CandidateSummaryResponse;
}

export interface TokenApprovalItem {
  token_address: string;
  token_name?: string;
  token_symbol?: string;
  token_decimals: number;
  spender: string;
  spender_label?: string;
  approval_value: string;
  approval_display?: string;
  approval_type: string;
  risk_level: string;
  tx_hash: string;
  timestamp: string;
}

export interface TokenApprovalResponse {
  address: string;
  items: TokenApprovalItem[];
  scanned_at: string;
  source: string;
  candidate_only: boolean;
  candidate_tier?: string;
  limit_applied: number;
  review_required: boolean;
}

export interface WhaleImportResponse {
  import_id: string;
  imported: number;
  skipped: number;
  source: string;
  imported_at: string;
}

export interface WatchlistItem {
  id: string;
  user_id: string;
  address: string;
  alias?: string;
  min_interaction_eth: string;
  notification_on: boolean;
  labels: AddressLabelResult[];
  created_at: string;
  updated_at: string;
}

export interface AlertEvent {
  id: string;
  user_id: string;
  address: string;
  type: string;
  severity: string;
  threshold_eth: string;
  title: string;
  description: string;
  evidence: Evidence[];
  labels: AddressLabelResult[];
  confidence: number;
  heuristic: boolean;
  status: string;
  dedupe_key: string;
  created_at: string;
}

export interface Evidence {
  tx_hash?: string;
  from?: string;
  to?: string;
  value_eth?: string;
  asset?: string;
  timestamp?: string;
  counterparty?: string;
  reason?: string;
}

export interface PricePoint {
  timestamp: string;
  open: number;
  high: number;
  low: number;
  close: number;
  volume?: number;
  source: string;
}

export interface PriceSeriesResponse {
  asset: string;
  interval: string;
  items: PricePoint[];
  source: string;
  cached_at: string;
}

export interface NewsItem {
  id: string;
  title: string;
  url: string;
  source: string;
  published_at: string;
  snippet: string;
}

export interface NewsResponse {
  items: NewsItem[];
  source: string;
  cached_at: string;
}

export interface FigureNewsItem {
  id: string;
  title: string;
  url: string;
  source: string;
  published_at: string;
  snippet: string;
  person: string;
}

export interface FigureNewsResponse {
  items: FigureNewsItem[];
  source: string;
  cached_at: string;
}

export interface RiskScore {
  score: number;
  level: string;
  source: string;
  confidence: number;
  heuristic: boolean;
  reasons: string[];
  last_checked_at: string;
}

export interface AddressDetailResponse {
  address: string;
  balance?: BalanceResponse;
  whale?: WhaleAccount;
  candidate?: CandidateAddress;
  labels: AddressLabelResult[];
  risk_score: RiskScore;
  is_tracked: boolean;
  last_checked_at: string;
}

export interface AISummaryResponse {
  address: string;
  summary: string;
  heuristic: boolean;
  confidence: number;
  evidence: Evidence[];
  created_at: string;
  labels: AddressLabelResult[];
}

export interface AppUser {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
  created_at: string;
}

export interface NotificationPreference {
  user_id: string;
  email: string;
  gmail_enabled: boolean;
  min_severity: string;
  updated_at: string;
}

export interface NotificationStatus {
  configured: boolean;
  provider: string;
  dry_run: boolean;
  from: string;
}

export interface TestNotificationResponse {
  ok?: boolean;
  log: {
    id: string;
    alert_id: string;
    user_id: string;
    channel: string;
    provider_message_id?: string;
    status: string;
    attempts: number;
    next_retry_at?: string;
    error?: string;
    created_at: string;
  };
  notification_status: NotificationStatus;
}

export interface WatchlistUpsertResponse {
  item: WatchlistItem;
  notification_log?: TestNotificationResponse['log'];
  notification_status?: NotificationStatus;
  notification_error?: string;
}
