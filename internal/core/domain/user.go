package domain

import "time"

type User struct {
	ID        int
	GoogleID  string
	Email     string
	Name      string
	CreatedAt time.Time
}

type UserProgress struct {
	UserID         int
	Level          int
	XP             int
	XPForNextLevel int
	Coins 		   int
	UpdatedAt      time.Time
}