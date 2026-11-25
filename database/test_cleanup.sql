-- ==============================================================================
-- UNDO database\test_update.sql
--   powershell -ExecutionPolicy Bypass -File .\scripts\sql.ps1 -File .\database\test_cleanup.sql
-- The DELETE is NOT synced automatically. Also remove the trade from Elasticsearch:
--   Invoke-RestMethod "http://localhost:9200/trade_reports/_doc/TRD-DEMO-001?refresh=true" -Method Delete
-- ==============================================================================
USE TradeDB;
GO

UPDATE client_details SET first_name = 'Priya' WHERE client_id = 'CL1002';
PRINT 'Client CL1002 name restored to Priya';
GO

DELETE FROM trade_details WHERE trade_id = 'TRD-DEMO-001';
PRINT 'Deleted TRD-DEMO-001 from SQL (remove it from Elasticsearch too, see top of file)';
GO
