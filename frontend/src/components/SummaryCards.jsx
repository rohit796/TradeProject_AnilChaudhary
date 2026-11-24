import React from 'react';
import { TrendingUp, TrendingDown, DollarSign, Layers, ArrowUpRight, ArrowDownRight, Hash } from 'lucide-react';

export default function SummaryCards({ trades = [], totalCount = 0 }) {
  // Compute real-time aggregates from the trades
  const displayTotal = totalCount > 0 ? totalCount : trades.length;

  const buyTrades = trades.filter(t => t.trade_type === 'BUY');
  const sellTrades = trades.filter(t => t.trade_type === 'SELL');

  const totalBuyQty = buyTrades.reduce((acc, t) => acc + Number(t.quantity || 0), 0);
  const totalSellQty = sellTrades.reduce((acc, t) => acc + Number(t.quantity || 0), 0);

  const totalBuyValue = buyTrades.reduce((acc, t) => acc + Number(t.trade_value || (t.quantity * t.price) || 0), 0);
  const totalSellValue = sellTrades.reduce((acc, t) => acc + Number(t.trade_value || (t.quantity * t.price) || 0), 0);

  const netTurnover = totalBuyValue + totalSellValue;
  const netPosition = totalBuyValue - totalSellValue;

  const formatCurrency = (val) => {
    return new Intl.NumberFormat('en-IN', {
      style: 'currency',
      currency: 'INR',
      maximumFractionDigits: 0
    }).format(val);
  };

  const cards = [
    {
      title: 'Total Executed Trades',
      value: displayTotal.toLocaleString(),
      subtext: totalCount > trades.length
        ? `${trades.length} loaded (${buyTrades.length} Buy / ${sellTrades.length} Sell)`
        : `${buyTrades.length} Buy / ${sellTrades.length} Sell Orders`,
      icon: Hash,
      color: 'blue',
      glow: 'from-blue-600/20 to-indigo-600/5',
      badge: totalCount > 0 ? `Total: ${totalCount.toLocaleString()}` : 'Real-Time'
    },
    {
      title: 'Total Buy Turnover',
      value: formatCurrency(totalBuyValue),
      subtext: `${totalBuyQty.toLocaleString()} Shares Purchased`,
      icon: TrendingUp,
      color: 'emerald',
      glow: 'from-emerald-600/20 to-teal-600/5',
      badge: `${buyTrades.length} Buys`
    },
    {
      title: 'Total Sell Turnover',
      value: formatCurrency(totalSellValue),
      subtext: `${totalSellQty.toLocaleString()} Shares Offloaded`,
      icon: TrendingDown,
      color: 'rose',
      glow: 'from-rose-600/20 to-pink-600/5',
      badge: `${sellTrades.length} Sells`
    },
    {
      title: 'Total Gross Volume',
      value: formatCurrency(netTurnover),
      subtext: `Net Flow: ${netPosition >= 0 ? '+' : ''}${formatCurrency(netPosition)}`,
      icon: DollarSign,
      color: 'violet',
      glow: 'from-violet-600/20 to-purple-600/5',
      badge: 'Cumulative'
    }
  ];

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
      {cards.map((c, i) => {
        const Icon = c.icon;
        return (
          <div
            key={i}
            className={`relative overflow-hidden rounded-xl bg-[#0F172A] border border-slate-800/80 p-3 sm:p-3.5 shadow-md hover:border-slate-700 transition-all`}
          >
            <div className={`absolute top-0 right-0 w-24 h-24 bg-gradient-to-br ${c.glow} rounded-bl-full pointer-events-none`} />
            
            <div className="flex items-center justify-between mb-1.5">
              <span className="text-[11px] font-medium text-slate-400">{c.title}</span>
              <span className={`text-[10px] font-semibold px-2 py-0.2 rounded-full border ${
                c.color === 'emerald' ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20' :
                c.color === 'rose' ? 'bg-rose-500/10 text-rose-400 border-rose-500/20' :
                c.color === 'violet' ? 'bg-violet-500/10 text-violet-400 border-violet-500/20' :
                'bg-blue-500/10 text-blue-400 border-blue-500/20'
              }`}>
                {c.badge}
              </span>
            </div>

            <div className="flex items-baseline justify-between">
              <div>
                <h3 className="text-xl font-bold text-white tracking-tight font-mono">{c.value}</h3>
                <p className="text-[11px] text-slate-400 mt-0.5">{c.subtext}</p>
              </div>
              <div className={`p-2 rounded-lg ${
                c.color === 'emerald' ? 'bg-emerald-500/10 text-emerald-400' :
                c.color === 'rose' ? 'bg-rose-500/10 text-rose-400' :
                c.color === 'violet' ? 'bg-violet-500/10 text-violet-400' :
                'bg-blue-500/10 text-blue-400'
              }`}>
                <Icon className="w-4 h-4" />
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}
