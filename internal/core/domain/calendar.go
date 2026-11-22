package domain

import (
	"fmt"
	"strings"
	"time"

	core_errors "github.com/NightFury2283/life_forge/internal/core/errors"
)

type Calendar struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
	Color   string `json:"backgroundColor"`
	Primary bool   `json:"primary"`
}

type CalendarEvent struct {
	ID          string    `json:"id"`
	CalendarID  string    `json:"calendar_id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Start       time.Time `json:"start_time"`
	End         time.Time `json:"end_time"`
	AllDay      bool      `json:"is_all_day"`
}

type EventDraft struct {
	Title       string    `json:"title" validate:"required,max=200"`
	Description string    `json:"description" validate:"max=4000"`
	Start       time.Time `json:"start_time"`
	Duration    float64   `json:"duration"`
	Recurrence  string    `json:"recurrence" validate:"max=200"`
	RequestID   string    `json:"request_id" validate:"max=64"`
}

func (d EventDraft) Validate() error {
	if strings.TrimSpace(d.Title) == "" || len([]rune(d.Title)) > 200 || len([]rune(d.Description)) > 4000 || d.Start.IsZero() || !(d.Duration > 0 && d.Duration <= 168) {
		return fmt.Errorf("event needs a title, start time and duration between 0 and 168 hours: %w", core_errors.ErrInvalidArgument)
	}
	switch strings.ToUpper(d.Recurrence) {
	case "", "DAILY", "WEEKLY", "MONTHLY", "YEARLY":
	default:
		return fmt.Errorf("unsupported recurrence: %w", core_errors.ErrInvalidArgument)
	}
	return nil
}
