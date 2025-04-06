package models

import "time"

type Booking struct {
	ID        string    `json:"id"`   // Уникальный идентификатор записи
	Date      string    `json:"date"` // Формат: "02.01.2006"
	Time      string    `json:"time"` // Формат: "15:04"
	CarModel  string    `json:"car_model"`
	CarNumber string    `json:"car_number"`
	UserID    int64     `json:"user_id"`
	Created   time.Time `json:"created_at"`
}

type UserState struct {
	AwaitingDay     bool
	AwaitingTime    bool
	AwaitingCarInfo bool
	SelectedDate    string
	SelectedTime    string
}
type TimeSlot struct {
	Time      string
	Available bool
	BookedBy  string
	CarModel  any
	CarNumber any
}

var weekdays = map[time.Weekday]string{
	time.Sunday:    "Воскресенье",
	time.Monday:    "Понедельник",
	time.Tuesday:   "Вторник",
	time.Wednesday: "Среда",
	time.Thursday:  "Четверг",
	time.Friday:    "Пятница",
	time.Saturday:  "Суббота",
}
