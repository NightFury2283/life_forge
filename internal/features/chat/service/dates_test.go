package chat_service

import (
	"context"
	"github.com/NightFury2283/life_forge/internal/core/domain"
	"testing"
	"time"
)

func TestRequestedDay(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 11, 22, 22, 0, 0, 0, moscow)
	for _, tt := range []struct {
		text, want string
		ok         bool
	}{{"поставь на четверг 19:00-21:00", "2026-11-26", true}, {"на понедельник", "2026-11-23", true}, {"завтра", "2026-11-23", true}, {"послезавтра", "2026-11-24", true}, {"в среду", "2026-11-25", true}, {"в следующую неделю в четверг", "2026-11-26", true}, {"во вторник и пятницу", "", false}, {"четверг 03.12.2026", "", false}, {"2026-12-03 в четверг", "", false}} {
		t.Run(tt.text, func(t *testing.T) {
			got, ok := requestedDay(tt.text, now)
			if ok != tt.ok || (ok && got.Format("2006-01-02") != tt.want) {
				t.Fatalf("got=%v,%v want=%s,%v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

type fixedModel struct{}

func (fixedModel) Generate(context.Context, string) (string, error) {
	return `В четверг.|||CALENDAR_EVENT|||[{"is_event":true,"title":"Встреча","start_time":"2026-11-23T19:00:00Z","duration":1}]`, nil
}

type emptyCalendar struct{}

func (emptyCalendar) ListEvents(context.Context, int, time.Time, time.Time, []string) ([]domain.CalendarEvent, error) {
	return nil, nil
}
func TestThursdayDraftCannotBecomeMonday(t *testing.T) {
	s := New(fixedModel{}, emptyCalendar{})
	s.now = func() time.Time { return time.Date(2026, 11, 22, 22, 0, 0, 0, moscow) }
	reply, err := s.Ask(t.Context(), 1, "поставь на четверг 19:00-21:00 встреча онлайн", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(reply.Drafts) != 1 {
		t.Fatal(reply)
	}
	d := reply.Drafts[0]
	if d.Start.Weekday() != time.Thursday || d.Start.Hour() != 19 || d.Duration != 2 || reply.Warning == "" {
		t.Fatalf("incorrect draft: %+v", d)
	}
}
