import { useCallback, useEffect, useRef, useState } from 'react';
import { Area, AreaChart, Bar, BarChart, CartesianGrid, Cell, Line, LineChart, Pie, PieChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import AddressInput from './components/AddressInput';
import SankeyChart from './components/SankeyChart';
import {
  captureGoogleOAuthCallback,
  getAddressAISummary,
  getAddressDetail,
  getAddressGraph,
  getAddressTransactions,
  getCryptoFigureNews,
  getETHNews,
  getETHPrices,
  getMe,
  getNotificationStatus,
  updateNotificationPreferences,
  listAlerts,
  listWatchlists,
  listWhales,
  sendTestNotification,
  startGoogleOAuthLogin,
  syncWhalesFromConfiguredURL,
  upsertWatchlist,
  deleteWatchlist,
} from './api/client';
import type {
  AISummaryResponse,
  AlertEvent,
  AppUser,
  AddressDetailResponse,
  FigureNewsItem,
  GraphNode,
  GraphResponse,
  NewsItem,
  NotificationStatus,
  NotificationPreference,
  PricePoint,
  Transaction,
  WatchlistItem,
  WhaleAccount,
  WhaleListResponse,
} from './types';

type SortKey = 'balance_desc' | 'balance_asc' | 'rank_asc' | 'rank_desc';

interface ThresholdParseResult {
  value: string;
  message: string;
}

const DEFAULT_THRESHOLD = '> 1000 ETH';
const DEFAULT_WATCH_THRESHOLD = '500';
const PAGE_SIZE = 25;
const GRAPH_TIMEOUT_MS = 90000;
const GRAPH_RETRY_TIMES = 2;
const GRAPH_RETRY_DELAY_MS = 1500;
const CATEGORY_COLORS = ['#34d399', '#60a5fa', '#f59e0b', '#f472b6', '#a78bfa', '#f87171', '#22d3ee'];

function App() {
  const [threshold, setThreshold] = useState(DEFAULT_THRESHOLD);
  const [thresholdNotice, setThresholdNotice] = useState('');
  const [sort, setSort] = useState<SortKey>('balance_desc');
  const [page, setPage] = useState(1);
  const [whales, setWhales] = useState<WhaleListResponse | null>(null);
  const [whaleListVersion, setWhaleListVersion] = useState(0);
  const [inputAddress, setInputAddress] = useState('');
  const [selectedAddress, setSelectedAddress] = useState('');
  const [detail, setDetail] = useState<AddressDetailResponse | null>(null);
  const [summary, setSummary] = useState<AISummaryResponse | null>(null);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [graph, setGraph] = useState<GraphResponse | null>(null);
  const [watchlists, setWatchlists] = useState<WatchlistItem[]>([]);
  const [alerts, setAlerts] = useState<AlertEvent[]>([]);
  const [priceInterval, setPriceInterval] = useState('5m');
  const [prices, setPrices] = useState<PricePoint[]>([]);
  const [news, setNews] = useState<NewsItem[]>([]);
  const [figureNews, setFigureNews] = useState<FigureNewsItem[]>([]);
  const [figureNewsSource, setFigureNewsSource] = useState('');
  const [selectedNewsItem, setSelectedNewsItem] = useState<NewsItem | null>(null);
  const [importLoading, setImportLoading] = useState(false);
  const [watchThreshold, setWatchThreshold] = useState(DEFAULT_WATCH_THRESHOLD);
  const [user, setUser] = useState<AppUser | null>(null);
  const [notificationStatus, setNotificationStatus] = useState<NotificationStatus | null>(null);
  const [notificationPref, setNotificationPref] = useState<NotificationPreference | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [transactionsLoading, setTransactionsLoading] = useState(false);
  const [graphLoading, setGraphLoading] = useState(false);
  const [graphEnriching, setGraphEnriching] = useState(false);
  const [testEmailLoading, setTestEmailLoading] = useState(false);
  const [error, setError] = useState('');
  const [graphMessage, setGraphMessage] = useState('');
  const [importMessage, setImportMessage] = useState('');
  const [notificationMessage, setNotificationMessage] = useState('');
  const thresholdRef = useRef(threshold);
  const sortRef = useRef(sort);
  const latestWhalesRequestId = useRef(0);

  useEffect(() => {
    thresholdRef.current = threshold;
  }, [threshold]);

  useEffect(() => {
    sortRef.current = sort;
  }, [sort]);

  const loadWhales = useCallback(
    async (nextPage = page) => {
      const requestId = ++latestWhalesRequestId.current;
      setError('');
      const normalized = parseThreshold(thresholdRef.current);
      setThresholdNotice(normalized.message);
      if (normalized.message.includes('TEH')) {
        const normalizedThreshold = `> ${normalized.value} ETH`;
        thresholdRef.current = normalizedThreshold;
        setThreshold(normalizedThreshold);
      }
      try {
        const resp = await listWhales({
          minBalanceEth: normalized.value,
          sort: sortRef.current,
          page: nextPage,
          pageSize: PAGE_SIZE,
        });
        if (requestId === latestWhalesRequestId.current) {
          setWhales(resp);
          setPage(nextPage);
          setWhaleListVersion((version: number) => version + 1);
        }
      } catch (err) {
        if (requestId === latestWhalesRequestId.current) setError(errorMessage(err));
      }
    },
    [page],
  );

  const handleUpdateWhales = useCallback(async () => {
    await loadWhales(1);
  }, [loadWhales]);

  const loadSideData = useCallback(async () => {
    const [priceResp, newsResp, figureNewsResp] = await Promise.allSettled([
      getETHPrices(priceInterval),
      getETHNews(),
      getCryptoFigureNews(),
    ]);
    if (priceResp.status === 'fulfilled') setPrices(priceResp.value.items);
    if (newsResp.status === 'fulfilled') setNews(newsResp.value.items);
    if (figureNewsResp.status === 'fulfilled') {
      setFigureNews(figureNewsResp.value.items);
      setFigureNewsSource(figureNewsResp.value.source);
    } else {
      setFigureNews([]);
      setFigureNewsSource('crypto_figure_news_unavailable');
    }
  }, [priceInterval]);

  const loadUserData = useCallback(async () => {
    const [meResp, watchResp, alertResp, statusResp] = await Promise.allSettled([
      getMe(),
      listWatchlists(),
      listAlerts(),
      getNotificationStatus(),
    ]);
    if (meResp.status === 'fulfilled') {
      setUser(meResp.value.user);
      setNotificationPref(meResp.value.notification_preferences);
    }
    if (watchResp.status === 'fulfilled') setWatchlists(watchResp.value.items);
    if (alertResp.status === 'fulfilled') setAlerts(alertResp.value.items);
    if (statusResp.status === 'fulfilled') setNotificationStatus(statusResp.value);
  }, []);

  useEffect(() => {
    const auth = captureGoogleOAuthCallback();
    if (auth.userId) setNotificationMessage('Gmail 登入授權成功，可以直接追蹤地址。');
    if (auth.error) setError(`Gmail 登入失敗：${auth.error}`);
    void (async () => {
      try {
        await syncWhalesFromConfiguredURL();
      } catch {
        // Keep demo/fallback rows if Etherscan is temporarily unavailable.
      }
      await loadWhales(1);
    })();
    loadUserData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    loadSideData();
  }, [loadSideData]);

  useEffect(() => {
    loadWhales(1);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sort]);

  const fetchAddressGraphWithRetry = useCallback(async (address: string) => {
    let lastError: unknown;
    for (let attempt = 0; attempt <= GRAPH_RETRY_TIMES; attempt++) {
      try {
        return await withTimeout(
          getAddressGraph(address),
          GRAPH_TIMEOUT_MS,
          `關聯標籤載入逾時（>${Math.floor(GRAPH_TIMEOUT_MS / 1000)} 秒），請稍後再試。`,
        );
      } catch (err) {
        lastError = err;
        if (attempt < GRAPH_RETRY_TIMES) {
          setGraphMessage(`關聯標籤載入較慢，正在重試（${attempt + 1}/${GRAPH_RETRY_TIMES}）...`);
          await sleep(GRAPH_RETRY_DELAY_MS);
          continue;
        }
      }
    }
    throw lastError instanceof Error ? lastError : new Error('關聯標籤載入失敗，請稍後再試。');
  }, []);

  const selectAddress = useCallback(async (address: string) => {
    setSelectedAddress(address);
    setInputAddress(address);
    setDetailLoading(true);
    setTransactionsLoading(true);
    setGraphLoading(false);
    setGraphEnriching(true);
    setGraphMessage('');
    setDetail(null);
    setSummary(null);
    setTransactions([]);
    setGraph(null);
    setError('');

    const [detailResp, summaryResp, txResp] = await Promise.allSettled([
      getAddressDetail(address),
      getAddressAISummary(address),
      getAddressTransactions(address),
    ]);

    if (detailResp.status === 'fulfilled') setDetail(detailResp.value);
    if (summaryResp.status === 'fulfilled') setSummary(summaryResp.value);
    if (txResp.status === 'fulfilled') setTransactions(txResp.value.transactions);
    if (detailResp.status === 'rejected') setError(errorMessage(detailResp.reason));
    if (txResp.status === 'rejected') setError(errorMessage(txResp.reason));
    setDetailLoading(false);
    setTransactionsLoading(false);

    void fetchAddressGraphWithRetry(address)
      .then((g) => {
        setGraph(g);
        setGraphMessage('');
      })
      .catch((err) => {
        setGraphMessage(errorMessage(err));
      })
      .finally(() => {
        setGraphEnriching(false);
      });
  }, [fetchAddressGraphWithRetry]);

  const retryAddressGraph = useCallback(async () => {
    const address = (selectedAddress || detail?.address || '').trim();
    if (!address) return;
    setGraphLoading(true);
    setGraphEnriching(true);
    setGraphMessage('');
    try {
      const graphResp = await fetchAddressGraphWithRetry(address);
      setGraph(graphResp);
      setGraphMessage('');
    } catch (err) {
      setGraphMessage(errorMessage(err));
    } finally {
      setGraphLoading(false);
      setGraphEnriching(false);
    }
  }, [selectedAddress, detail?.address, fetchAddressGraphWithRetry]);

  const prefillAddress = useCallback((address: string) => {
    setInputAddress(address);
    setSelectedAddress(address);
  }, []);

  const requireGoogleLogin = useCallback((): AppUser | null => {
    if (user) return user;
    setError('請先使用 Gmail 登入授權，通過後再按「+ 追蹤」。');
    return null;
  }, [user]);

  const handleLogin = () => {
    setError('');
    setNotificationMessage('');
    startGoogleOAuthLogin();
  };

  const handleTrack = async (address: string, alias?: string) => {
    const activeUser = requireGoogleLogin();
    if (!activeUser) {
      return;
    }

    const existing = watchlists.find((w) => w.address.toLowerCase() === address.toLowerCase());
    if (existing) {
      try {
        await deleteWatchlist(existing.id);
        setWatchlists((prev) => prev.filter((w) => w.id !== existing.id));
        setNotificationMessage(`已取消追蹤 ${shortAddress(address)}`);
        await loadWhales(page);
        if (selectedAddress) await selectAddress(selectedAddress);
      } catch (err) {
        setError(errorMessage(err));
      }
      return;
    }

    const normalizedThreshold = normalizeETHAmount(watchThreshold);
    if (!normalizedThreshold) {
      setError('請輸入有效的通知門檻，例如 500 或 1000。');
      return;
    }
    setWatchThreshold(normalizedThreshold);
    setError('');
    try {
      const resp = await upsertWatchlist({
        address,
        alias,
        min_interaction_eth: normalizedThreshold,
        notification_on: true,
      });
      const item = resp.item;
      setWatchlists((prev) => [item, ...prev.filter((w) => w.id !== item.id)]);
      if (resp.notification_status) setNotificationStatus(resp.notification_status);
      if (resp.notification_log?.status === 'sent') {
        setNotificationMessage(
          `已追蹤 ${shortAddress(address)}，確認信已寄出：${resp.notification_log.provider_message_id || 'sent'}`,
        );
      } else if (resp.notification_error) {
        setNotificationMessage(`已追蹤 ${shortAddress(address)}，但確認信寄送失敗：${resp.notification_error}`);
      } else {
        setNotificationMessage(`已追蹤 ${shortAddress(address)}，超過 ${normalizedThreshold} ETH 會建立通知事件。`);
      }
      await loadWhales(page);
      if (selectedAddress) await selectAddress(selectedAddress);
    } catch (err) {
      setError(errorMessage(err));
    }
  };

  const handleSendTestEmail = async () => {
    const activeUser = requireGoogleLogin();
    if (!activeUser) {
      return;
    }
    setTestEmailLoading(true);
    setError('');
    setNotificationMessage('');
    try {
      const resp = await sendTestNotification(activeUser.email);
      setNotificationStatus(resp.notification_status);
      setNotificationMessage(`測試信已送出：${resp.log.provider_message_id || resp.log.status}`);
    } catch (err) {
      setError(errorMessage(err));
      const status = await getNotificationStatus().catch(() => null);
      if (status) setNotificationStatus(status);
    } finally {
      setTestEmailLoading(false);
    }
  };

  const handleToggleNotificationPref = async () => {
    const activeUser = requireGoogleLogin();
    if (!activeUser || !notificationPref) return;
    
    setError('');
    try {
      const newPref = await updateNotificationPreferences({
        ...notificationPref,
        gmail_enabled: !notificationPref.gmail_enabled,
      });
      setNotificationPref(newPref);
      setNotificationMessage(newPref.gmail_enabled ? '已開啟通知。' : '已關閉通知。');
    } catch (err) {
      setError('更新通知設定失敗: ' + errorMessage(err));
    }
  };

  const handleImport = async () => {
    setImportLoading(true);
    setImportMessage('');
    setError('');
    try {
      const resp = await syncWhalesFromConfiguredURL();
      setImportMessage(`已同步 ${resp.imported} 筆，略過 ${resp.skipped} 筆。`);
      await loadWhales(1);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setImportLoading(false);
    }
  };

  const chartData = prices.map((item) => ({
    time: shortTime(item.timestamp, priceInterval),
    close: item.close,
    high: item.high,
    low: item.low,
  }));

  const selectedWhale = whales?.items.find((item) => item.address === selectedAddress);
  const newsTickerItems = news.length > 0 ? [...news, ...news] : [];

  return (
    <div className="min-h-screen bg-[#171a18] text-slate-100">
      <header className="border-b border-slate-700/70 bg-[#20231f]">
        <div className="mx-auto flex max-w-7xl flex-col gap-3 px-4 py-3 md:flex-row md:items-center md:justify-between">
          <div>
            <h1 className="text-xl font-semibold tracking-tight">鯨魚掃描器</h1>
            <p className="text-sm text-slate-400">公開鏈上資料分析與通知，不處理私鑰、不簽交易。</p>
          </div>
          <div className="flex flex-wrap items-center gap-2 text-sm">
            <span className="rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-3 py-1 text-emerald-200">
              免費資料源
            </span>
            <span className="rounded-lg border border-slate-600 px-3 py-1 text-slate-300">
              {user ? user.email : '尚未啟用通知'}
            </span>
          </div>
        </div>
      </header>

      <main className="mx-auto grid max-w-7xl grid-cols-1 gap-4 px-4 py-4 lg:grid-cols-[260px_1fr_340px]">
        <aside className="space-y-4">
          <section className="rounded-lg border border-slate-700 bg-[#20231f] p-4">
            <h2 className="text-sm font-semibold text-slate-200">掃描設定</h2>
            <label className="mt-4 block text-xs text-slate-400">最低 ETH 餘額</label>
            <input
              value={threshold}
              onChange={(event) => setThreshold(event.target.value)}
              className="mt-2 w-full rounded-lg border border-slate-700 bg-[#171a18] px-3 py-2 text-sm text-white outline-none focus:border-emerald-500"
              placeholder="> 1000 ETH"
            />
            {thresholdNotice && <p className="mt-2 text-xs text-emerald-300">{thresholdNotice}</p>}
            <WhaleListUpdateButton
              key={whaleListVersion}
              onUpdate={handleUpdateWhales}
            />
            <p className="mt-3 text-xs leading-5 text-slate-500">
              來源採 Etherscan Top Accounts 同步資料；若未設定 CSV URL，後端會嘗試讀取公開帳戶頁。
            </p>
          </section>

          <section className="rounded-lg border border-slate-700 bg-[#20231f] p-4">
            <h2 className="text-sm font-semibold text-slate-200">通知設定</h2>
            <p className="mt-3 rounded-lg border border-slate-700 bg-[#171a18] px-3 py-2 text-sm text-slate-300">
              {user ? user.email : '尚未使用 Gmail 登入授權'}
            </p>
            <label className="mt-3 block text-xs text-slate-400">通知門檻 ETH</label>
            <input
              value={watchThreshold}
              onChange={(event) => setWatchThreshold(event.target.value)}
              className="mt-2 w-full rounded-lg border border-slate-700 bg-[#171a18] px-3 py-2 text-sm text-white outline-none focus:border-emerald-500"
              placeholder="500"
            />
            <p className="mt-2 text-xs text-slate-500">之後按「追蹤」會用這個門檻；已追蹤地址可再次按追蹤更新門檻。</p>
            <button
              onClick={handleLogin}
              className="mt-3 w-full rounded-lg border border-slate-600 px-4 py-2 text-sm text-slate-100 hover:border-emerald-500"
            >
              {user ? '重新授權 Gmail' : '使用 Gmail 登入授權'}
            </button>
            <button
              onClick={handleSendTestEmail}
              disabled={testEmailLoading || !user}
              className="mt-2 w-full rounded-lg border border-emerald-500 px-4 py-2 text-sm text-emerald-200 hover:bg-emerald-500/10 disabled:opacity-50"
            >
              {testEmailLoading ? '寄送中...' : '寄送測試信'}
            </button>
            {notificationPref && (
              <button
                onClick={handleToggleNotificationPref}
                className={`mt-2 w-full rounded-lg border px-4 py-2 text-sm transition-colors ${
                  notificationPref.gmail_enabled
                    ? 'border-emerald-500 bg-emerald-500/20 text-emerald-200 hover:bg-emerald-500/30'
                    : 'border-slate-500 bg-slate-500/20 text-slate-300 hover:bg-slate-500/30'
                }`}
              >
                {notificationPref.gmail_enabled ? '已開啟通知 (點擊關閉)' : '已關閉通知 (點擊開啟)'}
              </button>
            )}
            {notificationMessage && <p className="mt-3 text-xs text-emerald-300">{notificationMessage}</p>}
            {notificationStatus && (
              <p className="mt-3 text-xs leading-5 text-slate-400">
                寄信狀態：{notificationStatus.provider}
                {notificationStatus.from ? ` · ${notificationStatus.from}` : ''}
                {!notificationStatus.configured && ' · 尚未設定真實寄信憑證'}
              </p>
            )}
            <p className="mt-3 text-xs leading-5 text-slate-500">
              登入會跳轉到 Google，通過後即可追蹤地址並寄送 Gmail 通知。
            </p>
          </section>

          <section className="rounded-lg border border-slate-700 bg-[#20231f] p-4">
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-semibold text-slate-200">巨鯨資料同步</h2>
            </div>
            <p className="mt-3 text-xs leading-5 text-slate-500">
              後端優先使用 <code>ETHERSCAN_TOP_ACCOUNTS_CSV_URL</code>；未設定時會同步公開帳戶頁，客戶不需要手動貼資料。
            </p>
            <button
              onClick={handleImport}
              disabled={importLoading}
              className="mt-3 w-full rounded-lg border border-slate-600 px-4 py-2 text-sm text-slate-100 hover:border-emerald-500 disabled:opacity-50"
            >
              {importLoading ? '同步中...' : '同步巨鯨種子'}
            </button>
            {importMessage && <p className="mt-3 text-xs text-emerald-300">{importMessage}</p>}
          </section>

          <section className="rounded-lg border border-slate-700 bg-[#20231f] p-4">
            <h2 className="text-sm font-semibold text-slate-200">追蹤清單</h2>
            <div className="mt-3 space-y-2">
              {watchlists.length === 0 && <p className="text-xs text-slate-500">尚未追蹤地址。</p>}
              {watchlists.slice(0, 6).map((item) => (
                <button
                  key={item.id}
                  onClick={() => prefillAddress(item.address)}
                  className="block w-full rounded-lg border border-slate-700 px-3 py-2 text-left text-xs text-slate-300 hover:border-emerald-500"
                >
                  <span className="block font-mono">{shortAddress(item.address)}</span>
                  <span className="text-slate-500">門檻 &gt; {item.min_interaction_eth} ETH</span>
                </button>
              ))}
            </div>
          </section>
        </aside>

        <section className="space-y-4">
          {error && (
            <div className="rounded-lg border border-red-500/50 bg-red-500/10 px-4 py-3 text-sm text-red-200">
              {error}
            </div>
          )}

          <div className="rounded-lg border border-slate-700 bg-[#20231f]">
            <div className="flex flex-col gap-3 border-b border-slate-700 px-4 py-3 md:flex-row md:items-center md:justify-between">
              <div>
                <h2 className="text-base font-semibold">巨鯨名單</h2>
                <p className="text-xs text-slate-500">
                  {whales
                    ? `總數 ${whales.total.toLocaleString()} 筆 · 目前 ${visibleRange(whales)} · 快照 ${
                        formatTaiwanDateTime(whales.snapshot_at) || 'demo'
                      }`
                    : '載入中'}
                </p>
                {whales?.source.includes('demo') && (
                  <p className="mt-1 text-xs text-amber-300">目前是 demo fallback 分布，不代表 Etherscan 真實排行。</p>
                )}
              </div>
              <select
                value={sort}
                onChange={(event) => setSort(event.target.value as SortKey)}
                className="rounded-lg border border-slate-700 bg-[#171a18] px-3 py-2 text-sm text-white outline-none"
              >
                <option value="balance_desc">餘額排序</option>
                <option value="balance_asc">餘額由低到高</option>
                <option value="tracked">追蹤中名單</option>
              </select>
            </div>

            <div className="overflow-x-auto">
              <table className="w-full min-w-[760px] text-left text-sm">
                <thead className="border-b border-slate-700 text-xs text-slate-500">
                  <tr>
                    <th className="px-4 py-3">地址</th>
                    <th className="px-4 py-3">ETH 餘額</th>
                    <th className="px-4 py-3">持有佔比</th>
                    <th className="px-4 py-3">分類</th>
                    <th className="px-4 py-3">追蹤</th>
                  </tr>
                </thead>
                <tbody>
                  {(whales?.items || []).map((whale) => (
                    <tr
                      key={whale.address}
                      className={`border-b border-slate-800 hover:bg-slate-800/40 ${
                        selectedAddress === whale.address ? 'bg-emerald-500/5' : ''
                      }`}
                    >
                      <td className="px-4 py-3">
                        <button onClick={() => prefillAddress(whale.address)} className="text-left">
                          <span className="block font-mono text-slate-100">{shortAddress(whale.address)}</span>
                          {whale.name_tag != null && String(whale.name_tag).trim() !== '' && (
                            <span className="text-xs text-slate-500">{String(whale.name_tag)}</span>
                          )}
                        </button>
                      </td>
                      <td className="px-4 py-3 font-semibold">{formatETH(whale.balance_eth)} ETH</td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          <span>
                            {whale.percentage != null && String(whale.percentage).trim() !== ''
                              ? String(whale.percentage)
                              : '-'}
                          </span>
                          <span className="h-1.5 w-16 rounded bg-slate-700">
                            <span
                              className="block h-1.5 rounded bg-emerald-400"
                              style={{
                                width: `${Math.min(
                                  parseFloat(String(whale.percentage || '0').replace(/%/g, '')) * 12,
                                  100,
                                )}%`,
                              }}
                            />
                          </span>
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex flex-wrap gap-1">
                          {whale.labels.slice(0, 2).map((label) => (
                            <span key={`${whale.address}-${label.category}`} className={tagClass(label.category)}>
                              {String(label.name ?? '')}
                            </span>
                          ))}
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <button
                          onClick={() => handleTrack(whale.address, whale.name_tag)}
                          className={`rounded-lg border px-3 py-1 text-xs ${
                            whale.is_tracked
                              ? 'border-emerald-400 bg-emerald-400/10 text-emerald-200'
                              : 'border-slate-600 text-slate-300 hover:border-emerald-500'
                          }`}
                        >
                          {whale.is_tracked ? '追蹤中' : '+ 追蹤'}
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div className="flex items-center justify-between px-4 py-3 text-sm text-slate-400">
              <button
                onClick={() => loadWhales(Math.max(page - 1, 1))}
                disabled={page <= 1}
                className="rounded-lg border border-slate-700 px-3 py-1 disabled:opacity-40"
              >
                上一頁
              </button>
              <span>
                第 {page} 頁{whales ? ` · 共 ${Math.ceil(whales.total / whales.page_size).toLocaleString()} 頁` : ''}
              </span>
              <button
                onClick={() => loadWhales(page + 1)}
                disabled={!whales?.has_next}
                className="rounded-lg border border-slate-700 px-3 py-1 disabled:opacity-40"
              >
                下一頁
              </button>
            </div>
          </div>

          <div className="rounded-lg border border-slate-700 bg-[#20231f] p-4">
            <h2 className="text-base font-semibold">單一地址搜尋</h2>
            <p className="mt-1 text-xs text-slate-500">輸入 ETH 地址後，按掃描載入單一地址分析。</p>
            <div className="mt-3">
              <AddressInput onScan={selectAddress} onChange={setInputAddress} loading={detailLoading} value={inputAddress} />
            </div>
          </div>

          <div className="flex h-80 flex-col rounded-lg border border-slate-700 bg-[#20231f] p-4">
            <div className="flex shrink-0 flex-col gap-3 md:flex-row md:items-center md:justify-between">
              <div>
                <h2 className="text-base font-semibold">ETH 價格走勢</h2>
                <p className="text-xs text-slate-500">CoinGecko 免費資料，後端快取。</p>
              </div>
              <div className="flex gap-2">
                {['5m', '1d', '1w'].map((interval) => (
                  <button
                    key={interval}
                    onClick={() => setPriceInterval(interval)}
                    className={`rounded-lg border px-3 py-1 text-xs ${
                      priceInterval === interval
                        ? 'border-emerald-400 text-emerald-200'
                        : 'border-slate-700 text-slate-400'
                    }`}
                  >
                    {interval}
                  </button>
                ))}
              </div>
            </div>
            <div className="mt-4 min-h-0 flex-1 w-full">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={chartData}>
                  <CartesianGrid stroke="#334155" strokeDasharray="3 3" />
                  <XAxis dataKey="time" stroke="#94a3b8" tick={{ fontSize: 11 }} />
                  <YAxis stroke="#94a3b8" tick={{ fontSize: 11 }} domain={['dataMin - 10', 'dataMax + 10']} />
                  <Tooltip contentStyle={{ background: '#171a18', border: '1px solid #475569', borderRadius: 8 }} />
                  <Area type="monotone" dataKey="close" stroke="#34d399" fill="#34d39922" strokeWidth={2} />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          </div>

          <AddressDetailPanel
            detail={detail}
            summary={summary}
            transactions={transactions}
            graph={graph}
            selectedWhale={selectedWhale}
            loading={detailLoading}
            transactionsLoading={transactionsLoading}
            graphLoading={graphLoading}
            graphEnriching={graphEnriching}
            graphMessage={graphMessage}
            onTrack={handleTrack}
            onRetryGraph={retryAddressGraph}
          />
        </section>

        <aside className="space-y-4">
          <section className="rounded-lg border border-slate-700 bg-[#20231f] p-4">
            <h2 className="text-sm font-semibold text-slate-200">通知事件</h2>
            <div className="mt-3 space-y-3">
              {alerts.length === 0 && <p className="text-xs text-slate-500">尚無通知事件。</p>}
              {alerts.slice(0, 6).map((alert) => (
                <button
                  key={alert.id}
                  onClick={() => prefillAddress(alert.address)}
                  className="block w-full rounded-lg border border-slate-700 px-3 py-2 text-left hover:border-emerald-500"
                >
                  <span className="block text-sm text-slate-100">{alert.title}</span>
                  <span className="text-xs text-slate-500">{alert.description}</span>
                </button>
              ))}
            </div>
          </section>

          <section className="rounded-lg border border-slate-700 bg-[#20231f] p-4">
            <h2 className="text-sm font-semibold text-slate-200">ETH 相關報導</h2>
            <div className="news-ticker mt-3 overflow-hidden rounded-lg border border-slate-700 bg-[#171a18]">
              {newsTickerItems.length === 0 ? (
                <p className="px-3 py-3 text-xs text-slate-500">新聞來源暫時不可用，稍後再試。</p>
              ) : (
                <div className="news-ticker-track flex gap-3 py-3">
                  {newsTickerItems.map((item, index) => (
                    <button
                      key={`${item.id}-${index}`}
                      type="button"
                      onClick={() => setSelectedNewsItem(item)}
                      className="w-72 shrink-0 rounded-lg border border-slate-700 px-3 py-2 text-left hover:border-emerald-500 focus:border-emerald-400 focus:outline-none"
                    >
                      <span className="block text-sm font-medium leading-5 text-slate-100">{item.title}</span>
                      <span className="mt-2 block text-xs text-slate-500">
                        {item.source} · {formatNewsTime(item.published_at)}
                      </span>
                    </button>
                  ))}
                </div>
              )}
            </div>
          </section>

          <section className="rounded-lg border border-slate-700 bg-[#20231f] p-4">
            <h2 className="text-sm font-semibold text-slate-200">重要人物相關新聞</h2>
            <p className="mt-1 text-xs text-slate-500">Vitalik / Trump 的加密貨幣相關新聞。</p>
            <div className="mt-3 space-y-3">
              {figureNews.length === 0 && (
                <p className="text-xs text-slate-500">{figureNewsEmptyMessage(figureNewsSource)}</p>
              )}
              {figureNews.slice(0, 3).map((item) => (
                <a
                  key={item.id}
                  href={item.url}
                  target="_blank"
                  rel="noreferrer"
                  className="block rounded-lg border border-slate-700 px-3 py-2 hover:border-emerald-500"
                >
                  <span className="block text-xs text-emerald-300">
                    {item.person} · {item.source} · {formatNewsTime(item.published_at)}
                  </span>
                  <span className="mt-2 block text-sm leading-5 text-slate-100">{item.title}</span>
                  <span className="mt-2 block text-xs leading-5 text-slate-500">{item.snippet}</span>
                </a>
              ))}
            </div>
          </section>

          <section className="rounded-lg border border-slate-700 bg-[#20231f] p-4">
            <h2 className="text-sm font-semibold text-slate-200">資料來源狀態</h2>
            <ul className="mt-3 space-y-2 text-xs text-slate-400">
              <li>Etherscan：單地址交易與餘額補查</li>
              <li>Top Accounts CSV：巨鯨種子匯入</li>
              <li>CoinGecko：ETH 價格快取</li>
              <li>GDELT / Cointelegraph：合規新聞連結</li>
              <li>Google News RSS：Vitalik / Trump 加密貨幣新聞</li>
              <li>Gmail：通知寄送，支援 dry-run</li>
            </ul>
          </section>
        </aside>
      </main>
      {selectedNewsItem && (
        <div className="fixed inset-0 z-50 flex items-center justify-center px-4 py-6">
          <button
            type="button"
            aria-label="關閉新聞詳情"
            onClick={() => setSelectedNewsItem(null)}
            className="absolute inset-0 bg-black/70"
          />
          <section
            role="dialog"
            aria-modal="true"
            aria-labelledby="news-dialog-title"
            className="relative w-full max-w-lg rounded-lg border border-slate-700 bg-[#20231f] p-5 shadow-2xl"
          >
            <div className="flex items-start justify-between gap-4">
              <div>
                <p className="text-xs text-emerald-300">
                  {selectedNewsItem.source} · {formatNewsTime(selectedNewsItem.published_at)}
                </p>
                <h2 id="news-dialog-title" className="mt-2 text-lg font-semibold leading-6 text-slate-100">
                  {selectedNewsItem.title}
                </h2>
              </div>
              <button
                type="button"
                onClick={() => setSelectedNewsItem(null)}
                className="rounded-lg border border-slate-600 px-2 py-1 text-xs text-slate-300 hover:border-emerald-500"
              >
                關閉
              </button>
            </div>
            <p className="mt-4 text-sm leading-6 text-slate-300">{selectedNewsItem.snippet}</p>
            <a
              href={selectedNewsItem.url}
              target="_blank"
              rel="noreferrer"
              className="mt-4 inline-flex rounded-lg border border-emerald-500 px-4 py-2 text-sm font-medium text-emerald-200 hover:bg-emerald-500/10"
            >
              開啟原文
            </a>
          </section>
        </div>
      )}
    </div>
  );
}

function AddressDetailPanel({
  detail,
  summary,
  transactions,
  graph,
  selectedWhale,
  loading,
  transactionsLoading,
  graphLoading,
  graphEnriching,
  graphMessage,
  onTrack,
  onRetryGraph,
}: {
  detail: AddressDetailResponse | null;
  summary: AISummaryResponse | null;
  transactions: Transaction[];
  graph: GraphResponse | null;
  selectedWhale?: WhaleAccount;
  loading: boolean;
  transactionsLoading: boolean;
  graphLoading: boolean;
  graphEnriching: boolean;
  graphMessage: string;
  onTrack: (address: string, alias?: string) => void;
  onRetryGraph: () => void;
}) {
  const [activeGraphCategory, setActiveGraphCategory] = useState<'center' | 'normal' | 'contract' | 'exchange' | 'bridge'>('center');
  const [copiedGraphAddress, setCopiedGraphAddress] = useState('');

  const copyGraphAddress = useCallback(async (address: string) => {
    try {
      await navigator.clipboard.writeText(address);
      setCopiedGraphAddress(address);
      window.setTimeout(() => {
        setCopiedGraphAddress((prev) => (prev === address ? '' : prev));
      }, 1200);
    } catch {
      setCopiedGraphAddress('');
    }
  }, []);

  if (loading) {
    return (
      <div className="rounded-lg border border-slate-700 bg-[#20231f] p-4 text-sm text-slate-400">
        地址資料載入中...
      </div>
    );
  }
  if (!detail) {
    return (
      <div className="rounded-lg border border-slate-700 bg-[#20231f] p-4 text-sm text-slate-400">
        請輸入一個地址後查看詳情。
      </div>
    );
  }

  const timelineData = buildTransactionTimeline(transactions);
  const categoryData = buildTransactionCategoryDistribution(transactions);
  const hasSankeyFlowData = transactions.some((tx) => {
    const from = (tx.from || '').trim().toLowerCase();
    const to = (tx.to || '').trim().toLowerCase();
    const asset = (tx.asset || '').trim().toUpperCase();
    const amount = Number(tx.value);
    return asset === 'ETH' && Number.isFinite(amount) && amount > 0 && (from === detail.address.toLowerCase() || to === detail.address.toLowerCase());
  });

  const hasBackendGraph = Boolean(graph && graph.nodes.length > 0);
  const graphNodes: GraphNode[] = hasBackendGraph
    ? graph!.nodes
    : buildTxCounterpartyGraphNodes(detail.address, transactions);
  const graphCategoryItems = {
    center: graphNodes.filter((node) => node.is_center),
    exchange: graphNodes.filter((node) => node.tag === 'exchange'),
    bridge: graphNodes.filter((node) => node.tag === 'bridge'),
    contract: graphNodes.filter((node) => node.is_contract),
    normal: graphNodes.filter((node) => !node.is_center && !node.is_contract && node.tag !== 'exchange' && node.tag !== 'bridge'),
  };
  const graphCategoryOptions: Array<{
    key: 'center' | 'normal' | 'contract' | 'exchange' | 'bridge';
    label: string;
    dotClassName: string;
  }> = [
    { key: 'center', label: '中心地址', dotClassName: 'rounded-full bg-amber-500' },
    { key: 'normal', label: '一般地址', dotClassName: 'rounded-full bg-indigo-500' },
    { key: 'contract', label: '合約', dotClassName: 'rounded bg-emerald-500' },
    { key: 'exchange', label: '交易所', dotClassName: 'rotate-45 bg-rose-500' },
    { key: 'bridge', label: '跨鏈橋', dotClassName: 'rounded-sm bg-cyan-500' },
  ];
  const activeGraphNodeList = graphCategoryItems[activeGraphCategory];

  return (
    <div className="rounded-lg border border-slate-700 bg-[#20231f] p-4">
      <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
        <div>
          <h2 className="text-base font-semibold">地址詳情</h2>
          <p className="font-mono text-sm text-emerald-200">{detail.address}</p>
          {selectedWhale?.name_tag && <p className="text-sm text-slate-400">{selectedWhale.name_tag}</p>}
        </div>
        <button
          onClick={() => onTrack(detail.address, selectedWhale?.name_tag)}
          className="rounded-lg border border-emerald-500 px-3 py-2 text-sm text-emerald-200"
        >
          {detail.is_tracked ? '已追蹤' : '+ 追蹤此地址'}
        </button>
      </div>

      <div className="mt-4 grid grid-cols-1 gap-3 md:grid-cols-3">
        <Metric
          label="ETH 餘額"
          value={
            detail.balance?.eth_balance
              ? `${formatETH(detail.balance.eth_balance)} ETH`
              : selectedWhale
                ? `${formatETH(selectedWhale.balance_eth)} ETH`
                : '待補查'
          }
        />
        <Metric label="風險分數" value={`${detail.risk_score.score} / 100`} hint={detail.risk_score.level} />
        <Metric
          label="分類信心"
          value={`${Math.round((detail.labels[0]?.confidence || 0) * 100)}%`}
          hint={detail.labels[0]?.source || 'local_rules'}
        />
      </div>

      <div className="mt-4 flex flex-wrap gap-2">
        {detail.labels.map((label) => (
          <span key={`${label.category}-${label.source}`} className={tagClass(label.category)}>
            {label.name} · {Math.round(label.confidence * 100)}%
          </span>
        ))}
      </div>

      {summary && (
        <div className="mt-4 rounded-lg border border-slate-700 bg-[#171a18] p-3">
          <h3 className="text-sm font-semibold text-slate-200">AI 簡短分析</h3>
          <p className="mt-2 text-sm leading-6 text-slate-300">{summary.summary}</p>
          <p className="mt-2 text-xs text-slate-500">Heuristic · confidence {Math.round(summary.confidence * 100)}%</p>
        </div>
      )}

      <div className="mt-4 grid grid-cols-1 gap-4 xl:grid-cols-2 xl:items-start">
        <div className="rounded-lg border border-slate-700 bg-[#171a18] p-3 xl:self-start">
          <h3 className="text-sm font-semibold text-slate-200">交易歷史</h3>
          <div className="mt-3 max-h-[420px] overflow-auto">
            {transactionsLoading && <p className="text-xs text-slate-500">交易資料載入中...</p>}
            {!transactionsLoading && transactions.length === 0 && (
              <p className="text-xs text-slate-500">尚無交易資料，或 Etherscan API key 尚未設定。</p>
            )}
            {transactions.slice(0, 25).map((tx) => (
              <div key={`${tx.hash}-${tx.category}`} className="border-b border-slate-800 py-2 text-xs">
                <div className="flex items-center justify-between gap-2">
                  <span className="font-mono text-slate-300">{shortAddress(tx.hash)}</span>
                  <span className="text-emerald-300">
                    {tx.value} {tx.asset}
                  </span>
                </div>
                <div className="mt-1 text-slate-500">
                  {tx.category} · {tx.timestamp}
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="rounded-lg border border-slate-700 bg-[#171a18] p-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold text-slate-200">地址關係列表</h3>
            <button
              type="button"
              onClick={onRetryGraph}
              disabled={graphLoading}
              className="rounded border border-slate-600 px-2 py-1 text-[11px] text-slate-300 hover:border-emerald-500 disabled:opacity-50"
            >
              {graphLoading ? '重整中...' : '重整關聯標籤'}
            </button>
          </div>
          <div className="mt-3 rounded border border-slate-700 bg-[#121512] p-3">
            {graphEnriching && (
              <p className="text-xs text-slate-500">正在從後端補齊合約／交易所／跨鏈橋等標籤（列表已可先從交易對手產生）。</p>
            )}
            {!graphEnriching && hasBackendGraph && (
              <p className="text-xs text-slate-400">已套用後端關聯資料，可用下方分類篩選。</p>
            )}
            {!graphEnriching && !hasBackendGraph && graphMessage && (
              <p className="text-xs text-amber-300">{graphMessage}</p>
            )}
            {!graphEnriching && !hasBackendGraph && !graphMessage && transactions.length === 0 && (
              <p className="text-xs text-slate-500">尚無交易資料，無法產生關係列表。</p>
            )}
            {!graphEnriching && !hasBackendGraph && !graphMessage && transactions.length > 0 && (
              <p className="text-xs text-slate-400">目前為交易摘要即時列表；按「重整關聯標籤」可再向後端請求完整分類。</p>
            )}
          </div>
          <div className="mt-3 flex flex-wrap gap-2 text-xs text-slate-300">
            {graphCategoryOptions.map((option) => {
              const selected = activeGraphCategory === option.key;
              const count = graphCategoryItems[option.key].length;
              return (
                <button
                  key={option.key}
                  type="button"
                  onClick={() => setActiveGraphCategory(option.key)}
                  className={`inline-flex items-center gap-1.5 rounded border px-2 py-1 transition ${
                    selected ? 'border-emerald-500 text-emerald-200' : 'border-slate-600 text-slate-400 hover:border-slate-500'
                  }`}
                >
                  <span className={`inline-block h-3 w-3 ${option.dotClassName}`} />
                  {option.label} ({count})
                </button>
              );
            })}
          </div>
          <div className="mt-3 rounded border border-slate-700 bg-[#121512] p-2">
            <p className="text-xs text-slate-400">
              {graphCategoryOptions.find((option) => option.key === activeGraphCategory)?.label} 地址列表
            </p>
            <div className="mt-2 max-h-52 overflow-auto">
              {activeGraphNodeList.length === 0 ? (
                <p className="text-xs text-slate-500">此分類目前無地址資料。</p>
              ) : (
                activeGraphNodeList.slice(0, 50).map((node) => (
                  <div key={`${activeGraphCategory}-${node.id}`} className="border-b border-slate-800 py-1.5 text-xs last:border-b-0">
                    <div className="flex items-center justify-between gap-2">
                      <p className="font-mono text-slate-200 break-all">{node.id}</p>
                      <button
                        type="button"
                        onClick={() => void copyGraphAddress(node.id)}
                        className="shrink-0 rounded border border-slate-600 px-2 py-0.5 text-[11px] text-slate-300 hover:border-emerald-500"
                      >
                        {copiedGraphAddress === node.id ? '已複製' : '複製'}
                      </button>
                    </div>
                    <p className="mt-0.5 text-slate-500">
                      {hasBackendGraph
                        ? `${(node.tag_name || node.label || node.id).trim()} · ${node.tx_count} tx`
                        : `交易對手 · ${node.tx_count} tx`}
                    </p>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      </div>

      <div className="mt-4 flex h-52 flex-col rounded-lg border border-slate-700 bg-[#171a18] p-3">
        <h3 className="shrink-0 text-sm font-semibold text-slate-200">最近 ETH 移動</h3>
        <div className="mt-2 min-h-0 flex-1 w-full">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart
              data={transactions
                .slice(0, 20)
                .reverse()
                .map((tx, index) => ({ index, value: Number(tx.value) || 0 }))}
            >
              <CartesianGrid stroke="#334155" strokeDasharray="3 3" />
              <XAxis dataKey="index" stroke="#94a3b8" tick={{ fontSize: 11 }} />
              <YAxis stroke="#94a3b8" tick={{ fontSize: 11 }} />
              <Tooltip contentStyle={{ background: '#171a18', border: '1px solid #475569', borderRadius: 8 }} />
              <Line type="monotone" dataKey="value" stroke="#34d399" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>

      <div className="mt-4 rounded-lg border border-slate-700 bg-[#171a18] p-3">
        <h3 className="text-sm font-semibold text-slate-200">資金流向圖 (ETH)</h3>
        <div className="mt-2">
          {hasSankeyFlowData ? (
            <SankeyChart transactions={transactions} centerAddress={detail.address} />
          ) : (
            <p className="text-xs text-slate-500">尚無可用 ETH 資金流向資料。</p>
          )}
        </div>
      </div>

      <div className="mt-4 grid grid-cols-1 gap-4 xl:grid-cols-2">
        <div className="rounded-lg border border-slate-700 bg-[#171a18] p-3">
          <h3 className="text-sm font-semibold text-slate-200">交易時間軸</h3>
          {timelineData.length === 0 ? (
            <p className="mt-2 text-xs text-slate-500">尚無可用時間軸資料。</p>
          ) : (
            <div className="mt-2 flex h-56 flex-col">
              <div className="min-h-0 flex-1 w-full">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={timelineData}>
                    <CartesianGrid stroke="#334155" strokeDasharray="3 3" />
                    <XAxis dataKey="time" stroke="#94a3b8" tick={{ fontSize: 11 }} />
                    <YAxis stroke="#94a3b8" tick={{ fontSize: 11 }} />
                    <Tooltip contentStyle={{ background: '#171a18', border: '1px solid #475569', borderRadius: 8 }} />
                    <Bar dataKey="count" fill="#34d399" radius={[4, 4, 0, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </div>
          )}
        </div>

        <div className="rounded-lg border border-slate-700 bg-[#171a18] p-3">
          <h3 className="text-sm font-semibold text-slate-200">交易金額分布圖</h3>
          {categoryData.length === 0 ? (
            <p className="mt-2 text-xs text-slate-500">尚無可用交易分類資料。</p>
          ) : (
            <div className="mt-2 flex h-64 flex-col">
              <div className="min-h-0 flex-1 w-full">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={categoryData}
                      dataKey="amount"
                      nameKey="name"
                      cx="50%"
                      cy="56%"
                      outerRadius={78}
                      labelLine={false}
                      label={false}
                    >
                      {categoryData.map((entry, index) => (
                        <Cell key={entry.name} fill={CATEGORY_COLORS[index % CATEGORY_COLORS.length]} />
                      ))}
                    </Pie>
                    <Tooltip contentStyle={{ background: '#171a18', border: '1px solid #475569', borderRadius: 8 }} />
                  </PieChart>
                </ResponsiveContainer>
              </div>
              <div className="mt-2 shrink-0 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-300">
                {categoryData.map((entry, index) => (
                  <span key={`legend-${entry.name}`} className="inline-flex items-center gap-1.5">
                    <span
                      className="inline-block h-2.5 w-2.5 rounded-full"
                      style={{ backgroundColor: CATEGORY_COLORS[index % CATEGORY_COLORS.length] }}
                    />
                    {entry.name} ({entry.count}筆 · {formatPercentForLegend(entry.amountPct)})
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function Metric({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <div className="rounded-lg border border-slate-700 bg-[#171a18] p-3">
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-1 text-lg font-semibold text-slate-100">{value}</p>
      {hint && <p className="mt-1 text-xs text-slate-500">{hint}</p>}
    </div>
  );
}

function WhaleListUpdateButton({ onUpdate }: { onUpdate: () => Promise<void> }) {
  const [updating, setUpdating] = useState(false);
  const fallbackTimerRef = useRef<number | null>(null);

  const stopUpdating = useCallback(() => {
    if (fallbackTimerRef.current !== null) {
      window.clearTimeout(fallbackTimerRef.current);
      fallbackTimerRef.current = null;
    }
    setUpdating(false);
  }, []);

  useEffect(
    () => () => {
      if (fallbackTimerRef.current !== null) {
        window.clearTimeout(fallbackTimerRef.current);
      }
    },
    [],
  );

  const handleClick = useCallback(async () => {
    setUpdating(true);
    if (fallbackTimerRef.current !== null) {
      window.clearTimeout(fallbackTimerRef.current);
    }
    fallbackTimerRef.current = window.setTimeout(() => {
      fallbackTimerRef.current = null;
      setUpdating(false);
    }, 5000);

    try {
      await onUpdate();
    } finally {
      stopUpdating();
    }
  }, [onUpdate, stopUpdating]);

  return (
    <button
      type="button"
      onClick={handleClick}
      disabled={updating}
      className="mt-4 w-full rounded-lg border border-emerald-500 bg-emerald-500/10 px-4 py-2 text-sm font-medium text-emerald-200 hover:bg-emerald-500/20 disabled:opacity-50"
    >
      {updating ? '掃描中...' : '更新列表'}
    </button>
  );
}

function parseThreshold(raw: string): ThresholdParseResult {
  let cleaned = raw.trim().toUpperCase();
  const hadTEH = cleaned.includes('TEH');
  cleaned = cleaned.replaceAll('TEH', 'ETH').replaceAll('ETH', '').replaceAll('>', '').replaceAll(',', '').trim();
  const value = cleaned && !Number.isNaN(Number(cleaned)) ? cleaned : '1000';
  return {
    value,
    message: hadTEH ? `已將 TEH 標準化為 ETH：> ${value} ETH` : '',
  };
}

function normalizeETHAmount(raw: string) {
  const cleaned = raw
    .trim()
    .toUpperCase()
    .replaceAll('TEH', 'ETH')
    .replaceAll('ETH', '')
    .replaceAll('>', '')
    .replaceAll(',', '')
    .trim();
  const value = Number(cleaned);
  if (!cleaned || !Number.isFinite(value) || value <= 0) return '';
  return cleaned;
}

function formatETH(value: string) {
  const n = Number(value);
  if (!Number.isFinite(n)) return value;
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}k`;
  return n.toLocaleString(undefined, { maximumFractionDigits: 4 });
}

function shortAddress(address: string) {
  if (!address || address.length <= 12) return address;
  return `${address.slice(0, 6)}...${address.slice(-4)}`;
}

function tagClass(category: string) {
  const base = 'rounded-lg px-2 py-1 text-xs border';
  switch (category) {
    case 'exchange':
      return `${base} border-sky-300 bg-sky-300/10 text-sky-100`;
    case 'bridge':
      return `${base} border-teal-300 bg-teal-300/10 text-teal-100`;
    case 'defi_protocol':
      return `${base} border-amber-300 bg-amber-300/10 text-amber-100`;
    case 'whale':
      return `${base} border-emerald-300 bg-emerald-300/10 text-emerald-100`;
    case 'safe':
      return `${base} border-blue-400 bg-blue-400/10 text-blue-200`;
    case 'scam':
    case 'high_risk':
      return `${base} border-red-300 bg-red-300/10 text-red-100`;
    default:
      return `${base} border-slate-500 bg-slate-500/10 text-slate-200`;
  }
}

function visibleRange(whales: WhaleListResponse) {
  if (whales.total === 0 || whales.items.length === 0) return '0-0';
  const start = (whales.page - 1) * whales.page_size + 1;
  const end = start + whales.items.length - 1;
  return `${start.toLocaleString()}-${end.toLocaleString()}`;
}

function shortTime(raw: string, interval: string) {
  const date = new Date(raw);
  if (Number.isNaN(date.getTime())) return raw;
  if (interval === '5m') return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  return date.toLocaleDateString([], { month: 'short', day: 'numeric' });
}

function formatNewsTime(raw: string) {
  const date = new Date(raw);
  if (Number.isNaN(date.getTime())) return raw;
  return date.toLocaleString('zh-TW', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function figureNewsEmptyMessage(source: string) {
  if (source === 'crypto_figure_news_no_matches') return '目前沒有找到 Vitalik / Trump 的加密貨幣相關新聞。';
  if (source === 'crypto_figure_news_unavailable') return '重要人物新聞來源暫時不可用，稍後再試。';
  return '重要人物新聞載入中。';
}
function formatTaiwanDateTime(raw?: string) {
  if (!raw) return '';
  const date = new Date(raw);
  if (Number.isNaN(date.getTime())) return raw;

  const parts = new Intl.DateTimeFormat('zh-TW', {
    timeZone: 'Asia/Taipei',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).formatToParts(date);

  const value = (type: Intl.DateTimeFormatPartTypes) => parts.find((part) => part.type === type)?.value ?? '';
  return `${value('year')}/${value('month')}/${value('day')} ${value('hour')}:${value('minute')}:${value('second')}`;
}

function errorMessage(err: unknown) {
  return err instanceof Error ? err.message : '發生未知錯誤';
}

/** 由交易 from/to 即時彙整關聯地址（不需等待慢速 graph API） */
function buildTxCounterpartyGraphNodes(center: string, transactions: Transaction[]): GraphNode[] {
  const c = center.trim().toLowerCase();
  if (!c.startsWith('0x') || c.length !== 42) return [];
  const counts = new Map<string, number>();
  for (const tx of transactions) {
    const from = (tx.from || '').trim().toLowerCase();
    const to = (tx.to || '').trim().toLowerCase();
    if (from && from !== c) counts.set(from, (counts.get(from) || 0) + 1);
    if (to && to !== c) counts.set(to, (counts.get(to) || 0) + 1);
  }
  const centerNode: GraphNode = {
    id: c,
    label: c,
    is_center: true,
    is_contract: false,
    tx_count: transactions.length,
  };
  const others: GraphNode[] = Array.from(counts.entries())
    .sort((a, b) => b[1] - a[1])
    .map(([addr, n]) => ({
      id: addr,
      label: addr,
      is_center: false,
      is_contract: false,
      tx_count: n,
    }));
  return [centerNode, ...others];
}

function buildTransactionTimeline(transactions: Transaction[]) {
  const bucket = new Map<number, number>();
  for (const tx of transactions) {
    const ms = parseTransactionTimestampMs(tx.timestamp);
    if (ms === null) continue;
    const minuteBucketMs = Math.floor(ms / 60000) * 60000;
    bucket.set(minuteBucketMs, (bucket.get(minuteBucketMs) || 0) + 1);
  }
  return Array.from(bucket.entries())
    .sort((a, b) => a[0] - b[0])
    .slice(-12)
    .map(([time, count]) => ({ time: formatTimelineLabel(time), count }));
}

function parseTransactionTimestampMs(raw?: string) {
  const text = (raw || '').trim();
  if (!text) return null;

  if (/^\d+$/.test(text)) {
    const n = Number(text);
    if (!Number.isFinite(n)) return null;
    return n > 1e12 ? n : n * 1000;
  }

  const parsed = new Date(text).getTime();
  if (!Number.isFinite(parsed)) return null;
  return parsed;
}

function formatTimelineLabel(ms: number) {
  const parts = new Intl.DateTimeFormat('zh-TW', {
    timeZone: 'Asia/Taipei',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).formatToParts(new Date(ms));

  const value = (type: Intl.DateTimeFormatPartTypes) => parts.find((part) => part.type === type)?.value ?? '';
  return `${value('year')}/${value('month')}/${value('day')} ${value('hour')}:${value('minute')}`;
}

function buildTransactionCategoryDistribution(transactions: Transaction[]) {
  const categoryMap = new Map<string, { count: number; amount: number }>();
  let totalAmount = 0;
  for (const tx of transactions) {
    const raw = (tx.category || '').trim();
    const category = mapTransactionCategoryName(raw || 'uncategorized');
    const value = Number(tx.value);
    const amount = Number.isFinite(value) && value > 0 ? value : 0;
    const current = categoryMap.get(category) || { count: 0, amount: 0 };
    categoryMap.set(category, { count: current.count + 1, amount: current.amount + amount });
    totalAmount += amount;
  }

  return Array.from(categoryMap.entries())
    .map(([name, item]) => ({
      name,
      count: item.count,
      amount: item.amount,
      amountPct: totalAmount > 0 ? (item.amount / totalAmount) * 100 : 0,
    }))
    .sort((a, b) => b.amount - a.amount)
    .slice(0, 7);
}

function mapTransactionCategoryName(category: string) {
  switch (category.toLowerCase()) {
    case 'normal':
      return '一般轉帳';
    case 'internal':
      return '內部交易';
    case 'external':
      return '外部轉帳';
    case 'erc20':
    case 'token':
      return '代幣轉帳';
    case 'erc721':
      return 'NFT (ERC721)';
    case 'erc1155':
      return 'NFT (ERC1155)';
    case 'bridge':
      return '跨鏈';
    case 'defi':
    case 'defi_protocol':
      return 'DeFi 互動';
    case 'uncategorized':
      return '未分類';
    default:
      return category;
  }
}

function formatPercentForLegend(percent: number) {
  if (percent > 0 && percent < 0.1) return '<0.1%';
  return `${percent.toFixed(1)}%`;
}

function withTimeout<T>(promise: Promise<T>, timeoutMs: number, timeoutMessage: string): Promise<T> {
  return new Promise((resolve, reject) => {
    const timeoutId = setTimeout(() => reject(new Error(timeoutMessage)), timeoutMs);
    promise
      .then((value) => {
        clearTimeout(timeoutId);
        resolve(value);
      })
      .catch((err) => {
        clearTimeout(timeoutId);
        reject(err);
      });
  });
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export default App;
