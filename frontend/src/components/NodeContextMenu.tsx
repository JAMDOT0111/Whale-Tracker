import { useState, useEffect, useRef } from 'react';

interface NodeContextMenuProps {
  x: number;
  y: number;
  nodeId: string;
  isMarked: boolean;
  isCenter: boolean;
  currentName: string;
  onMark: () => void;
  onRename: (name: string) => void;
  onShowTransactions: () => void;
  onClose: () => void;
}

export default function NodeContextMenu({
  x,
  y,
  nodeId,
  isMarked,
  isCenter,
  currentName,
  onMark,
  onRename,
  onShowTransactions,
  onClose,
}: NodeContextMenuProps) {
  const [renaming, setRenaming] = useState(false);
  const [name, setName] = useState(currentName);
  const menuRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (renaming && inputRef.current) {
      inputRef.current.focus();
      inputRef.current.select();
    }
  }, [renaming]);

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        onClose();
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [onClose]);

  const shortenAddr = (addr: string) => addr.slice(0, 8) + '...' + addr.slice(-4);

  const handleRenameSubmit = () => {
    onRename(name.trim());
    onClose();
  };

  const handleClearName = () => {
    onRename('');
    onClose();
  };

  return (
    <div
      ref={menuRef}
      className="fixed z-[100] bg-gray-800 border border-gray-600 rounded-lg shadow-2xl py-1 min-w-[180px]"
      style={{ left: x, top: y }}
    >
      {/* Header */}
      <div className="px-3 py-1.5 text-[10px] text-gray-500 border-b border-gray-700 font-mono">
        {shortenAddr(nodeId)}
      </div>

      {renaming ? (
        <div className="px-2 py-2">
          <input
            ref={inputRef}
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') handleRenameSubmit();
              if (e.key === 'Escape') onClose();
            }}
            placeholder="輸入名稱..."
            className="w-full px-2 py-1.5 bg-gray-900 border border-gray-600 rounded text-sm text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"
          />
          <div className="flex gap-1.5 mt-1.5">
            <button
              onClick={handleRenameSubmit}
              className="flex-1 px-2 py-1 bg-indigo-600 text-white text-xs rounded hover:bg-indigo-500 transition-colors"
            >
              確認
            </button>
            {currentName && (
              <button
                onClick={handleClearName}
                className="px-2 py-1 bg-gray-700 text-gray-300 text-xs rounded hover:bg-gray-600 transition-colors"
              >
                清除
              </button>
            )}
          </div>
        </div>
      ) : (
        <>
          <button
            onClick={() => {
              onMark();
              onClose();
            }}
            className="w-full text-left px-3 py-2 text-sm text-gray-200 hover:bg-gray-700 transition-colors flex items-center gap-2"
          >
            <svg
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill={isMarked ? 'currentColor' : 'none'}
              stroke="currentColor"
              strokeWidth="2"
              className={isMarked ? 'text-yellow-400' : 'text-gray-400'}
            >
              <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" />
            </svg>
            {isMarked ? '取消標記' : '標記'}
          </button>
          <button
            onClick={() => setRenaming(true)}
            className="w-full text-left px-3 py-2 text-sm text-gray-200 hover:bg-gray-700 transition-colors flex items-center gap-2"
          >
            <svg
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              className="text-gray-400"
            >
              <path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7" />
              <path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z" />
            </svg>
            {currentName ? '重新命名' : '命名'}
            {currentName && <span className="text-xs text-gray-500 ml-auto">{currentName}</span>}
          </button>
          {!isCenter && (
            <button
              onClick={() => {
                onShowTransactions();
                onClose();
              }}
              className="w-full text-left px-3 py-2 text-sm text-gray-200 hover:bg-gray-700 transition-colors flex items-center gap-2"
            >
              <svg
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                className="text-gray-400"
              >
                <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" />
                <polyline points="14 2 14 8 20 8" />
                <line x1="16" y1="13" x2="8" y2="13" />
                <line x1="16" y1="17" x2="8" y2="17" />
              </svg>
              顯示相關交易
            </button>
          )}
          <a
            href={`https://etherscan.io/address/${nodeId}`}
            target="_blank"
            rel="noopener noreferrer"
            onClick={onClose}
            className="w-full text-left px-3 py-2 text-sm text-gray-200 hover:bg-gray-700 transition-colors flex items-center gap-2"
          >
            <svg
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              className="text-gray-400"
            >
              <path d="M18 13v6a2 2 0 01-2 2H5a2 2 0 01-2-2V8a2 2 0 012-2h6" />
              <polyline points="15 3 21 3 21 9" />
              <line x1="10" y1="14" x2="21" y2="3" />
            </svg>
            在 Etherscan 查看
          </a>
        </>
      )}
    </div>
  );
}
