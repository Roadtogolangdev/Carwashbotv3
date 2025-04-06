package main

import (
	"carwash-bot/config"
	"carwash-bot/internal/bot"
	"github.com/joho/godotenv"
	"log"
)

func main() {
	// 1. Загрузка переменных окружения
	if err := godotenv.Load(); err != nil {
		log.Printf("Предупреждение: не удалось загрузить .env файл: %v", err)
	}

	// 2. Загрузка конфигурации
	cfg := config.Load()

	// 3. Валидация конфигурации
	if cfg.BotToken == "" {
		log.Fatal("ОШИБКА: Токен бота не установлен. Укажите TELEGRAM_BOT_TOKEN в .env файле")
	}

	// 4. Безопасное логирование токена (первые 5 символов)
	log.Printf("Токен бота загружен (первые 5 символов: %q)", cfg.BotToken[:5])

	// 5. Создание бота
	carWashBot, err := bot.New(cfg)
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}

	// 6. Запуск бота
	log.Println("Бот успешно запущен!")
	carWashBot.Start()
}
