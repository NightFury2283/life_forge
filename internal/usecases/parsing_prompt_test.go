package usecases_test

import (
	"github.com/NightFury2283/life_forge/internal/models"
	"github.com/NightFury2283/life_forge/internal/usecases"
	"reflect"
	"testing"
	"time"
)

func TestParseCalendarAIResponse(t *testing.T) {
	t.Parallel()
	separator := usecases.CALENDAR_SEPARATOR
	for _, tt := range []struct {
		name, input, answer string
		titles              []string
		wantErr             bool
	}{
		{"plain_text", "  Привет!  ", "Привет!", nil, false},
		{"empty", "", "", nil, false},
		{"answer_prefix", "Answer:\n Готово\n", "Готово", nil, false},
		{"empty_json", "Готово" + separator, "Готово", nil, false},
		{"empty_object", "Готово" + separator + "{}", "Готово", nil, false},
		{"empty_array", "Готово" + separator + "[]", "Готово", nil, false},
		{"single_event", "Готово" + separator + `{"is_event":true,"title":"Спорт"}`, "Готово", []string{"Спорт"}, false},
		{"skip_non_event", "Готово" + separator + `{"is_event":false,"title":"Не событие"}`, "Готово", nil, false},
		{"array_filters_non_events", "План" + separator + `[{"is_event":true,"title":"Спорт"},{"is_event":false,"start_time":"invalid"},{"is_event":true,"title":"Учёба"}]`, "План", []string{"Спорт", "Учёба"}, false},
		{"closing_separator", "План" + separator + `{"is_event":true,"title":"Учёба"}` + separator + "ignored", "План", []string{"Учёба"}, false},
		{"malformed_object", "План" + separator + `{"is_event":`, "План", nil, true},
		{"malformed_array", "План" + separator + `[{]`, "План", nil, true},
		{"invalid_time", "План" + separator + `{"is_event":true,"start_time":"tomorrow"}`, "План", nil, true},
		{"invalid_time_in_array", "План" + separator + `[{"is_event":true,"start_time":"tomorrow"}]`, "План", nil, true},
		{"wrong_field_type", "План" + separator + `{"is_event":"yes"}`, "План", nil, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			answer, events, err := usecases.ParseCalendarAIResponse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, want error %v", err, tt.wantErr)
			}
			if answer != tt.answer {
				t.Errorf("answer = %q, want %q", answer, tt.answer)
			}
			var titles []string
			for _, event := range events {
				titles = append(titles, event.Title)
			}
			if !reflect.DeepEqual(titles, tt.titles) {
				t.Errorf("event titles = %v, want %v", titles, tt.titles)
			}
		})
	}
}

func TestParseCalendarEventFields(t *testing.T) {
	t.Parallel()
	for _, start := range []string{"2026-11-11T15:00:00+03:00", "2026-11-11T12:00:00Z"} {
		t.Run(start, func(t *testing.T) {
			t.Parallel()
			input := `План` + usecases.CALENDAR_SEPARATOR + `{"is_event":true,"title":"Спорт","start_time":"` + start + `","duration":1.5,"recurrence":"daily","description":"Тренировка"}`
			_, events, err := usecases.ParseCalendarAIResponse(input)
			if err != nil {
				t.Fatal(err)
			}
			if len(events) != 1 {
				t.Fatalf("events = %d", len(events))
			}
			e := events[0]
			want, _ := time.Parse(time.RFC3339, start)
			if e.StartTime == nil || !e.StartTime.Equal(want) {
				t.Errorf("start = %v, want %v", e.StartTime, want)
			}
			if e.DurationHours == nil || *e.DurationHours != 1.5 || e.Recurrence == nil || *e.Recurrence != "daily" || e.Description == nil || *e.Description != "Тренировка" {
				t.Errorf("lost event fields: %+v", e)
			}
		})
	}
	t.Run("optional_null_fields", func(t *testing.T) {
		_, events, err := usecases.ParseCalendarAIResponse(usecases.CALENDAR_SEPARATOR + `{"is_event":true,"title":"Спорт","start_time":null,"duration":null}`)
		if err != nil {
			t.Fatal(err)
		}
		if len(events) != 1 {
			t.Fatalf("events = %d", len(events))
		}
		if e := events[0]; e.StartTime != nil || e.DurationHours != nil || e.Description != nil || e.Recurrence != nil {
			t.Errorf("optional fields = %+v", e)
		}
	})
}

func TestParseAIResponse(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, input, answer string
		want                models.Context
	}{
		{"plain_text", "  Привет  ", "Привет", models.Context{}},
		{"empty", "", "", models.Context{}},
		{"all_updates", "Ответ: Готово" + usecases.SEPARATOR + "\nЦели: Go, спорт, ,\nНедавние действия: тесты, прогулка\nПрогресс: Go:50%, спорт:2 из 3" + usecases.SEPARATOR + "ignored", "Готово", models.Context{Goals: []string{"Go", "спорт"}, Recent5: []string{"тесты", "прогулка"}, Progress: map[string]string{"Go": "50%", "спорт": "2 из 3"}}},
		{"ignore_empty_and_unknown", "Ответ: Готово" + usecases.SEPARATOR + "\nЦели: , ,\nПрогресс: invalid, :empty, key:\nUnknown: data", "Готово", models.Context{}},
		{"colon_inside_value", "План" + usecases.SEPARATOR + "Прогресс: время: 18:00, Go: 10%, Go: 20%", "План", models.Context{Progress: map[string]string{"время": "18:00", "Go": "20%"}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			answer, got := usecases.ParseAIResponse(tt.input)
			if answer != tt.answer || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ParseAIResponse() = (%q, %+v), want (%q, %+v)", answer, got, tt.answer, tt.want)
			}
		})
	}
}
