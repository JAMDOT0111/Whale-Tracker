import { useEffect, useState } from 'react';
import { getBalance } from '../api/client';
import type { BalanceResponse } from '../types';

interface BalanceCardProps {
  address: string;
}

export default function BalanceCard({ address }: BalanceCardProps) {
  const [balance, setBalance] = useState<BalanceResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [showAllTokens, setShowAllTokens] = useState(false);

  useEffect(() => {
    if (!address) return;
    let cancelled = false;
    setLoading(true);
    setBalance(null);
    getBalance(address)
      .then((data) => {
        if (!cancelled) setBalance(data);
      })
      .catch(() => {})
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [address]);

  if (loading) {
    return (
      <div className="bg-gray-900 rounded-xl p-4 border border-gray-800 animate-pulse">
        <div className="h-4 bg-gray-800 rounded w-24 mb-2" />
        <div className="h-6 bg-gray-800 rounded w-32" />
      </div>
    );
  }

  if (!balance) return null;

  const tokens = balance.tokens || [];
  const displayTokens = showAllTokens ? tokens : tokens.slice(0, 5);

  return (
    <div className="bg-gray-900 rounded-xl p-4 border border-gray-800">
      <div className="flex items-start justify-between">
        <div>
          <p className="text-xs text-gray-500 mb-1">ETH 餘額</p>
          <p className="text-xl font-bold text-white font-mono">{balance.eth_balance} ETH</p>
        </div>
        {tokens.length > 0 && (
          <div className="text-right">
            <p className="text-xs text-gray-500 mb-1">代幣持倉</p>
            <p className="text-sm text-gray-300">{tokens.length} 種</p>
          </div>
        )}
      </div>

      {tokens.length > 0 && (
        <div className="mt-3 pt-3 border-t border-gray-800">
          <div className="flex flex-wrap gap-1.5">
            {displayTokens.map((t, i) => (
              <span
                key={t.symbol + i}
                className="px-2 py-1 bg-gray-800 rounded text-xs text-gray-300"
                title={`${t.name}: ${t.balance}`}
              >
                <span className="text-white font-mono">
                  {Number(t.balance) > 1000 ? Number(t.balance).toLocaleString() : t.balance}
                </span>{' '}
                <span className="text-gray-500">{t.symbol}</span>
              </span>
            ))}
          </div>
          {tokens.length > 5 && (
            <button
              onClick={() => setShowAllTokens(!showAllTokens)}
              className="mt-2 text-xs text-gray-500 hover:text-gray-300 transition-colors"
            >
              {showAllTokens ? '收起' : `顯示全部 ${tokens.length} 種代幣`}
            </button>
          )}
        </div>
      )}
    </div>
  );
}
