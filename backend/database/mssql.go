package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	_ "github.com/microsoft/go-mssqldb"
	"trade-reporting-backend/models"
)

var DB *sql.DB

// EnsureDB guarantees an active connection to SQL Server, reconnecting if needed
func EnsureDB() (*sql.DB, error) {
	if DB != nil {
		if err := DB.Ping(); err == nil {
			return DB, nil
		}
	}
	return InitDB()
}

// InitDB initializes connection to Microsoft SQL Server strictly from .env with fallback
func InitDB() (*sql.DB, error) {
	server := os.Getenv("MSSQL_SERVER")
	if server == "" {
		server = `LAPTOP-OV8LM74T\MSSQLSERVERANIL`
	}
	dbName := os.Getenv("MSSQL_DATABASE")
	if dbName == "" {
		dbName = "TradeDB"
	}

	user := os.Getenv("MSSQL_USER")
	password := os.Getenv("MSSQL_PASSWORD")

	var connStr string
	if user != "" && password != "" {
		// SQL Authentication
		u := &url.URL{
			Scheme: "sqlserver",
			User:   url.UserPassword(user, password),
			Host:   server,
		}
		q := u.Query()
		q.Add("database", dbName)
		q.Add("encrypt", "disable")
		u.RawQuery = q.Encode()
		connStr = u.String()
	} else {
		// Windows Authentication (Trusted Connection)
		connStr = fmt.Sprintf("server=%s;database=%s;trusted_connection=yes;encrypt=disable;", server, dbName)
	}

	db, err := sql.Open("sqlserver", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open mssql driver: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Printf("[MSSQL Error] Could not connect to MSSQL Server (%v)", err)
		return nil, err
	}

	log.Printf("[MSSQL] Connected to Microsoft SQL Server: %s / Database: %s", server, dbName)
	DB = db
	return db, nil
}

// GetClients returns list of clients strictly from client_details table
func GetClients() ([]models.Client, error) {
	db, err := EnsureDB()
	if err != nil {
		return nil, err
	}

	query := `SELECT client_id, client_code, first_name, last_name, email, phone_number, pan_number, account_type, risk_category, status, created_at FROM client_details ORDER BY client_code ASC`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query client_details: %w", err)
	}
	defer rows.Close()

	var clients []models.Client
	for rows.Next() {
		var c models.Client
		if err := rows.Scan(&c.ClientID, &c.ClientCode, &c.FirstName, &c.LastName, &c.Email, &c.PhoneNumber, &c.PANNumber, &c.AccountType, &c.RiskCategory, &c.Status, &c.CreatedAt); err != nil {
			return nil, err
		}
		clients = append(clients, c)
	}
	return clients, nil
}

// GetStocks returns list of instruments strictly from stock_master table
func GetStocks() ([]models.Stock, error) {
	db, err := EnsureDB()
	if err != nil {
		return nil, err
	}

	query := `SELECT symbol, company_name, isin_code, series, exchange, sector, face_value, lot_size, tick_size, currency, is_active FROM stock_master ORDER BY symbol ASC`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query stock_master: %w", err)
	}
	defer rows.Close()

	var stocks []models.Stock
	for rows.Next() {
		var s models.Stock
		if err := rows.Scan(&s.Symbol, &s.CompanyName, &s.ISINCode, &s.Series, &s.Exchange, &s.Sector, &s.FaceValue, &s.LotSize, &s.TickSize, &s.Currency, &s.IsActive); err != nil {
			return nil, err
		}
		stocks = append(stocks, s)
	}
	return stocks, nil
}

// GetMaxTradeUpdatedOn returns the latest updated_on timestamp from trade_details
func GetMaxTradeUpdatedOn() (time.Time, error) {
	db, err := EnsureDB()
	if err != nil {
		return time.Now(), err
	}
	var maxTime sql.NullTime
	query := `SELECT MAX(ISNULL(updated_on, trade_time)) FROM trade_details`
	err = db.QueryRow(query).Scan(&maxTime)
	if err != nil || !maxTime.Valid {
		return time.Now(), nil
	}
	return maxTime.Time, nil
}

// GetMaxClientUpdatedOn returns the latest updated_on timestamp from client_details
func GetMaxClientUpdatedOn() (time.Time, error) {
	db, err := EnsureDB()
	if err != nil {
		return time.Now(), err
	}
	var maxTime sql.NullTime
	query := `SELECT MAX(ISNULL(updated_on, created_at)) FROM client_details`
	err = db.QueryRow(query).Scan(&maxTime)
	if err != nil || !maxTime.Valid {
		return time.Now(), nil
	}
	return maxTime.Time, nil
}

// GetMaxStockUpdatedOn returns the latest updated_on timestamp from stock_master
func GetMaxStockUpdatedOn() (time.Time, error) {
	db, err := EnsureDB()
	if err != nil {
		return time.Now(), err
	}
	var maxTime sql.NullTime
	query := `SELECT MAX(ISNULL(updated_on, created_at)) FROM stock_master`
	err = db.QueryRow(query).Scan(&maxTime)
	if err != nil || !maxTime.Valid {
		return time.Now(), nil
	}
	return maxTime.Time, nil
}

// GetMaxTradeTime returns the latest updated_on across trade_details, client_details, and stock_master
func GetMaxTradeTime() (time.Time, error) {
	db, err := EnsureDB()
	if err != nil {
		return time.Time{}, err
	}
	var maxTime sql.NullTime
	query := `
		SELECT MAX(v) FROM (
			SELECT MAX(ISNULL(updated_on, trade_time)) AS v FROM trade_details
			UNION ALL
			SELECT MAX(ISNULL(updated_on, '1970-01-01')) AS v FROM client_details
			UNION ALL
			SELECT MAX(ISNULL(updated_on, '1970-01-01')) AS v FROM stock_master
		) AS sub
	`
	err = db.QueryRow(query).Scan(&maxTime)
	if err != nil || !maxTime.Valid {
		// Fallback query if updated_on does not exist on master tables
		_ = db.QueryRow("SELECT MAX(trade_time) FROM trade_details").Scan(&maxTime)
		if !maxTime.Valid {
			return time.Time{}, err
		}
	}
	return maxTime.Time, nil
}

// GetIncrementalTrades retrieves new/updated trades strictly from trade_details table based on updated_on watermark
func GetIncrementalTrades(watermark time.Time) ([]models.TradeReport, time.Time, error) {
	db, err := EnsureDB()
	if err != nil {
		return nil, watermark, err
	}

	query := `
		SELECT 
			t.trade_id, ISNULL(t.order_id, ''), t.client_id, ISNULL(c.client_code, ''), 
			ISNULL(c.first_name, ''), ISNULL(c.last_name, ''), ISNULL(c.account_type, ''),
			t.symbol, ISNULL(s.company_name, ''), ISNULL(s.sector, ''), 
			ISNULL(t.exchange, 'NSE'), ISNULL(t.segment, 'EQUITY'), t.trade_type,
			t.quantity, t.price, t.trade_value, ISNULL(t.brokerage, 0), ISNULL(t.taxes_charges, 0), ISNULL(t.net_amount, 0),
			ISNULL(t.trade_status, 'EXECUTED'), t.trade_time,
			ISNULL(t.updated_on, t.trade_time) AS max_updated
		FROM trade_details t
		LEFT JOIN client_details c ON t.client_id = c.client_id
		LEFT JOIN stock_master s ON t.symbol = s.symbol
		WHERE ISNULL(t.updated_on, t.trade_time) > @p1
		ORDER BY ISNULL(t.updated_on, t.trade_time) ASC
	`
	rows, err := db.Query(query, watermark)
	if err != nil {
		// Fallback query if updated_on does not exist yet on all tables
		fallbackQuery := `
			SELECT 
				t.trade_id, ISNULL(t.order_id, ''), t.client_id, ISNULL(c.client_code, ''), 
				ISNULL(c.first_name, ''), ISNULL(c.last_name, ''), ISNULL(c.account_type, ''),
				t.symbol, ISNULL(s.company_name, ''), ISNULL(s.sector, ''), 
				ISNULL(t.exchange, 'NSE'), ISNULL(t.segment, 'EQUITY'), t.trade_type,
				t.quantity, t.price, t.trade_value, ISNULL(t.brokerage, 0), ISNULL(t.taxes_charges, 0), ISNULL(t.net_amount, 0),
				ISNULL(t.trade_status, 'EXECUTED'), t.trade_time, t.trade_time
			FROM trade_details t
			LEFT JOIN client_details c ON t.client_id = c.client_id
			LEFT JOIN stock_master s ON t.symbol = s.symbol
			WHERE t.trade_time > @p1
			ORDER BY t.trade_time ASC
		`
		rows, err = db.Query(fallbackQuery, watermark)
		if err != nil {
			return nil, watermark, fmt.Errorf("failed to read incremental trades: %w", err)
		}
	}
	defer rows.Close()

	var reports []models.TradeReport
	newWatermark := watermark
	for rows.Next() {
		var r models.TradeReport
		var updatedOn time.Time
		err := rows.Scan(
			&r.TradeID, &r.OrderID, &r.ClientID, &r.ClientCode, &r.FirstName, &r.LastName, &r.AccountType,
			&r.Symbol, &r.CompanyName, &r.Sector, &r.Exchange, &r.Segment, &r.TradeType,
			&r.Quantity, &r.Price, &r.TradeValue, &r.Brokerage, &r.TaxesCharges, &r.NetAmount,
			&r.TradeStatus, &r.TradeTime, &updatedOn,
		)
		if err != nil {
			continue
		}
		r.UpdatedOn = updatedOn
		if updatedOn.After(newWatermark) {
			newWatermark = updatedOn
		}
		reports = append(reports, r)
	}
	return reports, newWatermark, nil
}

// GetUpdatedClients returns clients modified since client watermark
func GetUpdatedClients(watermark time.Time) ([]models.Client, time.Time, error) {
	db, err := EnsureDB()
	if err != nil {
		return nil, watermark, err
	}

	query := `
		SELECT client_id, client_code, first_name, last_name, email, phone_number, pan_number, 
		       account_type, risk_category, status, created_at, ISNULL(updated_on, created_at) AS updated_on 
		FROM client_details 
		WHERE ISNULL(updated_on, created_at) > @p1 
		ORDER BY ISNULL(updated_on, created_at) ASC
	`
	rows, err := db.Query(query, watermark)
	if err != nil {
		return nil, watermark, err
	}
	defer rows.Close()

	var clients []models.Client
	newWatermark := watermark
	for rows.Next() {
		var c models.Client
		if err := rows.Scan(&c.ClientID, &c.ClientCode, &c.FirstName, &c.LastName, &c.Email, &c.PhoneNumber, &c.PANNumber, &c.AccountType, &c.RiskCategory, &c.Status, &c.CreatedAt, &c.UpdatedOn); err != nil {
			continue
		}
		if c.UpdatedOn.After(newWatermark) {
			newWatermark = c.UpdatedOn
		}
		clients = append(clients, c)
	}
	return clients, newWatermark, nil
}

// GetUpdatedStocks returns stocks modified since stock watermark
func GetUpdatedStocks(watermark time.Time) ([]models.Stock, time.Time, error) {
	db, err := EnsureDB()
	if err != nil {
		return nil, watermark, err
	}

	query := `
		SELECT symbol, company_name, isin_code, series, exchange, sector, face_value, 
		       lot_size, tick_size, currency, is_active, ISNULL(updated_on, created_at) AS updated_on 
		FROM stock_master 
		WHERE ISNULL(updated_on, created_at) > @p1 
		ORDER BY ISNULL(updated_on, created_at) ASC
	`
	rows, err := db.Query(query, watermark)
	if err != nil {
		return nil, watermark, err
	}
	defer rows.Close()

	var stocks []models.Stock
	newWatermark := watermark
	for rows.Next() {
		var s models.Stock
		if err := rows.Scan(&s.Symbol, &s.CompanyName, &s.ISINCode, &s.Series, &s.Exchange, &s.Sector, &s.FaceValue, &s.LotSize, &s.TickSize, &s.Currency, &s.IsActive, &s.UpdatedOn); err != nil {
			continue
		}
		if s.UpdatedOn.After(newWatermark) {
			newWatermark = s.UpdatedOn
		}
		stocks = append(stocks, s)
	}
	return stocks, newWatermark, nil
}

// GetTradeReportByID retrieves a fully enriched trade report by trade_id
func GetTradeReportByID(tradeID string) (*models.TradeReport, error) {
	db, err := EnsureDB()
	if err != nil {
		return nil, err
	}

	query := `
		SELECT 
			t.trade_id, ISNULL(t.order_id, ''), t.client_id, ISNULL(c.client_code, ''), 
			ISNULL(c.first_name, ''), ISNULL(c.last_name, ''), ISNULL(c.account_type, ''),
			t.symbol, ISNULL(s.company_name, ''), ISNULL(s.sector, ''), 
			ISNULL(t.exchange, 'NSE'), ISNULL(t.segment, 'EQUITY'), t.trade_type,
			t.quantity, t.price, t.trade_value, ISNULL(t.brokerage, 0), ISNULL(t.taxes_charges, 0), ISNULL(t.net_amount, 0),
			ISNULL(t.trade_status, 'EXECUTED'), t.trade_time, ISNULL(t.updated_on, t.trade_time)
		FROM trade_details t
		LEFT JOIN client_details c ON t.client_id = c.client_id
		LEFT JOIN stock_master s ON t.symbol = s.symbol
		WHERE t.trade_id = @p1
	`
	var r models.TradeReport
	var updatedOn time.Time
	err = db.QueryRow(query, tradeID).Scan(
		&r.TradeID, &r.OrderID, &r.ClientID, &r.ClientCode, &r.FirstName, &r.LastName, &r.AccountType,
		&r.Symbol, &r.CompanyName, &r.Sector, &r.Exchange, &r.Segment, &r.TradeType,
		&r.Quantity, &r.Price, &r.TradeValue, &r.Brokerage, &r.TaxesCharges, &r.NetAmount,
		&r.TradeStatus, &r.TradeTime, &updatedOn,
	)
	if err != nil {
		return nil, err
	}
	r.UpdatedOn = updatedOn
	return &r, nil
}

// GetAllTradesForSync retrieves all trades joined with client and stock metadata for streaming
func GetAllTradesForSync() ([]models.TradeReport, error) {
	db, err := EnsureDB()
	if err != nil {
		return nil, err
	}

	query := `
		SELECT 
			t.trade_id, t.order_id, t.client_id, c.client_code, c.first_name, c.last_name, c.account_type,
			t.symbol, s.company_name, s.sector, t.exchange, t.segment, t.trade_type,
			t.quantity, t.price, t.trade_value, t.brokerage, t.taxes_charges, t.net_amount,
			t.trade_status, t.trade_time
		FROM trade_details t
		INNER JOIN client_details c ON t.client_id = c.client_id
		INNER JOIN stock_master s ON t.symbol = s.symbol
		ORDER BY t.trade_time ASC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to read trades: %w", err)
	}
	defer rows.Close()

	var reports []models.TradeReport
	for rows.Next() {
		var r models.TradeReport
		err := rows.Scan(
			&r.TradeID, &r.OrderID, &r.ClientID, &r.ClientCode, &r.FirstName, &r.LastName, &r.AccountType,
			&r.Symbol, &r.CompanyName, &r.Sector, &r.Exchange, &r.Segment, &r.TradeType,
			&r.Quantity, &r.Price, &r.TradeValue, &r.Brokerage, &r.TaxesCharges, &r.NetAmount,
			&r.TradeStatus, &r.TradeTime,
		)
		if err != nil {
			continue
		}
		reports = append(reports, r)
	}
	return reports, nil
}

// InsertTrade persists an executed trade into trade_details (Clean business columns only)
func InsertTrade(t *models.Trade) error {
	db, err := EnsureDB()
	if err != nil {
		return err
	}

	query := `
		INSERT INTO trade_details (
			trade_id, order_id, client_id, symbol, exchange, segment, trade_type,
			quantity, price, trade_value, brokerage, taxes_charges, net_amount,
			trade_status, trade_time
		) VALUES (
			@p1, @p2, @p3, @p4, @p5, @p6, @p7,
			@p8, @p9, @p10, @p11, @p12, @p13,
			@p14, @p15
		)
	`
	_, err = db.Exec(
		query,
		t.TradeID, t.OrderID, t.ClientID, t.Symbol, t.Exchange, t.Segment, t.TradeType,
		t.Quantity, t.Price, t.TradeValue, t.Brokerage, t.TaxesCharges, t.NetAmount,
		t.TradeStatus, t.TradeTime,
	)
	return err
}
