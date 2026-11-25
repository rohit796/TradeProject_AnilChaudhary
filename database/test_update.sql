-- ==============================================================================
-- TEST DATA CHANGES: run this, then click Refresh in the dashboard (http://localhost:5173)
--   powershell -ExecutionPolicy Bypass -File .\scripts\sql.ps1 -File .\database\test_update.sql
-- Undo afterwards with database\test_cleanup.sql
-- ==============================================================================
USE TradeDB;
GO

-- 1) New trade: appears at the TOP of the dashboard as TRD-DEMO-001 (Alok Sharma, TCS, BUY 25)
INSERT INTO trade_details
    (trade_id, order_id, client_id, symbol, trade_type, quantity, price, trade_value, net_amount)
VALUES
    ('TRD-DEMO-001', 'ORD-DEMO-001', 'CL1001', 'TCS', 'BUY', 25, 4000.00, 100000.00, 100020.00);
PRINT 'Inserted TRD-DEMO-001';
GO

-- 2) Change that trade's price: dashboard shows price 4,250.00 and value 1,06,250.00
UPDATE trade_details
SET price = 4250.00, trade_value = 106250.00, net_amount = 106270.00
WHERE trade_id = 'TRD-DEMO-001';
PRINT 'Updated TRD-DEMO-001 price to 4250';
GO

-- 3) Rename a client: every Priya Verma trade shows "Priya DEMO"
UPDATE client_details SET first_name = 'Priya DEMO' WHERE client_id = 'CL1002';
PRINT 'Renamed client CL1002 to Priya DEMO';
GO
