package websocket

import (
	"sync"
)

// SubscriptionManager 管理客户端订阅
type SubscriptionManager struct {
	// 频道 -> 订阅的客户端集合
	channelSubscribers map[string]map[*Client]bool

	// 客户端 -> 订阅的频道列表
	clientSubscriptions map[*Client]map[string]bool

	// 读写锁
	mu sync.RWMutex
}

// NewSubscriptionManager 创建新的订阅管理器
func NewSubscriptionManager() *SubscriptionManager {
	return &SubscriptionManager{
		channelSubscribers:  make(map[string]map[*Client]bool),
		clientSubscriptions: make(map[*Client]map[string]bool),
	}
}

// Subscribe 订阅频道
func (sm *SubscriptionManager) Subscribe(client *Client, channel string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 添加到频道订阅者列表
	if _, exists := sm.channelSubscribers[channel]; !exists {
		sm.channelSubscribers[channel] = make(map[*Client]bool)
	}
	sm.channelSubscribers[channel][client] = true

	// 添加到客户端订阅列表
	if _, exists := sm.clientSubscriptions[client]; !exists {
		sm.clientSubscriptions[client] = make(map[string]bool)
	}
	sm.clientSubscriptions[client][channel] = true
}

// Unsubscribe 取消订阅频道
func (sm *SubscriptionManager) Unsubscribe(client *Client, channel string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 从频道订阅者列表中移除
	if subscribers, exists := sm.channelSubscribers[channel]; exists {
		delete(subscribers, client)
		// 如果频道没有订阅者了，删除频道
		if len(subscribers) == 0 {
			delete(sm.channelSubscribers, channel)
		}
	}

	// 从客户端订阅列表中移除
	if channels, exists := sm.clientSubscriptions[client]; exists {
		delete(channels, channel)
		// 如果客户端没有订阅任何频道，删除客户端记录
		if len(channels) == 0 {
			delete(sm.clientSubscriptions, client)
		}
	}
}

// UnsubscribeAll 取消客户端的所有订阅
func (sm *SubscriptionManager) UnsubscribeAll(client *Client) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 获取客户端的所有订阅
	channels, exists := sm.clientSubscriptions[client]
	if !exists {
		return
	}

	// 从每个频道的订阅者列表中移除该客户端
	for channel := range channels {
		if subscribers, exists := sm.channelSubscribers[channel]; exists {
			delete(subscribers, client)
			// 如果频道没有订阅者了，删除频道
			if len(subscribers) == 0 {
				delete(sm.channelSubscribers, channel)
			}
		}
	}

	// 删除客户端的订阅记录
	delete(sm.clientSubscriptions, client)
}

// GetSubscribers 获取订阅了指定频道的所有客户端
func (sm *SubscriptionManager) GetSubscribers(channel string) map[*Client]bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	subscribers, exists := sm.channelSubscribers[channel]
	if !exists {
		return make(map[*Client]bool)
	}

	// 返回副本，避免并发问题
	result := make(map[*Client]bool, len(subscribers))
	for client := range subscribers {
		result[client] = true
	}
	return result
}

// GetClientSubscriptions 获取客户端订阅的所有频道
func (sm *SubscriptionManager) GetClientSubscriptions(client *Client) []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	channels, exists := sm.clientSubscriptions[client]
	if !exists {
		return []string{}
	}

	result := make([]string, 0, len(channels))
	for channel := range channels {
		result = append(result, channel)
	}
	return result
}

// GetChannelCount 获取频道总数
func (sm *SubscriptionManager) GetChannelCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.channelSubscribers)
}

// GetSubscriberCount 获取指定频道的订阅者数量
func (sm *SubscriptionManager) GetSubscriberCount(channel string) int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	subscribers, exists := sm.channelSubscribers[channel]
	if !exists {
		return 0
	}
	return len(subscribers)
}

// IsSubscribed 检查客户端是否订阅了指定频道
func (sm *SubscriptionManager) IsSubscribed(client *Client, channel string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	channels, exists := sm.clientSubscriptions[client]
	if !exists {
		return false
	}
	return channels[channel]
}
