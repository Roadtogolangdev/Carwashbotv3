package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv" // Добавляем импорт
)

type Config struct {
	BotToken    string
	AdminID     int64
	StartTime   int
	EndTime     int
	ChannelName string `json:"channel_name"` // Для @username
	ChannelID   int64  `json:"channel_id"`   // Для числового ID // @channel_username или ID канала
	AdminIDs    []int64
}

// Инициализируем при первом вызове
func init() {
	// Загружаем .env файл
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not loaded: %v", err)
	}
}

func Load() *Config {
	return &Config{
		BotToken:  getEnv("TELEGRAM_BOT_TOKEN", ""), // Убрали хардкод
		AdminID:   getEnvAsInt64("ADMIN_CHAT_ID", 0),
		StartTime: getEnvAsInt("START_TIME", 8),
		EndTime:   getEnvAsInt("END_TIME", 20),
		ChannelID: getEnvAsInt64("CHANNEL_ID", 0), // 0 - значение по умолчанию
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value, exists := os.LookupEnv(key); exists {
		if num, err := strconv.ParseInt(value, 10, 64); err == nil {
			return num
		}
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if num, err := strconv.Atoi(value); err == nil {
			return num
		}
	}
	return defaultValue
}
