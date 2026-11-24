-- ==============================================================================
-- MASTER MIGRATION RUNNER: INITIALIZE TRADEDB SCHEMA & SEED DATA
-- Target Instance: LAPTOP-OV8LM74T\MSSQLSERVERANIL
-- Database: TradeDB
-- Usage: Run this single script in SSMS or via sqlcmd:
--        sqlcmd -S "LAPTOP-OV8LM74T\MSSQLSERVERANIL" -E -i "database\migrate.sql"
-- ==============================================================================

USE master;
GO

PRINT '==============================================================================';
PRINT 'STARTING TRADEDB COMPLETE DATABASE MIGRATION';
PRINT '==============================================================================';
GO

-- STEP 1: CREATE DATABASE IF NOT EXISTS
IF NOT EXISTS (SELECT name FROM sys.databases WHERE name = N'TradeDB')
BEGIN
    CREATE DATABASE TradeDB;
    PRINT '[OK] Database TradeDB created.';
END
ELSE
BEGIN
    PRINT '[INFO] Database TradeDB exists.';
END
GO

USE TradeDB;
GO

-- STEP 2: CREATE DOMAIN TABLES & CDC TRIGGERS
-- client_details
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'client_details')
BEGIN
    CREATE TABLE client_details (
        client_id NVARCHAR(32) NOT NULL PRIMARY KEY,
        client_code NVARCHAR(20) NOT NULL UNIQUE,
        first_name NVARCHAR(100) NOT NULL,
        last_name NVARCHAR(100) NOT NULL,
        email NVARCHAR(150) NOT NULL UNIQUE,
        phone_number NVARCHAR(20) NOT NULL,
        pan_number NVARCHAR(15) NOT NULL UNIQUE,
        account_type NVARCHAR(30) NOT NULL DEFAULT 'RETAIL',
        risk_category NVARCHAR(20) NOT NULL DEFAULT 'MEDIUM',
        status NVARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
        address NVARCHAR(MAX),
        created_at DATETIME2 NOT NULL DEFAULT GETDATE(),
        updated_on DATETIME2 NOT NULL DEFAULT GETDATE()
    );

    CREATE NONCLUSTERED INDEX idx_client_code ON client_details(client_code);
    CREATE NONCLUSTERED INDEX idx_client_status ON client_details(status);
    CREATE NONCLUSTERED INDEX idx_client_updated_on ON client_details(updated_on);
    PRINT '[OK] Table client_details created.';
END
GO

-- stock_master
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'stock_master')
BEGIN
    CREATE TABLE stock_master (
        symbol NVARCHAR(20) NOT NULL PRIMARY KEY,
        company_name NVARCHAR(200) NOT NULL,
        isin_code NVARCHAR(20) NOT NULL UNIQUE,
        series NVARCHAR(10) NOT NULL DEFAULT 'EQ',
        exchange NVARCHAR(10) NOT NULL DEFAULT 'NSE',
        sector NVARCHAR(100),
        face_value DECIMAL(10, 2) NOT NULL DEFAULT 10.00,
        lot_size INT NOT NULL DEFAULT 1,
        tick_size DECIMAL(6, 4) NOT NULL DEFAULT 0.05,
        currency NVARCHAR(10) NOT NULL DEFAULT 'INR',
        is_active BIT NOT NULL DEFAULT 1,
        created_at DATETIME2 NOT NULL DEFAULT GETDATE(),
        updated_on DATETIME2 NOT NULL DEFAULT GETDATE()
    );

    CREATE NONCLUSTERED INDEX idx_stock_sector ON stock_master(sector);
    CREATE NONCLUSTERED INDEX idx_stock_isin ON stock_master(isin_code);
    CREATE NONCLUSTERED INDEX idx_stock_updated_on ON stock_master(updated_on);
    PRINT '[OK] Table stock_master created.';
END
GO

-- trade_details
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'trade_details')
BEGIN
    CREATE TABLE trade_details (
        trade_id NVARCHAR(50) NOT NULL PRIMARY KEY,
        order_id NVARCHAR(50) NOT NULL,
        client_id NVARCHAR(32) NOT NULL,
        symbol NVARCHAR(20) NOT NULL,
        exchange NVARCHAR(10) NOT NULL DEFAULT 'NSE',
        segment NVARCHAR(20) NOT NULL DEFAULT 'EQUITY',
        trade_type NVARCHAR(10) NOT NULL,
        quantity INT NOT NULL CHECK (quantity > 0),
        price DECIMAL(18, 4) NOT NULL CHECK (price > 0),
        trade_value DECIMAL(18, 4) NOT NULL,
        brokerage DECIMAL(10, 4) NOT NULL DEFAULT 20.00,
        taxes_charges DECIMAL(10, 4) NOT NULL DEFAULT 0.00,
        net_amount DECIMAL(18, 4) NOT NULL,
        trade_status NVARCHAR(20) NOT NULL DEFAULT 'EXECUTED',
        trade_time DATETIME2 NOT NULL DEFAULT GETDATE(),
        settlement_date DATE,
        created_at DATETIME2 NOT NULL DEFAULT GETDATE(),
        updated_on DATETIME2 NOT NULL DEFAULT GETDATE(),
        CONSTRAINT fk_trade_client FOREIGN KEY (client_id) REFERENCES client_details(client_id) ON DELETE CASCADE,
        CONSTRAINT fk_trade_stock FOREIGN KEY (symbol) REFERENCES stock_master(symbol)
    );

    CREATE NONCLUSTERED INDEX idx_trades_client_id ON trade_details(client_id);
    CREATE NONCLUSTERED INDEX idx_trades_symbol ON trade_details(symbol);
    CREATE NONCLUSTERED INDEX idx_trades_time ON trade_details(trade_time DESC);
    CREATE NONCLUSTERED INDEX idx_trades_type ON trade_details(trade_type);
    CREATE NONCLUSTERED INDEX idx_trades_status ON trade_details(trade_status);
    CREATE NONCLUSTERED INDEX idx_trades_updated_on ON trade_details(updated_on);
    PRINT '[OK] Table trade_details created.';
END
GO

-- Triggers
CREATE OR ALTER TRIGGER trg_client_details_updated_on
ON client_details AFTER UPDATE AS
BEGIN
    SET NOCOUNT ON; IF TRIGGER_NESTLEVEL() > 1 RETURN;
    UPDATE c SET updated_on = GETDATE() FROM client_details c INNER JOIN inserted i ON c.client_id = i.client_id;
END;
GO

CREATE OR ALTER TRIGGER trg_stock_master_updated_on
ON stock_master AFTER UPDATE AS
BEGIN
    SET NOCOUNT ON; IF TRIGGER_NESTLEVEL() > 1 RETURN;
    UPDATE s SET updated_on = GETDATE() FROM stock_master s INNER JOIN inserted i ON s.symbol = i.symbol;
END;
GO

CREATE OR ALTER TRIGGER trg_trade_details_updated_on
ON trade_details AFTER UPDATE AS
BEGIN
    SET NOCOUNT ON; IF TRIGGER_NESTLEVEL() > 1 RETURN;
    UPDATE t SET updated_on = GETDATE() FROM trade_details t INNER JOIN inserted i ON t.trade_id = i.trade_id;
END;
GO
PRINT '[OK] All CDC triggers active.';
GO

-- STEP 3: SEED DATA IF TABLES ARE EMPTY
IF NOT EXISTS (SELECT 1 FROM client_details)
BEGIN
    INSERT INTO client_details (client_id, client_code, first_name, last_name, email, phone_number, pan_number, account_type, risk_category, status, address)
    VALUES 
    ('CL1001', 'ALOK001', 'Alok', 'Sharma', 'alok.sharma@trademail.com', '+919876543210', 'ABCPS1234F', 'RETAIL', 'MEDIUM', 'ACTIVE', '402 Sunrise Heights, Bandra West, Mumbai'),
    ('CL1002', 'PRIY002', 'Priya', 'Verma', 'priya.verma@trademail.com', '+919876543211', 'DEFPS5678G', 'HNI', 'HIGH', 'ACTIVE', '12 Emerald Park, Indiranagar, Bangalore'),
    ('CL1003', 'RAVI003', 'Ravi', 'Patel', 'ravi.patel@trademail.com', '+919876543212', 'GHIPS9012H', 'INSTITUTIONAL', 'LOW', 'ACTIVE', '78 Riverfront Road, Navrangpura, Ahmedabad');
    PRINT '[OK] Seed clients inserted.';
END
GO

IF NOT EXISTS (SELECT 1 FROM stock_master)
BEGIN
    INSERT INTO stock_master (symbol, company_name, isin_code, series, exchange, sector, face_value, lot_size, tick_size, currency, is_active)
    VALUES
    ('RELIANCE', 'Reliance Industries Limited', 'INE002A01018', 'EQ', 'NSE', 'Energy & Petrochemicals', 10.00, 1, 0.05, 'INR', 1),
    ('TCS', 'Tata Consultancy Services Limited', 'INE467B01029', 'EQ', 'NSE', 'Information Technology', 1.00, 1, 0.05, 'INR', 1),
    ('HDFCBANK', 'HDFC Bank Limited', 'INE040A01034', 'EQ', 'NSE', 'Banking & Financials', 1.00, 1, 0.05, 'INR', 1);
    PRINT '[OK] Seed stocks inserted.';
END
GO

IF NOT EXISTS (SELECT 1 FROM trade_details)
BEGIN
    PRINT 'Generating 1,200 initial trades...';
    SET NOCOUNT ON;
    DECLARE @i INT = 1;
    DECLARE @total INT = 1200;
    DECLARE @client_id NVARCHAR(32), @symbol NVARCHAR(20), @trade_type NVARCHAR(10);
    DECLARE @qty INT, @base_price DECIMAL(18, 4), @price DECIMAL(18, 4), @trade_val DECIMAL(18, 4);
    DECLARE @brokerage DECIMAL(10, 4) = 20.00, @taxes DECIMAL(10, 4), @net DECIMAL(18, 4);
    DECLARE @trade_time DATETIME2, @trade_id NVARCHAR(50), @order_id NVARCHAR(50);

    BEGIN TRANSACTION;
    WHILE @i <= @total
    BEGIN
        IF (@i % 3 = 1) SET @client_id = 'CL1001';
        ELSE IF (@i % 3 = 2) SET @client_id = 'CL1002';
        ELSE SET @client_id = 'CL1003';

        IF (@i % 3 = 1) BEGIN SET @symbol = 'RELIANCE'; SET @base_price = 2950.00; END
        ELSE IF (@i % 3 = 2) BEGIN SET @symbol = 'TCS'; SET @base_price = 4120.00; END
        ELSE BEGIN SET @symbol = 'HDFCBANK'; SET @base_price = 1680.00; END

        IF (@i % 2 = 1) SET @trade_type = 'BUY'; ELSE SET @trade_type = 'SELL';
        SET @qty = 10 + (@i % 140);
        SET @price = @base_price + ((@i % 40) - 20) * 1.5;
        SET @trade_val = @qty * @price;
        SET @taxes = ROUND(@trade_val * 0.001, 2);
        IF (@trade_type = 'BUY') SET @net = @trade_val + @brokerage + @taxes;
        ELSE SET @net = @trade_val - @brokerage - @taxes;

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
    PRINT '[OK] 1,200 initial trades generated.';
END
GO

PRINT '==============================================================================';
PRINT 'MIGRATION COMPLETE. CURRENT RECORD COUNTS:';
SELECT COUNT(*) AS total_clients FROM client_details;
SELECT COUNT(*) AS total_stocks FROM stock_master;
SELECT COUNT(*) AS total_trades FROM trade_details;
PRINT '==============================================================================';
GO
