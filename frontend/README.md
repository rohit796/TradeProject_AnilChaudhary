# TradeFlow Frontend Dashboard

Institutional financial trade execution, real-time analytics monitoring, and multi-dimensional reporting dashboard built with React 18, Tailwind CSS, and Vite.

---

## 1. Architectural Highlights

- **Framework**: React 18 with modern functional components and hooks (`useState`, `useEffect`, `useCallback`, `useRef`).
- **Build Tool**: Vite 6 providing sub-second HMR and optimized production bundling.
- **Styling**: Tailwind CSS with custom institutional dark-mode slate theme (`slate-900`, `slate-800`, `slate-700`).
- **Icons**: Lucide React icons.
- **Layout Paradigm**: **Fixed viewport layout**. Navigation bar, KPI summary cards, and the filter toolbar remain fixed at the top of the viewport; only the trade report table body scrolls vertically. Full-page scrolling is strictly prevented.
- **Real-Time Data Sync**: Dual update mechanism:
  1. **Background Polling**: Automatic lightweight polling every 4 seconds to reflect Kafka ➔ Elasticsearch CDC updates seamlessly.
  2. **Universal Refresh**: Circular refresh button that fetches master data (`clients`, `stocks`) and trade records concurrently with live spinning animation.
- **Universal Search**: Multi-field substring search with debounced typing and an immediate **Search** action button positioned directly adjacent to the **SELL** toggle button.

---

## 2. Component Hierarchy

```
App.jsx (Authentication Guard & Session State)
│
├── Login.jsx (Institutional Login Screen, auto-populates from .env)
│
└── Dashboard.jsx (Central Orchestrator, Master State, Auto-polling)
    ├── Navbar (Branding, Live Connection Pill, User Badge, New Trade Trigger)
    ├── SummaryCards.jsx (Dynamic KPIs: Total Trades, Turnover, Buy/Sell Count, Brokerage)
    ├── FilterBar.jsx (Search Bar, BUY/SELL Buttons, Search Button, Filters, Date Pickers, Refresh)
    ├── TradeTable.jsx (Fixed Header, Scrollable Table Body, Sorting, Badges, Pagination)
    └── NewTradeModal.jsx (Trade Execution Modal with Real-Time Validation & Calculation)
```

---

## 3. Directory Structure

```
frontend/
├── dist/                      # Production build output
├── node_modules/              # Node.js dependencies
├── src/
│   ├── components/
│   │   ├── Dashboard.jsx      # Top-level dashboard orchestrator & polling loop
│   │   ├── FilterBar.jsx      # Filter toolbar (Search, Buy/Sell, Search Button, Refresh)
│   │   ├── Login.jsx          # Institutional authentication view
│   │   ├── NewTradeModal.jsx  # Modal for placing BUY/SELL orders into SQL Server
│   │   ├── SummaryCards.jsx   # KPI metric cards aggregated from Elasticsearch
│   │   └── TradeTable.jsx     # Institutional table with sticky header & pagination
│   ├── services/
│   │   └── api.js             # Axios/Fetch client for REST endpoints with JWT handling
│   ├── App.jsx                # Application root with token verification
│   ├── index.css              # Tailwind CSS directives & custom scrollbar styles
│   └── main.jsx               # React DOM entrypoint
├── .env                       # Frontend environment configuration
├── index.html                 # HTML entrypoint
├── package.json               # Dependencies and scripts
├── tailwind.config.js         # Tailwind CSS configuration
└── vite.config.js             # Vite configuration and server port settings
```

---

## 4. Environment Configuration (`.env`)

```properties
# Backend REST API Base URL
VITE_API_BASE_URL=http://localhost:8080/api

# Authentication Credentials & Defaults
VITE_AUTH_USERNAME=admin
VITE_AUTH_PASSWORD=admin123
VITE_AUTH_ROLE=Trade Admin
VITE_COMPANY_NAME=TradeFlow Global Markets
```

---

## 5. Key UI & Interaction Specifications

### Search & Filter Bar Layout

The search and action button sequence in `FilterBar.jsx` follows strict usability guidelines:
```
[ 🔍 Multi-field Search Input ] ➔ [ ALL ] [ BUY ] [ SELL ] [ 🔍 Search ] ➔ [ Status ▾ ] [ Exchange ▾ ] [ Date Pickers ] ➔ [ 🔄 Refresh ]
```
- **BUY / SELL Toggles**: Instantly filter between buy orders, sell orders, or all trade flows.
- **Search Button**: Located immediately after the **SELL** button to allow instantaneous execution of the search query without waiting for the typing debounce.
- **Refresh Button**: Circular button with spinning animation during network requests; re-syncs both Elasticsearch trade reports and SQL Server master data.

### Fixed Layout & Table Scrolling

To ensure institutional financial ergonomics:
- The entire page body uses `h-screen overflow-hidden flex flex-col`.
- The `TradeTable` container uses `flex-1 overflow-hidden flex flex-col`.
- The table header (`<thead>`) is sticky (`sticky top-0 z-10 bg-slate-900`).
- Only `<tbody>` scrolls vertically inside `overflow-y-auto`.
- The pagination footer remains pinned at the bottom.

---

## 6. Development & Build Commands

### Install Dependencies
```powershell
cd D:\TradeProject\frontend
npm install
```

### Start Development Server
```powershell
cd D:\TradeProject\frontend
npm run dev
```
The application will launch at `http://localhost:5173/`.

### Build for Production
```powershell
cd D:\TradeProject\frontend
npm run build
```
Creates optimized static assets in the `dist/` directory.

### Preview Production Build
```powershell
cd D:\TradeProject\frontend
npm run preview
```

---

## 7. Frontend State Management & API Service (`services/api.js`)

All communication with the Go backend passes through `src/services/api.js`:

| Method | Function | Endpoint | Description |
|:---|:---|:---|:---|
| `POST` | `login(username, password)` | `/login` | Authenticates and returns JWT token |
| `GET` | `getTrades(params)` | `/trades` | Fetches filtered & paginated trades from ES |
| `GET` | `getTradeSummary(params)` | `/trades/summary` | Fetches dynamic KPI totals from ES |
| `POST` | `createTrade(tradeData)` | `/trades` | Submits new trade order to SQL Server |
| `GET` | `getClients()` | `/clients` | Fetches active clients list for order dropdown |
| `GET` | `getStocks()` | `/stocks` | Fetches active stocks list for order dropdown |
| `GET` | `getHealth()` | `/health` | Verifies backend, Kafka, and ES connectivity |
