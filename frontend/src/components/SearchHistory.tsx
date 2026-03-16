interface HistoryEntry {
  address: string;
  timestamp: number;
}

interface SearchHistoryProps {
  history: HistoryEntry[];
  onSelect: (address: string) => void;
  onClear: () => void;
}

export default function SearchHistory({ history, onSelect, onClear }: SearchHistoryProps) {
  const shortenAddr = (addr: string) => addr.slice(0, 10) + '...' + addr.slice(-6);
  const formatTime = (ts: number) => new Date(ts).toLocaleString();

  return (
    <div className="bg-gray-900 rounded-xl p-5 border border-gray-800">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-medium text-gray-300">最近搜尋</h3>
        <button onClick={onClear} className="text-xs text-gray-500 hover:text-gray-300 transition-colors">
          清除紀錄
        </button>
      </div>
      <div className="flex flex-wrap gap-2">
        {history.map((entry) => (
          <button
            key={entry.address + entry.timestamp}
            onClick={() => onSelect(entry.address)}
            className="group flex items-center gap-2 px-3 py-2 bg-gray-800 hover:bg-gray-700 border border-gray-700 hover:border-indigo-500/50 rounded-lg transition-all cursor-pointer"
          >
            <span className="font-mono text-xs text-gray-300 group-hover:text-white">{shortenAddr(entry.address)}</span>
            <span className="text-[10px] text-gray-600">{formatTime(entry.timestamp)}</span>
          </button>
        ))}
      </div>
    </div>
  );
}
