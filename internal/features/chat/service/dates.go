package chat_service

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var moscow = time.FixedZone("Europe/Moscow", 3*60*60)
var weekdays = []string{"воскресенье", "понедельник", "вторник", "среда", "четверг", "пятница", "суббота"}
var weekdayPattern = regexp.MustCompile(`(?i)(?:^|[^\p{L}])(понедельник[ауе]?|вторник[ауе]?|сред[ауые]|четверг[ауе]?|пятниц[ауые]|суббот[ауые]|воскресень[еяю])`)
var clockPattern = regexp.MustCompile(`\b(\d{1,2}):(\d{2})(?:\s*[-–—]\s*(\d{1,2}):(\d{2}))?\b`)
var isoDatePattern = regexp.MustCompile(`\b(\d{4}-\d{2}-\d{2})\b`)
var localDatePattern = regexp.MustCompile(`\b(\d{1,2})[./](\d{1,2})(?:[./](\d{4}))?\b`)

// requestedDay resolves only an unambiguous single date. Multiple weekdays or
// explicit dates are left to the model; we must not shift every draft to one day.
func requestedDay(text string, now time.Time) (time.Time, bool) {
	text = strings.ToLower(text)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, moscow)
	if isoDatePattern.MatchString(text) || localDatePattern.MatchString(text) {
		return time.Time{}, false
	}
	for _, relative := range []struct {
		word   string
		offset int
	}{{"послезавтра", 2}, {"завтра", 1}, {"сегодня", 0}} {
		if strings.Contains(text, relative.word) {
			return today.AddDate(0, 0, relative.offset), true
		}
	}
	var target time.Weekday
	found := false
	for _, match := range weekdayPattern.FindAllStringSubmatch(text, -1) {
		var day time.Weekday
		switch {
		case strings.HasPrefix(match[1], "понедельник"):
			day = time.Monday
		case strings.HasPrefix(match[1], "вторник"):
			day = time.Tuesday
		case strings.HasPrefix(match[1], "сред"):
			day = time.Wednesday
		case strings.HasPrefix(match[1], "четверг"):
			day = time.Thursday
		case strings.HasPrefix(match[1], "пятниц"):
			day = time.Friday
		case strings.HasPrefix(match[1], "суббот"):
			day = time.Saturday
		default:
			day = time.Sunday
		}
		if found && target != day {
			return time.Time{}, false
		}
		target = day
		found = true
	}
	if !found {
		return time.Time{}, false
	}
	if strings.Contains(text, "следующей неделе") || strings.Contains(text, "следующую неделю") {
		monday := today.AddDate(0, 0, -(int(today.Weekday())+6)%7)
		return monday.AddDate(0, 0, 7+(int(target)+6)%7), true
	}
	if strings.Contains(text, "этой неделе") {
		monday := today.AddDate(0, 0, -(int(today.Weekday())+6)%7)
		return monday.AddDate(0, 0, (int(target)+6)%7), true
	}
	return today.AddDate(0, 0, (int(target)-int(today.Weekday())+7)%7), true
}

func dateGuide(now time.Time) string {
	var b strings.Builder
	for i := 0; i < 15; i++ {
		day := now.AddDate(0, 0, i)
		fmt.Fprintf(&b, "%s = %s\n", weekdays[day.Weekday()], day.Format("2006-01-02"))
	}
	return b.String()
}

func requestedClock(text string) (hour, minute int, duration float64, ok bool) {
	matches := clockPattern.FindAllStringSubmatch(text, -1)
	if len(matches) != 1 {
		return 0, 0, 0, false
	}
	m := matches[0]
	hour, _ = strconv.Atoi(m[1])
	minute, _ = strconv.Atoi(m[2])
	if hour > 23 || minute > 59 {
		return 0, 0, 0, false
	}
	if m[3] != "" {
		endHour, _ := strconv.Atoi(m[3])
		endMinute, _ := strconv.Atoi(m[4])
		if endHour > 23 || endMinute > 59 {
			return 0, 0, 0, false
		}
		minutes := endHour*60 + endMinute - hour*60 - minute
		if minutes < 0 {
			minutes += 24 * 60
		}
		if minutes == 0 {
			return 0, 0, 0, false
		}
		duration = float64(minutes) / 60
	}
	return hour, minute, duration, true
}
