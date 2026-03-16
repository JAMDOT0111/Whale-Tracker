interface MarkedAddressesProps {
  addresses: Set<string>;
  customNames?: Record<string, string>;
  onNavigate: (address: string) => void;
  onRemove: (address: string) => void;
  onClearAll: () => void;
}

export default function MarkedAddresses({
  addresses,
  customNames,
  onNavigate,
  onRemove,
  onClearAll,
}: MarkedAddressesProps) {
  const shortenAddr = (addr: string) => addr.slice(0, 8) + '...' + addr.slice(-6);
  const names = customNames || {};

  return (
    <div className="bg-gray-900 rounded-xl p-4 border border-yellow-800/50">
      <div className="flex items-center justify-between mb-2.5">
        <h3 className="text-sm font-medium text-yellow-400 flex items-center gap-1.5">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" />
          </svg>
          已標記地址 ({addresses.size})
        </h3>
        <button onClick={onClearAll} className="text-xs text-gray-500 hover:text-gray-300 transition-colors">
          全部清除
        </button>
      </div>
      <div className="flex flex-wrap gap-1.5">
        {[...addresses].map((addr) => (
          <div
            key={addr}
            className="group flex items-center bg-gray-800 border border-yellow-700/30 rounded-lg overflow-hidden"
          >
            <button
              onClick={() => onNavigate(addr)}
              className="px-2.5 py-1.5 text-xs text-yellow-300 hover:text-white transition-colors"
              title={addr}
            >
              {names[addr] ? (
                <span className="font-medium">{names[addr]}</span>
              ) : (
                <span className="font-mono">{shortenAddr(addr)}</span>
              )}
            </button>
            <button
              onClick={() => onRemove(addr)}
              className="px-1.5 py-1.5 text-gray-600 hover:text-red-400 transition-colors border-l border-gray-700"
              title="取消標記"
            >
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M18 6L6 18M6 6l12 12" />
              </svg>
            </button>
          </div>
        ))}
      </div>
    </div>
  );
}
