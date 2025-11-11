package utils

import (
	"encoding/json"
)

// ToJSON 将对象转换为 JSON 字符串
func ToJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON 从 JSON 字符串解析对象
func FromJSON(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}

// ToJSONBytes 将对象转换为 JSON 字节数组
func ToJSONBytes(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// FromJSONBytes 从 JSON 字节数组解析对象
func FromJSONBytes(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// MustToJSON 将对象转换为 JSON 字符串（panic on error）
func MustToJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}
