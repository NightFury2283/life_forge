package domain

import (
	"fmt"
	"math"
	"strings"

	core_errors "github.com/NightFury2283/life_forge/internal/core/errors"
)

type Stat struct {
	ID          int
	UserID      int
	Title       string
	Description *string
	Value       int
}

func NewStat(
	id int,
	user_id int,
	title string,
	description *string,
	value int,
) Stat {
	return Stat{
		ID:          id,
		UserID:      user_id,
		Title:       title,
		Description: description,
		Value:       value,
	}
}

func NewStatUnitialized(
	user_id int,
	title string,
	description *string,
	value int,
) Stat {
	return NewStat(
		UnitializedID,
		user_id,
		title,
		description,
		value,
	)
}

func (s *Stat) Validate() error {
	if s.UserID <= 0 {
		return fmt.Errorf("invalid user ID: %w", core_errors.ErrInvalidArgument)
	}
	titleLen := len([]rune(s.Title))

	if titleLen < 1 || titleLen > 100 || strings.TrimSpace(s.Title) == "" {
		return fmt.Errorf("Invalid title len: %d. %w", titleLen, core_errors.ErrInvalidArgument)
	}

	if s.Description != nil {
		descriptionLen := len([]rune(*s.Description))

		if descriptionLen < 1 || descriptionLen > 1000 {
			return fmt.Errorf("Invalid description len: %d. %w", descriptionLen, core_errors.ErrInvalidArgument)
		}
	}

	if s.Value < 0 || s.Value > math.MaxInt32 {
		return fmt.Errorf("Invalid value of Value: %d. %w", s.Value, core_errors.ErrInvalidArgument)
	}

	return nil
}
