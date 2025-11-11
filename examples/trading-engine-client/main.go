package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"market-system/common/models"
	"market-system/common/utils"
	"net/http"
	"time"
)

// TradingEngineClient 交易引擎客户端
type TradingEngineClient struct {
	baseURL string
	client  *http.Client
}

// NewTradingEngineClient 创建客户端
func NewTradingEngineClient(baseURL string) *TradingEngineClient {
	return &TradingEngineClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// PushTrade 推送交易数据
func (c *TradingEngineClient) PushTrade(trade *models.InternalTradeMessage) error {
	url := fmt.Sprintf("%s/api/market/trade", c.baseURL)

	data, err := json.Marshal(trade)
	if err != nil {
		return err
	}

	resp, err := c.client.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	log.Printf("Trade pushed: %s @ %.2f x %.4f\n", trade.Symbol, trade.Price, trade.Amount)
	return nil
}

// PushDepth 推送深度数据
func (c *TradingEngineClient) PushDepth(depth *models.InternalDepthMessage) error {
	url := fmt.Sprintf("%s/api/market/depth", c.baseURL)

	data, err := json.Marshal(depth)
	if err != nil {
		return err
	}

	resp, err := c.client.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	log.Printf("Depth pushed: %s, bids: %d, asks: %d\n",
		depth.Symbol, len(depth.Bids), len(depth.Asks))
	return nil
}

// PushTicker 推送 Ticker 数据
func (c *TradingEngineClient) PushTicker(ticker *models.Ticker) error {
	url := fmt.Sprintf("%s/api/market/ticker", c.baseURL)

	data, err := json.Marshal(ticker)
	if err != nil {
		return err
	}

	resp, err := c.client.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	log.Printf("Ticker pushed: %s @ %.2f\n", ticker.Symbol, ticker.LastPrice)
	return nil
}

func main() {
	log.Println("========================================")
	log.Println("Trading Engine Client Example")
	log.Println("========================================")

	// 创建客户端
	client := NewTradingEngineClient("http://localhost:9001")

	// 模拟推送交易数据
	log.Println("\n[1] Pushing trade data...")
	trade := &models.InternalTradeMessage{
		Symbol:        "BTCUSDT",
		TradeID:       12345,
		Price:         45000.00,
		Amount:        0.5,
		Side:          "buy",
		BuyerID:       1001,
		SellerID:      1002,
		BuyOrderID:    2001,
		SellerOrderID: 2002,
		Timestamp:     utils.GetCurrentTimestamp(),
		IsMaker:       true,
	}

	if err := client.PushTrade(trade); err != nil {
		log.Printf("Failed to push trade: %v\n", err)
	}

	time.Sleep(1 * time.Second)

	// 模拟推送深度数据
	log.Println("\n[2] Pushing depth data...")
	depth := &models.InternalDepthMessage{
		Symbol: "BTCUSDT",
		Bids: []models.PriceLevel{
			{Price: 44999.00, Amount: 1.5},
			{Price: 44998.00, Amount: 2.3},
			{Price: 44997.00, Amount: 1.8},
		},
		Asks: []models.PriceLevel{
			{Price: 45001.00, Amount: 1.2},
			{Price: 45002.00, Amount: 2.1},
			{Price: 45003.00, Amount: 1.6},
		},
		Timestamp: utils.GetCurrentTimestamp(),
		SeqNum:    1,
	}

	if err := client.PushDepth(depth); err != nil {
		log.Printf("Failed to push depth: %v\n", err)
	}

	time.Sleep(1 * time.Second)

	// 模拟推送 Ticker 数据
	log.Println("\n[3] Pushing ticker data...")
	ticker := &models.Ticker{
		Symbol:    "BTCUSDT",
		LastPrice: 45000.00,
		BidPrice:  44999.00,
		AskPrice:  45001.00,
		High24h:   46000.00,
		Low24h:    44000.00,
		Volume24h: 100.5,
		Timestamp: utils.GetCurrentTimestamp(),
	}

	if err := client.PushTicker(ticker); err != nil {
		log.Printf("Failed to push ticker: %v\n", err)
	}

	log.Println("\n========================================")
	log.Println("All data pushed successfully!")
	log.Println("========================================")

	// 持续推送数据（模拟）
	log.Println("\n[4] Starting continuous push (Ctrl+C to stop)...")
	ticker.LastPrice = 45000.00

	for i := 0; i < 10; i++ {
		// 随机价格变动
		ticker.LastPrice += float64(i%3-1) * 10.0
		ticker.Timestamp = utils.GetCurrentTimestamp()

		if err := client.PushTicker(ticker); err != nil {
			log.Printf("Failed to push ticker: %v\n", err)
		}

		time.Sleep(2 * time.Second)
	}

	log.Println("Demo completed!")
}
