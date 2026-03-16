import { useState, useCallback, useEffect } from 'react';
import AddressInput from './components/AddressInput';
import TransactionPanel from './components/TransactionPanel';
import GraphView from './components/GraphView';
import SearchHistory from './components/SearchHistory';
import MarkedAddresses from './components/MarkedAddresses';
import NodeContextMenu from './components/NodeContextMenu';
import BalanceCard from './components/BalanceCard';
import TransactionTimeline from './components/TransactionTimeline';
import SankeyChart from './components/SankeyChart';
import { scanAddress, getGraph } from './api/client';
import type { Transaction, GraphResponse } from './types';

const HISTORY_KEY = 'eth-sweeper-history';
const MARKS_KEY = 'eth-sweeper-marks';
const NAMES_KEY = 'eth-sweeper-names';
const MAX_HISTORY = 10;

interface HistoryEntry {
  address: string;
  timestamp: number;
}

interface ContextMenuState {
  x: number;
  y: number;
  nodeId: string;
}

function App() {
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [graphData, setGraphData] = useState<GraphResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [currentAddress, setCurrentAddress] = useState('');
  const [history, setHistory] = useState<HistoryEntry[]>([]);
  const [panelOpen, setPanelOpen] = useState(false);
  const [markedAddresses, setMarkedAddresses] = useState<Set<string>>(new Set());
  const [customNames, setCustomNames] = useState<Record<string, string>>({});
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null);
  const [filterAddress, setFilterAddress] = useState<string | undefined>();

  const scanFromUrl = useCallback((url?: string) => {
    const params = new URLSearchParams(url || window.location.search);
    const addr = params.get('address');
    if (addr && /^0x[0-9a-fA-F]{40}$/.test(addr)) {
      return addr;
    }
    return null;
  }, []);

  useEffect(() => {
    try {
      const stored = localStorage.getItem(HISTORY_KEY);
      if (stored) setHistory(JSON.parse(stored));
    } catch {
      /* ignore */
    }
    try {
      const stored = localStorage.getItem(MARKS_KEY);
      if (stored) setMarkedAddresses(new Set(JSON.parse(stored)));
    } catch {
      /* ignore */
    }
    try {
      const stored = localStorage.getItem(NAMES_KEY);
      if (stored) setCustomNames(JSON.parse(stored));
    } catch {
      /* ignore */
    }

    const addrFromUrl = scanFromUrl();
    if (addrFromUrl) {
      handleScanInternal(addrFromUrl, false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    const onPopState = () => {
      const addr = scanFromUrl();
      if (addr) {
        handleScanInternal(addr, false);
      }
    };
    window.addEventListener('popstate', onPopState);
    return () => window.removeEventListener('popstate', onPopState);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const saveMarks = useCallback((marks: Set<string>) => {
    setMarkedAddresses(marks);
    localStorage.setItem(MARKS_KEY, JSON.stringify([...marks]));
  }, []);

  const toggleMark = useCallback((address: string) => {
    setMarkedAddresses((prev) => {
      const next = new Set(prev);
      if (next.has(address)) {
        next.delete(address);
      } else {
        next.add(address);
      }
      localStorage.setItem(MARKS_KEY, JSON.stringify([...next]));
      return next;
    });
  }, []);

  const setCustomName = useCallback((address: string, name: string) => {
    setCustomNames((prev) => {
      const next = { ...prev };
      if (name) {
        next[address] = name;
      } else {
        delete next[address];
      }
      localStorage.setItem(NAMES_KEY, JSON.stringify(next));
      return next;
    });
  }, []);

  const saveHistory = useCallback((address: string) => {
    setHistory((prev) => {
      const entry: HistoryEntry = { address, timestamp: Date.now() };
      const updated = [entry, ...prev.filter((h) => h.address !== address)].slice(0, MAX_HISTORY);
      localStorage.setItem(HISTORY_KEY, JSON.stringify(updated));
      return updated;
    });
  }, []);

  const handleScanInternal = useCallback(
    async (address: string, pushHistory: boolean = true) => {
      setLoading(true);
      setError(null);
      setCurrentAddress(address);
      setTransactions([]);
      setGraphData(null);
      setPanelOpen(false);
      setContextMenu(null);
      saveHistory(address);

      if (pushHistory) {
        const url = `${window.location.pathname}?address=${address}`;
        window.history.pushState({ address }, '', url);
      }

      try {
        const scanResult = await scanAddress({ address, page_size: 1000 });
        setTransactions(scanResult.transactions);

        const graphResult = await getGraph({ address });
        setGraphData(graphResult);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
      } finally {
        setLoading(false);
      }
    },
    [saveHistory],
  );

  const handleScan = useCallback(
    (address: string) => {
      handleScanInternal(address, true);
    },
    [handleScanInternal],
  );

  const handleNodeClick = useCallback(
    (address: string) => {
      handleScanInternal(address, true);
    },
    [handleScanInternal],
  );

  const hasResults = !loading && (transactions.length > 0 || graphData);

  return (
    <div className="min-h-screen bg-gray-950 text-gray-100 flex flex-col">
      <div className="flex-1 flex flex-col">
        {/* Top section */}
        <div className="max-w-7xl w-full mx-auto px-4 pt-8 pb-4 space-y-6">
          {/* Header */}
          <div className="text-center space-y-3">
            <div className="flex items-center justify-center gap-3">
              <svg width="40" height="40" viewBox="0 0 256 417" xmlns="http://www.w3.org/2000/svg" className="shrink-0">
                <path fill="#8C8C8C" d="M127.961 0l-2.795 9.5v275.668l2.795 2.79 127.962-75.638z" />
                <path fill="#fff" d="M127.962 0L0 212.32l127.962 75.639V154.158z" />
                <path fill="#8C8C8C" d="M127.961 312.187l-1.575 1.92v98.199l1.575 4.601L256 236.587z" />
                <path fill="#fff" d="M127.962 416.905v-104.72L0 236.585z" />
                <path fill="#6366f1" d="M127.961 287.958l127.96-75.637-127.96-58.162z" />
                <path fill="#818cf8" d="M0 212.32l127.96 75.638v-133.8z" />
              </svg>
              <h1 className="text-3xl font-bold text-white tracking-tight">ETH Sweeper</h1>
            </div>
            <p className="text-gray-400">掃描任意 Ethereum 錢包地址，查看交易紀錄與地址關係圖</p>
          </div>

          {/* Input */}
          <div className="bg-gray-900 rounded-xl p-6 border border-gray-800">
            <AddressInput onScan={handleScan} loading={loading} initialAddress={currentAddress} />
          </div>

          {/* Current address indicator + buttons */}
          {currentAddress && !loading && (
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2 text-sm text-gray-400">
                <span>目前查看：</span>
                <span className="font-mono text-indigo-400">{currentAddress}</span>
                <a
                  href={`https://etherscan.io/address/${currentAddress}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-gray-500 hover:text-white transition-colors"
                >
                  Etherscan ↗
                </a>
              </div>
              {transactions.length > 0 && (
                <button
                  onClick={() => setPanelOpen(!panelOpen)}
                  className="flex items-center gap-1.5 px-3 py-1.5 text-sm bg-gray-800 text-gray-300 rounded-lg hover:bg-gray-700 border border-gray-700 transition-colors"
                >
                  <span>{panelOpen ? '隱藏' : '顯示'}交易列表</span>
                  <span className="text-xs text-gray-500">({transactions.length})</span>
                </button>
              )}
            </div>
          )}

          {/* Balance + Charts */}
          {currentAddress && !loading && hasResults && (
            <>
              <BalanceCard address={currentAddress} />
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                <TransactionTimeline transactions={transactions} centerAddress={currentAddress} />
                <SankeyChart transactions={transactions} centerAddress={currentAddress} />
              </div>
            </>
          )}

          {/* Marked Addresses */}
          {markedAddresses.size > 0 && (
            <MarkedAddresses
              addresses={markedAddresses}
              customNames={customNames}
              onNavigate={handleScan}
              onRemove={toggleMark}
              onClearAll={() => saveMarks(new Set())}
            />
          )}

          {/* Search History */}
          {!hasResults && !loading && history.length > 0 && (
            <SearchHistory
              history={history}
              onSelect={(addr) => handleScan(addr)}
              onClear={() => {
                setHistory([]);
                localStorage.removeItem(HISTORY_KEY);
              }}
            />
          )}

          {/* Error */}
          {error && (
            <div className="px-4 py-3 bg-red-900/30 border border-red-700 rounded-lg text-red-400 text-sm">{error}</div>
          )}

          {/* Loading */}
          {loading && (
            <div className="flex items-center justify-center py-12">
              <div className="flex items-center gap-3 text-gray-400">
                <svg className="animate-spin h-5 w-5" viewBox="0 0 24 24" fill="none">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
                正在掃描交易資料，請稍候...
              </div>
            </div>
          )}
        </div>

        {/* Graph + Side Panel */}
        {hasResults && (
          <div className="flex-1 flex relative px-4 pb-4 max-w-7xl w-full mx-auto">
            <div className={`flex-1 transition-all duration-300 ${panelOpen ? 'mr-[420px]' : ''}`}>
              <div className="bg-gray-900 rounded-xl p-6 border border-gray-800 h-full">
                <GraphView
                  data={graphData}
                  onNodeClick={handleNodeClick}
                  onNodeRightClick={(nodeId, x, y) => setContextMenu({ nodeId, x, y })}
                  markedAddresses={markedAddresses}
                  customNames={customNames}
                />
              </div>
            </div>

            <TransactionPanel
              open={panelOpen}
              transactions={transactions}
              centerAddress={currentAddress}
              filterAddress={filterAddress}
              onClose={() => {
                setPanelOpen(false);
                setFilterAddress(undefined);
              }}
              onClearFilter={() => setFilterAddress(undefined)}
            />
          </div>
        )}
      </div>

      {/* Context Menu */}
      {contextMenu && (
        <NodeContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          nodeId={contextMenu.nodeId}
          isMarked={markedAddresses.has(contextMenu.nodeId)}
          isCenter={contextMenu.nodeId.toLowerCase() === currentAddress.toLowerCase()}
          currentName={customNames[contextMenu.nodeId] || ''}
          onMark={() => toggleMark(contextMenu.nodeId)}
          onRename={(name) => setCustomName(contextMenu.nodeId, name)}
          onShowTransactions={() => {
            setFilterAddress(contextMenu.nodeId);
            setPanelOpen(true);
          }}
          onClose={() => setContextMenu(null)}
        />
      )}

      {/* Footer */}
      <footer className="border-t border-gray-800 bg-gray-900/50 mt-auto">
        <div className="max-w-7xl mx-auto px-4 py-4 flex flex-col sm:flex-row items-center justify-between gap-2 text-xs text-gray-500">
          <span>ETH Sweeper — Ethereum Address Scanner</span>
          <div className="flex items-center gap-4">
            <span>
              Data by{' '}
              <a
                href="https://etherscan.io"
                target="_blank"
                rel="noopener noreferrer"
                className="text-gray-400 hover:text-white transition-colors"
              >
                Etherscan API
              </a>
            </span>
            <span className="text-gray-700">|</span>
            <span>Built with Go + React + Cytoscape.js</span>
          </div>
        </div>
      </footer>
    </div>
  );
}

export default App;
