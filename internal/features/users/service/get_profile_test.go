package users_service_test

import (
	"context"
	"errors"
	"github.com/NightFury2283/life_forge/internal/core/domain"
	users "github.com/NightFury2283/life_forge/internal/features/users/service"
	"testing"
)

type profileRepoFunc func(context.Context, int) (domain.UserProgress, error)

func (f profileRepoFunc) GetProfile(ctx context.Context, id int) (domain.UserProgress, error) {
	return f(ctx, id)
}

func TestGetProfile(t *testing.T) {
	t.Parallel()
	cause := errors.New("repository failed")
	for _, tt := range []struct {
		name string
		err  error
	}{{"success", nil}, {"repository_error", cause}} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			want := domain.UserProgress{UserID: 42, Level: 3, XP: 25, XPForNextLevel: 200, Coins: 10}
			calls := 0
			repo := profileRepoFunc(func(got context.Context, id int) (domain.UserProgress, error) {
				calls++
				if got != ctx || id != 42 {
					t.Fatal("unexpected repository arguments")
				}
				return want, tt.err
			})
			got, err := users.NewUsersService(repo).GetProfile(ctx, 42)
			if tt.err != nil {
				if !errors.Is(err, cause) || got != (domain.UserProgress{}) {
					t.Fatalf("GetProfile() = (%+v, %v)", got, err)
				}
			} else if err != nil || got != want {
				t.Fatalf("GetProfile() = (%+v, %v), want %+v", got, err, want)
			}
			if calls != 1 {
				t.Errorf("repository calls = %d", calls)
			}
		})
	}
}
