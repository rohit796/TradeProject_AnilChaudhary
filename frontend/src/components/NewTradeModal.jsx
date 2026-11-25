import React, { useState } from 'react';
import { X, Send, Sparkles, TrendingUp, TrendingDown, CheckCircle2 } from 'lucide-react';
import { api } from '../services/api';

export default function NewTradeModal({ isOpen, onClose, onTradeCreated, clients = [], stocks = [] }) {
  if (!isOpen) return null;

  const [formData, setFormData] = useState({
    client_id: clients[0]?.client_id || 'CL1001',
    symbol: stocks[0]?.symbol || 'RELIANCE',
    trade_type: 'BUY',
    quantity: 10,
    price: 2950.00,
    exchange: 'NSE'
  });

  const [loading, setLoading] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState('');

  const selectedStock = stocks.find(s => s.symbol === formData.symbol);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError('');

    const tradePayload = {
      trade_id: `TRD-${Date.now().toString().slice(-6)}`,
      order_id: `ORD-${Math.floor(1000 + Math.random() * 9000)}`,
      client_id: formData.client_id,
      symbol: formData.symbol,
      exchange: formData.exchange,
      trade_type: formData.trade_type,
      quantity: Number(formData.quantity),
      price: Number(formData.price),
      trade_value: Number(formData.quantity) * Number(formData.price),
      trade_status: 'EXECUTED',
      trade_time: new Date().toISOString()
    };

    try {
      await api.createTrade(tradePayload);
      setSuccess(true);
      setTimeout(() => {
        setSuccess(false);
        onTradeCreated(tradePayload);
        onClose();
      }, 1200);
    } catch (err) {
      // Never pretend a failed save worked: keep the form open and show why it failed
      const offline = err instanceof TypeError; // fetch() throws TypeError when the backend is unreachable
      setError(offline
        ? 'Could not reach the trading server. Trade was NOT placed. Please try again.'
        : `Trade was NOT placed: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-[#0F172A] border border-slate-700 rounded-2xl w-full max-w-lg shadow-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-150">
        {/* Header */}
        <div className="px-6 py-4 border-b border-slate-800 flex items-center justify-between">
          <div>
            <h3 className="text-base font-bold text-white flex items-center gap-2">
              <Sparkles className="w-4 h-4 text-blue-400" />
              Execute Live Trade
            </h3>
            <p className="text-xs text-slate-400 mt-0.5">
              Direct Market Access Order Execution
            </p>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Success Banner */}
        {success && (
          <div className="p-4 bg-emerald-500/10 border-b border-emerald-500/30 flex items-center gap-3 text-emerald-400 text-sm">
            <CheckCircle2 className="w-5 h-5" />
            <span>Trade executed successfully and confirmed.</span>
          </div>
        )}

        {/* Error Banner */}
        {error && (
          <div role="alert" className="p-4 bg-red-500/10 border-b border-red-500/30 flex items-center gap-3 text-red-400 text-sm">
            <X className="w-5 h-5 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {/* Form */}
        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          {/* Side Selector (BUY vs SELL) */}
          <div>
            <label className="block text-xs font-semibold text-slate-300 mb-1.5 uppercase tracking-wider">
              Trade Action (Side)
            </label>
            <div className="grid grid-cols-2 gap-3">
              <button
                type="button"
                onClick={() => setFormData(p => ({ ...p, trade_type: 'BUY' }))}
                className={`py-2.5 rounded-xl text-xs font-bold flex items-center justify-center gap-2 border transition-all ${
                  formData.trade_type === 'BUY'
                    ? 'bg-emerald-600 text-white border-emerald-500 shadow-lg shadow-emerald-600/30'
                    : 'bg-slate-900 text-slate-400 border-slate-800 hover:border-slate-700'
                }`}
              >
                <TrendingUp className="w-4 h-4" />
                BUY
              </button>
              <button
                type="button"
                onClick={() => setFormData(p => ({ ...p, trade_type: 'SELL' }))}
                className={`py-2.5 rounded-xl text-xs font-bold flex items-center justify-center gap-2 border transition-all ${
                  formData.trade_type === 'SELL'
                    ? 'bg-rose-600 text-white border-rose-500 shadow-lg shadow-rose-600/30'
                    : 'bg-slate-900 text-slate-400 border-slate-800 hover:border-slate-700'
                }`}
              >
                <TrendingDown className="w-4 h-4" />
                SELL
              </button>
            </div>
          </div>

          {/* Client Selection */}
          <div>
            <label className="block text-xs font-medium text-slate-300 mb-1">
              Select Client
            </label>
            <select
              value={formData.client_id}
              onChange={(e) => setFormData(p => ({ ...p, client_id: e.target.value }))}
              className="w-full px-3 py-2.5 bg-slate-900 border border-slate-700 rounded-xl text-xs text-white focus:outline-none focus:ring-1 focus:ring-blue-500"
            >
              {clients.map(c => (
                <option key={c.client_id} value={c.client_id}>
                  {c.client_code} - {c.first_name} {c.last_name} ({c.account_type})
                </option>
              ))}
            </select>
          </div>

          {/* Stock Symbol Selection */}
          <div>
            <label className="block text-xs font-medium text-slate-300 mb-1">
              Stock Instrument (from Stock Master)
            </label>
            <select
              value={formData.symbol}
              onChange={(e) => setFormData(p => ({ ...p, symbol: e.target.value }))}
              className="w-full px-3 py-2.5 bg-slate-900 border border-slate-700 rounded-xl text-xs text-white focus:outline-none focus:ring-1 focus:ring-blue-500 font-mono"
            >
              {stocks.map(s => (
                <option key={s.symbol} value={s.symbol}>
                  {s.symbol} - {s.company_name} ({s.sector})
                </option>
              ))}
            </select>
          </div>

          {/* Quantity & Price Grid */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-medium text-slate-300 mb-1">
                Quantity (Units)
              </label>
              <input
                type="number"
                min="1"
                value={formData.quantity}
                onChange={(e) => setFormData(p => ({ ...p, quantity: e.target.value }))}
                className="w-full px-3 py-2.5 bg-slate-900 border border-slate-700 rounded-xl text-xs text-white focus:outline-none focus:ring-1 focus:ring-blue-500 font-mono"
                required
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-300 mb-1">
                Execution Price (₹)
              </label>
              <input
                type="number"
                step="0.05"
                min="0.05"
                value={formData.price}
                onChange={(e) => setFormData(p => ({ ...p, price: e.target.value }))}
                className="w-full px-3 py-2.5 bg-slate-900 border border-slate-700 rounded-xl text-xs text-white focus:outline-none focus:ring-1 focus:ring-blue-500 font-mono"
                required
              />
            </div>
          </div>

          {/* Value Summary */}
          <div className="p-3 bg-slate-950/80 rounded-xl border border-slate-800/80 flex items-center justify-between text-xs">
            <span className="text-slate-400">Total Calculated Trade Value:</span>
            <span className="font-mono font-bold text-base text-white">
              ₹{(formData.quantity * formData.price).toLocaleString('en-IN', { minimumFractionDigits: 2 })}
            </span>
          </div>

          {/* Submit */}
          <div className="pt-2 flex items-center justify-end gap-3">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-xs font-medium text-slate-400 hover:text-white transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading}
              className="px-5 py-2.5 bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white text-xs font-semibold rounded-xl shadow-lg shadow-blue-600/30 flex items-center gap-1.5 disabled:opacity-50"
            >
              <Send className="w-3.5 h-3.5" />
              <span>{loading ? 'Processing...' : 'Submit Trade'}</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
