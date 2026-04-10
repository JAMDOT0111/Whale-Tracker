import type {
  AISummaryResponse,
  AlertEvent,
  AddressDetailResponse,
  AppUser,
  BalanceResponse,
  CandidateAddress,
  CandidateListResponse,
  CandidateRebuildResponse,
  CandidateSummaryResponse,
  GraphRequest,
  GraphResponse,
  FigureNewsResponse,
  NewsResponse,
  NotificationPreference,
  NotificationStatus,
  PriceSeriesResponse,
  ScanRequest,
  ScanResponse,
  TestNotificationResponse,
  TokenApprovalResponse,
  WatchlistItem,
  WatchlistUpsertResponse,
  WhaleImportResponse,
  WhaleListResponse,
} from '../types';

const BASE_URL = '/api';
const USER_ID_KEY = 'eth-scanner-user-id';
const USER_EMAIL_KEY = 'eth-scanner-user-email';

export function startGoogleOAuthLogin() {
  window.location.href = `${BASE_URL}/auth/google/start`;
}

export function captureGoogleOAuthCallback(): { userId?: string; error?: string } {
  const url = new URL(window.location.href);
  const userId = url.searchParams.get('auth_user_id') || '';
  const error = url.searchParams.get('auth_error') || '';
  if (userId) {
    localStorage.setItem(USER_ID_KEY, userId);
    url.searchParams.delete('auth_user_id');
    url.searchParams.delete('auth');
    window.history.replaceState({}, '', `${url.pathname}${url.search}${url.hash}`);
    return { userId };
  }
  if (error) {
    url.searchParams.delete('auth_error');
    window.history.replaceState({}, '', `${url.pathname}${url.search}${url.hash}`);
    return { error };
  }
  return {};
}

function userHeaders(): HeadersInit {
  const userId = localStorage.getItem(USER_ID_KEY);
  return userId ? { 'X-User-ID': userId } : {};
}

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    headers: userHeaders(),
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Unknown error' }));
    throw new Error(err.error || `HTTP ${res.status}`);
  }

  return res.json();
}

async function post<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...userHeaders() },
    body: JSON.stringify(body),
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Unknown error' }));
    throw new Error(err.error || `HTTP ${res.status}`);
  }

  return res.json();
}

async function patch<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json', ...userHeaders() },
    body: JSON.stringify(body),
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Unknown error' }));
    throw new Error(err.error || `HTTP ${res.status}`);
  }

  return res.json();
}

async function del<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    method: 'DELETE',
    headers: userHeaders(),
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Unknown error' }));
    throw new Error(err.error || `HTTP ${res.status}`);
  }

  return res.json();
}

export async function scanAddress(req: ScanRequest): Promise<ScanResponse> {
  return post<ScanResponse>('/scan', req);
}

export async function getGraph(req: GraphRequest): Promise<GraphResponse> {
  return post<GraphResponse>('/graph', req);
}

export async function getBalance(address: string): Promise<BalanceResponse> {
  return post<BalanceResponse>('/balance', { address });
}

// TODO: 待實作的 API

export async function resolveENS(name: string): Promise<{ address: string }> {
  return post('/resolve-ens', { name });
}

export async function exportCSV(address: string): Promise<Blob> {
  const res = await fetch(`${BASE_URL}/export`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ address }),
  });
  return res.blob();
}

export async function getGasAnalytics(address: string): Promise<unknown> {
  return post('/gas-analytics', { address });
}

export async function getTokenApprovals(address: string, limit = 12): Promise<TokenApprovalResponse> {
  return post('/token-approvals', { address, limit });
}

export async function getRiskScore(address: string): Promise<unknown> {
  return post('/risk-score', { address });
}

export async function decodeContract(txHash: string): Promise<unknown> {
  return post('/contract-decode', { tx_hash: txHash });
}

export async function loginGoogle(email: string, name = ''): Promise<{ user: AppUser }> {
  const resp = await post<{ user: AppUser }>('/auth/email', { email, name });
  localStorage.setItem(USER_ID_KEY, resp.user.id);
  localStorage.setItem(USER_EMAIL_KEY, resp.user.email);
  return resp;
}

export async function getMe(): Promise<{ user: AppUser | null; notification_preferences: NotificationPreference }> {
  return get('/me');
}

export async function listWhales(params: {
  minBalanceEth?: string;
  sort?: string;
  page?: number;
  pageSize?: number;
}): Promise<WhaleListResponse> {
  const search = new URLSearchParams();
  if (params.minBalanceEth) search.set('min_balance_eth', params.minBalanceEth);
  if (params.sort) search.set('sort', params.sort);
  if (params.page) search.set('page', String(params.page));
  if (params.pageSize) search.set('page_size', String(params.pageSize));
  return get(`/whales?${search.toString()}`);
}

export async function listCandidates(params?: {
  tier?: string;
  limit?: number;
  minScore?: number;
}): Promise<CandidateListResponse> {
  const search = new URLSearchParams();
  if (params?.tier) search.set('tier', params.tier);
  if (params?.limit) search.set('limit', String(params.limit));
  if (params?.minScore) search.set('min_score', String(params.minScore));
  const query = search.toString();
  return get(`/candidates${query ? `?${query}` : ''}`);
}

export async function getCandidateSummary(): Promise<CandidateSummaryResponse> {
  return get('/candidates/summary');
}

export async function getCandidate(address: string): Promise<CandidateAddress> {
  return get(`/candidates/${address}`);
}

export async function rebuildCandidates(): Promise<CandidateRebuildResponse> {
  return post('/admin/candidates/rebuild', {});
}

export async function importWhalesCSV(csv: string): Promise<WhaleImportResponse> {
  const res = await fetch(`${BASE_URL}/admin/whales/import-etherscan-csv`, {
    method: 'POST',
    headers: { 'Content-Type': 'text/csv', 'X-Import-Filename': 'etherscan-top-accounts.csv', ...userHeaders() },
    body: csv,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Unknown error' }));
    throw new Error(err.error || `HTTP ${res.status}`);
  }
  return res.json();
}

export async function syncWhalesFromConfiguredURL(): Promise<WhaleImportResponse> {
  return post('/admin/whales/import-etherscan-url', {});
}

export async function getAddressDetail(address: string): Promise<AddressDetailResponse> {
  return get(`/addresses/${address}`);
}

export async function getAddressAISummary(address: string): Promise<AISummaryResponse> {
  return get(`/addresses/${address}/ai-summary`);
}

export async function getAddressTransactions(address: string): Promise<ScanResponse> {
  return get(`/addresses/${address}/transactions?page_size=100`);
}

export async function getAddressGraph(address: string): Promise<GraphResponse> {
  return get(`/addresses/${address}/graph`);
}

export async function getETHPrices(interval: string): Promise<PriceSeriesResponse> {
  return get(`/prices/eth/ohlc?interval=${encodeURIComponent(interval)}`);
}

export async function getETHNews(): Promise<NewsResponse> {
  return get('/news/eth');
}

export async function getCryptoFigureNews(): Promise<FigureNewsResponse> {
  return get('/news/crypto-figures');
}

export async function listWatchlists(): Promise<{ items: WatchlistItem[] }> {
  return get('/watchlists');
}

export async function upsertWatchlist(body: {
  address: string;
  alias?: string;
  min_interaction_eth?: string;
  notification_on?: boolean;
}): Promise<WatchlistUpsertResponse> {
  return post('/watchlists/confirm', body);
}

export async function deleteWatchlist(id: string): Promise<{ ok: boolean }> {
  return del(`/watchlists/${id}`);
}

export async function listAlerts(): Promise<{ items: AlertEvent[] }> {
  return get('/alerts');
}

export async function markAlertRead(id: string): Promise<{ ok: boolean }> {
  return patch(`/alerts/${id}/read`, {});
}

export async function updateNotificationPreferences(pref: NotificationPreference): Promise<NotificationPreference> {
  return post('/notification-preferences', pref);
}

export async function getNotificationStatus(): Promise<NotificationStatus> {
  return get('/notifications/status');
}

export async function sendTestNotification(email: string): Promise<TestNotificationResponse> {
  return post('/notifications/test', { email });
}
