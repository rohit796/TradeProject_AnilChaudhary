import React, { useState, useEffect, useCallback } from 'react';
import { LogOut, Activity } from 'lucide-react';
import SummaryCards from './SummaryCards';
import FilterBar from './FilterBar';
import TradeTable from './TradeTable';
import NewTradeModal from './NewTradeModal';
import { api } from '../services/api';

export default function Dashboard({ user, onLogout }) {
  // Start with clean empty array - NO HARDCODED TRADES
  const [trades, setTrades] = useState([]);
  const [totalCount, setTotalCount] = useState(0);
  const [clients, setClients] = useState([]);
  const [stocks, setStocks] = useState([]);
  const [loading, setLoading] = useState(false);
  const [isNewTradeOpen, setIsNewTradeOpen] = useState(false);

  const [filters, setFilters] = useState({
    search: '',
    client_id: '',
    symbol: '',
    trade_type: 'ALL',
    sort_by: 'trade_time',
    order: 'desc',
    page: 1,
    limit: 100
  });

  // Load master data strictly from MSSQL tables
  const loadMasterData = useCallback(async () => {
    try {
      const [c, s] = await Promise.all([api.getClients(), api.getStocks()]);
      if (c) setClients(c);
      if (s) setStocks(s);
    } catch (err) {
      console.error('Failed to load master data from MSSQL:', err);
    }
  }, []);

  useEffect(() => {
    loadMasterData();
  }, [loadMasterData]);

  // Fetch trade reports strictly from Elasticsearch
  const fetchTrades = useCallback(async (activeFilters) => {
    setLoading(true);
    try {
      console.log('%c[REALTIME SYNC] 🔄 Querying Elasticsearch for trade reports...', 'color: #38bdf8; font-weight: bold;', activeFilters);
      const data = await api.getTradeReports(activeFilters);
      if (data && data.trades) {
        console.log(`%c[REALTIME SYNC] ✅ Loaded ${data.trades.length} of ${data.total || data.trades.length} records from Elasticsearch (Source: ${data.source || 'elasticsearch'})`, 'color: #34d399; font-weight: bold;');
        setTrades(data.trades);
        setTotalCount(data.total || data.trades.length);
      } else {
        setTrades([]);
        setTotalCount(0);
      }
    } catch (err) {
      console.error('[REALTIME SYNC ERROR] Failed to fetch from Elasticsearch:', err);
      setTrades([]);
      setTotalCount(0);
    } finally {
      setLoading(false);
    }
  }, []);

  // Automatic Real-Time Refresh on ANY change (instant for dropdowns/side toggle, 300ms debounce for typing search)
  useEffect(() => {
    const delay = filters.search ? 300 : 0;
    const timer = setTimeout(() => {
      fetchTrades(filters);
    }, delay);

    return () => clearTimeout(timer);
  }, [filters, fetchTrades]);

  // Refresh all data on page: master data (clients, stocks) from MSSQL + trade reports from Elasticsearch
  const handleRefreshAll = useCallback(async () => {
    setLoading(true);
    try {
      console.log('%c[REFRESH ALL] 🔄 Refreshing all data on page (MSSQL Master Data + Elasticsearch Reports)...', 'color: #38bdf8; font-weight: bold;');
      await Promise.all([
        loadMasterData(),
        fetchTrades(filters)
      ]);
      console.log('%c[REFRESH ALL] ✅ Successfully refreshed all page data!', 'color: #34d399; font-weight: bold;');
    } catch (err) {
      console.error('[REFRESH ALL ERROR]', err);
    } finally {
      setLoading(false);
    }
  }, [loadMasterData, fetchTrades, filters]);

  const handleTradeCreated = () => {
    // Re-query from Elasticsearch to show the newly indexed trade
    fetchTrades(filters);
  };

  const handleExportCSV = () => {
    if (trades.length === 0) return;
    const headers = ['Trade ID', 'Order ID', 'Client ID', 'Client Code', 'Stock', 'Side', 'Qty', 'Price', 'Value', 'Time'];
    const rows = trades.map(t => [
      t.trade_id,
      t.order_id,
      t.client_id,
      t.client_code || '',
      t.symbol,
      t.trade_type,
      t.quantity,
      t.price,
      t.trade_value,
      t.trade_time
    ]);

    const csvContent = "data:text/csv;charset=utf-8," 
      + [headers.join(','), ...rows.map(e => e.join(','))].join('\n');
    
    const encodedUri = encodeURI(csvContent);
    const link = document.createElement("a");
    link.setAttribute("href", encodedUri);
    link.setAttribute("download", `trade_report_${new Date().toISOString().slice(0, 10)}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  return (
    <div className="h-screen max-h-screen overflow-hidden bg-[#070B14] flex flex-col">
      {/* Top Navbar - Fixed / Static */}
      <header className="flex-shrink-0 bg-[#0B0F19]/90 backdrop-blur-md border-b border-slate-800 px-6 py-2.5">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-xl bg-gradient-to-tr from-blue-600 to-indigo-500 flex items-center justify-center shadow-md shadow-blue-500/20 ring-1 ring-white/10">
              <Activity className="w-4 h-4 text-white" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-sm font-bold text-white tracking-tight">TradeFlow Analytics</h1>
                <span className="text-[10px] px-2 py-0.5 rounded-full font-mono bg-blue-500/10 text-blue-400 border border-blue-500/20">
                  Live Engine
                </span>
              </div>
              <p className="text-[11px] text-slate-400 hidden sm:block">
                Enterprise Trading & Reporting Engine
              </p>
            </div>
          </div>

          <div className="flex items-center gap-3">
            {/* User Profile & Logout */}
            <div className="flex items-center gap-3">
              <div className="text-right hidden sm:block">
                <p className="text-xs font-semibold text-white">{user?.name || user?.username || 'Admin'}</p>
                <p className="text-[10px] text-blue-400 font-medium">{user?.role || 'Trade Admin'}</p>
              </div>

              <button
                onClick={onLogout}
                title="Sign Out"
                className="p-2 text-slate-400 hover:text-rose-400 hover:bg-rose-500/10 rounded-xl transition-all cursor-pointer"
              >
                <LogOut className="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Main Container - Full height flex with static controls + scrollable table */}
      <main className="flex-1 min-h-0 max-w-7xl w-full mx-auto px-4 sm:px-6 py-3 flex flex-col gap-3">
        {/* KPI Cards - Static */}
        <div className="flex-shrink-0">
          <SummaryCards trades={trades} totalCount={totalCount} />
        </div>

        {/* Filters - Static */}
        <div className="flex-shrink-0">
          <FilterBar
            filters={filters}
            setFilters={setFilters}
            clients={clients}
            stocks={stocks}
            onFetch={fetchTrades}
            onRefresh={handleRefreshAll}
            onOpenNewTrade={() => setIsNewTradeOpen(true)}
            onExportCSV={handleExportCSV}
            loading={loading}
          />
        </div>

        {/* Real-Time Table - ONLY this component scrolls */}
        <div className="flex-1 min-h-0 flex flex-col">
          <TradeTable
            trades={trades}
            totalCount={totalCount}
            loading={loading}
            filters={filters}
            setFilters={setFilters}
          />
        </div>
      </main>

      <NewTradeModal
        isOpen={isNewTradeOpen}
        onClose={() => setIsNewTradeOpen(false)}
        onTradeCreated={handleTradeCreated}
        clients={clients}
        stocks={stocks}
      />
    </div>
  );
}
