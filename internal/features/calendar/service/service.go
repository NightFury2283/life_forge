package calendar_service

import (
	"context"
	"fmt"
	"github.com/NightFury2283/life_forge/internal/core/domain"
	core_errors "github.com/NightFury2283/life_forge/internal/core/errors"
	"time"
)

type Repository interface {
	ListCalendars(context.Context, int) ([]domain.Calendar, error)
	ListEvents(context.Context, int, time.Time, time.Time, []string) ([]domain.CalendarEvent, error)
	CreateEvent(context.Context, int, domain.EventDraft) (domain.CalendarEvent, error)
}
type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }
func (s *Service) ListCalendars(ctx context.Context, userID int) ([]domain.Calendar, error) {
	if userID <= 0 {
		return nil, core_errors.ErrUnauthorized
	}
	return s.repo.ListCalendars(ctx, userID)
}
func (s *Service) ListEvents(ctx context.Context, userID int, start, end time.Time, ids []string) ([]domain.CalendarEvent, error) {
	if userID <= 0 || start.IsZero() || !end.After(start) || end.Sub(start) > 93*24*time.Hour || len(ids) > 10 {
		return nil, fmt.Errorf("invalid calendar window (max 93 days, 10 calendars): %w", core_errors.ErrInvalidArgument)
	}
	if len(ids) == 0 {
		ids = []string{"primary"}
	}
	return s.repo.ListEvents(ctx, userID, start, end, ids)
}
func (s *Service) CreateEvent(ctx context.Context, userID int, draft domain.EventDraft) (domain.CalendarEvent, error) {
	if err := draft.Validate(); err != nil {
		return domain.CalendarEvent{}, err
	}
	if userID <= 0 {
		return domain.CalendarEvent{}, core_errors.ErrUnauthorized
	}
	return s.repo.CreateEvent(ctx, userID, draft)
}
