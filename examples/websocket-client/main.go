package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

var (
	addr   = flag.String("addr", "localhost:8888", "WebSocket server address")
	symbol = flag.String("symbol", "BTCUSDT", "Trading symbol")
)

type Message struct {
	Type    string      `json:"type,omitempty"`
	Channel string      `json:"channel,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	url := "ws://" + *addr + "/ws"
	log.Printf("Connecting to %s...\n", url)

	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	// 读取消息
	go func() {
		defer close(done)
		for {
			var msg Message
			err := c.ReadJSON(&msg)
			if err != nil {
				log.Println("Read error:", err)
				return
			}

			switch msg.Type {
			case "welcome":
				log.Printf("✓ Welcome message: %+v\n", msg)
				subscribeAll(c)
			case "subscribed":
				log.Printf("✓ Subscribed: %+v\n", msg.Data)
			case "unsubscribed":
				log.Printf("✓ Unsubscribed: %+v\n", msg.Data)
			case "pong":
				log.Println("✓ Pong received")
			case "error":
				log.Printf("✗ Error: %s\n", msg.Error)
			default:
				// 市场数据
				displayMarketData(msg)
			}
		}
	}()

	// 定时发送ping
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			pingMsg := map[string]string{"action": "ping"}
			err := c.WriteJSON(pingMsg)
			if err != nil {
				log.Println("Ping error:", err)
				return
			}
			log.Println("→ Ping sent")
		case <-interrupt:
			log.Println("Interrupt signal received, closing connection...")
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("Write close error:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

func subscribeAll(c *websocket.Conn) {
	channels := []string{"ticker", "depth", "trade", "kline"}

	for _, channel := range channels {
		msg := map[string]string{
			"action":  "subscribe",
			"channel": channel,
			"symbol":  *symbol,
		}
		err := c.WriteJSON(msg)
		if err != nil {
			log.Printf("Failed to subscribe to %s: %v\n", channel, err)
		} else {
			log.Printf("→ Subscribed to %s:%s\n", channel, *symbol)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func displayMarketData(msg Message) {
	// 将data转换为map以便访问
	dataMap, ok := msg.Data.(map[string]interface{})
	if !ok {
		log.Printf("[%s] %+v\n", msg.Channel, msg.Data)
		return
	}

	switch {
	case contains(msg.Channel, "ticker"):
		displayTicker(msg.Channel, dataMap)
	case contains(msg.Channel, "depth"):
		displayDepth(msg.Channel, dataMap)
	case contains(msg.Channel, "trade"):
		displayTrade(msg.Channel, dataMap)
	case contains(msg.Channel, "kline"):
		displayKline(msg.Channel, dataMap)
	default:
		log.Printf("[%s] %+v\n", msg.Channel, msg.Data)
	}
}

func displayTicker(channel string, data map[string]interface{}) {
	log.Printf("📊 [%s] Price: %.2f | Bid: %.2f | Ask: %.2f | 24h: %.2f~%.2f | Vol: %.2f",
		channel,
		getFloat(data, "last_price"),
		getFloat(data, "bid_price"),
		getFloat(data, "ask_price"),
		getFloat(data, "low_24h"),
		getFloat(data, "high_24h"),
		getFloat(data, "volume_24h"),
	)
}

func displayDepth(channel string, data map[string]interface{}) {
	bids, _ := data["bids"].([]interface{})
	asks, _ := data["asks"].([]interface{})

	log.Printf("📈 [%s] Depth: %d bids, %d asks",
		channel,
		len(bids),
		len(asks),
	)

	// 显示最优买卖价
	if len(bids) > 0 {
		if bid, ok := bids[0].(map[string]interface{}); ok {
			log.Printf("   Best Bid: %.2f x %.4f", getFloat(bid, "price"), getFloat(bid, "amount"))
		}
	}
	if len(asks) > 0 {
		if ask, ok := asks[0].(map[string]interface{}); ok {
			log.Printf("   Best Ask: %.2f x %.4f", getFloat(ask, "price"), getFloat(ask, "amount"))
		}
	}
}

func displayTrade(channel string, data map[string]interface{}) {
	side := getString(data, "side")
	sideIcon := "🟢"
	if side == "sell" {
		sideIcon = "🔴"
	}

	log.Printf("%s [%s] %s %.2f @ %.4f | ID: %s",
		sideIcon,
		channel,
		side,
		getFloat(data, "price"),
		getFloat(data, "amount"),
		getString(data, "trade_id"),
	)
}

func displayKline(channel string, data map[string]interface{}) {
	log.Printf("📉 [%s] O: %.2f | H: %.2f | L: %.2f | C: %.2f | V: %.2f",
		channel,
		getFloat(data, "open"),
		getFloat(data, "high"),
		getFloat(data, "low"),
		getFloat(data, "close"),
		getFloat(data, "volume"),
	)
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case int:
			return float64(val)
		}
	}
	return 0
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
