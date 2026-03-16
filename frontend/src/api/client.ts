import type { ScanRequest, ScanResponse, GraphRequest, GraphResponse, BalanceResponse } from '../types';

const BASE_URL = '/api';

async function post<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
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

export async function getTokenApprovals(address: string): Promise<unknown> {
  return post('/token-approvals', { address });
}

export async function getRiskScore(address: string): Promise<unknown> {
  return post('/risk-score', { address });
}

export async function decodeContract(txHash: string): Promise<unknown> {
  return post('/contract-decode', { tx_hash: txHash });
}
