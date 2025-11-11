package utils

import "time"

// GetCurrentTimestamp 获取当前时间戳（毫秒）
func GetCurrentTimestamp() int64 {
	return time.Now().UnixMilli()
}

// TimestampToTime 时间戳转时间对象
func TimestampToTime(timestamp int64) time.Time {
	return time.UnixMilli(timestamp)
}

// GetKlineOpenTime 获取K线开盘时间
func GetKlineOpenTime(timestamp int64, interval string) int64 {
	t := time.UnixMilli(timestamp)

	switch interval {
	case "1m":
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, t.Location()).UnixMilli()
	case "5m":
		minute := t.Minute() / 5 * 5
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), minute, 0, 0, t.Location()).UnixMilli()
	case "15m":
		minute := t.Minute() / 15 * 15
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), minute, 0, 0, t.Location()).UnixMilli()
	case "30m":
		minute := t.Minute() / 30 * 30
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), minute, 0, 0, t.Location()).UnixMilli()
	case "1h":
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location()).UnixMilli()
	case "4h":
		hour := t.Hour() / 4 * 4
		return time.Date(t.Year(), t.Month(), t.Day(), hour, 0, 0, 0, t.Location()).UnixMilli()
	case "1d":
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).UnixMilli()
	default:
		return timestamp
	}
}

// GetKlineCloseTime 获取K线收盘时间
func GetKlineCloseTime(openTime int64, interval string) int64 {
	t := time.UnixMilli(openTime)

	switch interval {
	case "1m":
		return t.Add(1 * time.Minute).UnixMilli() - 1
	case "5m":
		return t.Add(5 * time.Minute).UnixMilli() - 1
	case "15m":
		return t.Add(15 * time.Minute).UnixMilli() - 1
	case "30m":
		return t.Add(30 * time.Minute).UnixMilli() - 1
	case "1h":
		return t.Add(1 * time.Hour).UnixMilli() - 1
	case "4h":
		return t.Add(4 * time.Hour).UnixMilli() - 1
	case "1d":
		return t.Add(24 * time.Hour).UnixMilli() - 1
	default:
		return openTime
	}
}

// IsNewKline 判断是否需要生成新K线
func IsNewKline(currentOpenTime, tradeTime int64, interval string) bool {
	return GetKlineOpenTime(tradeTime, interval) != currentOpenTime
}
