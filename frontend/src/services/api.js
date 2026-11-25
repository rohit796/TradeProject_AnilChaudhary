const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api';

export const api = {
  // Authentication via Backend Go API
  async login(username, password) {
    const response = await fetch(`${API_BASE_URL}/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password }),
    });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'Login failed');
    }
    return await response.json();
  },

  // Health & Pipeline Status
  async getStatus() {
    const res = await fetch(`${API_BASE_URL}/health`);
    if (!res.ok) throw new Error('Health check failed');
    return await res.json();
  },

  // Get Clients list strictly from MSSQL
  async getClients() {
    const res = await fetch(`${API_BASE_URL}/clients`);
    if (!res.ok) throw new Error('Failed to load clients from MSSQL');
    return await res.json();
  },

  // Get Stocks list strictly from MSSQL
  async getStocks() {
    const res = await fetch(`${API_BASE_URL}/stocks`);
    if (!res.ok) throw new Error('Failed to load stocks from MSSQL');
    return await res.json();
  },

  // Get Filtered Trade Reports strictly from Elasticsearch
  async getTradeReports(filters = {}) {
    const params = new URLSearchParams();
    if (filters.client_id) params.append('client_id', filters.client_id);
    if (filters.symbol) params.append('symbol', filters.symbol);
    if (filters.trade_type && filters.trade_type !== 'ALL') params.append('trade_type', filters.trade_type);
    if (filters.search) params.append('search', filters.search);
    if (filters.sort_by) params.append('sort_by', filters.sort_by);
    if (filters.order) params.append('order', filters.order);
    if (filters.page) params.append('page', filters.page);
    if (filters.limit) params.append('limit', filters.limit);

    const res = await fetch(`${API_BASE_URL}/reports/trades?${params.toString()}`);
    if (!res.ok) {
      const err = await res.json();
      throw new Error(err.message || err.error || 'Failed to fetch trade reports from Elasticsearch');
    }
    return await res.json();
  },

  // Execute / Submit New Trade (Writes MSSQL -> Kafka -> Elastic)
  async createTrade(tradeData) {
    const res = await fetch(`${API_BASE_URL}/trades`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(tradeData)
    });
    if (!res.ok) {
      const err = await res.json();
      throw new Error(err.message || err.error || 'Failed to place trade');
    }
    return await res.json();
  },

  // Trigger Full Pipeline Sync: MSSQL ➔ Kafka ➔ Elasticsearch
  async syncTrades() {
    const res = await fetch(`${API_BASE_URL}/sync/trades`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' }
    });
    if (!res.ok) {
      const err = await res.json();
      throw new Error(err.error || 'Pipeline sync failed');
    }
    return await res.json();
  }
};
