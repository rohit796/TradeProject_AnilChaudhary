# TradeFlow: Developer Setup & Operations Guide

Comprehensive guide for configuring, building, testing, developing, and operating the **TradeFlow** trading and real-time reporting platform on Windows.

---

## 1. Prerequisites

Ensure the following runtimes and tools are installed:
- **OS**: Windows 10/11 x64
- **Go**: Version 1.23 or higher
- **Node.js**: Version 18.x or 20.x with `npm`
- **Java JDK**: Version 17 or 21 (required by Kafka)
- **Database**: Microsoft SQL Server 2022 (Developer or Express edition)
- **SSMS**: SQL Server Management Studio or Azure Data Studio
- **Git / PowerShell 7+**

---

## 2. Infrastructure Setup

### A. Microsoft SQL Server Setup

1. Open SSMS and connect to instance `LAPTOP-OV8LM74T\MSSQLSERVERANIL` using Windows Authentication.
2. Verify or create the database:
   ```sql
   IF NOT EXISTS (SELECT * FROM sys.databases WHERE name = 'TradeDB')
   BEGIN
       CREATE DATABASE TradeDB;
   END
   GO
   USE TradeDB;
   GO
   ```
3. Verify domain tables exist:
   - `client_details` (PK: `client_id`)
   - `stock_master` (PK: `symbol`)
   - `trade_details` (PK: `trade_id`, FKs: `client_id`, `symbol`)

### B. Apache Kafka 4.x Setup (KRaft Mode)

Kafka runs without ZooKeeper using KRaft mode:
1. Verify storage initialization:
   ```powershell
   cd D:\TradeProject\kafka
   # Format storage (only if data dir does not exist):
   # .\bin\windows\kafka-storage.bat format -t <cluster-id> -c .\config\kraft\server.properties
   ```
2. Start Kafka broker:
   ```powershell
   cd D:\TradeProject\kafka
   .\bin\windows\kafka-server-start.bat .\config\kraft\server.properties
   ```
3. Kafka broker listens on `localhost:9092`. Topics (`trade-events`, `client-events`, `stock-events`) are automatically created by Go backend `kafka/admin.go` on startup.

### C. Elasticsearch 9.5.7 Setup

1. Launch Elasticsearch:
   ```powershell
   cd D:\TradeProject\elasticsearch-9.5.7-windows-x86_64\elasticsearch-9.5.7
   .\bin\elasticsearch.bat
   ```
2. Test cluster health:
   ```powershell
   Invoke-RestMethod -Uri "http://localhost:9200/_cat/health?v"
   ```
   Elasticsearch listens on `http://localhost:9200`.

---

## 3. Launching Services (Local Development)

### Terminal 1: Kafka
```powershell
cd D:\TradeProject\kafka
.\bin\windows\kafka-server-start.bat .\config\kraft\server.properties
```

### Terminal 2: Elasticsearch
```powershell
cd D:\TradeProject\elasticsearch-9.5.7-windows-x86_64\elasticsearch-9.5.7
.\bin\elasticsearch.bat
```

### Terminal 3: Go Backend
```powershell
cd D:\TradeProject\backend
go run main.go
# Or run precompiled binary:
# .\trade-backend.exe
```

Expected startup logs:
```
[MSSQL] Connected to Microsoft SQL Server: TradeDB (LAPTOP-OV8LM74T\MSSQLSERVERANIL)
[Elasticsearch] Connected to Elasticsearch 9.5.7 at http://localhost:9200
[Kafka Admin] Topic 'trade-events' ready
[Kafka Admin] Topic 'client-events' ready
[Kafka Admin] Topic 'stock-events' ready
[Kafka Consumer] Loaded 50 clients and 100 stocks into local reference cache
[Kafka Consumer] Goroutines started on trade-events, client-events, stock-events
[CDC Engine] Delta watermarks initialized:
  - Trades: 2026-11-24 16:45:10
  - Clients: 2026-11-24 16:45:10
  - Stocks: 2026-11-24 16:45:10
[Server] HTTP server listening on http://localhost:8080
```

### Terminal 4: React Frontend
```powershell
cd D:\TradeProject\frontend
npm run dev
```
Open browser at `http://localhost:5173/`. Login with `admin` / `admin123`.

---

## 4. Initial Historical Data Sync

If starting with an empty Elasticsearch index or after a fresh database import:
```powershell
cd D:\TradeProject\backend
go run cmd/sync/main.go
```
This utility:
1. Verifies Elasticsearch mapping `trade_reports`.
2. Reads all historical records from `trade_details` joined with `client_details` and `stock_master`.
3. Performs a bulk index upsert directly into Elasticsearch.

---

## 5. API Testing & Verification

### 1. Authenticate (Login)
```powershell
$response = Invoke-RestMethod -Uri "http://localhost:8080/api/login" -Method Post -ContentType "application/json" -Body '{"username":"admin","password":"admin123"}'
$token = $response.token
Write-Host "JWT Token: $token"
```

### 2. Query Trades (with Filters and Search)
```powershell
$headers = @{ Authorization = "Bearer $token" }
$trades = Invoke-RestMethod -Uri "http://localhost:8080/api/trades?search=RELIANCE&trade_type=BUY&page=1&page_size=10" -Headers $headers
$trades.data | Format-Table trade_id, client_code, symbol, trade_type, quantity, price, net_amount
```

### 3. Fetch KPI Summary Aggregations
```powershell
$summary = Invoke-RestMethod -Uri "http://localhost:8080/api/trades/summary?trade_type=BUY" -Headers $headers
$summary | Format-List
```

### 4. Execute a New Trade
```powershell
$body = @{
    client_id  = "CL1001"
    symbol     = "INFY"
    exchange   = "NSE"
    segment    = "EQUITY"
    trade_type = "BUY"
    quantity   = 25
    price      = 1850.00
} | ConvertTo-Json

$tradeResult = Invoke-RestMethod -Uri "http://localhost:8080/api/trades" -Method Post -ContentType "application/json" -Body $body -Headers $headers
$tradeResult | Format-List
```

### 5. Live Delta CDC Verification
Run in SSMS:
```sql
UPDATE client_details 
SET first_name = 'Dev', updated_on = GETDATE() 
WHERE client_id = 'CL1001';
```
Backend console will instantly log:
```
[REALTIME CDC] ⚡ CLIENT TABLE UPDATE DETECTED: 1 record(s)
[REALTIME CDC] 🔄 Client: CL1001 | Code: ALOK001 | Name: Dev Kumar
[Kafka Producer] 📤 Streamed CLIENT_UPDATED to topic 'client-events' (Key: CL1001)
[REALTIME CDC] 🚀 Elasticsearch synced client CL1001 ('Dev Kumar') across 401 trade report(s)!
```

---

## 6. Troubleshooting Common Issues

### Issue 1: Port Already in Use (e.g. 8080 or 9092)
Check which process owns the port:
```powershell
Get-NetTCPConnection -LocalPort 8080, 9092, 9200 -State Listen | Select-Object LocalPort, OwningProcess
```
Kill the rogue process if necessary:
```powershell
Stop-Process -Id <OwningProcessId> -Force
```

### Issue 2: SQL Server Connection Fails in Sandbox
- Sandboxed execution environments block UDP port 1434 (SQL Browser) and named pipe resolution.
- Always run database migration or testing commands with sandbox elevation if executing from automation tools.

### Issue 3: Elasticsearch HTTP 400 on Substring Search
- Check that the query in `elastic/client.go` does not apply `case_insensitive: true` to analyzed `text` fields (`company_name`, `first_name`, `last_name`). Text fields must use `match_phrase_prefix` or `query_string`.

### Issue 4: React Blank Screen
- Check the browser developer console (F12 ➔ Console).
- A missing icon import in `TradeTable.jsx` or undefined prop can break the React render tree.
- Run `npm run build` inside `frontend/` to check for unresolved imports or syntax issues.
