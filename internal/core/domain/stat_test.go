package domain_test

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/NightFury2283/life_forge/internal/core/domain"
	core_errors "github.com/NightFury2283/life_forge/internal/core/errors"
)

func TestStatValidate(t *testing.T) {
	t.Parallel()
	ptr := func(s string) *string { return &s }
	tests := []struct {
		name          string
		title         string
		description   *string
		value, userID int
		invalid       bool
	}{
		{"minimal", "A", nil, 0, 1, false},
		{"unicode_title_limit", strings.Repeat("я", 100), nil, 1, 1, false},
		{"description_limit", "Сила", ptr(strings.Repeat("界", 1000)), 1, 1, false},
		{"one_character_description", "Сила", ptr("я"), 0, 1, false},
		{"maximum_value", "Сила", nil, math.MaxInt32, 1, false},
		{"empty_title", "", nil, 0, 1, true},
		{"blank_title", " \t", nil, 0, 1, true},
		{"title_too_long", strings.Repeat("я", 101), nil, 0, 1, true},
		{"empty_description", "Сила", ptr(""), 0, 1, true},
		{"description_too_long", "Сила", ptr(strings.Repeat("я", 1001)), 0, 1, true},
		{"negative_value", "Сила", nil, -1, 1, true},
		{"database_integer_overflow", "Сила", nil, math.MaxInt32 + 1, 1, true},
		{"missing_owner", "Сила", nil, 0, 0, true},
		{"negative_owner", "Сила", nil, 0, -1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			stat := domain.NewStatUnitialized(tt.userID, tt.title, tt.description, tt.value)
			err := stat.Validate()
			if tt.invalid {
				if !errors.Is(err, core_errors.ErrInvalidArgument) {
					t.Fatalf("Validate() error = %v, want ErrInvalidArgument", err)
				}
			} else if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}
