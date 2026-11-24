package models

import "time"

// Client represents investor/client details from client_details table
type Client struct {
	ClientID     string    `json:"client_id"`
	ClientCode   string    `json:"client_code"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Email        string    `json:"email"`
	PhoneNumber  string    `json:"phone_number"`
	PANNumber    string    `json:"pan_number"`
	AccountType  string    `json:"account_type"`
	RiskCategory string    `json:"risk_category"`
	Status       string    `json:"status"`
	Address      string    `json:"address,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedOn    time.Time `json:"updated_on,omitempty"`
}

// Stock represents instrument master info from stock_master table
type Stock struct {
	Symbol      string    `json:"symbol"`
	CompanyName string    `json:"company_name"`
	ISINCode    string    `json:"isin_code"`
	Series      string    `json:"series"`
	Exchange    string    `json:"exchange"`
	Sector      string    `json:"sector"`
	FaceValue   float64   `json:"face_value"`
	LotSize     int       `json:"lot_size"`
	TickSize    float64   `json:"tick_size"`
	Currency    string    `json:"currency"`
	IsActive    bool      `json:"is_active"`
	UpdatedOn   time.Time `json:"updated_on,omitempty"`
}

// Trade represents executed transaction from trade_details table
type Trade struct {
	TradeID         string    `json:"trade_id"`
	OrderID         string    `json:"order_id"`
	ClientID        string    `json:"client_id"`
	Symbol          string    `json:"symbol"`
	Exchange        string    `json:"exchange"`
	Segment         string    `json:"segment"`
	TradeType       string    `json:"trade_type"` // BUY or SELL
	Quantity        int       `json:"quantity"`
	Price           float64   `json:"price"`
	TradeValue      float64   `json:"trade_value"`
	Brokerage       float64   `json:"brokerage"`
	TaxesCharges    float64   `json:"taxes_charges"`
	NetAmount       float64   `json:"net_amount"`
	TradeStatus     string    `json:"trade_status"`
	TradeTime       time.Time `json:"trade_time"`
	SettlementDate  string    `json:"settlement_date,omitempty"`
	UpdatedOn       time.Time `json:"updated_on,omitempty"`
}

// TradeReport is the enriched trade document stored in Elasticsearch and returned in reports
type TradeReport struct {
	TradeID         string    `json:"trade_id"`
	OrderID         string    `json:"order_id"`
	ClientID        string    `json:"client_id"`
	ClientCode      string    `json:"client_code"`
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	AccountType     string    `json:"account_type"`
	Symbol          string    `json:"symbol"`
	CompanyName     string    `json:"company_name"`
	Sector          string    `json:"sector"`
	Exchange        string    `json:"exchange"`
	Segment         string    `json:"segment"`
	TradeType       string    `json:"trade_type"`
	Quantity        int       `json:"quantity"`
	Price           float64   `json:"price"`
	TradeValue      float64   `json:"trade_value"`
	Brokerage       float64   `json:"brokerage"`
	TaxesCharges    float64   `json:"taxes_charges"`
	NetAmount       float64   `json:"net_amount"`
	TradeStatus     string    `json:"trade_status"`
	TradeTime       time.Time `json:"trade_time"`
	UpdatedOn       time.Time `json:"updated_on,omitempty"`
	IndexedAt       time.Time `json:"indexed_at"`
}

// TradeFilter defines filter criteria for querying trade reports
type TradeFilter struct {
	ClientID  string
	Symbol    string
	TradeType string
	Search    string
	FromDate  string
	ToDate    string
	Page      int
	Limit     int
	SortBy    string
	Order     string
}

// LoginRequest received on /api/login
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse sent on successful /api/login
type LoginResponse struct {
	Token string      `json:"token"`
	User  UserProfile `json:"user"`
}

type UserProfile struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	Name     string `json:"name"`
}
