package usecases

import (
	"encoding/json"
	"fmt"
	"life_forge/internal/models"
	"strings"
	"time"
)

const CALENDAR_SEPARATOR = "|||CALENDAR_EVENT|||"

func ParseCalendarAIResponse(response string) (string, []*models.EventRequest, error) {
	logParse("Parsing...")

	cleanedResponse := cleanResponse(response)

	if !strings.Contains(cleanedResponse, CALENDAR_SEPARATOR) {
		text := strings.TrimSpace(cleanedResponse)
		logParse("No separator, return text: %s", text)
		return text, nil, nil
	}

	// split
	parts := strings.SplitN(cleanedResponse, CALENDAR_SEPARATOR, 3)
	if len(parts) < 2 {
		text := strings.TrimSpace(cleanedResponse)
		logParse("Not enough parts, return text: %s", text)
		return text, nil, nil
	}

	userAnswer := strings.TrimSpace(parts[0])

	jsonPart := strings.TrimSpace(parts[1])

	// clean json
	jsonPart = strings.ReplaceAll(jsonPart, CALENDAR_SEPARATOR, "")
	jsonPart = strings.TrimSpace(jsonPart)

	if jsonPart == "" || jsonPart == "{}" || jsonPart == "[]" {
		logParse("Empty json")
		return userAnswer, nil, nil
	}

	events, err := parseCalendarJSON(jsonPart)
	if err != nil {
		logParse("Error parsing JSON: %v", err)
		return userAnswer, nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	logParse("Succesfully parsed %d events", len(events))
	return userAnswer, events, nil
}

func cleanResponse(response string) string {
	lines := strings.Split(response, "\n")
	var cleanedLines []string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if i == 0 && (strings.HasPrefix(trimmed, "Ответ ИИ") ||
			strings.HasPrefix(trimmed, "Ответ:") ||
			strings.HasPrefix(trimmed, "Answer:")) {
			continue
		}
		if trimmed != "" {
			cleanedLines = append(cleanedLines, trimmed)
		}
	}

	return strings.Join(cleanedLines, "\n")
}

func parseCalendarJSON(jsonText string) ([]*models.EventRequest, error) {
	if strings.HasPrefix(jsonText, "[") {
		return parseJSONArray(jsonText)
	}
	return parseSingleEvent(jsonText)
}

func parseJSONArray(jsonText string) ([]*models.EventRequest, error) {
	var tempEvents []struct {
		IsEvent     bool     `json:"is_event"`
		Title       string   `json:"title"`
		StartTime   *string  `json:"start_time"`
		Duration    *float64 `json:"duration"`
		Recurrence  *string  `json:"recurrence"`
		Description *string  `json:"description"`
	}

	if err := json.Unmarshal([]byte(jsonText), &tempEvents); err != nil {
		logParse("error parsing array: %v", err)
		return nil, fmt.Errorf("wrong JSON array: %w", err)
	}

	var events []*models.EventRequest
	for i, tempEvent := range tempEvents {
		if !tempEvent.IsEvent {
			logParse("Event %d skiped (is_event=false)", i+1)
			continue
		}

		event, err := createEventFromTemp(tempEvent)
		if err != nil {
			logParse("Error create event %d: %v", i+1, err)
			return nil, err
		}

		events = append(events, event)
		logParse("Event %d created: %s", i+1, event.Title)
	}

	return events, nil
}

func parseSingleEvent(jsonText string) ([]*models.EventRequest, error) {
	var tempEvent struct {
		IsEvent     bool     `json:"is_event"`
		Title       string   `json:"title"`
		StartTime   *string  `json:"start_time"`
		Duration    *float64 `json:"duration"`
		Recurrence  *string  `json:"recurrence"`
		Description *string  `json:"description"`
	}

	if err := json.Unmarshal([]byte(jsonText), &tempEvent); err != nil {
		logParse("Error parsing object: %v", err)
		return nil, fmt.Errorf("wrong json object: %w \n%s", err, jsonText)
	}

	if !tempEvent.IsEvent {
		logParse("Event skiped (is_event=false)")
		return nil, nil
	}

	event, err := createEventFromTemp(tempEvent)
	if err != nil {
		logParse("Error create event: %v", err)
		return nil, err
	}

	logParse("Event created: %s", event.Title)
	return []*models.EventRequest{event}, nil
}

func createEventFromTemp(tempEvent struct {
	IsEvent     bool     `json:"is_event"`
	Title       string   `json:"title"`
	StartTime   *string  `json:"start_time"`
	Duration    *float64 `json:"duration"`
	Recurrence  *string  `json:"recurrence"`
	Description *string  `json:"description"`
}) (*models.EventRequest, error) {

	event := &models.EventRequest{
		IsEvent:       tempEvent.IsEvent,
		Title:         tempEvent.Title,
		DurationHours: tempEvent.Duration,
		Recurrence:    tempEvent.Recurrence,
		Description:   tempEvent.Description,
	}

	//time parsing
	if tempEvent.StartTime != nil && *tempEvent.StartTime != "" {
		parsedTime, err := time.Parse(time.RFC3339, *tempEvent.StartTime)
		if err != nil {
			parsedTime, err = time.Parse("2006-01-02T15:04:05Z", *tempEvent.StartTime)
			if err != nil {
				logParse("wrong time format: %s", *tempEvent.StartTime)
				return nil, fmt.Errorf("wrong time format: %s", *tempEvent.StartTime)
			}
		}
		event.StartTime = &parsedTime
		logParse("Event startTime: %v", event.StartTime)
	}

	return event, nil
}

func logParse(format string, args ...interface{}) {
	fmt.Printf("[PARSE] "+format+"\n", args...)
}
