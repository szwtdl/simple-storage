package utils

import (
	"crypto/rand"
	"github.com/google/uuid"
	"math/big"
	"time"
)

// 生成随机字符串

func RandomString(n int) string {
	var letters = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	result := make([]byte, n)
	for i := range result {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			panic("secure random source unavailable")
		}
		result[i] = letters[index.Int64()]
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
