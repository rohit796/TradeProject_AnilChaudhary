# TradeFlow: Real-Time Financial Trading & Reporting Platform

TradeFlow is an enterprise financial trading, Change Data Capture (CDC) streaming, and sub-millisecond analytics reporting platform. It bridges transactional relational databases with distributed search engines using an event-driven architecture powered by Go, Apache Kafka, Elasticsearch, and React.

---

## 1. System Architecture

```
[ Microsoft SQL Server 2022 ] ── (OLTP Transactional Store: TradeDB)
              │
              ▼ (Every 3s High-Watermark Delta Poller: updated_on > watermark)
[ Go CDC Engine & Backend ] ── (Port 8080)
              │
              ▼ (Produces Keyed Domain Events)
[ Apache Kafka 4.x (KRaft) ] ── (Port 9092)
     ├── topic: trade-events   (Key: trade_id)
     ├── topic: client-events  (Key: client_id)
     └── topic: stock-events   (Key: symbol)
              │
              ▼ (Dedicated Goroutine Consumers with Reference Caches)
[ Elasticsearch 9.5.7 ] ── (Port 9200) (Index: trade_reports)
              │
              ▼ (Sub-millisecond Search, Filtering, Dynamic KPI Aggregations)
[ Go REST API Service ] ── (Port 8080)
              │
              ▼ (REST JSON / Auto-refresh Polling)
[ React 18 Institutional Dashboard ] ── (Port 5173)
```

---

## 2. Technology Stack & Port Allocations

| Component | Technology | Default Port | Role & Responsibility |
|:---|:---|:---|:---|
| **OLTP Database** | MS SQL Server 2022 | `1433` | ACID transactional source of truth (`TradeDB`) |
| **CDC & API Backend** | Go (Golang 1.23+) | `8080` | Multi-table delta CDC, Kafka streaming, REST API |
| **Message Broker** | Apache Kafka 4.x (KRaft) | `9092` | Durable distributed message broker for domain events |
| **Search Engine** | Elasticsearch 9.5.7 | `9200` | Sub-millisecond denormalized trade reporting index |
| **Trading UI** | React 18 + Vite + Tailwind | `5173` | Institutional dashboard, trade entry, KPI monitoring |

---

## 3. Directory Structure

```
TradeProject/
├── .agent/                    # AI Agent skills and configuration
│   └── skills/
│       └── tradeflow-pipeline/
│           └── SKILL.md       # Operational & architectural runbook skill
├── backend/                   # Go 1.23+ Backend Service
│   ├── cmd/sync/              # Initial full historical sync CLI
│   ├── database/              # SQL Server connector & delta query engine
│   ├── elastic/               # Elasticsearch 9.5.7 client & query builder
│   ├── handlers/              # HTTP REST API route handlers
│   ├── kafka/                 # Kafka producers, consumers, and topic admin
│   ├── models/                # Go struct data models
│   ├── .env                   # Backend environment configuration
│   ├── main.go                # Server entrypoint & CDC poller loop
│   ├── trade-backend.exe      # Compiled Windows executable
│   └── README.md              # Backend technical documentation
├── frontend/                  # React 18 Frontend Dashboard
│   ├── src/
│   │   ├── components/        # Dashboard, FilterBar, TradeTable, Modal, Login
│   │   ├── services/          # REST API client
│   │   └── App.jsx            # Application entry & auth guard
│   ├── .env                   # Frontend environment configuration
│   ├── package.json           # npm dependencies & build scripts
│   └── README.md              # Frontend technical documentation
├── Bkp/                       # Archived legacy scripts, docker, & old backups
├── AGENTS.md                  # Autonomous agent instructions & coding constraints
├── ARCHITECTURE_AND_WORKFLOW.md # Comprehensive architectural reference
├── DEVELOPMENT.md             # Developer setup, testing, and troubleshooting guide
└── README.md                  # This repository overview
```

---

## 4. Key Architectural Principles

### 1. Zero Extra Columns on Domain Tables
- Production domain tables (`trade_details`, `client_details`, `stock_master`) use their natural business columns and standard timestamps (`updated_on`, `created_at`).
- **Never** add tracking flags (`is_synced`), status columns, or database triggers to the SQL domain tables.

### 2. High-Watermark Delta Syncing
- On backend startup, watermarks initialize to the current database `MAX(ISNULL(updated_on, created_at))`.
- **Zero historical records are re-streamed to Kafka on startup.**
- Polling runs every 3 seconds: only rows modified *after* backend launch are produced to Kafka.

### 3. Elasticsearch Query Rules (Keyword vs. Text)
- Keyword fields (`symbol`, `client_code`, `trade_id`, `exchange`, `sector`) use `wildcard` queries with `case_insensitive: true`.
- Analyzed text fields (`first_name`, `last_name`, `company_name`) use `match_phrase_prefix` or `query_string`. Elasticsearch 9.x strictly rejects `case_insensitive` on text fields.

### 4. Fixed-Layout Institutional UI
- The dashboard employs a **fixed viewport layout**. Navigation, KPI summary cards, and the filter toolbar stay pinned at the top. Only the trade table body scrolls vertically.
- The **Search** button is positioned immediately after the **SELL** toggle button.
- The **Refresh** button refreshes both Elasticsearch reports and SQL Server master data concurrently.

---

## 5. Quick Start Runbook

### Step 1: Start Microsoft SQL Server
Ensure SQL Server instance `LAPTOP-OV8LM74T\MSSQLSERVERANIL` is running with database `TradeDB`.

### Step 2: Start Apache Kafka (KRaft Mode)
```powershell
cd D:\TradeProject\kafka
.\bin\windows\kafka-server-start.bat .\config\kraft\server.properties
```

### Step 3: Start Elasticsearch
```powershell
cd D:\TradeProject\elasticsearch-9.5.7-windows-x86_64\elasticsearch-9.5.7
.\bin\elasticsearch.bat
```

### Step 4: Start Go Backend
```powershell
cd D:\TradeProject\backend
.\trade-backend.exe
# Or run with Go: go run main.go
```

### Step 5: Start React Frontend
```powershell
cd D:\TradeProject\frontend
npm run dev
```
Open your browser at `http://localhost:5173/`. Login with `admin` / `admin123`.

---

## 6. Real-Time Verification Test

1. Open SQL Server Management Studio (SSMS) and run:
   ```sql
   UPDATE client_details 
   SET first_name = 'Ravi', updated_on = GETDATE() 
   WHERE client_id = 'CL1001';
   ```
2. Observe backend console output:
   ```
   [REALTIME CDC] ⚡ CLIENT TABLE UPDATE DETECTED: 1 record(s)
   [REALTIME CDC] 🔄 Client: CL1001 | Code: ALOK001 | Name: Ravi Sharma
   [Kafka Producer] 📤 Streamed CLIENT_UPDATED to topic 'client-events' (Key: CL1001)
   [REALTIME CDC] 🚀 Elasticsearch synced client CL1001 ('Ravi Sharma') across 401 trade report(s)!
   ```
3. In the React UI (`http://localhost:5173/`), all trades for `CL1001` update to `Ravi Sharma` within 4 seconds without any manual reload!

---

## 7. Further Documentation

- **Full System Architecture & Workflow**: [ARCHITECTURE_AND_WORKFLOW.md](file:///D:/TradeProject/ARCHITECTURE_AND_WORKFLOW.md)
- **Developer Guide & Runbook**: [DEVELOPMENT.md](file:///D:/TradeProject/DEVELOPMENT.md)
- **Agent Instructions & Development Rules**: [AGENTS.md](file:///D:/TradeProject/AGENTS.md)
- **Backend Technical Documentation**: [backend/README.md](file:///D:/TradeProject/backend/README.md)
- **Frontend Technical Documentation**: [frontend/README.md](file:///D:/TradeProject/frontend/README.md)
- **Agent Skill Definition**: [.agent/skills/tradeflow-pipeline/SKILL.md](file:///D:/TradeProject/.agent/skills/tradeflow-pipeline/SKILL.md)
