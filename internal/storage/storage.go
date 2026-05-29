package storage

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/Soarkey/worktime/internal/config"
	_ "modernc.org/sqlite"
)

type Attendance struct {
	ID                int
	WorkDate          string
	StartTime         string
	ExpectedLeaveTime string
	ActualLeaveTime   string
	LateMinutes       int
	CreatedAt         time.Time
}

type Store struct {
	db *sql.DB
}

func New() (*Store, error) {
	if err := os.MkdirAll(config.DBDir(), 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", config.DBPath()))
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &Store{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS attendance (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			work_date TEXT NOT NULL UNIQUE,
			start_time TEXT,
			expected_leave_time TEXT,
			actual_leave_time TEXT,
			late_minutes INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) UpsertStart(workDate, startTime, expectedLeave string, lateMinutes int) error {
	_, err := s.db.Exec(`
		INSERT INTO attendance (work_date, start_time, expected_leave_time, late_minutes)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(work_date) DO UPDATE SET
			start_time = excluded.start_time,
			expected_leave_time = excluded.expected_leave_time,
			late_minutes = excluded.late_minutes
	`, workDate, startTime, expectedLeave, lateMinutes)
	return err
}

func (s *Store) UpdateActualLeave(workDate, actualLeave string) error {
	_, err := s.db.Exec(`
		UPDATE attendance SET actual_leave_time = ? WHERE work_date = ?
	`, actualLeave, workDate)
	return err
}

func (s *Store) GetByDate(workDate string) (*Attendance, error) {
	a := &Attendance{}
	err := s.db.QueryRow(`
		SELECT id, work_date, COALESCE(start_time,''), COALESCE(expected_leave_time,''),
		       COALESCE(actual_leave_time,''), late_minutes, created_at
		FROM attendance WHERE work_date = ?
	`, workDate).Scan(&a.ID, &a.WorkDate, &a.StartTime, &a.ExpectedLeaveTime,
		&a.ActualLeaveTime, &a.LateMinutes, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return a, err
}

func (s *Store) GetWeek(mondayDate string) ([]Attendance, error) {
	rows, err := s.db.Query(`
		SELECT id, work_date, COALESCE(start_time,''), COALESCE(expected_leave_time,''),
		       COALESCE(actual_leave_time,''), late_minutes, created_at
		FROM attendance
		WHERE work_date >= ? AND work_date < date(?, '+7 days')
		ORDER BY work_date
	`, mondayDate, mondayDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Attendance
	for rows.Next() {
		var a Attendance
		if err := rows.Scan(&a.ID, &a.WorkDate, &a.StartTime, &a.ExpectedLeaveTime,
			&a.ActualLeaveTime, &a.LateMinutes, &a.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (s *Store) GetAll() ([]Attendance, error) {
	rows, err := s.db.Query(`
		SELECT id, work_date, COALESCE(start_time,''), COALESCE(expected_leave_time,''),
		       COALESCE(actual_leave_time,''), late_minutes, created_at
		FROM attendance ORDER BY work_date
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Attendance
	for rows.Next() {
		var a Attendance
		if err := rows.Scan(&a.ID, &a.WorkDate, &a.StartTime, &a.ExpectedLeaveTime,
			&a.ActualLeaveTime, &a.LateMinutes, &a.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}
