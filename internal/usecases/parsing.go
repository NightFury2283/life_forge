// parsers.go
package usecases

import (
	"life_forge/internal/models"
	"strings"
)

const SEPARATOR = "|||UPDATE_DATA|||"

func ParseAIResponse(response string) (userAnswer string, updatedContext models.Context) {
	if !strings.Contains(response, SEPARATOR) {
		return strings.TrimSpace(response), models.Context{}
	}

	parts := strings.SplitN(response, SEPARATOR, 3)
	if len(parts) < 2 {
		return strings.TrimSpace(parts[0]), models.Context{}
	}

	// Ответ (часть до первого SEPARATOR)
	userAnswer = strings.TrimSpace(parts[0])
	userAnswer = strings.TrimPrefix(userAnswer, "Ответ:")
	userAnswer = strings.TrimSpace(userAnswer)

	// Обновления (между SEPARATOR)
	updatesText := parts[1]
	parseUpdates(updatesText, &updatedContext)

	return userAnswer, updatedContext
}

func parseUpdates(updatesText string, ctx *models.Context) {
	tempCtx := models.Context{
		Goals:    []string{},
		Recent5:  []string{},
		Progress: make(map[string]string),
	}

	for _, line := range strings.Split(updatesText, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "Цели:"):
			// Цели через запятую
			goalsText := strings.TrimPrefix(line, "Цели:")
			goalsText = strings.TrimSpace(goalsText)
			tempCtx.Goals = parseGoals(goalsText)

		case strings.HasPrefix(line, "Недавние действия:"):
			// Недавние действия через запятую
			recentText := strings.TrimPrefix(line, "Недавние действия:")
			recentText = strings.TrimSpace(recentText)
			tempCtx.Recent5 = parseGoals(recentText)

		case strings.HasPrefix(line, "Прогресс:"):
			// Прогресс в формате "ключ:значение"
			progressText := strings.TrimPrefix(line, "Прогресс:")
			progressText = strings.TrimSpace(progressText)
			progressMap := parseProgress(progressText)

			for k, v := range progressMap {
				tempCtx.Progress[k] = v
			}
		}
	}

	if len(tempCtx.Goals) > 0 {
		ctx.Goals = tempCtx.Goals
	}
	if len(tempCtx.Recent5) > 0 {
		ctx.Recent5 = tempCtx.Recent5
	}
	if len(tempCtx.Progress) > 0 {
		ctx.Progress = tempCtx.Progress
	}
}

func parseGoals(text string) []string {
	var result []string

	items := strings.Split(text, ",")
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func parseProgress(text string) map[string]string {
	result := make(map[string]string)

	items := strings.Split(text, ",")
	for _, item := range items {
		item = strings.TrimSpace(item)

		if idx := strings.Index(item, ":"); idx > 0 {
			key := strings.TrimSpace(item[:idx])
			val := strings.TrimSpace(item[idx+1:])
			if key != "" && val != "" {
				result[key] = val
			}
		}
	}
	return result
}
