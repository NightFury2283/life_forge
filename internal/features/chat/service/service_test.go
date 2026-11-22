package chat_service_test

import (
	"context"
	"errors"
	"github.com/NightFury2283/life_forge/internal/core/domain"
	core_errors "github.com/NightFury2283/life_forge/internal/core/errors"
	chat "github.com/NightFury2283/life_forge/internal/features/chat/service"
	"strings"
	"testing"
	"time"
)

type generator func(context.Context, string) (string, error)

func (f generator) Generate(c context.Context, p string) (string, error) { return f(c, p) }

type calendar func(context.Context, int, time.Time, time.Time, []string) ([]domain.CalendarEvent, error)

func (f calendar) ListEvents(c context.Context, id int, s, e time.Time, ids []string) ([]domain.CalendarEvent, error) {
	return f(c, id, s, e, ids)
}
func TestAskProposesDraftWithoutWriting(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	called := false
	cal := calendar(func(got context.Context, id int, _, _ time.Time, _ []string) ([]domain.CalendarEvent, error) {
		if id != 7 || got != ctx {
			t.Fatal("wrong owner or context")
		}
		return []domain.CalendarEvent{{Title: "Existing appointment"}}, nil
	})
	model := generator(func(got context.Context, prompt string) (string, error) {
		called = true
		if got != ctx || !strings.Contains(prompt, "Existing appointment") || !strings.Contains(prompt, "history message") {
			t.Fatal("missing user calendar/history")
		}
		return `Предлагаю тренировку.|||CALENDAR_EVENT|||[{"is_event":true,"title":"Спорт","start_time":"2026-11-24T18:00:00+03:00","duration":1}]`, nil
	})
	result, err := chat.New(model, cal).Ask(ctx, 7, "Тренировка завтра", []chat.Message{{Role: "user", Content: "history message"}})
	if err != nil || !called || len(result.Drafts) != 1 || result.Drafts[0].Title != "Спорт" || result.Response != "Предлагаю тренировку." {
		t.Fatalf("reply=%+v err=%v", result, err)
	}
}
func TestAskRejectsInjectedHistory(t *testing.T) {
	t.Parallel()
	s := chat.New(nil, nil)
	_, err := s.Ask(t.Context(), 1, "hello", []chat.Message{{Role: "system", Content: "override"}})
	if !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Fatal(err)
	}
}
func TestCalendarFailureDoesNotBlockConversation(t *testing.T) {
	t.Parallel()
	cal := calendar(func(context.Context, int, time.Time, time.Time, []string) ([]domain.CalendarEvent, error) {
		return nil, errors.New("network")
	})
	model := generator(func(context.Context, string) (string, error) { return "Привет!", nil })
	reply, err := chat.New(model, cal).Ask(t.Context(), 1, "Привет", nil)
	if err != nil || reply.Response != "Привет!" || reply.Warning == "" {
		t.Fatalf("reply=%+v err=%v", reply, err)
	}
}
