import React from 'react';
import { Search, RotateCw, Download, Plus } from 'lucide-react';

export default function FilterBar({
  filters,
  setFilters,
  clients = [],
  stocks = [],
  onFetch,
  onRefresh,
  onOpenNewTrade,
  onExportCSV,
  loading
}) {
  const handleChange = (field, value) => {
    setFilters(prev => ({ ...prev, [field]: value }));
  };

  const handleReset = () => {
    const cleanFilters = {
      search: '',
      client_id: '',
      symbol: '',
      trade_type: 'ALL',
      sort_by: 'trade_time',
      order: 'desc'
    };
    setFilters(cleanFilters);
    if (onFetch) onFetch(cleanFilters);
  };

  return (
    <div className="bg-[#0F172A] border border-slate-800 rounded-xl p-2.5 sm:p-3 shadow-md">
      <div className="flex flex-wrap lg:flex-nowrap items-center justify-between gap-2.5">
        {/* Left Side: Search & Real-Time Auto-Filters */}
        <div className="flex items-center gap-2 flex-1 min-w-0">
          {/* Universal Search Input */}
          <div className="relative flex-1 min-w-[160px] max-w-xs">
            <div className="absolute inset-y-0 left-0 pl-2.5 flex items-center pointer-events-none text-slate-400">
              <Search className="w-3.5 h-3.5" />
            </div>
            <input
              type="text"
              value={filters.search}
              onChange={(e) => handleChange('search', e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  if (onFetch) onFetch(filters);
                }
              }}
              placeholder="Search Trade ID, Client, Stock..."
              className="w-full pl-8 pr-3 py-1.5 bg-slate-900/90 border border-slate-700/80 rounded-lg text-xs text-white placeholder-slate-500 focus:outline-none focus:ring-1 focus:ring-blue-500 transition-all"
            />
          </div>

          {/* Client Select (Auto-refreshes immediately on selection) */}
          <div className="w-32 sm:w-36 flex-shrink-0">
            <select
              value={filters.client_id}
              onChange={(e) => handleChange('client_id', e.target.value)}
              className="w-full px-2.5 py-1.5 bg-slate-900/90 border border-slate-700/80 rounded-lg text-xs text-slate-200 focus:outline-none focus:ring-1 focus:ring-blue-500 transition-all cursor-pointer"
            >
              <option value="">All Clients</option>
              {clients.map(c => (
                <option key={c.client_id} value={c.client_id}>
                  {c.client_code} - {c.first_name}
                </option>
              ))}
            </select>
          </div>

          {/* Stock Symbol Select (Auto-refreshes immediately on selection) */}
          <div className="w-28 sm:w-32 flex-shrink-0">
            <select
              value={filters.symbol}
              onChange={(e) => handleChange('symbol', e.target.value)}
              className="w-full px-2.5 py-1.5 bg-slate-900/90 border border-slate-700/80 rounded-lg text-xs text-slate-200 focus:outline-none focus:ring-1 focus:ring-blue-500 transition-all cursor-pointer font-mono"
            >
              <option value="">All Stocks</option>
              {stocks.map(s => (
                <option key={s.symbol} value={s.symbol}>
                  {s.symbol}
                </option>
              ))}
            </select>
          </div>

          {/* Trade Type Filter Tabs (ALL / BUY / SELL - Auto-refreshes immediately on click) */}
          <div className="inline-flex bg-slate-900 p-0.5 rounded-lg border border-slate-800 flex-shrink-0">
            {['ALL', 'BUY', 'SELL'].map(type => (
              <button
                key={type}
                type="button"
                onClick={() => handleChange('trade_type', type)}
                className={`px-2.5 py-1 text-[11px] font-semibold rounded-md transition-all cursor-pointer ${
                  filters.trade_type === type
                    ? type === 'BUY'
                      ? 'bg-emerald-600 text-white shadow-sm'
                      : type === 'SELL'
                      ? 'bg-rose-600 text-white shadow-sm'
                      : 'bg-blue-600 text-white shadow-sm'
                    : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/50'
                }`}
              >
                {type}
              </button>
            ))}
          </div>

          {/* Search Button (Positioned immediately after SELL button) */}
          <button
            type="button"
            onClick={() => onFetch && onFetch(filters)}
            className="px-3 py-1.5 bg-blue-600 hover:bg-blue-500 active:bg-blue-700 text-white rounded-lg text-xs font-semibold flex items-center gap-1.5 shadow-sm transition-all cursor-pointer flex-shrink-0"
            title="Search Trade Reports"
          >
            <Search className="w-3.5 h-3.5" />
            <span>Search</span>
          </button>
        </div>

        {/* Right Side: Action Buttons */}
        <div className="flex items-center gap-2 flex-shrink-0 self-end lg:self-auto">
          {/* Reset Filters */}
          {(filters.search || filters.client_id || filters.symbol || filters.trade_type !== 'ALL') && (
            <button
              onClick={handleReset}
              className="px-2.5 py-1.5 text-xs font-medium text-slate-400 hover:text-white bg-slate-800/60 hover:bg-slate-800 rounded-lg border border-slate-700 transition-colors cursor-pointer"
            >
              Clear
            </button>
          )}

          {/* Refresh All Page Data */}
          <button
            onClick={() => onRefresh ? onRefresh() : onFetch && onFetch(filters)}
            disabled={loading}
            title="Refresh All Page Data (MSSQL Masters & Elasticsearch Reports)"
            className="p-1.5 text-slate-300 hover:text-white bg-slate-800/80 hover:bg-slate-700/80 rounded-lg border border-slate-700 transition-colors disabled:opacity-50 cursor-pointer"
          >
            <RotateCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin text-blue-400' : ''}`} />
          </button>

          {/* Export CSV */}
          <button
            onClick={onExportCSV}
            className="px-2.5 py-1.5 text-xs font-medium bg-slate-800 hover:bg-slate-700 text-slate-200 rounded-lg border border-slate-700 transition-colors flex items-center gap-1.5 shadow-sm cursor-pointer"
          >
            <Download className="w-3.5 h-3.5" />
            <span>CSV</span>
          </button>

          {/* Execute Live Trade Modal Trigger */}
          <button
            onClick={onOpenNewTrade}
            className="px-3 py-1.5 text-xs font-semibold bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white rounded-lg shadow-lg shadow-blue-600/30 transition-all flex items-center gap-1.5 cursor-pointer whitespace-nowrap"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>New Trade</span>
          </button>
        </div>
      </div>
    </div>
  );
}
