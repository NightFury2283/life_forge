package users_postgres_repository

import "time"

type UserProgressModel struct {
	UserID         int       `db:"user_id"`
	Level          int       `db:"level"`
	XP             int       `db:"xp"`
	XPForNextLevel int       `db:"xp_for_next_level"`
	UpdatedAt      time.Time `db:"updated_at"`
	Coins          int       `db:"coins"`
}
