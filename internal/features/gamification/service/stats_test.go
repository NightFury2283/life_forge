package gamification_service_test

import (
	"context"
	"errors"
	"github.com/NightFury2283/life_forge/internal/core/domain"
	core_errors "github.com/NightFury2283/life_forge/internal/core/errors"
	stats "github.com/NightFury2283/life_forge/internal/features/gamification/service"
	"reflect"
	"testing"
)

type repositoryStub struct {
	adjust func(context.Context, int, int, int) (domain.Stat, error)
	create func(context.Context, domain.Stat) (domain.Stat, error)
	list   func(context.Context, int) ([]domain.Stat, error)
	remove func(context.Context, int, int) error
}

func (r repositoryStub) AdjustStat(c context.Context, u, id, d int) (domain.Stat, error) {
	return r.adjust(c, u, id, d)
}

func (r repositoryStub) CreateStat(ctx context.Context, s domain.Stat) (domain.Stat, error) {
	return r.create(ctx, s)
}
func (r repositoryStub) ListStats(ctx context.Context, id int) ([]domain.Stat, error) {
	return r.list(ctx, id)
}
func (r repositoryStub) DeleteStat(ctx context.Context, userID, statID int) error {
	return r.remove(ctx, userID, statID)
}

func TestCreateStat(t *testing.T) {
	t.Parallel()
	cause := errors.New("database unavailable")
	for _, tt := range []struct {
		name, title      string
		repoErr, wantErr error
		wantCalls        int
	}{
		{"success", "Сила", nil, nil, 1}, {"empty_title", "", nil, core_errors.ErrInvalidArgument, 0},
		{"repository_error", "Сила", cause, cause, 1}, {"duplicate", "Сила", core_errors.ErrConflict, core_errors.ErrConflict, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			calls := 0
			input := domain.NewStatUnitialized(7, tt.title, nil, 10)
			want := input
			want.ID = 42
			repo := repositoryStub{create: func(got context.Context, s domain.Stat) (domain.Stat, error) {
				calls++
				if got != ctx || !reflect.DeepEqual(s, input) {
					t.Fatal("unexpected repository arguments")
				}
				return want, tt.repoErr
			}}
			got, err := stats.NewGamificationService(repo).CreateStat(ctx, input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) || got != (domain.Stat{}) {
					t.Fatalf("CreateStat() = (%+v, %v)", got, err)
				}
			} else if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("CreateStat() = (%+v, %v), want %+v", got, err, want)
			}
			if calls != tt.wantCalls {
				t.Errorf("repository calls = %d, want %d", calls, tt.wantCalls)
			}
		})
	}
}

func TestDeleteStat(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name             string
		userID, statID   int
		repoErr, wantErr error
		calls            int
	}{
		{"success", 7, 42, nil, nil, 1}, {"foreign_or_missing", 7, 42, core_errors.ErrNotFound, core_errors.ErrNotFound, 1},
		{"invalid_stat", 7, 0, nil, core_errors.ErrInvalidArgument, 0}, {"invalid_user", 0, 42, nil, core_errors.ErrInvalidArgument, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			calls := 0
			ctx := t.Context()
			repo := repositoryStub{remove: func(got context.Context, userID, statID int) error {
				calls++
				if got != ctx || userID != tt.userID || statID != tt.statID {
					t.Fatal("unexpected delete arguments")
				}
				return tt.repoErr
			}}
			err := stats.NewGamificationService(repo).DeleteStat(ctx, tt.userID, tt.statID)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("DeleteStat() = %v, want %v", err, tt.wantErr)
			}
			if calls != tt.calls {
				t.Errorf("calls = %d, want %d", calls, tt.calls)
			}
		})
	}
}

func TestListStats(t *testing.T) {
	t.Parallel()
	cause := errors.New("database unavailable")
	for _, tt := range []struct {
		name string
		err  error
	}{{"success", nil}, {"repository_error", cause}} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			want := []domain.Stat{{ID: 42, UserID: 7, Title: "Сила", Value: 10}}
			repo := repositoryStub{list: func(got context.Context, id int) ([]domain.Stat, error) {
				if got != ctx || id != 7 {
					t.Fatal("unexpected list arguments")
				}
				return want, tt.err
			}}
			got, err := stats.NewGamificationService(repo).ListStats(ctx, 7)
			if tt.err != nil {
				if !errors.Is(err, cause) || got != nil {
					t.Fatalf("ListStats() = (%v, %v)", got, err)
				}
			} else if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("ListStats() = (%v, %v)", got, err)
			}
		})
	}
}
