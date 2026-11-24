-- ==============================================================================
-- TRADE REPORTING SYSTEM - DATABASE MIGRATION SCRIPT 02: SEED DATA
-- Purpose: Seeds baseline clients, stocks, and generates 1,200 realistic trades
-- Database: TradeDB
-- ==============================================================================

USE TradeDB;
GO

PRINT 'Starting seed migration...';
GO

-- 1. CLEAN EXISTING DATA (Optional safe cleanup for reproducible state)
DELETE FROM trade_details;
DELETE FROM client_details;
DELETE FROM stock_master;
GO

-- ------------------------------------------------------------------------------
-- 2. INSERT SEED CLIENTS
-- ------------------------------------------------------------------------------
INSERT INTO client_details (
    client_id, client_code, first_name, last_name, email, phone_number,
    pan_number, account_type, risk_category, status, address, created_at, updated_on
) VALUES 
('CL1001', 'ALOK001', 'Alok', 'Sharma', 'alok.sharma@trademail.com', '+919876543210', 'ABCPS1234F', 'RETAIL', 'MEDIUM', 'ACTIVE', '402 Sunrise Heights, Bandra West, Mumbai', GETDATE(), GETDATE()),
('CL1002', 'PRIY002', 'Priya', 'Verma', 'priya.verma@trademail.com', '+919876543211', 'DEFPS5678G', 'HNI', 'HIGH', 'ACTIVE', '12 Emerald Park, Indiranagar, Bangalore', GETDATE(), GETDATE()),
('CL1003', 'RAVI003', 'Ravi', 'Patel', 'ravi.patel@trademail.com', '+919876543212', 'GHIPS9012H', 'INSTITUTIONAL', 'LOW', 'ACTIVE', '78 Riverfront Road, Navrangpura, Ahmedabad', GETDATE(), GETDATE());
GO
PRINT '[OK] Inserted 3 seed clients.';
GO

-- ------------------------------------------------------------------------------
-- 3. INSERT SEED STOCKS (INSTRUMENTS)
-- ------------------------------------------------------------------------------
INSERT INTO stock_master (
    symbol, company_name, isin_code, series, exchange, sector,
    face_value, lot_size, tick_size, currency, is_active, created_at, updated_on
) VALUES
('RELIANCE', 'Reliance Industries Limited', 'INE002A01018', 'EQ', 'NSE', 'Energy & Petrochemicals', 10.00, 1, 0.05, 'INR', 1, GETDATE(), GETDATE()),
('TCS', 'Tata Consultancy Services Limited', 'INE467B01029', 'EQ', 'NSE', 'Information Technology', 1.00, 1, 0.05, 'INR', 1, GETDATE(), GETDATE()),
('HDFCBANK', 'HDFC Bank Limited', 'INE040A01034', 'EQ', 'NSE', 'Banking & Financials', 1.00, 1, 0.05, 'INR', 1, GETDATE(), GETDATE());
GO
PRINT '[OK] Inserted 3 seed stocks.';
GO

-- ------------------------------------------------------------------------------
-- 4. GENERATE 1,200 REALISTIC TRADES DISTRIBUTED ACROSS CLIENTS & STOCKS
-- ------------------------------------------------------------------------------
SET NOCOUNT ON;
DECLARE @i INT = 1;
DECLARE @total INT = 1200;

DECLARE @client_id NVARCHAR(32);
DECLARE @symbol NVARCHAR(20);
DECLARE @trade_type NVARCHAR(10);
DECLARE @qty INT;
DECLARE @base_price DECIMAL(18, 4);
DECLARE @price DECIMAL(18, 4);
DECLARE @trade_val DECIMAL(18, 4);
DECLARE @brokerage DECIMAL(10, 4) = 20.00;
DECLARE @taxes DECIMAL(10, 4);
DECLARE @net DECIMAL(18, 4);
DECLARE @trade_time DATETIME2;
DECLARE @trade_id NVARCHAR(50);
DECLARE @order_id NVARCHAR(50);

BEGIN TRANSACTION;

WHILE @i <= @total
BEGIN
    -- Rotate across 3 clients
    IF (@i % 3 = 1) SET @client_id = 'CL1001';
    ELSE IF (@i % 3 = 2) SET @client_id = 'CL1002';
    ELSE SET @client_id = 'CL1003';

    -- Rotate across 3 stocks
    IF (@i % 3 = 1) 
    BEGIN
        SET @symbol = 'RELIANCE';
        SET @base_price = 2950.00;
    END
    ELSE IF (@i % 3 = 2) 
    BEGIN
        SET @symbol = 'TCS';
        SET @base_price = 4120.00;
    END
    ELSE 
    BEGIN
        SET @symbol = 'HDFCBANK';
        SET @base_price = 1680.00;
    END

    -- Alternate Buy and Sell
    IF (@i % 2 = 1) SET @trade_type = 'BUY';
    ELSE SET @trade_type = 'SELL';

    -- Quantity between 10 and 150 shares
    SET @qty = 10 + (@i % 140);

    -- Realistic price fluctuation (+/- 2%)
    SET @price = @base_price + ((@i % 40) - 20) * 1.5;

    -- Financial calculations
    SET @trade_val = @qty * @price;
    SET @taxes = ROUND(@trade_val * 0.001, 2); -- 0.1% STT & regulatory turnover charges
    
    IF (@trade_type = 'BUY')
        SET @net = @trade_val + @brokerage + @taxes;
    ELSE
        SET @net = @trade_val - @brokerage - @taxes;

    -- Spread trade timestamps across the past 30 days
    SET @trade_time = DATEADD(MINUTE, -((@total - @i) * 35), GETDATE());

    SET @trade_id = 'TRD-' + RIGHT('000000' + CAST(@i AS NVARCHAR(10)), 6);
    SET @order_id = 'ORD-' + RIGHT('000000' + CAST(10000 + @i AS NVARCHAR(10)), 6);

    INSERT INTO trade_details (
        trade_id, order_id, client_id, symbol, exchange, segment,
        trade_type, quantity, price, trade_value, brokerage, taxes_charges,
        net_amount, trade_status, trade_time, created_at, updated_on
    ) VALUES (
        @trade_id, @order_id, @client_id, @symbol, 'NSE', 'EQUITY',
        @trade_type, @qty, @price, @trade_val, @brokerage, @taxes,
        @net, 'EXECUTED', @trade_time, @trade_time, @trade_time
    );

    SET @i = @i + 1;
END;

COMMIT TRANSACTION;
GO

PRINT '[OK] Inserted 1,200 realistic historical trades.';
GO

-- ------------------------------------------------------------------------------
-- 5. VERIFICATION
-- ------------------------------------------------------------------------------
PRINT '==============================================================================';
PRINT 'SEEDING VERIFICATION SUMMARY:';
SELECT COUNT(*) AS total_clients FROM client_details;
SELECT COUNT(*) AS total_stocks FROM stock_master;
SELECT 
    COUNT(*) AS total_trades,
    SUM(CASE WHEN trade_type = 'BUY' THEN 1 ELSE 0 END) AS buy_trades,
    SUM(CASE WHEN trade_type = 'SELL' THEN 1 ELSE 0 END) AS sell_trades,
    SUM(trade_value) AS total_gross_turnover,
    SUM(brokerage) AS total_brokerage,
    SUM(net_amount) AS total_net_amount
FROM trade_details;
PRINT '==============================================================================';
GO
