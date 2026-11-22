package calendar_service_test

import (
	"context"
	"errors"
	"github.com/NightFury2283/life_forge/internal/core/domain"
	core_errors "github.com/NightFury2283/life_forge/internal/core/errors"
	calendar "github.com/NightFury2283/life_forge/internal/features/calendar/service"
	"math"
	"testing"
	"time"
)

type repo struct {
	list   func(context.Context, int, time.Time, time.Time, []string) ([]domain.CalendarEvent, error)
	create func(context.Context, int, domain.EventDraft) (domain.CalendarEvent, error)
}

func (r repo) ListCalendars(context.Context, int) ([]domain.Calendar, error) { return nil, nil }
func (r repo) ListEvents(c context.Context, id int, s, e time.Time, ids []string) ([]domain.CalendarEvent, error) {
	return r.list(c, id, s, e, ids)
}
func (r repo) CreateEvent(c context.Context, id int, d domain.EventDraft) (domain.CalendarEvent, error) {
	return r.create(c, id, d)
}
func TestListEventsValidation(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name     string
		user     int
		duration time.Duration
		invalid  bool
	}{{"week", 7, 7 * 24 * time.Hour, false}, {"no_user", 0, time.Hour, true}, {"empty_window", 7, 0, true}, {"reversed", 7, -time.Hour, true}, {"too_long", 7, 94 * 24 * time.Hour, true}} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			calls := 0
			ctx := t.Context()
			s := calendar.New(repo{list: func(got context.Context, id int, a, b time.Time, ids []string) ([]domain.CalendarEvent, error) {
				calls++
				if got != ctx || id != 7 || len(ids) != 1 || ids[0] != "primary" {
					t.Fatal("wrong context, user or default calendar")
				}
				return []domain.CalendarEvent{}, nil
			}})
			_, err := s.ListEvents(ctx, tt.user, start, start.Add(tt.duration), nil)
			if tt.invalid {
				if !errors.Is(err, core_errors.ErrInvalidArgument) || calls != 0 {
					t.Fatalf("error=%v calls=%d", err, calls)
				}
			} else if err != nil || calls != 1 {
				t.Fatalf("error=%v calls=%d", err, calls)
			}
		})
	}
}
func TestCreateEventValidation(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name       string
		duration   float64
		recurrence string
		invalid    bool
	}{{"valid", 1, "", false}, {"recurring", .5, "WEEKLY", false}, {"negative", -1, "", true}, {"nan", math.NaN(), "", true}, {"infinite", math.Inf(1), "", true}, {"unknown_recurrence", 1, "hack", true}} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			calls := 0
			draft := domain.EventDraft{Title: "Test", Start: time.Now(), Duration: tt.duration, Recurrence: tt.recurrence}
			s := calendar.New(repo{create: func(ctx context.Context, id int, d domain.EventDraft) (domain.CalendarEvent, error) {
				calls++
				if id != 42 || d.Title != draft.Title {
					t.Fatal("wrong owner/draft")
				}
				return domain.CalendarEvent{ID: "created"}, nil
			}})
			result, err := s.CreateEvent(t.Context(), 42, draft)
			if tt.invalid {
				if !errors.Is(err, core_errors.ErrInvalidArgument) || calls != 0 {
					t.Fatalf("error=%v calls=%d", err, calls)
				}
			} else if err != nil || result.ID != "created" || calls != 1 {
				t.Fatalf("result=%v err=%v", result, err)
			}
		})
	}
}
