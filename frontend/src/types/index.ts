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
