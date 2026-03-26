package handlers

import (
	"encoding/json"
	"life_forge/internal/storage"
	"log"
	"net/http"
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
}

func NewCalendarHandler(cs *storage.GoogleCalendarStorage) *CalendarHandler {
	return &CalendarHandler{calendarStorage: cs}
}

func (cal *CalendarHandler) HandleGanttDiagramm(w http.ResponseWriter, r *http.Request) {
	op := "internal/handlers/calendar.go HandleGranttDiagramm"

	eventsArr, err := cal.calendarStorage.ListEvents(r.Context(), daysToShowTasks)

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
				end, err := time.Parse(time.RFC3339, val.End.DateTime)
				if err != nil {
					log.Printf("Failed to parse time in event in %s with err: %v", op, err)
					continue
				}

				endDay := normalizeToDay(end)
				resEvents <- GanttTask{
					Name:      val.Summary,
					StartTime: time.Now(),
					EndTime:   endDay,
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
