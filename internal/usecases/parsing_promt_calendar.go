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
	if !strings.Contains(response, CALENDAR_SEPARATOR) {
		return strings.TrimSpace(response), nil, nil
	}

	parts := strings.SplitN(response, CALENDAR_SEPARATOR, 3)
	if len(parts) < 2 {
		return strings.TrimSpace(parts[0]), nil, fmt.Errorf("некорректный формат ответа, отправьте запрос ещё раз")
	}

	// answer to user
	userAnswer := strings.TrimSpace(parts[0])
	userAnswer = strings.TrimPrefix(userAnswer, "Ответ:")
	userAnswer = strings.TrimSpace(userAnswer)

	// events
	jsonText := strings.TrimSpace(parts[1])

	jsonText = strings.Split(jsonText, CALENDAR_SEPARATOR)[0]
	jsonText = strings.TrimSpace(jsonText)

	events, err := parseCalendarJSON(jsonText)
	if err != nil {
		return userAnswer, nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	return userAnswer, events, nil
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
		return nil, err
	}

	var events []*models.EventRequest
	for _, tempEvent := range tempEvents {
		if !tempEvent.IsEvent {
			continue
		}

		event := &models.EventRequest{
			IsEvent:       tempEvent.IsEvent,
			Title:         tempEvent.Title,
			DurationHours: tempEvent.Duration,
			Recurrence:    tempEvent.Recurrence,
			Description:   tempEvent.Description,
		}

		if tempEvent.StartTime != nil && *tempEvent.StartTime != "" {
			parsedTime, err := time.Parse(time.RFC3339, *tempEvent.StartTime)
			if err != nil {
				parsedTime, err = time.Parse("2006-01-02T15:04:05Z", *tempEvent.StartTime)
				if err != nil {
					return nil, fmt.Errorf("неверный формат времени: %s", *tempEvent.StartTime)
				}
			}
			event.StartTime = &parsedTime
		}

		events = append(events, event)
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
		return nil, err
	}

	if !tempEvent.IsEvent {
		return nil, nil
	}

	event := &models.EventRequest{
		IsEvent:       tempEvent.IsEvent,
		Title:         tempEvent.Title,
		DurationHours: tempEvent.Duration,
		Recurrence:    tempEvent.Recurrence,
		Description:   tempEvent.Description,
	}

	if tempEvent.StartTime != nil && *tempEvent.StartTime != "" {
		parsedTime, err := time.Parse(time.RFC3339, *tempEvent.StartTime)
		if err != nil {
			parsedTime, err = time.Parse("2006-01-02T15:04:05Z", *tempEvent.StartTime)
			if err != nil {
				return nil, fmt.Errorf("неверный формат времени: %s", *tempEvent.StartTime)
			}
		}
		event.StartTime = &parsedTime
	}

	return []*models.EventRequest{event}, nil
}
