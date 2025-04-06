package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv" // Добавляем импорт
)

type Config struct {
	BotToken  string
	AdminID   int64   // Оставляем для обратной совместимости
	AdminIDs  []int64 // Добавляем поддержку нескольких админов
	StartTime int
	EndTime   int
	ChannelID int64
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
		BotToken:  getEnv("TELEGRAM_BOT_TOKEN", ""),
		AdminID:   getEnvAsInt64("ADMIN_CHAT_ID", 0),
		AdminIDs:  parseAdminIDs(getEnv("ADMIN_IDS", "")),
		StartTime: getEnvAsInt("START_TIME", 8),
		EndTime:   getEnvAsInt("END_TIME", 20),
		ChannelID: getEnvAsInt64("CHANNEL_ID", 0),
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
func getEnvAsInt64Slice(key string, defaultValue []int64) []int64 {
	if value, exists := os.LookupEnv(key); exists {
		parts := strings.Split(value, ",")
		var result []int64
		for _, part := range parts {
			num, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
			if err == nil {
				result = append(result, num)
			}
		}
		return result
	}
	return defaultValue
}

func parseAdminIDs(idsStr string) []int64 {
	if idsStr == "" {
		return nil
	}

	var ids []int64
	for _, idStr := range strings.Split(idsStr, ",") {
		id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
		if err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}
func getAdminIDs() []int64 {
	idsStr := getEnv("ADMIN_IDS", "123")
	if idsStr == "" {
		return []int64{}
	}

	var ids []int64
	for _, idStr := range strings.Split(idsStr, ",") {
		id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
		if err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}
