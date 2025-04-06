package bot

import (
	"carwash-bot/storage"
	"errors"
	"fmt"
	"log"
	"sync"

	"carwash-bot/config"
	"carwash-bot/internal/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CarWashBot struct {
	botAPI        *tgbotapi.BotAPI
	storage       *storage.SQLiteStorage // Меняем тип на SQLiteStorage
	userStates    map[int64]models.UserState
	adminID       int64
	lastMessageID map[int64]int
	msgIDLock     sync.Mutex
	config        *config.Config
}

func New(config *config.Config) (*CarWashBot, error) {
	botAPI, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		return nil, err
	}

	botAPI.Debug = true

	// Заменяем ScheduleService на SQLiteStorage
	storageService, err := storage.NewSQLiteStorage(
		"bookings.db",
		config.StartTime,
		config.EndTime,
	)
	if err != nil {
		return nil, err
	}

	return &CarWashBot{
		botAPI:        botAPI,
		storage:       storageService,
		userStates:    make(map[int64]models.UserState),
		adminID:       config.AdminID,
		lastMessageID: make(map[int64]int),
		config:        config,
	}, nil
}
func (b *CarWashBot) Start() {
	log.Printf("Бот запущен: @%s", b.botAPI.Self.UserName)
	log.Printf("Admin IDs: %v", b.config.AdminIDs) // Правильное логирование

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.botAPI.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			b.handleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			b.handleCallbackQuery(update.CallbackQuery)
		}
	}
}

func (b *CarWashBot) notifyChannel(booking models.Booking) error {
	// Проверка на случай, если метод вызван без проверки ChannelID
	if b.config.ChannelID == 0 {
		return errors.New("channel ID not configured")
	}

	msgText := fmt.Sprintf(`🆕 Новая запись на мойку:
📅 <b>%s</b> в <code>%s</code>
🚗 <i>%s %s</i>
👤 ID: %d`,
		booking.Date,
		booking.Time,
		booking.CarModel,
		booking.CarNumber,
		booking.UserID)

	msg := tgbotapi.NewMessage(b.config.ChannelID, msgText)
	msg.ParseMode = "HTML"

	// Добавляем кнопку отмены для админов
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"❌ Отменить",
				fmt.Sprintf("admin_cancel:%s", booking.ID)),
		),
	)

	_, err := b.botAPI.Send(msg)
	return err
}

func (b *CarWashBot) answerCallback(callbackID string, text string, showAlert bool) {
	callback := tgbotapi.NewCallback(callbackID, text)
	if showAlert {
		callback.ShowAlert = true
	}
	if _, err := b.botAPI.Request(callback); err != nil {
		log.Printf("Ошибка ответа на callback: %v", err)
	}
}

func (b *CarWashBot) isAdmin(userID int64) bool {
	// Проверяем в списке админов
	for _, adminID := range b.config.AdminIDs {
		if userID == adminID {
			return true
		}
	}
	// Для обратной совместимости
	if b.config.AdminID != 0 && userID == b.config.AdminID {
		return true
	}
	return false
}

// ... остальные методы ...
