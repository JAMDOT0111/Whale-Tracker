import { useEffect, useRef } from 'react';
import cytoscape from 'cytoscape';
import type { GraphResponse } from '../types';

interface GraphViewProps {
  data: GraphResponse | null;
  onNodeClick?: (address: string) => void;
  onNodeRightClick?: (nodeId: string, x: number, y: number) => void;
  markedAddresses?: Set<string>;
  customNames?: Record<string, string>;
  embedded?: boolean;
  containerClassName?: string;
}

export default function GraphView({
  data,
  onNodeClick,
  onNodeRightClick,
  markedAddresses,
  customNames,
  embedded = false,
  containerClassName = 'h-[600px]',
}: GraphViewProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const cyRef = useRef<cytoscape.Core | null>(null);
  const onNodeClickRef = useRef(onNodeClick);
  const onNodeRightClickRef = useRef(onNodeRightClick);

  onNodeClickRef.current = onNodeClick;
  onNodeRightClickRef.current = onNodeRightClick;

  // Only rebuild graph when data changes
  useEffect(() => {
    if (!containerRef.current || !data || data.nodes.length === 0) return;

    if (cyRef.current) {
      cyRef.current.destroy();
    }

    const maxTxCount = Math.max(...data.nodes.map((n) => n.tx_count), 1);
    const maxEdgeValue = Math.max(...data.edges.map((e) => parseFloat(e.value) || 0), 1);
    const marks = markedAddresses || new Set<string>();
    const names = customNames || {};

    const elements: cytoscape.ElementDefinition[] = [
      ...data.nodes.map((node) => ({
        data: {
          id: node.id,
          label: names[node.id] || node.label,
          originalLabel: node.label,
          isCenter: node.is_center,
          isContract: node.is_contract,
          tag: node.tag || '',
          tagName: node.tag_name || '',
          isExchange: node.tag === 'exchange',
          isBridge: node.tag === 'bridge',
          isMarked: marks.has(node.id),
          txCount: node.tx_count,
          size: 30 + (node.tx_count / maxTxCount) * 50,
        },
      })),
      ...data.edges.map((edge, i) => ({
        data: {
          id: `e${i}`,
          source: edge.source,
          target: edge.target,
          value: edge.value,
          txCount: edge.tx_count,
          width: 1 + (parseFloat(edge.value) / maxEdgeValue) * 6,
          label: `${edge.tx_count} tx`,
        },
      })),
    ];

    const cy = cytoscape({
      container: containerRef.current,
      elements,
      style: [
        {
          selector: 'node',
          style: {
            label: 'data(label)',
            'background-color': '#6366f1',
            width: 'data(size)',
            height: 'data(size)',
            color: '#e5e7eb',
            'font-size': '10px',
            'text-valign': 'bottom',
            'text-margin-y': 8,
            'font-family': 'ui-monospace, monospace',
            'border-width': 2,
            'border-color': '#4f46e5',
          },
        },
        {
          selector: 'node[?isExchange]',
          style: {
            'background-color': '#f43f5e',
            'border-color': '#e11d48',
            shape: 'round-diamond',
            'font-size': '11px',
          },
        },
        {
          selector: 'node[?isBridge]',
          style: {
            'background-color': '#06b6d4',
            'border-color': '#0891b2',
            shape: 'round-hexagon' as cytoscape.Css.NodeShape,
            'font-size': '11px',
          },
        },
        {
          selector: 'node[?isContract]',
          style: {
            'background-color': '#10b981',
            'border-color': '#059669',
            shape: 'round-rectangle',
          },
        },
        {
          selector: 'node[?isCenter]',
          style: {
            'background-color': '#f59e0b',
            'border-color': '#d97706',
            'border-width': 3,
            'font-size': '12px',
            'font-weight': 'bold' as const,
            color: '#fbbf24',
          },
        },
        {
          selector: 'node[?isMarked]',
          style: {
            'border-color': '#facc15',
            'border-width': 4,
            'border-style': 'double',
          } as unknown as cytoscape.Css.Node,
        },
        {
          selector: 'edge',
          style: {
            width: 'data(width)',
            'line-color': '#4b5563',
            'target-arrow-color': '#4b5563',
            'target-arrow-shape': 'triangle',
            'curve-style': 'bezier',
            label: 'data(label)',
            'font-size': '8px',
            color: '#9ca3af',
            'text-rotation': 'autorotate',
            'text-margin-y': -10,
          },
        },
        {
          selector: 'node:selected',
          style: {
            'background-color': '#22d3ee',
            'border-color': '#06b6d4',
          },
        },
      ],
      layout: {
        name: 'cose',
        animate: true,
        animationDuration: 800,
        nodeRepulsion: () => 8000,
        idealEdgeLength: () => 150,
        gravity: 0.3,
        padding: 50,
      } as cytoscape.LayoutOptions,
      minZoom: 0.2,
      maxZoom: 3,
    });

    cy.on('tap', 'node', (evt) => {
      const nodeId = evt.target.id();
      onNodeClickRef.current?.(nodeId);
    });

    cy.on('cxttap', 'node', (evt) => {
      evt.originalEvent.preventDefault();
      const nodeId = evt.target.id();
      const { clientX, clientY } = evt.originalEvent as MouseEvent;
      onNodeRightClickRef.current?.(nodeId, clientX, clientY);
    });

    containerRef.current.addEventListener('contextmenu', (e) => e.preventDefault());

    cyRef.current = cy;

    return () => {
      if (cyRef.current) {
        cyRef.current.destroy();
        cyRef.current = null;
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data]);

  // Update marks and names without rebuilding graph
  useEffect(() => {
    const cy = cyRef.current;
    if (!cy || !data) return;

    const marks = markedAddresses || new Set<string>();
    const names = customNames || {};

    data.nodes.forEach((node) => {
      const el = cy.getElementById(node.id);
      if (el.length === 0) return;

      el.data('isMarked', marks.has(node.id));
      el.data('label', names[node.id] || node.label);
    });
  }, [markedAddresses, customNames, data]);

  if (!data || data.nodes.length === 0) {
    return (
      <div className={`${containerClassName} flex items-center justify-center text-gray-500 border border-gray-700 rounded-lg bg-gray-900/50`}>
        掃描地址後，這裡會顯示地址關係圖。
      </div>
    );
  }

  if (embedded) {
    return <div ref={containerRef} className={`${containerClassName} border border-gray-700 rounded-lg bg-gray-900/80`} />;
  }

  return (
    <div className="space-y-3">
      <div>
        <h2 className="text-lg font-semibold text-white">
          地址關聯圖譜
          <span className="text-gray-400 font-normal text-sm ml-2">
            ({data.nodes.length} 個節點, {data.edges.length} 條邊)
          </span>
        </h2>
      </div>
      <div ref={containerRef} className={`${containerClassName} border border-gray-700 rounded-lg bg-gray-900/80`} />
      <div className="flex flex-wrap gap-3 text-xs text-gray-400">
        <span className="flex items-center gap-1.5">
          <span className="w-3 h-3 rounded-full bg-amber-500 inline-block" />
          中心地址
        </span>
        <span className="flex items-center gap-1.5">
          <span className="w-3 h-3 rounded-full bg-indigo-500 inline-block" />
          一般地址
        </span>
        <span className="flex items-center gap-1.5">
          <span className="w-3 h-3 rounded bg-emerald-500 inline-block" />
          合約
        </span>
        <span className="flex items-center gap-1.5">
          <span className="w-3 h-3 rotate-45 bg-rose-500 inline-block" />
          交易所
        </span>
        <span className="flex items-center gap-1.5">
          <span className="w-3 h-3 rounded-sm bg-cyan-500 inline-block" />
          跨鏈橋
        </span>
      </div>
      <p className="text-gray-500 text-xs text-center">左鍵點擊節點導航 / 右鍵選單操作 / 拖拉移動 / 滾輪縮放</p>
    </div>
  );
}
