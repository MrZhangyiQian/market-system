package adapters

import "market-system/common/models"

// ExchangeAdapter 交易所适配器接口
type ExchangeAdapter interface {
	// Connect 建立连接
	Connect() error

	// Subscribe 订阅数据
	Subscribe(symbols []string, channels []string) error

	// OnMessage 设置消息处理器
	OnMessage(handler MessageHandler)

	// Close 关闭连接
	Close() error

	// IsConnected 检查连接状态
	IsConnected() bool

	// GetName 获取交易所名称
	GetName() string
}

// MessageHandler 消息处理器
type MessageHandler func(data *models.MarketData)

// AdapterFactory 适配器工厂
type AdapterFactory struct {
	adapters map[string]func(wsURL string) ExchangeAdapter
}

// NewAdapterFactory 创建适配器工厂
func NewAdapterFactory() *AdapterFactory {
	factory := &AdapterFactory{
		adapters: make(map[string]func(wsURL string) ExchangeAdapter),
	}

	// 注册适配器
	factory.Register("binance", func(wsURL string) ExchangeAdapter {
		return NewBinanceAdapter(wsURL)
	})

	factory.Register("okx", func(wsURL string) ExchangeAdapter {
		return NewOKXAdapter(wsURL)
	})

	// 注册内部适配器（wsURL 用于传递端口，格式：:9001）
	factory.Register("internal", func(wsURL string) ExchangeAdapter {
		port := 9001 // 默认端口
		// 可以从 wsURL 解析端口，这里简化处理
		return NewInternalAdapter(port)
	})

	return factory
}

// Register 注册适配器
func (f *AdapterFactory) Register(name string, creator func(wsURL string) ExchangeAdapter) {
	f.adapters[name] = creator
}

// Create 创建适配器
func (f *AdapterFactory) Create(name, wsURL string) ExchangeAdapter {
	creator, ok := f.adapters[name]
	if !ok {
		return nil
	}
	return creator(wsURL)
}
