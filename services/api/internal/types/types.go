package types

type TickerRequest struct {
	Symbol string `path:"symbol"`
}

type TickerResponse struct {
	Symbol    string  `json:"symbol"`
	LastPrice float64 `json:"last_price"`
	BidPrice  float64 `json:"bid_price"`
	AskPrice  float64 `json:"ask_price"`
	High24h   float64 `json:"high_24h"`
	Low24h    float64 `json:"low_24h"`
	Volume24h float64 `json:"volume_24h"`
	Timestamp int64   `json:"timestamp"`
}

type KlineRequest struct {
	Symbol   string `form:"symbol"`
	Interval string `form:"interval,default=1m"`
	Limit    int64  `form:"limit,default=100"`
}

type Kline struct {
	OpenTime  int64   `json:"open_time"`
	CloseTime int64   `json:"close_time"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    float64 `json:"volume"`
	QuoteVol  float64 `json:"quote_vol"`
	TradeNum  int64   `json:"trade_num"`
}

type KlineResponse struct {
	Symbol   string  `json:"symbol"`
	Interval string  `json:"interval"`
	Data     []Kline `json:"data"`
}

type DepthRequest struct {
	Symbol string `path:"symbol"`
	Limit  int64  `form:"limit,default=20"`
}

type PriceLevel struct {
	Price  float64 `json:"price"`
	Amount float64 `json:"amount"`
}

type DepthResponse struct {
	Symbol    string       `json:"symbol"`
	Bids      []PriceLevel `json:"bids"`
	Asks      []PriceLevel `json:"asks"`
	Timestamp int64        `json:"timestamp"`
}

type BaseResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}
