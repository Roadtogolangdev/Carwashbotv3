package services

import (
	"carwash-bot/internal/models"
	"fmt"
	_ "modernc.org/sqlite"
	"sort"
	"sync"
	"time"
)

type ScheduleService struct {
	bookings     []models.Booking
	bookingsLock sync.Mutex
	StartTime    int // Начальное время (часы)
	EndTime      int // Конечное время (часы)
	adminID      int64
}

func NewScheduleService(dbPath string, start, end int, adminID int64) (*ScheduleService, error) {
	return &ScheduleService{
		StartTime: start,
		EndTime:   end,
		adminID:   adminID,
	}, nil
}

func (s *ScheduleService) BookDateTime(date, timeStr, carModel, carNumber string, userID int64) bool {
	s.bookingsLock.Lock()
	defer s.bookingsLock.Unlock()

	// Проверяем, свободно ли время
	for _, booking := range s.bookings {
		if booking.Date == date && booking.Time == timeStr {
			return false
		}
	}

	// Добавляем новую запись
	s.bookings = append(s.bookings, models.Booking{
		ID:        fmt.Sprintf("%d-%s-%s", userID, date, timeStr), // Генерируем простой ID
		Date:      date,
		Time:      timeStr,
		CarModel:  carModel,
		CarNumber: carNumber,
		UserID:    userID,
		Created:   time.Now(),
	})

	return true
}

func (s *ScheduleService) IsTimeAvailable(date, timeStr string) bool {
	s.bookingsLock.Lock()
	defer s.bookingsLock.Unlock()

	for _, booking := range s.bookings {
		if booking.Date == date && booking.Time == timeStr {
			return false
		}
	}
	return true
}

func (s *ScheduleService) CancelBooking(bookingID string, userID int64) (bool, *models.Booking) {
	s.bookingsLock.Lock()
	defer s.bookingsLock.Unlock()

	for i, booking := range s.bookings {
		if booking.ID == bookingID {
			// Проверяем права (владелец или админ)
			if booking.UserID == userID || userID == s.adminID {
				// Удаляем запись
				deletedBooking := s.bookings[i]
				s.bookings = append(s.bookings[:i], s.bookings[i+1:]...)
				return true, &deletedBooking
			}
			return false, nil
		}
	}
	return false, nil
}

func (s *ScheduleService) GetBookingsGroupedByDate() map[string][]models.Booking {
	s.bookingsLock.Lock()
	defer s.bookingsLock.Unlock()

	result := make(map[string][]models.Booking)
	for _, booking := range s.bookings {
		result[booking.Date] = append(result[booking.Date], booking)
	}
	return result
}

func (s *ScheduleService) GetAvailableTimeSlots(date string) []string {
	s.bookingsLock.Lock()
	defer s.bookingsLock.Unlock()

	var slots []string
	bookedTimes := make(map[string]bool)

	for _, booking := range s.bookings {
		if booking.Date == date {
			bookedTimes[booking.Time] = true
		}
	}

	for hour := s.StartTime; hour <= s.EndTime; hour++ {
		timeStr := fmt.Sprintf("%02d:00", hour)
		if !bookedTimes[timeStr] {
			slots = append(slots, timeStr)
		}
	}

	return slots
}

func (s *ScheduleService) GetUserBookings(userID int64) []models.Booking {
	s.bookingsLock.Lock()
	defer s.bookingsLock.Unlock()

	var result []models.Booking
	for _, booking := range s.bookings {
		if booking.UserID == userID {
			result = append(result, booking)
		}
	}

	// Сортируем по дате и времени
	sort.Slice(result, func(i, j int) bool {
		dateI, _ := time.Parse("02.01.2006", result[i].Date)
		dateJ, _ := time.Parse("02.01.2006", result[j].Date)
		if dateI.Equal(dateJ) {
			return result[i].Time < result[j].Time
		}
		return dateI.Before(dateJ)
	})

	return result
}

func (s *ScheduleService) GetBooking(userID int64, date, time string) *models.Booking {
	s.bookingsLock.Lock()
	defer s.bookingsLock.Unlock()

	for _, booking := range s.bookings {
		if booking.UserID == userID && booking.Date == date && booking.Time == time {
			return &booking
		}
	}
	return nil
}
