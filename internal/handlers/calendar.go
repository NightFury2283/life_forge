package handlers

import (
	"encoding/json"
	"life_forge/internal/storage"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"google.golang.org/api/calendar/v3"
)

const (
	daysToShowTasks = 14 // 2 недели
	workers         = 7
)

type CalendarHandler struct {
	calendarStorage *storage.GoogleCalendarStorage
}

type GanttTask struct {
	Name      string    `json:"name"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	IsAllDay  bool      `json:"is_all_day"`
}

func NewCalendarHandler(cs *storage.GoogleCalendarStorage) *CalendarHandler {
	return &CalendarHandler{calendarStorage: cs}
}

func (cal *CalendarHandler) HandleGanttDiagramm(w http.ResponseWriter, r *http.Request) {
	op := "internal/handlers/calendar.go HandleGranttDiagramm"

	timeMin := time.Now().UTC()
	timeMax := time.Now().AddDate(0, 0, daysToShowTasks).UTC()

	if startStr := r.URL.Query().Get("start"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			timeMin = t
		}
	}
	if endStr := r.URL.Query().Get("end"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			timeMax = t
		}
	}

	var calIDs []string
	if cals := r.URL.Query().Get("calendars"); cals != "" {
		calIDs = strings.Split(cals, ",")
	}

	eventsArr, err := cal.calendarStorage.ListEvents(r.Context(), timeMin, timeMax, calIDs...)

	if err != nil {
		http.Error(w, "Failed to load user events: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Failed to load user events in %s with err: %v", op, err)
		return
	}

	eventsChan := make(chan *calendar.Event, len(eventsArr))

	var wgWorkers sync.WaitGroup

	go func() {
		defer close(eventsChan)

		for i := 0; i < len(eventsArr); i++ {
			eventsChan <- eventsArr[i]
		}
	}()

	resEvents := make(chan GanttTask, len(eventsArr))

	for range workers {
		wgWorkers.Add(1)
		go func() {
			defer wgWorkers.Done()

			for val := range eventsChan {
				var start, end time.Time
				var err error
				isAllDay := false

				if val.Start != nil && val.Start.DateTime != "" {
					start, err = time.Parse(time.RFC3339, val.Start.DateTime)
				} else if val.Start != nil && val.Start.Date != "" {
					start, err = time.Parse("2006-01-02", val.Start.Date)
					isAllDay = true
				}
				if err != nil {
					log.Printf("Failed to parse start time in event '%s': %v", val.Summary, err)
					continue
				}

				if val.End != nil && val.End.DateTime != "" {
					end, err = time.Parse(time.RFC3339, val.End.DateTime)
				} else if val.End != nil && val.End.Date != "" {
					end, err = time.Parse("2006-01-02", val.End.Date)
					isAllDay = true
				}
				if err != nil {
					log.Printf("Failed to parse end time in event '%s': %v", val.Summary, err)
					continue
				}

				resEvents <- GanttTask{
					Name:      val.Summary,
					StartTime: start,
					EndTime:   end,
					IsAllDay:  isAllDay,
				}
			}
		}()
	}

	var wg sync.WaitGroup
	w.Header().Set("Content-Type", "application/json")

	events := make([]GanttTask, 0, len(eventsArr))
	wg.Add(1)
	go func() {
		defer wg.Done()
		for val := range resEvents {
			events = append(events, val)
		}
	}()

	wgWorkers.Wait()
	close(resEvents)

	wg.Wait()
	err = json.NewEncoder(w).Encode(events)
	if err != nil {
		http.Error(w, "Failed to encode events: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Failed to encode events in %s with err: %v", op, err)
		return
	}
}

func normalizeToDay(t time.Time) time.Time {
	return time.Date(
		t.Year(),
		t.Month(),
		t.Day(),
		0, 0, 0, 0,
		t.Location(),
	)
}

func (cal *CalendarHandler) HandleGetCalendars(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	calendars, err := cal.calendarStorage.GetUserCalendars(r.Context())
	if err != nil {
		http.Error(w, "Failed to load calendars: "+err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(calendars)
}
