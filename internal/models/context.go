package models

type Context struct {
	ID       int               `json:"id" db:"id"`
	Goals    []string          `json:"goals" db:"goals"`
	Recent5  []string          `json:"recent5" db:"recent5"`
	Progress map[string]string `json:"progress" db:"progress"`
}
