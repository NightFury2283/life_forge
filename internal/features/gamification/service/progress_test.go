package gamification_service_test

import (
	"context"
	"errors"
	"github.com/NightFury2283/life_forge/internal/core/domain"
	core_errors "github.com/NightFury2283/life_forge/internal/core/errors"
	service "github.com/NightFury2283/life_forge/internal/features/gamification/service"
	"testing"
	"time"
)

type progressRepo struct {
	calls       int
	completeErr error
}

func (r *progressRepo) GetHeroSummary(context.Context, int) (domain.HeroSummary, error) {
	return domain.HeroSummary{}, nil
}
func (r *progressRepo) CompletedEvents(context.Context, int) (map[string]bool, error) {
	return map[string]bool{}, nil
}
func (r *progressRepo) CompleteEvent(_ context.Context, id int, c domain.Completion) (domain.UserProgress, error) {
	r.calls++
	if id != 7 || c.XP != 10 || c.Coins != 2 || c.Title != "Real event" {
		panic("wrong owner or server rewards")
	}
	return domain.UserProgress{UserID: id, XP: 10, Coins: 2}, r.completeErr
}

type progressCalendar struct{}

func (progressCalendar) ListEvents(_ context.Context, id int, _, _ time.Time, _ []string) ([]domain.CalendarEvent, error) {
	if id != 7 {
		panic("wrong calendar owner")
	}
	return []domain.CalendarEvent{{ID: "real", CalendarID: "primary", Title: "Real event"}}, nil
}
func TestCompleteUsesServerCalendarAndRewards(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, event      string
		repoErr, wantErr error
		calls            int
	}{{"valid", "real", nil, nil, 1}, {"invented_event", "fake", nil, core_errors.ErrNotFound, 0}, {"duplicate", "real", core_errors.ErrConflict, core_errors.ErrConflict, 1}} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &progressRepo{completeErr: tt.repoErr}
			s := service.NewProgress(repo, progressCalendar{})
			p, err := s.Complete(t.Context(), 7, "primary", tt.event)
			if !errors.Is(err, tt.wantErr) || repo.calls != tt.calls {
				t.Fatalf("err=%v calls=%d", err, repo.calls)
			}
			if err == nil && (p.XP != 10 || p.Coins != 2) {
				t.Fatal(p)
			}
		})
	}
}
