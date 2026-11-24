# AGENTS.md: AI Agent Development Guidelines & Guardrails

This document establishes the mandatory architectural rules, development constraints, operational guidelines, and verification workflows for any AI agent working on the **TradeFlow** codebase.

---

## 1. Non-Negotiable Architectural Constraints

### Constraint 1: Zero Tracking Columns on SQL Domain Tables
- **Rule**: Never add sync flags (e.g. `is_synced`, `sync_status`, `exported`), triggers, or shadow tables to the SQL Server production domain tables (`trade_details`, `client_details`, `stock_master`).
- **Rationale**: Production OLTP schemas must remain clean, ACID-compliant, and free of vendor/tooling side effects.
- **Implementation**: Change Data Capture (CDC) must rely strictly on high-watermark timestamps using existing natural columns: `ISNULL(updated_on, created_at)` or `ISNULL(updated_on, trade_time)`.

### Constraint 2: Delta-Only Syncing (No Full Startup Re-streams)
- **Rule**: On backend startup, the CDC watermarks must initialize to the current database maximum modification timestamps:
  ```go
  tradeWatermark, _  := database.GetMaxTradeUpdatedOn()
  clientWatermark, _ := database.GetMaxClientUpdatedOn()
  stockWatermark, _  := database.GetMaxStockUpdatedOn()
  ```
- **Rationale**: Initializing watermarks to `time.Now().Add(-1 * time.Hour)` or `time.Time{}` forces Kafka and Elasticsearch to re-produce and re-index all historical records (1,200+ rows) on every process restart.
- **Implementation**: Only rows with timestamps strictly greater than these startup watermarks may be sent to Kafka.

### Constraint 3: SQL Server Windows Authentication
- **Rule**: Preserve SQL Server connection configuration using Windows Authentication (`trusted_connection=yes`) to the local instance `LAPTOP-OV8LM74T\MSSQLSERVERANIL` and database `TradeDB`.
- **Implementation**: Connection string must dynamically handle empty `MSSQL_USER` and `MSSQL_PASSWORD` by falling back to `trusted_connection=yes;encrypt=disable;`.

### Constraint 4: Elasticsearch 9.x Query Rules (Keyword vs. Text)
- **Rule**: In Elasticsearch 9.x, passing `case_insensitive: true` inside a `wildcard` clause on an analyzed `text` field (e.g. `first_name`, `last_name`, `company_name`) triggers a fatal HTTP 400 error (`case_insensitive is not supported for field of type text`).
- **Implementation**:
  - **Keyword fields** (`trade_id`, `client_code`, `symbol`, `exchange`, `sector`, `trade_status`): Use `wildcard` with `case_insensitive: true`.
  - **Text fields** (`first_name`, `last_name`, `company_name`): Use `match_phrase_prefix`, `match`, or `query_string` without `case_insensitive`.

### Constraint 5: Fixed Viewport UI Layout
- **Rule**: The React trading interface must never exhibit full-page scrolling.
- **Implementation**:
  - The viewport must be fixed using `h-screen overflow-hidden flex flex-col`.
  - Top sections (Navbar, KPI SummaryCards, FilterBar) remain static.
  - Only the table body (`<tbody>` inside `TradeTable.jsx`) scrolls vertically.
  - The universal **Search** button must remain immediately adjacent to the **SELL** toggle button in `FilterBar.jsx`.
  - The **Refresh** button must refresh both Elasticsearch reports and SQL Server master data concurrently.

### Constraint 6: Environment-Driven Configuration
- **Rule**: Never hardcode database credentials, Kafka broker addresses, Elasticsearch URLs, or JWT secrets in application code.
- **Implementation**: Load all runtime values from `backend/.env` and `frontend/.env`.

### Constraint 7: Elasticsearch Data Cleanups
- **Rule**: When clearing or rebuilding test data in Elasticsearch, never delete the index mapping (`DELETE /trade_reports`).
- **Implementation**: Use `_delete_by_query` with `{"query": {"match_all": {}}}` to preserve field mappings and performance settings.

---

## 2. Directory Structure & File Placement Rules

- Root directory must maintain minimal clutter:
  - `backend/`: All Go backend code, configs, and executable.
  - `frontend/`: All React UI code, components, styles, and configs.
  - `Bkp/`: Archived legacy scripts, docker-compose, and backups.
  - Active runtime folders: `kafka/` and `elasticsearch-9.5.7-windows-x86_64/` (do not attempt to move active running service directories).
- Agent skills must reside in `.agent/skills/<skill-name>/SKILL.md`.

---

## 3. Code Modification Checklist for Agents

Before concluding any code change turn:

1. **Backend Verification**:
   - Run `go build -o trade-backend.exe main.go` inside `backend/` to verify zero compile or import errors.
   - Ensure all imported packages are actively utilized (Go disallows unused imports).
2. **Frontend Verification**:
   - Run `npm run build` inside `frontend/` to verify zero bundling or JSX syntax errors.
   - Verify all Lucide icon imports exist in `lucide-react`.
3. **Delta CDC Check**:
   - If modifying database pollers, ensure `watermark` updates occur **after** successful Kafka event production.
4. **Elasticsearch Query Check**:
   - If editing search queries in `elastic/client.go`, test that substring queries (e.g. `search=e`) return HTTP 200 and valid JSON hits.

---

## 4. Runbook for Agents

### Check Process Status
```powershell
Get-Process -Name "trade-backend", "java", "node" -ErrorAction SilentlyContinue | Select-Object Id, ProcessName, Path
```

### Check Port Bindings
```powershell
Get-NetTCPConnection -LocalPort 1433, 8080, 9092, 9200, 5173 -State Listen -ErrorAction SilentlyContinue | Select-Object LocalPort, OwningProcess
```

### Test Backend Build
```powershell
cd D:\TradeProject\backend
go build -o trade-backend.exe main.go
```

### Test Frontend Build
```powershell
cd D:\TradeProject\frontend
npm run build
```
