package utils

import (
	"github.com/google/uuid"
	"math/rand"
	"time"
)

// 生成随机字符串

func RandomString(n int) string {
	var letters = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	result := make([]byte, n)
	for i := range result {
		result[i] = letters[rand.Intn(len(letters))]
	}
	return string(result)
}

// 生成年月日

func GenerateDatePath() string {
	now := time.Now()
	return now.Format("20060102")
}

// 生成uuid

func GenerateUUID() string {
	return uuid.New().String()
}
