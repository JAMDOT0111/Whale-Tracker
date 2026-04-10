import type { CandidateAddress, CandidateSummaryResponse } from '../types';

interface CandidateQueueProps {
  items: CandidateAddress[];
  summary: CandidateSummaryResponse | null;
  loading: boolean;
  refreshing: boolean;
  onRefresh: () => Promise<void>;
  onSelect: (address: string) => void;
}

export default function CandidateQueue({
  items,
  summary,
  loading,
  refreshing,
  onRefresh,
  onSelect,
}: CandidateQueueProps) {
  return (
    <section className="rounded-lg border border-slate-700 bg-[#20231f] p-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h2 className="text-sm font-semibold text-slate-200">值得詳細分析地址</h2>
          <p className="mt-1 text-xs leading-5 text-slate-500">
            先排除已知服務型地址，再依最近資金異動、歷史活躍度與協議互動挑出值得深挖的候選池。
          </p>
        </div>
        <button
          type="button"
          onClick={() => void onRefresh()}
          disabled={refreshing}
          className="rounded-lg border border-emerald-500 px-3 py-2 text-xs font-medium text-emerald-200 hover:bg-emerald-500/10 disabled:opacity-50"
        >
          {refreshing ? '掃描中...' : '重算候選池'}
        </button>
      </div>

      {summary && (
        <div className="mt-3 grid grid-cols-3 gap-2 text-xs">
          <SummaryCard label="可用池" value={summary.available_total.toLocaleString()} />
          <SummaryCard label="Review" value={summary.review_total.toLocaleString()} />
          <SummaryCard label="Watch" value={summary.watch_total.toLocaleString()} />
        </div>
      )}

      {summary && (
        <div className="mt-3 rounded-lg border border-slate-700 bg-[#171a18] px-3 py-3 text-xs text-slate-400">
          <p>{buildStatusText(summary)}</p>
          {summary.build.total > 0 && (
            <p className="mt-1">
              progress {summary.build.processed.toLocaleString()} / {summary.build.total.toLocaleString()}
            </p>
          )}
          {summary.refreshed_at && <p className="mt-1">最後快照 {formatTime(summary.refreshed_at)}</p>}
        </div>
      )}

      {loading ? (
        <p className="mt-4 text-xs text-slate-500">候選池資料載入中...</p>
      ) : items.length === 0 ? (
        <p className="mt-4 text-xs text-slate-500">目前沒有候選地址。先匯入 whale 名單，然後執行一次 full scan。</p>
      ) : (
        <div className="mt-4 space-y-3">
          {items.map((item) => {
            const displayTier = effectiveCandidateTier(item);
            return (
              <button
                key={item.address}
                type="button"
                onClick={() => onSelect(item.address)}
                className="block w-full rounded-lg border border-slate-700 px-3 py-3 text-left hover:border-emerald-500"
              >
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-mono text-sm text-slate-100">{shortAddress(item.address)}</span>
                      <span className={tierClass(displayTier)}>{displayTier}</span>
                    </div>
                    {item.name_tag && <p className="mt-1 text-xs text-slate-400">{item.name_tag}</p>}
                  </div>
                  <div className="text-right">
                    <p className="text-lg font-semibold text-emerald-200">{item.score}</p>
                    <p className="text-[11px] text-slate-500">score</p>
                  </div>
                </div>

                <div className="mt-3 grid grid-cols-2 gap-2 text-xs text-slate-300">
                  <StatLine label="Balance" value={`${formatETH(item.balance_eth)} ETH`} />
                  <StatLine label="7d tx" value={String(item.activity.tx_count_7d)} />
                  <StatLine label="7d netflow" value={formatSignedETH(item.activity.netflow_eth_7d)} />
                  <StatLine label="Largest tx" value={`${formatETH(item.activity.largest_tx_eth_7d)} ETH`} />
                </div>

                <div className="mt-3 flex flex-wrap gap-2 text-[11px] text-slate-400">
                  <span>protocol {item.activity.protocol_interactions_7d}</span>
                  <span>reactivated {item.activity.is_reactivated ? 'yes' : 'no'}</span>
                  <span>{item.activity.activity_loaded ? item.activity.activity_source : 'pending full activity scan'}</span>
                </div>
              </button>
            );
          })}
        </div>
      )}
    </section>
  );
}

function SummaryCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-slate-700 bg-[#171a18] px-3 py-2">
      <p className="text-[11px] text-slate-500">{label}</p>
      <p className="mt-1 text-sm font-semibold text-slate-100">{value}</p>
    </div>
  );
}

function StatLine({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-slate-700/70 bg-[#171a18] px-3 py-2">
      <p className="text-[11px] text-slate-500">{label}</p>
      <p className="mt-1 text-sm text-slate-100">{value}</p>
    </div>
  );
}

function tierClass(tier: string) {
  const base = 'rounded-full border px-2 py-0.5 text-[11px] font-medium';
  if (tier === 'review') return `${base} border-emerald-400/60 bg-emerald-500/10 text-emerald-200`;
  if (tier === 'watch') return `${base} border-amber-400/60 bg-amber-500/10 text-amber-200`;
  return `${base} border-slate-500/60 bg-slate-500/10 text-slate-300`;
}

function shortAddress(address: string) {
  if (!address || address.length <= 12) return address;
  return `${address.slice(0, 6)}...${address.slice(-4)}`;
}

function formatETH(value: string) {
  const n = Number(value);
  if (!Number.isFinite(n)) return value;
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}k`;
  return n.toLocaleString(undefined, { maximumFractionDigits: 2 });
}

function formatSignedETH(value: string) {
  const n = Number(value);
  if (!Number.isFinite(n)) return value;
  if (n > 0) return `+${formatETH(value)} ETH`;
  return `${formatETH(value)} ETH`;
}

function effectiveCandidateTier(item: CandidateAddress) {
  if (!hasRecentActivitySignal(item)) return 'backlog';
  return item.priority_tier;
}

function hasRecentActivitySignal(item: CandidateAddress) {
  if (!item.activity.activity_loaded) return false;
  return (
    item.activity.tx_count_7d >= 2 ||
    Math.abs(Number(item.activity.netflow_eth_7d) || 0) >= 300 ||
    (Number(item.activity.largest_tx_eth_7d) || 0) >= 150 ||
    item.activity.protocol_interactions_7d >= 1 ||
    item.activity.is_reactivated
  );
}

function formatTime(raw: string) {
  const date = new Date(raw);
  if (Number.isNaN(date.getTime())) return raw;
  return date.toLocaleString('zh-TW', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function buildStatusText(summary: CandidateSummaryResponse) {
  if (summary.build.status === 'queued' || summary.build.status === 'running') {
    const mode = summary.build.mode === 'incremental' ? 'incremental' : 'full';
    return `${mode} scan 進行中：${summary.build.message || '背景工作執行中'}`;
  }
  if (summary.build.status === 'failed') {
    return `scan 失敗：${summary.build.error || summary.build.message || 'unknown error'}`;
  }
  if (!summary.full_snapshot_ready) {
    return '尚未完成 full scan；目前清單只代表待補 recent activity 的 backlog，review/watch 會在 full 或 incremental 完成後才顯示。';
  }
  return `${summary.last_build_mode} snapshot 已就緒，已補 activity ${summary.activity_enriched_count.toLocaleString()} / ${summary.scan_limit.toLocaleString()}。`;
}
