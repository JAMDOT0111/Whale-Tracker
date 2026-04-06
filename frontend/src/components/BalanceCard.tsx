import { useEffect, useState } from 'react';
import { getBalance } from '../api/client';
import type { BalanceResponse } from '../types';

interface BalanceCardProps {
  address: string;
}

interface BalanceState {
  data: BalanceResponse | null;
  loading: boolean;
}

export default function BalanceCard({ address }: BalanceCardProps) {
  const [state, setState] = useState<BalanceState>({ data: null, loading: false });
  const [showAllTokens, setShowAllTokens] = useState(false);

  useEffect(() => {
    if (!address) return;
    let cancelled = false;

    void (async () => {
      setState({ data: null, loading: true });
      try {
        const data = await getBalance(address);
        if (!cancelled) setState({ data, loading: false });
      } catch {
        if (!cancelled) setState({ data: null, loading: false });
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [address]);

  if (state.loading) {
    return (
      <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
        <div className="h-4 w-24 rounded bg-gray-800" />
        <div className="mt-2 h-6 w-32 rounded bg-gray-800" />
      </div>
    );
  }

  if (!state.data) return null;

  const tokens = state.data.tokens || [];
  const displayTokens = showAllTokens ? tokens : tokens.slice(0, 5);

  return (
    <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
      <div className="flex items-start justify-between">
        <div>
          <p className="mb-1 text-xs text-gray-500">ETH 餘額</p>
          <p className="font-mono text-xl font-bold text-white">{state.data.eth_balance} ETH</p>
        </div>
        {tokens.length > 0 && (
          <div className="text-right">
            <p className="mb-1 text-xs text-gray-500">代幣持倉</p>
            <p className="text-sm text-gray-300">{tokens.length} 種</p>
          </div>
        )}
      </div>

      {tokens.length > 0 && (
        <div className="mt-3 border-t border-gray-800 pt-3">
          <div className="flex flex-wrap gap-1.5">
            {displayTokens.map((token, index) => (
              <span
                key={token.symbol + index}
                className="rounded bg-gray-800 px-2 py-1 text-xs text-gray-300"
                title={`${token.name}: ${token.balance}`}
              >
                <span className="font-mono text-white">
                  {Number(token.balance) > 1000 ? Number(token.balance).toLocaleString() : token.balance}
                </span>{' '}
                <span className="text-gray-500">{token.symbol}</span>
              </span>
            ))}
          </div>
          {tokens.length > 5 && (
            <button
              onClick={() => setShowAllTokens(!showAllTokens)}
              className="mt-2 text-xs text-gray-500 transition-colors hover:text-gray-300"
            >
              {showAllTokens ? '收合' : `顯示全部 ${tokens.length} 種代幣`}
            </button>
          )}
        </div>
      )}
    </div>
  );
}
