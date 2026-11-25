# TradeFlow: Run, Update Data, and Test (this PC)

This PC has no SQL Server install, so the database runs as **SQL Server 2022 (Developer edition) in Docker**.
Everything else runs from folders next to the project:

```
C:\Users\Pratham\Desktop\Anil\
├── elasticsearch-9.5.7\              Elasticsearch (uses its own bundled JDK)
├── kafka_2.13-3.9.2\                 Kafka 3.9.2, KRaft mode (uses system Java 8)
└── TradeProject_AnilChaudhary-main\  this project  <- run all commands from here
```

## How data flows

```
SQL Server (TradeDB, Docker, localhost:1433)
   │  backend polls every 3 s for rows where updated_on > last seen value
   ▼
Go backend (localhost:8081) ──► Kafka topics (localhost:9092)
                                  trade-events / client-events / stock-events
                                          │ backend consumer
                                          ▼
                                Elasticsearch index trade_reports (localhost:9200)
                                          │
React dashboard (localhost:5173) ◄── /api/reports/trades (reads Elasticsearch only)
```

**Write data in SQL Server.** The dashboard reads Elasticsearch, and SQL changes reach it within about 3 seconds.

---

## 1. Files you run

All scripts are in `scripts\`. Run them from the project folder with
`powershell -ExecutionPolicy Bypass -File .\scripts\<name>.ps1`.

| When | Script | What it does |
|---|---|---|
| **Every day** | `start-all.ps1` | Starts everything in order, one window each (needs Docker Desktop running) |
| Or one by one | `start-elasticsearch.ps1` | Elasticsearch on 9200 |
| | `start-kafka.ps1` | Kafka on 9092 (formats storage on the very first run) |
| | `start-sqlserver.ps1` | Starts or creates the Docker container `tradeflow-sql` on 1433 |
| | `backend\trade-backend.exe` | Backend API on 8081 (run from inside `backend\`) |
| | `npm run dev` | Dashboard on 5173 (run from inside `frontend\`) |
| **First time only** | `init-database.ps1` | Creates TradeDB (tables + triggers) and seed data |
| **First time only** | `Invoke-RestMethod http://localhost:8081/api/sync/trades -Method Post` | Loads all SQL trades into Elasticsearch through Kafka |
| Anytime | `sql.ps1` | Runs SQL against TradeDB (see section 2) |
| Anytime | `test-pipeline.ps1` | Automatic end-to-end test (see section 3) |
| Rarely | `create-kafka-topics.ps1` | Creates the 3 topics by hand (the backend also does this on startup) |

Start order matters: **Elasticsearch, Kafka and SQL Server first, then the backend, then the frontend.**
`start-all.ps1` does this for you.

Open **http://localhost:5173** and log in with `VITE_AUTH_USERNAME` / `VITE_AUTH_PASSWORD` from `frontend\.env`
(they must match `AUTH_USERNAME` / `AUTH_PASSWORD` in `backend\.env`).

If you change Go code, rebuild first: `cd backend; go build -o trade-backend.exe main.go`.

**Stop:** press Ctrl+C in each window. SQL Server keeps running in Docker until `docker stop tradeflow-sql`.

---

## 2. Updating data

### Where to run queries

- **Easiest:** `.\scripts\sql.ps1 "your query"` (reads the password from `backend\.env`)
- **Interactive:** `.\scripts\sql.ps1` with no query. Type a query, press Enter, type `GO`, press Enter. `EXIT` quits.
- **A .sql file:** `.\scripts\sql.ps1 -File .\database\my_changes.sql`
- **GUI:** SSMS, Azure Data Studio, or the VS Code "SQL Server (mssql)" extension. Server `localhost,1433`,
  SQL login `sa`, password = `MSSQL_PASSWORD` in `backend\.env`, database `TradeDB`, turn on "Trust server certificate".

Current master data: clients `CL1001` (Alok), `CL1002` (Priya), `CL1003` (Ravi); stocks `TCS`, `RELIANCE`, `HDFCBANK`.

### Add a trade

```sql
INSERT INTO trade_details
  (trade_id, order_id, client_id, symbol, trade_type, quantity, price, trade_value, net_amount)
VALUES
  ('TRD-900001', 'ORD-900001', 'CL1001', 'TCS', 'BUY', 10, 4000, 40000, 40020);
```

`client_id` and `symbol` must already exist (foreign keys). `trade_time`, `updated_on`, `exchange`, `segment`,
`brokerage` and `trade_status` have defaults. Or use the **New Trade** button in the dashboard, which writes to SQL
and sends the trade straight to Kafka and Elasticsearch.

### Change a trade

```sql
UPDATE trade_details
SET price = 4200, trade_value = 42000, net_amount = 42020
WHERE trade_id = 'TRD-900001';
```

### Change a client (all of that client's trades update in the dashboard)

```sql
UPDATE client_details SET first_name = 'Alok K', account_type = 'HNI' WHERE client_id = 'CL1001';
```

### Change a stock (all trades for that symbol update)

```sql
UPDATE stock_master SET company_name = 'TCS Ltd', sector = 'IT Services' WHERE symbol = 'TCS';
```

### Add a new client / stock (then you can trade them)

```sql
INSERT INTO client_details (client_id, client_code, first_name, last_name, email, phone_number, pan_number, account_type)
VALUES ('CL1004', 'NEHA004', 'Neha', 'Singh', 'neha.singh@trademail.com', '+919800000004', 'ABCPN4444Q', 'RETAIL');

INSERT INTO stock_master (symbol, company_name, isin_code, sector)
VALUES ('INFY', 'Infosys Limited', 'INE009A01021', 'Information Technology');
```

The dashboard's client and stock dropdowns read SQL directly, so new entries appear after **Refresh**.

### Rules to know

- **UPDATEs sync automatically:** a trigger on each table sets `updated_on` to now, and the backend picks the row up.
- **Don't set `updated_on` yourself** to an older time, or the change will be skipped.
- **DELETEs are NOT synced.** Change detection only sees new or updated rows. After deleting in SQL, also remove the trade
  from Elasticsearch: `Invoke-RestMethod "http://localhost:9200/trade_reports/_doc/TRD-900001?refresh=true" -Method Delete`
- **Bulk changes made while the backend is stopped** are skipped on restart, because it starts from the newest
  `updated_on` and only streams changes after that. To catch up, run a full re-sync:
  `Invoke-RestMethod http://localhost:8081/api/sync/trades -Method Post`.

---

## 3. Testing

### Automatic (recommended)

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\test-pipeline.ps1
```

It inserts a trade in SQL, updates its price, renames a client, changes a stock's sector, places a trade through
the API, searches, and checks Kafka. Each step must reach Elasticsearch within 12 s. Afterwards it undoes every
change and prints `ALL TESTS PASSED`.

### Watch the sync live

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\watch-sync.ps1
```

This refreshes every 2 s and shows the trade count in SQL vs Elasticsearch (`IN SYNC`), plus the last 5 changed
trades on each side. Run `database\test_update.sql` in another window, and `TRD-DEMO-001` appears in both lists within about 3 s.
The dashboard does **not** auto-refresh (the "Live Pipeline" dot is decoration), so click **Refresh** there.

### By hand

1. **Health:** `Invoke-RestMethod http://localhost:8081/api/health`. `database.connected` and
   `elasticsearch.connected` must both be `True`.
2. **Watch Kafka live** (leave this running in a second window):
   ```powershell
   ..\kafka_2.13-3.9.2\bin\windows\kafka-console-consumer.bat --topic trade-events --bootstrap-server localhost:9092 --property print.key=true
   ```
   Use `client-events` / `stock-events` for client and stock changes.
3. **Make a change** with one of the queries in section 2.
4. **Within about 3 s:** the backend window prints the row as JSON, the Kafka window shows the event, and the
   trade is in Elasticsearch:
   ```powershell
   Invoke-RestMethod http://localhost:9200/trade_reports/_doc/TRD-900001
   ```
5. **Dashboard:** click **Refresh**. The trade shows up, and searching its ID finds it.
6. **Totals check:** these two numbers should match:
   ```powershell
   .\scripts\sql.ps1 "SELECT COUNT(*) FROM trade_details"
   (Invoke-RestMethod http://localhost:9200/trade_reports/_count).count
   ```

### Useful API calls

| Call | Purpose |
|---|---|
| `GET  /api/health` | Status of SQL / Elasticsearch |
| `GET  /api/reports/trades?search=TCS&trade_type=BUY&page=1&limit=50` | Report query (Elasticsearch) |
| `GET  /api/clients`, `GET /api/stocks` | Master data (SQL) |
| `POST /api/trades` | Place a trade (SQL → Kafka → Elasticsearch) |
| `POST /api/sync/trades` | Full re-sync of all trades |
| `GET/POST/DELETE /api/kafka/topics` | Manage Kafka topics |

### Reset

- **Clear Elasticsearch** (keeps the index and its mapping) and re-sync:
  ```powershell
  Invoke-RestMethod "http://localhost:9200/trade_reports/_delete_by_query?refresh=true" -Method Post -ContentType 'application/json' -Body '{"query":{"match_all":{}}}'
  Invoke-RestMethod http://localhost:8081/api/sync/trades -Method Post
  ```
- **Fresh database:** `docker rm -f tradeflow-sql; docker volume rm tradeflow-sql-data`, then `start-sqlserver.ps1`,
  `init-database.ps1`, restart the backend, and clear and re-sync Elasticsearch as above.

---

## 4. Troubleshooting

| Problem | Fix |
|---|---|
| `start-sqlserver.ps1` fails | Start Docker Desktop first |
| Dashboard: "Elasticsearch is not connected" | Start Elasticsearch, then restart the backend (it checks Elasticsearch only at startup) |
| SQL changes don't appear | Backend window running? Check `api/health`. A DELETE never syncs (see section 2) |
| Kafka: "Cluster ID does not match" | Delete `kafka_2.13-3.9.2\data\` and run `start-kafka.ps1` again |
| Port already in use | `Get-NetTCPConnection -LocalPort 8081 -State Listen` to find the process |
| Backend on 8080 instead of 8081 | Apache (httpd) uses 8080 on this PC, so keep `PORT=8081` in `backend\.env` |

---

## 5. What was changed to make this work

- **Elasticsearch** (`config\elasticsearch.yml`): single node, security off (the backend uses plain HTTP and no login), 1 GB heap. Original saved as `elasticsearch.yml.orig`.
- **Kafka** (`config\kraft\server.properties`): data in `kafka_2.13-3.9.2\data\` instead of `/tmp`, 3 partitions by default. Original saved as `.orig`.
- **Kafka Windows fix** (`bin\windows\kafka-run-class.bat`): uses `libs\*` instead of listing every jar, which avoided *"The input line is too long"*. Original saved as `.orig`.
- **Kafka `wmic` fix** (`bin\windows\kafka-server-start.bat`): the original used `wmic` (missing on newer Windows 11) to choose the heap size, and failed. It now sets the 64-bit heap (1 GB) directly, so you can run `.\bin\windows\kafka-server-start.bat .\config\kraft\server.properties` straight from the Kafka folder. Original saved as `.orig`.
- **SQL Server in Docker**: `backend\.env` uses `MSSQL_SERVER=localhost:1433`, `MSSQL_USER=sa`, and a generated password. The container uses `TZ=Asia/Kolkata`, so `GETDATE()` is local time like a Windows install. To go back to the Windows instance, restore `backend\.env.orig` (empty user/password = Windows Authentication).
- **Backend port 8081** (8080 is used by Apache). `frontend\.env` points there.
- **`backend\kafka\` restored**: `.gitignore` had `kafka/`, which also ignored this Go package. It was rewritten (`admin.go`, `producer.go`, `consumer.go`), and `.gitignore` now uses `/kafka/`.
- **`backend\cmd\sync`** uses the shared `database.InitDB()` and `GetAllTradesForSync()`.
- **Trade times** (`backend\database\mssql.go`): SQL `DATETIME2` stores local time with no timezone, but the Go driver labelled it UTC, so times showed 5½ h ahead. Values sent to the API, Kafka and Elasticsearch are now labelled with the local timezone, and new trades are written in local time. CDC watermarks are left untouched on purpose.
- **Login error** no longer mentions `.env`.
- **Full sync data loss fixed** (`handlers.go`, `cmd\sync`, `GetAllTradesForSync`): the sync sent Kafka a partial trade (no status, brokerage, taxes or updated_on), and the consumer overwrote the complete Elasticsearch document with it. It now sends the full report, including `updated_on`.
- **`elasticsearch\mappings\trade_reports_mapping.json`** added. The backend creates the `trade_reports` index from it, using keyword fields for IDs, symbol and status, and text fields for names (AGENTS.md, Constraint 4).
- **Go** installed per-user at `%LOCALAPPDATA%\Programs\go` (no admin needed).
