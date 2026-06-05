package polymarket

// Market represents a single prediction market from the Polymarket CLOB API.
type Market struct {
	ConditionID  string   `json:"condition_id"`
	QuestionID   string   `json:"question_id"`
	Question     string   `json:"question"`
	Description  string   `json:"description"`
	MarketSlug   string   `json:"market_slug"`
	EndDateISO   string   `json:"end_date_iso"`
	Category     string   `json:"category"`
	Tags         []string `json:"tags"`
	Active       bool     `json:"active"`
	Closed       bool     `json:"closed"`
	Tokens       []Token  `json:"tokens"`
	Volume       string   `json:"volume"`
	Volume24hr   string   `json:"volume_24hr"`
	Liquidity    string   `json:"liquidity"`
	StartDateISO string   `json:"start_date_iso"`
}

// Token represents one outcome token within a market (e.g. YES or NO).
type Token struct {
	TokenID string  `json:"token_id"`
	Outcome string  `json:"outcome"`
	Price   float64 `json:"price"`
	Winner  bool    `json:"winner"`
}

// MarketList is the paginated response from GET /markets.
type MarketList struct {
	Data       []Market `json:"data"`
	NextCursor string   `json:"next_cursor"`
	Limit      int      `json:"limit"`
	Count      int      `json:"count"`
}

// PricePoint is a single (timestamp, price) observation from the price-history endpoint.
type PricePoint struct {
	T int64   `json:"t"`
	P float64 `json:"p"`
}

// PriceHistoryResponse wraps the price history list.
type PriceHistoryResponse struct {
	History []PricePoint `json:"history"`
}

// OrderbookEntry is a single bid or ask level.
type OrderbookEntry struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

// Orderbook holds bids and asks for a given outcome token.
type Orderbook struct {
	Market  string           `json:"market"`
	AssetID string           `json:"asset_id"`
	Bids    []OrderbookEntry `json:"bids"`
	Asks    []OrderbookEntry `json:"asks"`
}
