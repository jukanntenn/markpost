package main

import (
	"crypto/rand"
	"encoding/base64"
)

// GenerateShortKey 生成短密钥（按照 Python 算法改写）
func GenerateShortKey(byteLength int) (string, error) {
	if byteLength <= 0 {
		byteLength = 8
	}
	
	// 生成随机字节
	randomBytes := make([]byte, byteLength)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	
	// Base64编码并替换非URL安全字符，去除填充字符
	encoded := base64.URLEncoding.EncodeToString(randomBytes)
	// 移除末尾的 '=' 填充字符
	for len(encoded) > 0 && encoded[len(encoded)-1] == '=' {
		encoded = encoded[:len(encoded)-1]
	}
	
	return encoded, nil
} 