package gamification_service

import (
	"context"
	"fmt"
	"github.com/NightFury2283/life_forge/internal/core/domain"
	core_errors "github.com/NightFury2283/life_forge/internal/core/errors"
	"time"
)

type ProgressRepository interface {
	GetHeroSummary(context.Context, int) (domain.HeroSummary, error)
	CompletedEvents(context.Context, int) (map[string]bool, error)
	CompleteEvent(context.Context, int, domain.Completion) (domain.UserProgress, error)
}
type CalendarReader interface {
	ListEvents(context.Context, int, time.Time, time.Time, []string) ([]domain.CalendarEvent, error)
}
type ProgressService struct {
	repo     ProgressRepository
	calendar CalendarReader
	now      func() time.Time
}

func NewProgress(repo ProgressRepository, calendar CalendarReader) *ProgressService {
	return &ProgressService{repo, calendar, time.Now}
}
func (s *ProgressService) Dashboard(ctx context.Context, userID int) (domain.HeroDashboard, error) {
	if userID <= 0 {
		return domain.HeroDashboard{}, core_errors.ErrUnauthorized
	}
	summary, err := s.repo.GetHeroSummary(ctx, userID)
	if err != nil {
		return domain.HeroDashboard{}, err
	}
	completed, err := s.repo.CompletedEvents(ctx, userID)
	if err != nil {
		return domain.HeroDashboard{}, err
	}
	now := s.now()
	events, calendarErr := s.calendar.ListEvents(ctx, userID, now.AddDate(0, 0, -1), now.AddDate(0, 0, 14), nil)
	result := domain.HeroDashboard{Summary: summary, Tasks: make([]domain.HeroTask, 0)}
	if calendarErr != nil {
		result.Warning = "Для задач подключи Google Calendar."
		return result, nil
	}
	for _, event := range events {
		result.Tasks = append(result.Tasks, domain.HeroTask{CalendarEvent: event, Completed: completed[event.CalendarID+"/"+event.ID]})
	}
	return result, nil
}
func (s *ProgressService) Complete(ctx context.Context, userID int, calendarID, eventID string) (domain.UserProgress, error) {
	if userID <= 0 {
		return domain.UserProgress{}, core_errors.ErrUnauthorized
	}
	if eventID == "" || len(eventID) > 1024 || len(calendarID) > 1024 {
		return domain.UserProgress{}, core_errors.ErrInvalidArgument
	}
	if calendarID == "" {
		calendarID = "primary"
	}
	// The hero dashboard currently awards only primary-calendar tasks. Requiring
	// this canonical identifier also prevents rewards through calendar-ID aliases.
	if calendarID != "primary" {
		return domain.UserProgress{}, core_errors.ErrInvalidArgument
	}
	now := s.now()
	events, err := s.calendar.ListEvents(ctx, userID, now.AddDate(0, 0, -14), now.AddDate(0, 0, 14), []string{calendarID})
	if err != nil {
		return domain.UserProgress{}, err
	}
	for _, event := range events {
		if event.ID == eventID {
			return s.repo.CompleteEvent(ctx, userID, domain.Completion{CalendarID: calendarID, EventID: eventID, Title: event.Title, XP: 10, Coins: 2})
		}
	}
	return domain.UserProgress{}, fmt.Errorf("event does not belong to your available calendar window: %w", core_errors.ErrNotFound)
}
