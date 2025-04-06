package bot

import (
	_ "carwash-bot/config"
	"carwash-bot/internal/models"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"sort"
	"strings"
	"time"
)

func (b *CarWashBot) handleMessage(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	userID := msg.From.ID
	text := msg.Text

	// Проверяем состояние пользователя (ожидание данных авто)
	if state, exists := b.userStates[userID]; exists {
		if state.AwaitingCarInfo {
			b.handleCarInfoInput(chatID, userID, text)
			return
		}
	}

	// Обрабатываем команды
	switch {
	case text == "/start" || text == "/menu" || text == "🏠 Главное меню":
		b.sendWelcomeMessage(chatID)

	case text == "📝 Записаться" || text == "/book":
		b.showDaySelection(chatID)

	case text == "🕒 Расписание" || text == "/schedule":
		b.showSchedule(chatID)

	case text == "❌ Мои записи" || text == "/mybookings":
		b.showUserBookings(msg.Chat.ID, msg.From.ID)

	case text == "❌ Отменить запись" || text == "/cancel":
		b.handleCancelCommand(chatID, userID)

	default:
		b.sendMessage(chatID, "Я не понимаю эту команду. Используйте кнопки меню.")
	}
}

func (b *CarWashBot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	chatID := query.Message.Chat.ID
	userID := query.From.ID
	data := query.Data

	// Отвечаем на callback (убираем "часы ожидания")
	callback := tgbotapi.NewCallback(query.ID, "")
	if _, err := b.botAPI.Request(callback); err != nil {
		log.Printf("Ошибка ответа на callback: %v", err)
	}

	switch {
	case strings.HasPrefix(data, "day_"):
		dateStr := strings.TrimPrefix(data, "day_")
		b.handleDaySelection(chatID, userID, dateStr)

	case strings.HasPrefix(data, "time_"):
		timeStr := strings.TrimPrefix(data, "time_")
		b.handleTimeSelection(chatID, userID, timeStr)

	case data == "main_menu":
		b.sendWelcomeMessage(chatID)

	case strings.HasPrefix(data, "cancel_"):
		bookingID := strings.TrimPrefix(data, "cancel_")
		b.handleBookingCancellation(chatID, userID, bookingID)

	case data == "back_to_dates":
		b.showDaySelection(chatID)

	case strings.HasPrefix(data, "admin_cancel:"):
		if !b.isAdmin(query.From.ID) {
			b.answerCallback(query.ID, "❌ Только администратор может отменять записи", true)
			return
		}
		bookingID := strings.TrimPrefix(data, "admin_cancel:")
		err := b.storage.DeleteBooking(bookingID)
		if err != nil {
			b.answerCallback(query.ID, "⚠️ Не удалось отменить запись", true)
			return
		}
		b.answerCallback(query.ID, "✅ Запись отменена", false)

		// Обновляем сообщение в канале
		editMsg := tgbotapi.NewEditMessageText(
			b.config.ChannelID,
			query.Message.MessageID,
			fmt.Sprintf("❌ ОТМЕНЕНО АДМИНОМ\n%s", query.Message.Text),
		)
		editMsg.ParseMode = "HTML"
		b.botAPI.Send(editMsg)
	default:
		b.answerCallback(query.ID, "", false) // Просто убираем "часы ожидания"
	}

}
func (b *CarWashBot) sendWelcomeMessage(chatID int64) {

	msgText := `🚗 *Добро пожаловать в бота автомойки!* 🧼

    
Выберите действие:`

	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📝 Записаться"),
			tgbotapi.NewKeyboardButton("🕒 Расписание"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("❌ Отменить запись"),
			tgbotapi.NewKeyboardButton("ℹ️ Помощь"),
		),
	)
	b.sendMessageWithSave(chatID, msg)
}

func (b *CarWashBot) handleTimeSelection(chatID, userID int64, timeStr string) {
	state := b.userStates[userID]
	now := time.Now()
	today := now.Format("02.01.2006")
	if state.SelectedDate == today {
		selectedTime, err := time.Parse("15:04", timeStr)
		if err == nil {
			// Получаем текущее время + 1 час (чтобы нельзя было записаться прямо сейчас)
			currentTime := now.Add(time.Hour)
			selectedDateTime := time.Date(
				now.Year(), now.Month(), now.Day(),
				selectedTime.Hour(), selectedTime.Minute(), 0, 0, now.Location())

			if selectedDateTime.Before(currentTime) {
				b.sendMessage(chatID, "❌ Нельзя записаться на прошедшее время")
				b.showTimeSlots(chatID, state.SelectedDate)
				return
			}
		}
	}
	// Проверяем доступность времени
	available, err := b.storage.IsTimeAvailable(state.SelectedDate, timeStr)
	if err != nil || !available {
		b.sendMessage(chatID, "❌ Это время уже занято! Выберите другое время.")
		b.showTimeSlots(chatID, state.SelectedDate)
		return
	}

	b.userStates[userID] = models.UserState{
		AwaitingCarInfo: true,
		SelectedDate:    state.SelectedDate,
		SelectedTime:    timeStr,
	}

	msg := tgbotapi.NewMessage(chatID, "Введите марку и номер машины через пробел\nПример: Лада 123")
	b.sendMessageWithSave(chatID, msg)
}

func (b *CarWashBot) handleCarInfoInput(chatID, userID int64, text string) {
	// Удаляем предыдущее сообщение
	b.deleteLastMessage(chatID)

	// Упрощенная проверка - просто разделяем по первому пробелу
	parts := strings.SplitN(text, " ", 2)
	if len(parts) < 2 {
		msg := tgbotapi.NewMessage(chatID, "Нужно ввести и марку, и номер!\nПример: Газель 123")
		b.sendMessageWithSave(chatID, msg)
		return
	}

	carModel := parts[0]
	carNumber := parts[1]
	state := b.userStates[userID]

	// Записываем в расписание
	err := b.storage.AddBooking(models.Booking{
		ID:        fmt.Sprintf("%d-%s-%s", userID, state.SelectedDate, state.SelectedTime),
		Date:      state.SelectedDate,
		Time:      state.SelectedTime,
		CarModel:  carModel,
		CarNumber: carNumber,
		UserID:    userID,
		Created:   time.Now(),
	})
	if err != nil {
		b.sendMessage(chatID, "⚠️ Ошибка при сохранении записи")
		return
	}

	if b.config.ChannelID != 0 {
		booking, err := b.storage.GetBookingByID(fmt.Sprintf("%d-%s-%s", userID, state.SelectedDate, state.SelectedTime))
		if err != nil {
			log.Printf("Ошибка получения записи: %v", err)
		} else if booking != nil {
			if err := b.notifyChannel(*booking); err != nil {
				log.Printf("Ошибка оповещения канала: %v", err)
				b.sendMessage(b.adminID, fmt.Sprintf("Ошибка отправки в канал: %v", err))
			}
		}
	}
	delete(b.userStates, userID)

	// Отправляем подтверждение
	confirmMsg := fmt.Sprintf(`✅ Вы успешно записаны на мойку!

	📅 Дата: %s
	🕒 Время: %s
	🚗 Автомобиль: %s %s
	
	Спасибо за выбор нашей услуги!`,
		state.SelectedDate, state.SelectedTime, carModel, carNumber)

	msg := tgbotapi.NewMessage(chatID, confirmMsg)
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🏠 Главное меню"),
		),
	)
	b.sendMessageWithSave(chatID, msg)

	// Уведомляем админа
	b.notifyAdminAboutNewBooking(state.SelectedTime, carModel, carNumber)
}
func (b *CarWashBot) showSchedule(chatID int64) {
	// Русские названия дней недели и месяцев
	weekdayNames := map[time.Weekday]string{
		time.Monday:    "Понедельник",
		time.Tuesday:   "Вторник",
		time.Wednesday: "Среда",
		time.Thursday:  "Четверг",
		time.Friday:    "Пятница",
		time.Saturday:  "Суббота",
		time.Sunday:    "Воскресенье",
	}

	monthNames := map[time.Month]string{
		time.January:   "Января",
		time.February:  "Февраля",
		time.March:     "Марта",
		time.April:     "Апреля",
		time.May:       "Мая",
		time.June:      "Июня",
		time.July:      "Июля",
		time.August:    "Августа",
		time.September: "Сентября",
		time.October:   "Октября",
		time.November:  "Ноября",
		time.December:  "Декабря",
	}

	allBookings, err := b.storage.GetAllBookings()
	if err != nil {
		b.sendMessage(chatID, "⚠️ Ошибка при получении расписания")
		return
	}
	bookingsByDate := make(map[string][]models.Booking)
	for _, booking := range allBookings {
		bookingsByDate[booking.Date] = append(bookingsByDate[booking.Date], booking)
	}

	// Сортируем даты
	var dates []time.Time
	for dateStr := range bookingsByDate {
		date, _ := time.Parse("02.01.2006", dateStr)
		dates = append(dates, date)
	}
	sort.Slice(dates, func(i, j int) bool {
		return dates[i].Before(dates[j])
	})

	var sb strings.Builder
	sb.WriteString("📅 *Расписание моек*\n\n")

	now := time.Now()
	today := now.Format("02.01.2006")
	tomorrow := now.AddDate(0, 0, 1).Format("02.01.2006")

	for _, date := range dates {
		dateStr := date.Format("02.01.2006")
		dayName := weekdayNames[date.Weekday()]
		monthName := monthNames[date.Month()]

		// Форматируем заголовок
		switch dateStr {
		case today:
			sb.WriteString(fmt.Sprintf("=== Сегодня, %d %s ===\n", date.Day(), monthName))
		case tomorrow:
			sb.WriteString(fmt.Sprintf("=== Завтра, %d %s ===\n", date.Day(), monthName))
		default:
			sb.WriteString(fmt.Sprintf("=== %s, %d %s ===\n", dayName, date.Day(), monthName))
		}

		// Сортируем записи по времени
		bookings := bookingsByDate[dateStr]
		sort.Slice(bookings, func(i, j int) bool {
			return bookings[i].Time < bookings[j].Time
		})

		// Добавляем записи
		for _, booking := range bookings {
			sb.WriteString(fmt.Sprintf("🕒 %s - %s %s\n",
				booking.Time,
				booking.CarModel,
				booking.CarNumber))
		}
		sb.WriteString("\n")
	}

	if len(dates) == 0 {
		sb.WriteString("На данный момент нет записей\n")
	}

	msg := tgbotapi.NewMessage(chatID, sb.String())
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📝 Записаться"),
			tgbotapi.NewKeyboardButton("🏠 Главное меню"),
		),
	)
	b.sendMessageWithSave(chatID, msg)
}

func (b *CarWashBot) notifyAdminAboutNewBooking(timeStr, carModel, carNumber string) {
	msgText := fmt.Sprintf(`🆕 Новая запись:
Время: %s
Авто: %s %s`, timeStr, carModel, carNumber)

	msg := tgbotapi.NewMessage(b.adminID, msgText)
	b.botAPI.Send(msg)
}

func (b *CarWashBot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := b.botAPI.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}

func (b *CarWashBot) sendMessageWithSave(chatID int64, msg tgbotapi.MessageConfig) {
	sentMsg, err := b.botAPI.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
		return
	}

	b.msgIDLock.Lock()
	b.lastMessageID[chatID] = sentMsg.MessageID
	b.msgIDLock.Unlock()
}

func (b *CarWashBot) deleteLastMessage(chatID int64) {
	b.msgIDLock.Lock()
	msgID := b.lastMessageID[chatID]
	b.msgIDLock.Unlock()

	if msgID != 0 {
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, msgID)
		b.botAPI.Request(deleteMsg)
	}
}
func (b *CarWashBot) showTimeSlots(chatID int64, dateStr string) {
	now := time.Now()
	todayStr := now.Format("02.01.2006")

	date, err := time.Parse("02.01.2006", dateStr)
	if err != nil {
		b.sendMessage(chatID, "Ошибка формата даты")
		return
	}

	weekdayNames := [...]string{
		"Воскресенье", "Понедельник", "Вторник",
		"Среда", "Четверг", "Пятница", "Суббота",
	}

	header := fmt.Sprintf("Выберите время на %s, %s:",
		weekdayNames[date.Weekday()],
		date.Format("02.01.2006"))

	var rows [][]tgbotapi.InlineKeyboardButton

	for hour := b.storage.StartTime; hour <= b.storage.EndTime; hour++ {
		timeStr := fmt.Sprintf("%02d:00", hour)
		available, err := b.storage.IsTimeAvailable(dateStr, timeStr)
		if err != nil {
			available = false
		}

		// Если это сегодня и время уже прошло
		if dateStr == todayStr && hour <= now.Hour() {
			available = false
		}

		btnText := fmt.Sprintf("🕒 %s", timeStr)
		if !available {
			btnText = "🔴 " + timeStr + " (Недоступно)"
		} else {
			btnText = "🟢 " + timeStr + " (Свободно)"
		}

		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnText, "time_"+timeStr),
		))
	}

	// Добавляем кнопки навигации
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к выбору даты", "back_to_dates"),
		tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "main_menu"),
	))

	msg := tgbotapi.NewMessage(chatID, header)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	b.sendMessageWithSave(chatID, msg)
}
func (b *CarWashBot) showDaySelection(chatID int64) {
	now := time.Now()
	var buttons [][]tgbotapi.InlineKeyboardButton

	weekdayNames := []string{
		"Воскресенье", "Понедельник", "Вторник",
		"Среда", "Четверг", "Пятница", "Суббота",
	}

	for i := 0; i < 7; i++ {
		date := now.AddDate(0, 0, i)
		dateStr := date.Format("02.01.2006")
		weekday := weekdayNames[date.Weekday()]

		dayDesc := ""
		switch i {
		case 0:
			dayDesc = "Сегодня"
		case 1:
			dayDesc = "Завтра"
		default:
			dayDesc = weekday
		}

		btnText := fmt.Sprintf("📅 %s (%s)", dayDesc, date.Format("02.01"))
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnText, "day_"+dateStr),
		))
	}

	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "main_menu"),
	))

	msg := tgbotapi.NewMessage(chatID, "Выберите день для записи:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	b.sendMessageWithSave(chatID, msg)
}
func (b *CarWashBot) handleDaySelection(chatID, userID int64, dateStr string) {
	now := time.Now()
	todayStr := now.Format("02.01.2006")

	// Парсим выбранную дату
	selectedDate, err := time.Parse("02.01.2006", dateStr)
	if err != nil {
		b.sendMessage(chatID, "❌ Ошибка формата даты")
		b.showDaySelection(chatID)
		return
	}

	// Проверяем, что дата не в прошлом (используем parsedDate)
	if selectedDate.Before(now.Truncate(24 * time.Hour)) {
		b.sendMessage(chatID, "❌ Нельзя записаться на прошедшую дату")
		b.showDaySelection(chatID)
		return
	}

	// Если выбрана сегодняшняя дата
	if dateStr == todayStr {
		currentHour := now.Hour()
		if currentHour >= b.storage.EndTime {
			b.sendMessage(chatID, "❌ На сегодня время записи уже закончилось")
			b.showDaySelection(chatID)
			return
		}
	}

	b.userStates[userID] = models.UserState{
		AwaitingTime: true,
		SelectedDate: dateStr,
	}

	b.showTimeSlots(chatID, dateStr)
}
func (b *CarWashBot) showUserBookings(chatID, userID int64) {
	bookings, err := b.storage.GetUserBookings(userID)
	if err != nil {
		b.sendMessage(chatID, "⚠️ Ошибка при получении ваших записей")
		return
	}

	if len(bookings) == 0 {
		b.sendMessage(chatID, "У вас нет активных записей.")
		return
	}

	var sb strings.Builder
	sb.WriteString("📋 *Ваши записи:*\n\n")

	var buttons [][]tgbotapi.InlineKeyboardButton

	for _, booking := range bookings {
		sb.WriteString(fmt.Sprintf(
			"📅 %s\n🕒 %s\n🚗 %s %s\n\n",
			booking.Date,
			booking.Time,
			booking.CarModel,
			booking.CarNumber,
		))

		// Добавляем кнопку отмены для каждой записи
		btnData := fmt.Sprintf("cancel_%s_%s", booking.Date, booking.Time)
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отменить эту запись", btnData),
		))
	}

	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🏠 В главное меню", "main_menu"),
	))

	msg := tgbotapi.NewMessage(chatID, sb.String())
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	b.sendMessageWithSave(chatID, msg)
}
func (b *CarWashBot) handleCancelCommand(chatID, userID int64) {
	userBookings, _ := b.storage.GetUserBookings(userID)
	if len(userBookings) == 0 {
		b.sendMessage(chatID, "У вас нет активных записей.")
		return
	}

	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, booking := range userBookings {
		btnText := fmt.Sprintf("%s %s - %s %s",
			booking.Date, booking.Time, booking.CarModel, booking.CarNumber)
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnText, "cancel_"+booking.ID),
		))
	}

	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "main_menu"),
	))

	msg := tgbotapi.NewMessage(chatID, "Выберите запись для отмены:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	b.sendMessageWithSave(chatID, msg)
}

func (b *CarWashBot) handleBookingCancellation(chatID, userID int64, bookingID string) {
	booking, err := b.storage.GetBookingByID(bookingID)
	if err != nil {
		b.sendMessage(chatID, "❌ Ошибка при отмене записи")
		return
	}

	if booking == nil {
		b.sendMessage(chatID, "❌ Запись не найдена")
		return
	}

	err = b.storage.DeleteBooking(bookingID)
	if err != nil {
		b.sendMessage(chatID, "❌ Не удалось отменить запись")
		return
	}

	msg := fmt.Sprintf("✅ Запись отменена:\n📅 %s\n🕒 %s\n🚗 %s %s",
		booking.Date,
		booking.Time,
		booking.CarModel,
		booking.CarNumber)
	b.sendMessage(chatID, msg)

	// Уведомление администратора
	if userID != b.adminID {
		adminMsg := fmt.Sprintf("ℹ️ Пользователь отменил запись:\n%s %s - %s %s",
			booking.Date, booking.Time, booking.CarModel, booking.CarNumber)
		b.sendMessage(b.adminID, adminMsg)
	}
}
func (b *CarWashBot) notifyChannelWithButtons(booking models.Booking) {
	msgText := fmt.Sprintf(`🆕 Новая запись:
🗓 %s в %s
🚘 %s %s`,
		booking.Date, booking.Time,
		booking.CarModel, booking.CarNumber)

	// Создаем кнопку "Отменить" для админов
	btn := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"❌ Отменить запись",
				fmt.Sprintf("admin_cancel_%s", booking.ID)),
		),
	)

	msg := tgbotapi.NewMessageToChannel("ваш_канал", msgText)
	msg.ReplyMarkup = btn
	b.botAPI.Send(msg)
}
