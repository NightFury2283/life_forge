package models

import (
	"time"
)

type EventRequest struct {
	ID            int        `json:"id" db:"id"`
	IsEvent       bool       `json:"is_event" db:"is_event"`
	Title         string     `json:"title" db:"title"`
	StartTime     *time.Time `json:"start_time,omitempty" db:"start_time"`
	DurationHours *float64   `json:"duration,omitempty" db:"duration_hours"`
	Recurrence    *string    `json:"recurrence,omitempty" db:"recurrence"`
	Description   *string    `json:"description,omitempty" db:"description"`
}
