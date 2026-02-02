package models

import(
	"time"
)

type JournalEntry struct {
	ID		int `json:"id" db:"id"`
	EntryText	string `json:"entry_text" db:"entry_text"`
	MoodScore	int `json:"mood_score" db:"mood_score"`
	CreatedAt	time.Time `json:"created_at" db:"created_at"`
}