package storage

import (
	"carwash-bot/internal/models"
	"database/sql"
	_ "modernc.org/sqlite"
)

type SQLiteStorage struct {
	db        *sql.DB
	StartTime int
	EndTime   int
}

func NewSQLiteStorage(dbPath string, startTime, endTime int) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS bookings (
            id TEXT PRIMARY KEY,
            date TEXT NOT NULL,
            time TEXT NOT NULL,
            car_model TEXT NOT NULL,
            car_number TEXT NOT NULL,
            user_id INTEGER NOT NULL,
            created_at TIMESTAMP NOT NULL
        );
    `); err != nil {
		return nil, err
	}

	return &SQLiteStorage{
		db:        db,
		StartTime: startTime,
		EndTime:   endTime,
	}, nil
}

func (s *SQLiteStorage) AddBooking(booking models.Booking) error {
	_, err := s.db.Exec(`
		INSERT INTO bookings (id, date, time, car_model, car_number, user_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, booking.ID, booking.Date, booking.Time, booking.CarModel, booking.CarNumber, booking.UserID, booking.Created)
	return err
}

func (s *SQLiteStorage) IsTimeAvailable(date, time string) (bool, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM bookings
		WHERE date = ? AND time = ?
	`, date, time).Scan(&count)
	return count == 0, err
}

func (s *SQLiteStorage) GetAllBookings() ([]models.Booking, error) {
	rows, err := s.db.Query(`
		SELECT id, date, time, car_model, car_number, user_id, created_at
		FROM bookings
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []models.Booking
	for rows.Next() {
		var b models.Booking
		if err := rows.Scan(&b.ID, &b.Date, &b.Time, &b.CarModel, &b.CarNumber, &b.UserID, &b.Created); err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}
	return bookings, nil
}

func (s *SQLiteStorage) GetBookingsByDate(date string) ([]models.Booking, error) {
	rows, err := s.db.Query(`
		SELECT id, date, time, car_model, car_number, user_id, created_at
		FROM bookings
		WHERE date = ?
	`, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []models.Booking
	for rows.Next() {
		var b models.Booking
		if err := rows.Scan(&b.ID, &b.Date, &b.Time, &b.CarModel, &b.CarNumber, &b.UserID, &b.Created); err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}
	return bookings, nil
}

func (s *SQLiteStorage) GetBookingByID(id string) (*models.Booking, error) {
	var booking models.Booking

	err := s.db.QueryRow(`
        SELECT id, date, time, car_model, car_number, user_id, created_at
        FROM bookings
        WHERE id = ?
    `, id).Scan(
		&booking.ID,
		&booking.Date,
		&booking.Time,
		&booking.CarModel,
		&booking.CarNumber,
		&booking.UserID,
		&booking.Created,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Запись не найдена
		}
		return nil, err // Произошла ошибка при запросе
	}

	return &booking, nil
}

func (s *SQLiteStorage) DeleteBooking(id string) error {
	_, err := s.db.Exec("DELETE FROM bookings WHERE id = ?", id)
	return err
}

func (s *SQLiteStorage) GetUserBookings(userID int64) ([]models.Booking, error) {
	rows, err := s.db.Query(`
		SELECT id, date, time, car_model, car_number, user_id, created_at
		FROM bookings
		WHERE user_id = ?
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []models.Booking
	for rows.Next() {
		var b models.Booking
		if err := rows.Scan(&b.ID, &b.Date, &b.Time, &b.CarModel, &b.CarNumber, &b.UserID, &b.Created); err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}
	return bookings, nil
}

func (s *SQLiteStorage) GetBookingsByDateTime(date, time string) ([]models.Booking, error) {
	rows, err := s.db.Query(`
		SELECT id, date, time, car_model, car_number, user_id, created_at
		FROM bookings
		WHERE date = ? AND time = ?
	`, date, time)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []models.Booking
	for rows.Next() {
		var b models.Booking
		if err := rows.Scan(&b.ID, &b.Date, &b.Time, &b.CarModel, &b.CarNumber, &b.UserID, &b.Created); err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}
	return bookings, nil
}
