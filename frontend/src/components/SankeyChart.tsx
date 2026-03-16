import { useMemo, useRef, useEffect } from 'react';
import { sankey, sankeyLinkHorizontal } from 'd3-sankey';
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
  source: number;
  target: number;
  value: number;
}

const MAX_NODES = 15;

export default function SankeyChart({ transactions, centerAddress }: SankeyChartProps) {
  const svgRef = useRef<SVGSVGElement>(null);
  const center = centerAddress.toLowerCase();

  const { nodes: sankeyNodes, links: sankeyLinks } = useMemo(() => {
    const flowMap = new Map<string, number>();

    for (const tx of transactions) {
      const from = tx.from.toLowerCase();
      const to = tx.to.toLowerCase();
      const value = parseFloat(tx.value) || 0;
      if (value <= 0 || !from || !to || from === to) continue;
      if (tx.asset !== 'ETH') continue;

      const key = `${from}|${to}`;
      flowMap.set(key, (flowMap.get(key) || 0) + value);
    }

    const addressValues = new Map<string, number>();
    for (const [key, val] of flowMap) {
      const [from, to] = key.split('|');
      if (from !== center) addressValues.set(from, (addressValues.get(from) || 0) + val);
      if (to !== center) addressValues.set(to, (addressValues.get(to) || 0) + val);
    }

    const topAddresses = new Set([center]);
    [...addressValues.entries()]
      .sort((a, b) => b[1] - a[1])
      .slice(0, MAX_NODES - 1)
      .forEach(([addr]) => topAddresses.add(addr));

    const shortenAddr = (addr: string) => (addr === center ? 'Center' : addr.slice(0, 6) + '...' + addr.slice(-4));

    const nodeList: SNode[] = [];
    const nodeIndex = new Map<string, number>();

    for (const addr of topAddresses) {
      nodeIndex.set(addr, nodeList.length);
      nodeList.push({ id: addr, name: shortenAddr(addr) });
    }

    const linkList: SLink[] = [];
    for (const [key, val] of flowMap) {
      const [from, to] = key.split('|');
      if (!nodeIndex.has(from) || !nodeIndex.has(to)) continue;
      linkList.push({
        source: nodeIndex.get(from)!,
        target: nodeIndex.get(to)!,
        value: val,
      });
    }

    return { nodes: nodeList, links: linkList };
  }, [transactions, center]);

  useEffect(() => {
    if (!svgRef.current || sankeyNodes.length === 0 || sankeyLinks.length === 0) return;

    const width = svgRef.current.clientWidth || 800;
    const height = 300;

    const generator = sankey<SNode, SLink>()
      .nodeId((_d, i) => i)
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
      path.setAttribute('stroke', (link.source as SankeyNodeExt).id === center ? '#f87171' : '#34d399');
      path.setAttribute('stroke-opacity', '0.4');
      path.setAttribute('stroke-width', String(Math.max(1, link.width || 1)));
      svg.appendChild(path);
    }

    for (const node of nodes as SankeyNodeExt[]) {
      const rect = document.createElementNS('http://www.w3.org/2000/svg', 'rect');
      rect.setAttribute('x', String(node.x0 || 0));
      rect.setAttribute('y', String(node.y0 || 0));
      rect.setAttribute('width', String((node.x1 || 0) - (node.x0 || 0)));
      rect.setAttribute('height', String(Math.max(1, (node.y1 || 0) - (node.y0 || 0))));
      rect.setAttribute('fill', node.id === center ? '#f59e0b' : '#6366f1');
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
  }, [sankeyNodes, sankeyLinks, center]);

  if (sankeyLinks.length === 0) return null;

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
    </div>
  );
}
