---
name: tradeflow-pipeline
description: >-
  Standard operating procedures, verification runbooks, and architectural reference
  for the TradeFlow real-time financial trading, CDC streaming, and reporting pipeline
  (MS SQL Server -> Kafka -> Elasticsearch -> Go Backend -> React Frontend).
---

# TradeFlow Pipeline Skill

This skill defines the technical specifications, development standards, operational workflows, and verification procedures for the **TradeFlow Financial Trading & Reporting Platform**.

---

## 1. Pipeline Architecture Overview

TradeFlow is an institutional trade reporting platform engineered with a decoupled event-driven architecture:

```
[MS SQL Server: TradeDB] 
        │
        ▼ (Every 3s Delta Poller: ISNULL(updated_on, trade_time) > watermark)
[Go Backend CDC Engine]
        │
        ▼ (Streams Delta Keyed Events)
[Apache Kafka: Port 9092]
   ├── topic: trade-events   (Key: trade_id)
   ├── topic: client-events  (Key: client_id)
   └── topic: stock-events   (Key: symbol)
        │
        ▼ (Kafka Consumer Goroutines with Local Reference Cache)
[Elasticsearch 9.5.7: Port 9200] (Index: trade_reports)
        │
        ▼ (Sub-millisecond Search & Analytics)
[Go REST API: Port 8080]
        │
        ▼ (REST JSON API / Polling)
[React 18 Trading UI: Port 5173]
```

---

## 2. Multi-Table Change Data Capture (CDC) Rules

### Core Principle: Zero Extra Columns on Domain Tables
- Domain tables (`trade_details`, `client_details`, `stock_master`) use their natural business columns and standard `updated_on` / `created_at` timestamps.
- **Never** add boolean sync flags (`is_synced`), extra status columns, or triggers to the production domain tables.

### High-Watermark Delta Sync Algorithm
Each table maintains an **independent watermark** initialized at backend startup to the table's current `MAX(ISNULL(updated_on, created_at))`:
1. `clientWatermark`: Initialized via `SELECT MAX(ISNULL(updated_on, created_at)) FROM client_details`.
2. `stockWatermark`: Initialized via `SELECT MAX(ISNULL(updated_on, created_at)) FROM stock_master`.
3. `tradeWatermark`: Initialized via `SELECT MAX(ISNULL(updated_on, trade_time)) FROM trade_details`.

### Event Routing & Processing
- **`client_details` modification**:
  - Captured via `WHERE ISNULL(updated_on, created_at) > clientWatermark`.
  - Streamed to Kafka topic `client-events` with key `client_id`.
  - Consumer updates `clientCache` and updates all matching documents in Elasticsearch via `_update_by_query`.
- **`stock_master` modification**:
  - Captured via `WHERE ISNULL(updated_on, created_at) > stockWatermark`.
  - Streamed to Kafka topic `stock-events` with key `symbol`.
  - Consumer updates `stockCache` and updates all matching documents in Elasticsearch via `_update_by_query`.
- **`trade_details` modification / insert**:
  - Captured via `WHERE ISNULL(t.updated_on, t.trade_time) > tradeWatermark`.
  - Joined with reference data and streamed to Kafka topic `trade-events` with key `trade_id`.
  - Consumer enriches and upserts document into Elasticsearch `trade_reports` index.

---

## 3. Elasticsearch Querying Standards

### Keyword vs. Text Fields
In the `trade_reports` mapping:
- **`keyword` fields**: `trade_id`, `order_id`, `client_id`, `client_code`, `symbol`, `exchange`, `segment`, `trade_type`, `sector`, `trade_status`.
- **`text` fields**: `first_name`, `last_name`, `company_name`.

### Substring Search Query Rule
- **On `keyword` fields**: Use `wildcard` queries with `case_insensitive: true` (e.g. `*term*`).
- **On `text` fields**: Use `match_phrase_prefix`, `match`, or `query_string` with wildcard analysis. **Never** pass `case_insensitive: true` on `text` fields as Elasticsearch will reject the query with HTTP 400.

---

## 4. Runbook: Starting & Verifying the Platform

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
```

### Step 5: Start React Frontend
```powershell
cd D:\TradeProject\frontend
npm run dev
```

### Step 6: Verify Real-Time Syncing
Run an update in SQL Server Management Studio:
```sql
UPDATE client_details 
SET updated_on = GETDATE(), first_name = 'Ravi' 
WHERE client_id = 'CL1001';
```
**Expected Results**:
1. Go terminal logs:
   ```
   [REALTIME CDC] ⚡ CLIENT TABLE UPDATE DETECTED: 1 record(s)
   [REALTIME CDC] 🔄 Client: CL1001 | Code: ALOK001 | Name: Ravi Sharma
   [Kafka Producer] 📤 Streamed CLIENT_UPDATED to topic 'client-events' (Key: CL1001)
   [REALTIME CDC] 🚀 Elasticsearch synced client CL1001 ('Ravi Sharma') across 401 trade report(s)!
   ```
2. In browser at `http://localhost:5173/`, all trades for `CL1001` automatically update to show client name `Ravi Sharma`.
