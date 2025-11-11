package utils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math"
)

// GenerateID 生成唯一ID
func GenerateID(parts ...string) string {
	str := ""
	for _, part := range parts {
		str += part
	}
	hash := md5.Sum([]byte(str))
	return hex.EncodeToString(hash[:])
}

// RoundFloat 浮点数四舍五入
func RoundFloat(val float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

// MaxFloat 返回最大浮点数
func MaxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// MinFloat 返回最小浮点数
func MinFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// FormatSymbol 格式化交易对
func FormatSymbol(symbol string) string {
	return symbol
}

// CalculateSpread 计算买卖价差
func CalculateSpread(bidPrice, askPrice float64) float64 {
	if bidPrice <= 0 || askPrice <= 0 {
		return 0
	}
	return RoundFloat((askPrice-bidPrice)/bidPrice*100, 4)
}

// CalculateChange24h 计算24小时涨跌幅
func CalculateChange24h(current, open24h float64) float64 {
	if open24h <= 0 {
		return 0
	}
	return RoundFloat((current-open24h)/open24h*100, 2)
}

// RetryBackoff 计算重试退避时间
func RetryBackoff(retryCount int, initialDelay, maxDelay int64) int64 {
	delay := initialDelay * int64(math.Pow(2, float64(retryCount)))
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}

// ValidateSymbol 验证交易对格式
func ValidateSymbol(symbol string) error {
	if len(symbol) < 3 {
		return fmt.Errorf("invalid symbol: %s", symbol)
	}
	return nil
}

// ValidateInterval 验证K线周期
func ValidateInterval(interval string) bool {
	validIntervals := map[string]bool{
		"1m": true, "5m": true, "15m": true, "30m": true,
		"1h": true, "4h": true, "1d": true, "1w": true,
	}
	return validIntervals[interval]
}
