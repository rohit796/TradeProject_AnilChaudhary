-- ==============================================================================
-- TRADE REPORTING SYSTEM - DATABASE MIGRATION SCRIPT 01: SCHEMA DEFINITION
-- Target Instance: LAPTOP-OV8LM74T\MSSQLSERVERANIL (or standard MSSQL 2019/2022)
-- Database: TradeDB
-- Architectural Constraint: Clean domain tables with natural high-watermark timestamps
-- (Zero vendor sync tracking columns or status flags on production tables)
-- ==============================================================================

USE master;
GO

-- 1. CREATE DATABASE
IF NOT EXISTS (SELECT name FROM sys.databases WHERE name = N'TradeDB')
BEGIN
    CREATE DATABASE TradeDB;
    PRINT '[OK] Database TradeDB created successfully.';
END
ELSE
BEGIN
    PRINT '[INFO] Database TradeDB already exists.';
END
GO

USE TradeDB;
GO

-- ------------------------------------------------------------------------------
-- 2. CLIENT DETAILS TABLE (Client / Investor Master)
-- ------------------------------------------------------------------------------
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
        account_type NVARCHAR(30) NOT NULL DEFAULT 'RETAIL',   -- RETAIL, HNI, INSTITUTIONAL, CORPORATE
        risk_category NVARCHAR(20) NOT NULL DEFAULT 'MEDIUM',  -- LOW, MEDIUM, HIGH
        status NVARCHAR(20) NOT NULL DEFAULT 'ACTIVE',         -- ACTIVE, INACTIVE, SUSPENDED
        address NVARCHAR(MAX),
        created_at DATETIME2 NOT NULL DEFAULT GETDATE(),
        updated_on DATETIME2 NOT NULL DEFAULT GETDATE()
    );

    CREATE NONCLUSTERED INDEX idx_client_code ON client_details(client_code);
    CREATE NONCLUSTERED INDEX idx_client_status ON client_details(status);
    CREATE NONCLUSTERED INDEX idx_client_updated_on ON client_details(updated_on);
    PRINT '[OK] Table client_details created.';
END
ELSE
BEGIN
    PRINT '[INFO] Table client_details already exists.';
    -- Ensure updated_on exists
    IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('client_details') AND name = 'updated_on')
    BEGIN
        ALTER TABLE client_details ADD updated_on DATETIME2 NOT NULL DEFAULT GETDATE();
        PRINT '[OK] Added updated_on column to client_details.';
    END
END
GO

-- ------------------------------------------------------------------------------
-- 3. STOCK MASTER TABLE (Financial Instrument / Security Master)
-- ------------------------------------------------------------------------------
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'stock_master')
BEGIN
    CREATE TABLE stock_master (
        symbol NVARCHAR(20) NOT NULL PRIMARY KEY,
        company_name NVARCHAR(200) NOT NULL,
        isin_code NVARCHAR(20) NOT NULL UNIQUE,
        series NVARCHAR(10) NOT NULL DEFAULT 'EQ',             -- EQ (Equity), BE, F&O
        exchange NVARCHAR(10) NOT NULL DEFAULT 'NSE',          -- NSE, BSE, NASDAQ
        sector NVARCHAR(100),                                 -- Energy, IT, Banking, FMCG, Auto
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
ELSE
BEGIN
    PRINT '[INFO] Table stock_master already exists.';
    -- Ensure updated_on exists
    IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('stock_master') AND name = 'updated_on')
    BEGIN
        ALTER TABLE stock_master ADD updated_on DATETIME2 NOT NULL DEFAULT GETDATE();
        PRINT '[OK] Added updated_on column to stock_master.';
    END
END
GO

-- ------------------------------------------------------------------------------
-- 4. TRADE DETAILS TABLE (Transactional Trade Executions)
-- ------------------------------------------------------------------------------
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'trade_details')
BEGIN
    CREATE TABLE trade_details (
        trade_id NVARCHAR(50) NOT NULL PRIMARY KEY,
        order_id NVARCHAR(50) NOT NULL,
        client_id NVARCHAR(32) NOT NULL,
        symbol NVARCHAR(20) NOT NULL,
        exchange NVARCHAR(10) NOT NULL DEFAULT 'NSE',
        segment NVARCHAR(20) NOT NULL DEFAULT 'EQUITY',       -- EQUITY, FUTURES, OPTIONS, COMMODITY
        trade_type NVARCHAR(10) NOT NULL,                     -- BUY or SELL
        quantity INT NOT NULL CHECK (quantity > 0),
        price DECIMAL(18, 4) NOT NULL CHECK (price > 0),
        trade_value DECIMAL(18, 4) NOT NULL,                -- quantity * price
        brokerage DECIMAL(10, 4) NOT NULL DEFAULT 20.00,
        taxes_charges DECIMAL(10, 4) NOT NULL DEFAULT 0.00,   -- STT, GST, Stamp Duty
        net_amount DECIMAL(18, 4) NOT NULL,                 -- Buy: Value + Charges | Sell: Value - Charges
        trade_status NVARCHAR(20) NOT NULL DEFAULT 'EXECUTED', -- EXECUTED, SETTLED, CANCELLED
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
ELSE
BEGIN
    PRINT '[INFO] Table trade_details already exists.';
    -- Ensure updated_on exists
    IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('trade_details') AND name = 'updated_on')
    BEGIN
        ALTER TABLE trade_details ADD updated_on DATETIME2 NOT NULL DEFAULT GETDATE();
        PRINT '[OK] Added updated_on column to trade_details.';
    END
END
GO

-- ------------------------------------------------------------------------------
-- 5. AUTO-UPDATE TRIGGERS FOR NATURAL CDC WATERMARK
-- Ensures updated_on automatically advances on any direct UPDATE statement
-- ------------------------------------------------------------------------------

-- Trigger on client_details
CREATE OR ALTER TRIGGER trg_client_details_updated_on
ON client_details
AFTER UPDATE
AS
BEGIN
    SET NOCOUNT ON;
    IF TRIGGER_NESTLEVEL() > 1 RETURN;

    UPDATE c
    SET updated_on = GETDATE()
    FROM client_details c
    INNER JOIN inserted i ON c.client_id = i.client_id;
END;
GO
PRINT '[OK] Trigger trg_client_details_updated_on configured.';
GO

-- Trigger on stock_master
CREATE OR ALTER TRIGGER trg_stock_master_updated_on
ON stock_master
AFTER UPDATE
AS
BEGIN
    SET NOCOUNT ON;
    IF TRIGGER_NESTLEVEL() > 1 RETURN;

    UPDATE s
    SET updated_on = GETDATE()
    FROM stock_master s
    INNER JOIN inserted i ON s.symbol = i.symbol;
END;
GO
PRINT '[OK] Trigger trg_stock_master_updated_on configured.';
GO

-- Trigger on trade_details
CREATE OR ALTER TRIGGER trg_trade_details_updated_on
ON trade_details
AFTER UPDATE
AS
BEGIN
    SET NOCOUNT ON;
    IF TRIGGER_NESTLEVEL() > 1 RETURN;

    UPDATE t
    SET updated_on = GETDATE()
    FROM trade_details t
    INNER JOIN inserted i ON t.trade_id = i.trade_id;
END;
GO
PRINT '[OK] Trigger trg_trade_details_updated_on configured.';
GO

PRINT '==============================================================================';
PRINT 'MIGRATION 01 COMPLETED: Schema, indexes, and CDC triggers are ready.';
PRINT '==============================================================================';
GO
