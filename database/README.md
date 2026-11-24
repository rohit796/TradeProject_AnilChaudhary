# TradeFlow Database Migrations (Microsoft SQL Server)

This directory contains the official database migrations, schema definitions, and seed data scripts for the **TradeFlow** transactional OLTP database (`TradeDB`).

---

## 1. Migration Files

| File | Purpose | Description |
|:---|:---|:---|
| [`01_schema.sql`](file:///d:/TradeProject/database/01_schema.sql) | **DDL Schema & Triggers** | Creates database `TradeDB`, production tables (`client_details`, `stock_master`, `trade_details`), primary/foreign keys, indexes, and automatic CDC `updated_on` triggers. |
| [`02_seed_data.sql`](file:///d:/TradeProject/database/02_seed_data.sql) | **DML Seed Data** | Cleans existing tables and seeds 3 clients, 3 stocks, and generates 1,200 realistic trades distributed across clients and symbols. |
| [`migrate.sql`](file:///d:/TradeProject/database/migrate.sql) | **Master Runner** | Single idempotent script that runs the full migration and seeds the database if empty. |

---

## 2. Architectural Design Rules

1. **Zero Tracking Columns**:
   - Production domain tables (`trade_details`, `client_details`, `stock_master`) use strictly natural business columns.
   - **Never** add sync flags (e.g. `is_synced`, `sync_status`, `exported`) or shadow tables to domain tables.
2. **High-Watermark Change Data Capture (CDC)**:
   - Each table maintains an `updated_on DATETIME2` column with a default of `GETDATE()`.
   - Automatic triggers (`trg_*_updated_on`) update `updated_on = GETDATE()` on every `UPDATE` statement.
   - The Go CDC engine polls with `WHERE updated_on > @watermark` to capture delta modifications.

---

## 3. How to Execute Migrations

### Method 1: Using SQL Server Management Studio (SSMS)
1. Open SSMS and connect to instance: `LAPTOP-OV8LM74T\MSSQLSERVERANIL` (Windows Authentication).
2. Open `database\migrate.sql` (or `01_schema.sql` followed by `02_seed_data.sql`).
3. Click **Execute** (or press `F5`).

### Method 2: Using `sqlcmd` (Command Line)
```powershell
sqlcmd -S "LAPTOP-OV8LM74T\MSSQLSERVERANIL" -E -i "D:\TradeProject\database\migrate.sql"
```

---

## 4. Verification Queries

```sql
USE TradeDB;
GO

-- 1. Check Clients (Expect: 3)
SELECT client_id, client_code, first_name, last_name, status FROM client_details;

-- 2. Check Stocks (Expect: 3)
SELECT symbol, company_name, exchange, sector, is_active FROM stock_master;

-- 3. Check Trades Summary (Expect: 1,200)
SELECT 
    COUNT(*) AS total_trades,
    SUM(CASE WHEN trade_type = 'BUY' THEN 1 ELSE 0 END) AS buy_trades,
    SUM(CASE WHEN trade_type = 'SELL' THEN 1 ELSE 0 END) AS sell_trades,
    SUM(trade_value) AS total_gross_turnover,
    SUM(net_amount) AS total_net_amount
FROM trade_details;
```
