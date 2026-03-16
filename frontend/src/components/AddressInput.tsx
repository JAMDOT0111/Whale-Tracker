import { useState, useMemo } from 'react';

interface AddressInputProps {
  onScan: (address: string) => void;
  loading: boolean;
  initialAddress?: string;
}

export default function AddressInput({ onScan, loading, initialAddress }: AddressInputProps) {
  const [localInput, setLocalInput] = useState('');
  const address = useMemo(() => localInput || initialAddress || '', [localInput, initialAddress]);

  const isValidAddress = /^0x[0-9a-fA-F]{40}$/.test(address);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (isValidAddress) {
      onScan(address);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="flex gap-3">
        <input
          type="text"
          value={address}
          onChange={(e) => setLocalInput(e.target.value.trim())}
          placeholder="輸入 ETH 錢包地址 (0x...)"
          className="flex-1 px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-white font-mono text-sm placeholder-gray-500 focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-colors"
        />
        <button
          type="submit"
          disabled={!isValidAddress || loading}
          className="px-6 py-3 bg-indigo-600 text-white rounded-lg font-medium text-sm hover:bg-indigo-500 disabled:opacity-40 disabled:cursor-not-allowed transition-colors cursor-pointer"
        >
          {loading ? '掃描中...' : '掃描'}
        </button>
      </div>

      {address && !isValidAddress && <p className="text-red-400 text-sm">請輸入有效的 ETH 地址（0x 開頭，42 字元）</p>}
    </form>
  );
}
