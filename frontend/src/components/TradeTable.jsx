import React from 'react';
import { ArrowUpDown, CheckCircle, ChevronLeft, ChevronRight, Database } from 'lucide-react';

export default function TradeTable({
  trades = [],
  totalCount = 0,
  loading,
  filters,
  setFilters
}) {
  const currentPage = filters?.page || 1;
  const currentLimit = filters?.limit || 100;
  const effectiveTotal = totalCount > 0 ? totalCount : trades.length;
  const totalPages = Math.max(1, Math.ceil(effectiveTotal / currentLimit));

  const handlePageChange = (newPage) => {
    if (newPage < 1 || newPage > totalPages) return;
    setFilters(prev => ({ ...prev, page: newPage }));
  };

  const handleLimitChange = (newLimit) => {
    setFilters(prev => ({ ...prev, limit: newLimit, page: 1 }));
  };

  const startRecord = effectiveTotal > 0 ? (currentPage - 1) * currentLimit + 1 : 0;
  const endRecord = Math.min(startRecord + trades.length - 1, effectiveTotal);
  const handleSort = (field) => {
    if (filters.sort_by === field) {
      setFilters(prev => ({
        ...prev,
        order: prev.order === 'asc' ? 'desc' : 'asc'
      }));
    } else {
      setFilters(prev => ({
        ...prev,
        sort_by: field,
        order: 'desc'
      }));
    }
  };

  const formatNumber = (num, decimals = 2) => {
    return Number(num || 0).toLocaleString('en-IN', {
      minimumFractionDigits: decimals,
      maximumFractionDigits: decimals
    });
  };

  const formatDateTime = (dateStr) => {
    if (!dateStr) return '-';
    try {
      const d = new Date(dateStr);
      return d.toLocaleString('en-IN', {
        month: 'short',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: true
      });
    } catch {
      return dateStr;
    }
  };

  return (
    <div className="flex-1 min-h-0 bg-[#0F172A] border border-slate-800 rounded-xl shadow-xl flex flex-col overflow-hidden">
      {/* Scrollable Container - ONLY this body scrolls */}
      <div className="flex-1 min-h-0 overflow-y-auto overflow-x-auto relative">
        <table className="w-full text-left border-collapse text-xs">
          <thead className="sticky top-0 z-20 bg-slate-900 shadow-sm border-b border-slate-800">
            <tr className="bg-slate-900 text-slate-400 font-semibold tracking-wider uppercase text-[11px]">
              <th className="py-3 px-4 bg-slate-900 cursor-pointer hover:text-white" onClick={() => handleSort('trade_id')}>
                <div className="flex items-center gap-1.5">
                  <span>Trade ID</span>
                  <ArrowUpDown className="w-3 h-3 text-slate-500" />
                </div>
              </th>
              <th className="py-3 px-4 bg-slate-900 cursor-pointer hover:text-white" onClick={() => handleSort('trade_time')}>
                <div className="flex items-center gap-1.5">
                  <span>Execution Time</span>
                  <ArrowUpDown className="w-3 h-3 text-slate-500" />
                </div>
              </th>
              <th className="py-3 px-4 bg-slate-900">Client Details</th>
              <th className="py-3 px-4 bg-slate-900 cursor-pointer hover:text-white" onClick={() => handleSort('symbol')}>
                <div className="flex items-center gap-1.5">
                  <span>Instrument / Stock</span>
                  <ArrowUpDown className="w-3 h-3 text-slate-500" />
                </div>
              </th>
              <th className="py-3 px-4 bg-slate-900 cursor-pointer hover:text-white" onClick={() => handleSort('trade_type')}>
                <div className="flex items-center gap-1.5">
                  <span>Side</span>
                  <ArrowUpDown className="w-3 h-3 text-slate-500" />
                </div>
              </th>
              <th className="py-3 px-4 bg-slate-900 text-right cursor-pointer hover:text-white" onClick={() => handleSort('quantity')}>
                <div className="flex items-center justify-end gap-1.5">
                  <span>Quantity</span>
                  <ArrowUpDown className="w-3 h-3 text-slate-500" />
                </div>
              </th>
              <th className="py-3 px-4 bg-slate-900 text-right cursor-pointer hover:text-white" onClick={() => handleSort('price')}>
                <div className="flex items-center justify-end gap-1.5">
                  <span>Trade Price (₹)</span>
                  <ArrowUpDown className="w-3 h-3 text-slate-500" />
                </div>
              </th>
              <th className="py-3 px-4 bg-slate-900 text-right cursor-pointer hover:text-white" onClick={() => handleSort('trade_value')}>
                <div className="flex items-center justify-end gap-1.5">
                  <span>Trade Value (₹)</span>
                  <ArrowUpDown className="w-3 h-3 text-slate-500" />
                </div>
              </th>
              <th className="py-3 px-4 bg-slate-900 text-center">Status</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-800/60 font-sans">
            {loading ? (
              <tr>
                <td colSpan="9" className="py-12 text-center text-slate-400">
                  <div className="flex flex-col items-center justify-center gap-2">
                    <div className="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
                    <span>Loading trade records...</span>
                  </div>
                </td>
              </tr>
            ) : trades.length === 0 ? (
              <tr>
                <td colSpan="9" className="py-12 text-center text-slate-400">
                  <div className="flex flex-col items-center justify-center gap-2">
                    <Database className="w-8 h-8 text-slate-600" />
                    <p className="text-slate-300 font-medium">No Trade Reports Found</p>
                    <p className="text-slate-500 text-xs">Adjust your search filters or click "New Trade" to create one.</p>
                  </div>
                </td>
              </tr>
            ) : (
              trades.map((trade) => {
                const isBuy = trade.trade_type === 'BUY';
                return (
                  <tr
                    key={trade.trade_id}
                    className="hover:bg-slate-800/40 transition-colors group"
                  >
                    {/* Trade ID */}
                    <td className="py-3.5 px-4 font-mono font-medium text-slate-200">
                      <div className="flex flex-col">
                        <span className="text-blue-400 group-hover:underline cursor-pointer">{trade.trade_id}</span>
                        <span className="text-[10px] text-slate-500 font-sans">{trade.order_id || 'Direct'}</span>
                      </div>
                    </td>

                    {/* Execution Time */}
                    <td className="py-3.5 px-4 text-slate-300 whitespace-nowrap">
                      {formatDateTime(trade.trade_time)}
                    </td>

                    {/* Client */}
                    <td className="py-3.5 px-4">
                      <div className="flex flex-col">
                        <span className="font-medium text-white">{trade.client_name || trade.first_name ? `${trade.first_name} ${trade.last_name || ''}` : trade.client_id}</span>
                        <div className="flex items-center gap-1.5 mt-0.5">
                          <span className="font-mono text-[10px] text-slate-400">{trade.client_code || trade.client_id}</span>
                          {trade.account_type && (
                            <span className="text-[9px] px-1.5 py-0.2 bg-slate-800 text-slate-400 rounded border border-slate-700">
                              {trade.account_type}
                            </span>
                          )}
                        </div>
                      </div>
                    </td>

                    {/* Stock Symbol & Master info */}
                    <td className="py-3.5 px-4">
                      <div className="flex flex-col">
                        <div className="flex items-center gap-1.5">
                          <span className="font-mono font-bold text-slate-100">{trade.symbol}</span>
                          <span className="text-[10px] px-1 py-0.2 bg-blue-900/40 text-blue-300 rounded font-mono">
                            {trade.exchange || 'NSE'}
                          </span>
                        </div>
                        <span className="text-[10px] text-slate-400 truncate max-w-[180px]">
                          {trade.company_name || trade.sector || 'Equity'}
                        </span>
                      </div>
                    </td>

                    {/* Side (BUY / SELL) */}
                    <td className="py-3.5 px-4">
                      <span className={`inline-flex items-center px-2.5 py-1 rounded-full text-[11px] font-bold tracking-wide ${
                        isBuy
                          ? 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30'
                          : 'bg-rose-500/15 text-rose-400 border border-rose-500/30'
                      }`}>
                        {trade.trade_type}
                      </span>
                    </td>

                    {/* Quantity */}
                    <td className="py-3.5 px-4 text-right font-mono font-medium text-slate-200">
                      {formatNumber(trade.quantity, 0)}
                    </td>

                    {/* Price */}
                    <td className="py-3.5 px-4 text-right font-mono font-medium text-slate-200">
                      ₹{formatNumber(trade.price, 2)}
                    </td>

                    {/* Trade Value */}
                    <td className="py-3.5 px-4 text-right font-mono font-semibold text-white">
                      ₹{formatNumber(trade.trade_value || (trade.quantity * trade.price), 2)}
                    </td>

                    {/* Execution Status */}
                    <td className="py-3.5 px-4 text-center">
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-medium bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                        <CheckCircle className="w-2.5 h-2.5" /> Executed
                      </span>
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>

      {/* Footer & Interactive Pagination Bar */}
      <div className="flex-shrink-0 px-4 py-2 bg-slate-900/95 border-t border-slate-800 flex flex-wrap items-center justify-between gap-3 text-xs text-slate-400">
        {/* Left: Record Range & Rows per page selector */}
        <div className="flex items-center gap-3">
          <span>
            Showing <strong className="text-white font-mono">{startRecord}</strong> - <strong className="text-white font-mono">{endRecord}</strong> of <strong className="text-white font-mono">{effectiveTotal.toLocaleString()}</strong> trade records
          </span>

          <div className="flex items-center gap-1.5 pl-3 border-l border-slate-800">
            <span className="text-[11px] text-slate-500">Rows:</span>
            <select
              value={currentLimit}
              onChange={(e) => handleLimitChange(Number(e.target.value))}
              className="bg-slate-800 border border-slate-700 text-slate-200 text-xs rounded-lg px-2 py-0.5 focus:outline-none focus:ring-1 focus:ring-blue-500 cursor-pointer font-mono"
            >
              <option value={50}>50</option>
              <option value={100}>100</option>
              <option value={250}>250</option>
              <option value={500}>500</option>
              <option value={1200}>All (1,200)</option>
            </select>
          </div>
        </div>

        {/* Right: Page Navigation & Live Status */}
        <div className="flex items-center gap-2">
          <button
            onClick={() => handlePageChange(currentPage - 1)}
            disabled={currentPage <= 1 || loading}
            title="Previous Page"
            className="px-2.5 py-1 bg-slate-800 hover:bg-slate-700 disabled:opacity-30 disabled:hover:bg-slate-800 text-slate-200 rounded-lg text-xs transition-all flex items-center gap-1 cursor-pointer disabled:cursor-not-allowed"
          >
            <ChevronLeft className="w-3.5 h-3.5" />
            <span className="hidden sm:inline">Prev</span>
          </button>

          <span className="px-2 text-xs font-mono text-slate-300">
            Page <strong className="text-white">{currentPage}</strong> of <strong className="text-white">{totalPages}</strong>
          </span>

          <button
            onClick={() => handlePageChange(currentPage + 1)}
            disabled={currentPage >= totalPages || loading}
            title="Next Page"
            className="px-2.5 py-1 bg-slate-800 hover:bg-slate-700 disabled:opacity-30 disabled:hover:bg-slate-800 text-slate-200 rounded-lg text-xs transition-all flex items-center gap-1 cursor-pointer disabled:cursor-not-allowed"
          >
            <span className="hidden sm:inline">Next</span>
            <ChevronRight className="w-3.5 h-3.5" />
          </button>

          <span className="inline-flex items-center gap-1.5 pl-3 border-l border-slate-800 text-slate-400 text-[11px]">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
            <span className="hidden sm:inline">Live Pipeline</span>
          </span>
        </div>
      </div>
    </div>
  );
}
