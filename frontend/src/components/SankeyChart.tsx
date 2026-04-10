import { useEffect, useMemo, useRef, useState } from 'react';
import { sankey, sankeyCenter, sankeyLinkHorizontal } from 'd3-sankey';
import type { SankeyNode, SankeyLink } from 'd3-sankey';
import type { Transaction } from '../types';

interface SankeyChartProps {
  transactions: Transaction[];
  centerAddress: string;
}

interface SNode {
  id: string;
  name: string;
}

interface SLink {
  source: string;
  target: string;
  value: number;
}

const MAX_NODES = 15;
const CENTER_IN_ID = '__center_in__';
const CENTER_OUT_ID = '__center_out__';

export default function SankeyChart({ transactions, centerAddress }: SankeyChartProps) {
  const svgRef = useRef<SVGSVGElement>(null);
  const [renderError, setRenderError] = useState('');
  const center = centerAddress.toLowerCase();

  const { nodes: sankeyNodes, links: sankeyLinks, fallbackFlows } = useMemo(() => {
    try {
      const flowMap = new Map<string, number>();

      for (const tx of transactions) {
        const from = (tx.from || '').trim().toLowerCase();
        const to = (tx.to || '').trim().toLowerCase();
        const value = Number(tx.value);
        const asset = (tx.asset || '').trim().toUpperCase();
        if (!from || !to || from === to) continue;
        if (!from.startsWith('0x') || !to.startsWith('0x')) continue;
        if (!Number.isFinite(value) || value <= 0) continue;
        if (asset !== 'ETH') continue;
        if (from !== center && to !== center) continue;

        const source = from === center ? CENTER_OUT_ID : from;
        const target = to === center ? CENTER_IN_ID : to;
        if (source === target) continue;

        const key = `${source}|${target}`;
        flowMap.set(key, (flowMap.get(key) || 0) + value);
      }

      if (flowMap.size === 0) return { nodes: [] as SNode[], links: [] as SLink[], fallbackFlows: [] as SLink[] };

      const addressValues = new Map<string, number>();
      for (const [key, val] of flowMap) {
        const [from, to] = key.split('|');
        if (from !== CENTER_OUT_ID && from !== CENTER_IN_ID) {
          addressValues.set(from, (addressValues.get(from) || 0) + val);
        }
        if (to !== CENTER_OUT_ID && to !== CENTER_IN_ID) {
          addressValues.set(to, (addressValues.get(to) || 0) + val);
        }
      }

      const selectedAddresses = new Set<string>();
      [...addressValues.entries()]
        .sort((a, b) => b[1] - a[1])
        .slice(0, MAX_NODES - 2)
        .forEach(([addr]) => selectedAddresses.add(addr));

      const allowed = new Set<string>([CENTER_IN_ID, CENTER_OUT_ID, ...selectedAddresses]);
      const shortenAddr = (addr: string) => {
        if (addr === CENTER_IN_ID) return '中心流入';
        if (addr === CENTER_OUT_ID) return '中心流出';
        return `${addr.slice(0, 6)}...${addr.slice(-4)}`;
      };

      const links = [...flowMap.entries()]
        .map(([key, value]) => {
          const [source, target] = key.split('|');
          return { source, target, value };
        })
        .filter((l) => allowed.has(l.source) && allowed.has(l.target))
        .sort((a, b) => b.value - a.value);

      const nodes = [...allowed].map((id) => ({ id, name: shortenAddr(id) }));
      return { nodes, links, fallbackFlows: links.slice(0, 8) };
    } catch (error) {
      console.error('[SankeyChart] data build failed', error);
      return { nodes: [] as SNode[], links: [] as SLink[], fallbackFlows: [] as SLink[] };
    }
  }, [transactions, center]);

  useEffect(() => {
    setRenderError('');
    if (!svgRef.current || sankeyNodes.length === 0 || sankeyLinks.length === 0) return;

    try {
      const width = svgRef.current.clientWidth || 800;
      const height = 300;

      const generator = sankey<SNode, SLink>()
        .nodeId((d) => d.id)
        .nodeAlign(sankeyCenter)
        .nodeWidth(12)
        .nodePadding(8)
        .extent([
          [1, 1],
          [width - 1, height - 5],
        ]);

      const { nodes, links } = generator({
        nodes: sankeyNodes.map((d) => ({ ...d })),
        links: sankeyLinks.map((d) => ({ ...d })),
      });

      const svg = svgRef.current;
      svg.innerHTML = '';
      svg.setAttribute('viewBox', `0 0 ${width} ${height}`);

      type SankeyNodeExt = SankeyNode<SNode, SLink>;
      type SankeyLinkExt = SankeyLink<SNode, SLink>;

      for (const link of links as SankeyLinkExt[]) {
        const path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
        path.setAttribute('d', sankeyLinkHorizontal()(link as never) || '');
        path.setAttribute('fill', 'none');
        path.setAttribute('stroke', (link.source as SankeyNodeExt).id === CENTER_OUT_ID ? '#f87171' : '#34d399');
        path.setAttribute('stroke-opacity', '0.45');
        path.setAttribute('stroke-width', String(Math.min(28, Math.max(1, link.width || 1))));
        svg.appendChild(path);
      }

      for (const node of nodes as SankeyNodeExt[]) {
        const rect = document.createElementNS('http://www.w3.org/2000/svg', 'rect');
        rect.setAttribute('x', String(node.x0 || 0));
        rect.setAttribute('y', String(node.y0 || 0));
        rect.setAttribute('width', String((node.x1 || 0) - (node.x0 || 0)));
        rect.setAttribute('height', String(Math.max(1, (node.y1 || 0) - (node.y0 || 0))));
        rect.setAttribute('fill', node.id === CENTER_IN_ID || node.id === CENTER_OUT_ID ? '#f59e0b' : '#6366f1');
        rect.setAttribute('rx', '2');
        svg.appendChild(rect);

        const text = document.createElementNS('http://www.w3.org/2000/svg', 'text');
        const isLeft = (node.x0 || 0) < width / 2;
        text.setAttribute('x', String(isLeft ? (node.x1 || 0) + 6 : (node.x0 || 0) - 6));
        text.setAttribute('y', String(((node.y0 || 0) + (node.y1 || 0)) / 2));
        text.setAttribute('dy', '0.35em');
        text.setAttribute('text-anchor', isLeft ? 'start' : 'end');
        text.setAttribute('fill', '#9ca3af');
        text.setAttribute('font-size', '10');
        text.setAttribute('font-family', 'ui-monospace, monospace');
        text.textContent = node.name;
        svg.appendChild(text);
      }
    } catch (error) {
      console.error('[SankeyChart] render failed', error);
      setRenderError('資金流向圖資料暫時無法繪製。');
    }
  }, [sankeyNodes, sankeyLinks, center]);

  if (sankeyLinks.length === 0) {
    return (
      <div className="bg-gray-900 rounded-xl p-5 border border-gray-800">
        <h3 className="text-sm font-semibold text-white mb-2">資金流向圖 (ETH)</h3>
        <p className="text-xs text-slate-500">尚無可用 ETH 資金流向資料。</p>
      </div>
    );
  }

  return (
    <div className="bg-gray-900 rounded-xl p-5 border border-gray-800">
      <h3 className="text-sm font-semibold text-white mb-4">資金流向圖 (ETH)</h3>
      <svg ref={svgRef} className="w-full" style={{ height: 300 }} />
      <div className="flex justify-center gap-4 mt-2 text-xs text-gray-400">
        <span className="flex items-center gap-1.5">
          <span className="w-2.5 h-2.5 rounded-sm bg-emerald-400 inline-block" />
          轉入
        </span>
        <span className="flex items-center gap-1.5">
          <span className="w-2.5 h-2.5 rounded-sm bg-red-400 inline-block" />
          轉出
        </span>
        <span className="flex items-center gap-1.5">
          <span className="w-2.5 h-2.5 rounded-sm bg-amber-500 inline-block" />
          中心地址
        </span>
      </div>
      {renderError && (
        <div className="mt-3 rounded border border-amber-500/40 bg-amber-500/10 p-2 text-xs text-amber-300">
          {renderError}
          {fallbackFlows.length > 0 && (
            <div className="mt-2 space-y-1 text-slate-300">
              {fallbackFlows.slice(0, 6).map((flow, index) => (
                <div key={`${flow.source}-${flow.target}-${index}`}>
                  {flow.source === CENTER_OUT_ID ? '中心流出' : `${flow.source.slice(0, 6)}...${flow.source.slice(-4)}`} {'->'}{' '}
                  {flow.target === CENTER_IN_ID ? '中心流入' : `${flow.target.slice(0, 6)}...${flow.target.slice(-4)}`} ·{' '}
                  {flow.value.toFixed(4)} ETH
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
