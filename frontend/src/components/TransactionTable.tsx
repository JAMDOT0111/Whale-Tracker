import { useState, useMemo } from 'react';
import type { Transaction } from '../types';

interface TransactionTableProps {
  transactions: Transaction[];
  centerAddress: string;
}

type SortField = 'timestamp' | 'value' | 'asset' | 'category';
type SortDir = 'asc' | 'desc';

const categoryColors: Record<string, string> = {
  external: 'bg-blue-900/40 text-blue-300',
  internal: 'bg-purple-900/40 text-purple-300',
  erc20: 'bg-green-900/40 text-green-300',
  erc721: 'bg-orange-900/40 text-orange-300',
};

export default function TransactionTable({ transactions, centerAddress }: TransactionTableProps) {
  const [sortField, setSortField] = useState<SortField>('timestamp');
  const [sortDir, setSortDir] = useState<SortDir>('desc');
  const [page, setPage] = useState(0);
  const pageSize = 20;

  const sorted = useMemo(() => {
    const arr = [...transactions];
    arr.sort((a, b) => {
      let cmp = 0;
      switch (sortField) {
        case 'timestamp':
          cmp = a.timestamp.localeCompare(b.timestamp);
          break;
        case 'value':
          cmp = parseFloat(a.value || '0') - parseFloat(b.value || '0');
          break;
        case 'asset':
          cmp = a.asset.localeCompare(b.asset);
          break;
        case 'category':
          cmp = a.category.localeCompare(b.category);
          break;
      }
      return sortDir === 'asc' ? cmp : -cmp;
    });
    return arr;
  }, [transactions, sortField, sortDir]);

  const paged = sorted.slice(page * pageSize, (page + 1) * pageSize);
  const totalPages = Math.ceil(sorted.length / pageSize);

  const toggleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDir(sortDir === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortDir('desc');
    }
    setPage(0);
  };

  const sortIcon = (field: SortField) => {
    if (sortField !== field) return '↕';
    return sortDir === 'asc' ? '↑' : '↓';
  };

  const shortenHash = (hash: string) => hash.slice(0, 10) + '...' + hash.slice(-6);
  const shortenAddr = (addr: string) => addr.slice(0, 8) + '...' + addr.slice(-6);

  const center = centerAddress.toLowerCase();

  if (transactions.length === 0) {
    return <div className="text-gray-500 text-center py-8">尚無交易資料。請輸入地址並點擊掃描。</div>;
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-white">
          交易列表 <span className="text-gray-400 font-normal text-sm">({transactions.length} 筆)</span>
        </h2>
      </div>

      <div className="overflow-x-auto rounded-lg border border-gray-700">
        <table className="w-full text-sm text-left">
          <thead className="bg-gray-800 text-gray-400 text-xs uppercase">
            <tr>
              <th className="px-4 py-3">Tx Hash</th>
              <th className="px-4 py-3">方向</th>
              <th className="px-4 py-3">From</th>
              <th className="px-4 py-3">To</th>
              <th className="px-4 py-3 cursor-pointer select-none" onClick={() => toggleSort('value')}>
                金額 {sortIcon('value')}
              </th>
              <th className="px-4 py-3 cursor-pointer select-none" onClick={() => toggleSort('asset')}>
                資產 {sortIcon('asset')}
              </th>
              <th className="px-4 py-3 cursor-pointer select-none" onClick={() => toggleSort('category')}>
                類型 {sortIcon('category')}
              </th>
              <th className="px-4 py-3 cursor-pointer select-none" onClick={() => toggleSort('timestamp')}>
                時間 {sortIcon('timestamp')}
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-700">
            {paged.map((tx) => {
              const isOut = tx.from.toLowerCase() === center;
              return (
                <tr key={tx.hash + tx.from + tx.to + tx.category} className="hover:bg-gray-800/50 transition-colors">
                  <td className="px-4 py-2.5">
                    <a
                      href={`https://etherscan.io/tx/${tx.hash}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-indigo-400 hover:text-indigo-300 font-mono text-xs"
                    >
                      {shortenHash(tx.hash)}
                    </a>
                  </td>
                  <td className="px-4 py-2.5">
                    <span
                      className={`inline-block px-2 py-0.5 rounded text-xs font-medium ${isOut ? 'bg-red-900/40 text-red-300' : 'bg-emerald-900/40 text-emerald-300'}`}
                    >
                      {isOut ? 'OUT' : 'IN'}
                    </span>
                  </td>
                  <td className="px-4 py-2.5 font-mono text-xs text-gray-300">{shortenAddr(tx.from)}</td>
                  <td className="px-4 py-2.5 font-mono text-xs text-gray-300">{shortenAddr(tx.to)}</td>
                  <td className="px-4 py-2.5 text-white font-mono text-xs">{tx.value}</td>
                  <td className="px-4 py-2.5 text-gray-300 text-xs">{tx.asset || '-'}</td>
                  <td className="px-4 py-2.5">
                    <span
                      className={`inline-block px-2 py-0.5 rounded text-xs ${categoryColors[tx.category] || 'bg-gray-700 text-gray-300'}`}
                    >
                      {tx.category}
                    </span>
                  </td>
                  <td className="px-4 py-2.5 text-gray-400 text-xs whitespace-nowrap">
                    {tx.timestamp ? new Date(tx.timestamp).toLocaleString() : '-'}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2 pt-2">
          <button
            onClick={() => setPage(Math.max(0, page - 1))}
            disabled={page === 0}
            className="px-3 py-1.5 text-sm bg-gray-800 text-gray-300 rounded hover:bg-gray-700 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          >
            上一頁
          </button>
          <span className="text-gray-400 text-sm">
            {page + 1} / {totalPages}
          </span>
          <button
            onClick={() => setPage(Math.min(totalPages - 1, page + 1))}
            disabled={page >= totalPages - 1}
            className="px-3 py-1.5 text-sm bg-gray-800 text-gray-300 rounded hover:bg-gray-700 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          >
            下一頁
          </button>
        </div>
      )}
    </div>
  );
}
