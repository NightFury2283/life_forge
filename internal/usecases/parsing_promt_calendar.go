package usecases

import (
	"encoding/json"
	"fmt"
	"life_forge/internal/models"
	"strings"
	"time"
)

const CALENDAR_SEPARATOR = "|||CALENDAR_EVENT|||" // Разделитель ответа ИИ

func ParseCalendarAIResponse(response string) (string, []*models.EventRequest, error) {
	// Если нет разделителя — обычный текст
	if !strings.Contains(response, CALENDAR_SEPARATOR) {
		return strings.TrimSpace(response), nil, nil
	}

	// Разбиваем ответ ИИ на части
	parts := strings.SplitN(response, CALENDAR_SEPARATOR, 3)
	if len(parts) < 2 {
		return strings.TrimSpace(parts[0]), nil, fmt.Errorf("некорректный формат ответа, отправьте запрос ещё раз")
	}

	// Очищаем ответ пользователю
	userAnswer := strings.TrimSpace(parts[0])
	userAnswer = strings.TrimPrefix(userAnswer, "Ответ:")
	userAnswer = strings.TrimSpace(userAnswer)

	// ✅ ИСПРАВЛЕНИЕ: правильно извлекаем JSON без лишнего текста
	jsonText := strings.TrimSpace(parts[1])
	jsonParts := strings.SplitN(jsonText, CALENDAR_SEPARATOR, 2)
	jsonText = strings.TrimSpace(jsonParts[0])
	jsonText = strings.Trim(jsonText, " {}[]\t\n\r")

	if jsonText == "" || jsonText == "{}" {
		return userAnswer, nil, nil // Пустой JSON = нет событий
	}

	fmt.Println("Ответ ИИ перед парсингом Json: \n\n", response)

	// ✅ ИСПРАВЛЕНИЕ: убираем проверку isValidJSON — она ломает null значения
	events, err := parseCalendarJSON(jsonText)
	if err != nil {
		return userAnswer, nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	return userAnswer, events, nil
}

func parseCalendarJSON(jsonText string) ([]*models.EventRequest, error) {
	// Массив или одиночный объект?
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

	// ✅ Проверяем валидность JSON массива
	if err := json.Unmarshal([]byte(jsonText), &tempEvents); err != nil {
		return nil, fmt.Errorf("неверный JSON массив: %w", err)
	}

	var events []*models.EventRequest
	for _, tempEvent := range tempEvents {
		if !tempEvent.IsEvent {
			continue // Пропускаем не-события
		}

		event := &models.EventRequest{
			IsEvent:       tempEvent.IsEvent,
			Title:         tempEvent.Title,
			DurationHours: tempEvent.Duration,
			Recurrence:    tempEvent.Recurrence,
			Description:   tempEvent.Description,
		}

		// ✅ Безопасный парсинг времени
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

	// ✅ Проверяем валидность одиночного JSON
	if err := json.Unmarshal([]byte(jsonText), &tempEvent); err != nil {
		return nil, fmt.Errorf("неверный JSON объект: %w \n%s", err, jsonText)
	}

	if !tempEvent.IsEvent {
		return nil, nil // Не событие
	}

	event := &models.EventRequest{
		IsEvent:       tempEvent.IsEvent,
		Title:         tempEvent.Title,
		DurationHours: tempEvent.Duration,
		Recurrence:    tempEvent.Recurrence,
		Description:   tempEvent.Description,
	}

	// ✅ Безопасный парсинг времени
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
