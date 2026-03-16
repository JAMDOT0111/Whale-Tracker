import { useState, useMemo } from 'react';
import type { Transaction } from '../types';

interface TransactionPanelProps {
  open: boolean;
  transactions: Transaction[];
  centerAddress: string;
  filterAddress?: string;
  onClose: () => void;
  onClearFilter?: () => void;
}

const categoryColors: Record<string, string> = {
  external: 'bg-blue-900/40 text-blue-300',
  internal: 'bg-purple-900/40 text-purple-300',
  erc20: 'bg-green-900/40 text-green-300',
  erc721: 'bg-orange-900/40 text-orange-300',
};

type FilterCategory = 'all' | 'external' | 'internal' | 'erc20' | 'erc721';

export default function TransactionPanel({
  open,
  transactions,
  centerAddress,
  filterAddress,
  onClose,
  onClearFilter,
}: TransactionPanelProps) {
  const [filter, setFilter] = useState<FilterCategory>('all');
  const center = centerAddress.toLowerCase();
  const counterparty = filterAddress?.toLowerCase();

  const filtered = useMemo(() => {
    let result = transactions;
    if (counterparty) {
      result = result.filter((tx) => tx.from.toLowerCase() === counterparty || tx.to.toLowerCase() === counterparty);
    }
    if (filter !== 'all') {
      result = result.filter((tx) => tx.category === filter);
    }
    return result;
  }, [transactions, filter, counterparty]);

  const shortenHash = (hash: string) => hash.slice(0, 8) + '...' + hash.slice(-4);
  const shortenAddr = (addr: string) => addr.slice(0, 6) + '...' + addr.slice(-4);

  return (
    <div
      className={`fixed top-0 right-0 h-full w-[420px] bg-gray-900 border-l border-gray-800 shadow-2xl z-50 flex flex-col transition-transform duration-300 ${
        open ? 'translate-x-0' : 'translate-x-full'
      }`}
    >
      {/* Panel Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-800">
        <div>
          <h3 className="text-sm font-semibold text-white">
            交易列表
            <span className="text-gray-400 font-normal ml-1.5">({filtered.length})</span>
          </h3>
          {counterparty && (
            <div className="flex items-center gap-1.5 mt-1">
              <span className="text-[10px] text-gray-500">篩選：</span>
              <span className="text-[10px] font-mono text-indigo-400">{shortenAddr(counterparty)}</span>
              <button onClick={onClearFilter} className="text-[10px] text-gray-500 hover:text-white transition-colors">
                ✕ 清除
              </button>
            </div>
          )}
        </div>
        <button onClick={onClose} className="p-1 text-gray-500 hover:text-white transition-colors shrink-0">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M18 6L6 18M6 6l12 12" />
          </svg>
        </button>
      </div>

      {/* Filters */}
      <div className="flex gap-1 px-4 py-2 border-b border-gray-800 overflow-x-auto">
        {(['all', 'external', 'internal', 'erc20', 'erc721'] as FilterCategory[]).map((cat) => (
          <button
            key={cat}
            onClick={() => setFilter(cat)}
            className={`px-2.5 py-1 rounded text-xs whitespace-nowrap transition-colors ${
              filter === cat ? 'bg-indigo-600 text-white' : 'bg-gray-800 text-gray-400 hover:text-white'
            }`}
          >
            {cat === 'all' ? '全部' : cat}
          </button>
        ))}
      </div>

      {/* Transaction List */}
      <div className="flex-1 overflow-y-auto">
        {filtered.length === 0 ? (
          <div className="text-gray-500 text-sm text-center py-8">無交易紀錄</div>
        ) : (
          <ul className="divide-y divide-gray-800">
            {filtered.map((tx, i) => {
              const isOut = tx.from.toLowerCase() === center;
              return (
                <li key={tx.hash + tx.category + i} className="px-4 py-3 hover:bg-gray-800/50 transition-colors">
                  {/* Row 1: hash + direction + category */}
                  <div className="flex items-center justify-between mb-1.5">
                    <a
                      href={`https://etherscan.io/tx/${tx.hash}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-indigo-400 hover:text-indigo-300 font-mono text-xs"
                    >
                      {shortenHash(tx.hash)}
                    </a>
                    <div className="flex items-center gap-1.5">
                      <span
                        className={`px-1.5 py-0.5 rounded text-[10px] font-medium ${isOut ? 'bg-red-900/40 text-red-300' : 'bg-emerald-900/40 text-emerald-300'}`}
                      >
                        {isOut ? 'OUT' : 'IN'}
                      </span>
                      <span
                        className={`px-1.5 py-0.5 rounded text-[10px] ${categoryColors[tx.category] || 'bg-gray-700 text-gray-300'}`}
                      >
                        {tx.category}
                      </span>
                    </div>
                  </div>

                  {/* Row 2: from → to */}
                  <div className="flex items-center gap-1 text-xs text-gray-400 mb-1">
                    <span className={`font-mono ${!isOut ? 'text-gray-300' : ''}`}>{shortenAddr(tx.from)}</span>
                    <span className="text-gray-600">→</span>
                    <span className={`font-mono ${isOut ? 'text-gray-300' : ''}`}>{shortenAddr(tx.to)}</span>
                  </div>

                  {/* Row 3: value + asset + time */}
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-white font-mono">
                      {tx.value} <span className="text-gray-500">{tx.asset || ''}</span>
                    </span>
                    <span className="text-gray-600 text-[10px]">
                      {tx.timestamp ? new Date(tx.timestamp).toLocaleString() : ''}
                    </span>
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </div>
  );
}
