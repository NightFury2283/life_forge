package domain

type Completion struct {
	EventID    string `json:"event_id"`
	CalendarID string `json:"calendar_id"`
	Title      string `json:"title"`
	XP         int    `json:"xp"`
	Coins      int    `json:"coins"`
}
type ActivityDay struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}
type HeroSummary struct {
	Completed int           `json:"completed"`
	EarnedXP  int           `json:"earned_xp"`
	Activity  []ActivityDay `json:"activity"`
}
type HeroTask struct {
	CalendarEvent
	Completed bool `json:"completed"`
}
type HeroDashboard struct {
	Summary HeroSummary `json:"summary"`
	Tasks   []HeroTask  `json:"tasks"`
	Warning string      `json:"warning,omitempty"`
}
