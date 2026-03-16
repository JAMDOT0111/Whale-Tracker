import { useMemo } from 'react';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid } from 'recharts';
import type { Transaction } from '../types';

interface TransactionTimelineProps {
  transactions: Transaction[];
  centerAddress: string;
}

interface DayData {
  date: string;
  inCount: number;
  outCount: number;
  inValue: number;
  outValue: number;
}

export default function TransactionTimeline({ transactions, centerAddress }: TransactionTimelineProps) {
  const data = useMemo(() => {
    const center = centerAddress.toLowerCase();
    const dayMap = new Map<string, DayData>();

    for (const tx of transactions) {
      if (!tx.timestamp) continue;
      const date = tx.timestamp.slice(0, 10);
      if (!dayMap.has(date)) {
        dayMap.set(date, { date, inCount: 0, outCount: 0, inValue: 0, outValue: 0 });
      }
      const day = dayMap.get(date)!;
      const value = parseFloat(tx.value) || 0;
      const isOut = tx.from.toLowerCase() === center;

      if (isOut) {
        day.outCount++;
        day.outValue += value;
      } else {
        day.inCount++;
        day.inValue += value;
      }
    }

    return [...dayMap.values()].sort((a, b) => a.date.localeCompare(b.date));
  }, [transactions, centerAddress]);

  if (data.length === 0) return null;

  return (
    <div className="bg-gray-900 rounded-xl p-5 border border-gray-800">
      <h3 className="text-sm font-semibold text-white mb-4">交易時間軸</h3>
      <ResponsiveContainer width="100%" height={200}>
        <BarChart data={data} margin={{ top: 0, right: 0, left: -20, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
          <XAxis dataKey="date" tick={{ fill: '#6b7280', fontSize: 10 }} tickFormatter={(v: string) => v.slice(5)} />
          <YAxis tick={{ fill: '#6b7280', fontSize: 10 }} />
          <Tooltip
            contentStyle={{ backgroundColor: '#1f2937', border: '1px solid #374151', borderRadius: 8, fontSize: 12 }}
            labelStyle={{ color: '#e5e7eb' }}
            formatter={(value: number, name: string) => {
              const label = name === 'inCount' ? '轉入' : '轉出';
              return [value, label];
            }}
          />
          <Bar dataKey="inCount" fill="#34d399" name="inCount" radius={[2, 2, 0, 0]} />
          <Bar dataKey="outCount" fill="#f87171" name="outCount" radius={[2, 2, 0, 0]} />
        </BarChart>
      </ResponsiveContainer>
      <div className="flex justify-center gap-4 mt-2 text-xs text-gray-400">
        <span className="flex items-center gap-1.5">
          <span className="w-2.5 h-2.5 rounded-sm bg-emerald-400 inline-block" />
          轉入
        </span>
        <span className="flex items-center gap-1.5">
          <span className="w-2.5 h-2.5 rounded-sm bg-red-400 inline-block" />
          轉出
        </span>
      </div>
    </div>
  );
}
