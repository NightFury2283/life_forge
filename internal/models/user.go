package models

import "time"

type User struct {
	ID        int       `json:"id" db:"id"`
	GoogleID  string    `json:"google_id" db:"google_id"`
	Email     string    `json:"email" db:"email"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type UserProgress struct {
	ID             int       `json:"id" db:"id"`
	UserID         int       `json:"user_id" db:"user_id"`
	Level          int       `json:"level" db:"level"`
	XP             int       `json:"xp" db:"xp"`
	XPForNextLevel int       `json:"xp_for_next_level" db:"xp_for_next_level"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

type UserStat struct {
	ID          int       `json:"id" db:"id"`
	UserID      int       `json:"user_id" db:"user_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Value       int       `json:"value" db:"value"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type CompletedTask struct {
	ID          int       `json:"id" db:"id"`
	UserID      int       `json:"user_id" db:"user_id"`
	EventID     string    `json:"event_id" db:"event_id"`
	EventTitle  string    `json:"event_title" db:"event_title"`
	XPGained    int       `json:"xp_gained" db:"xp_gained"`
	CompletedAt time.Time `json:"completed_at" db:"completed_at"`
}
