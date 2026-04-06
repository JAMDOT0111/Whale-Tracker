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
