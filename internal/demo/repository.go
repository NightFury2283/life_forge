// Package demo provides volatile storage for an explicitly enabled local demo.
package demo

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/NightFury2283/life_forge/internal/core/domain"
	core_errors "github.com/NightFury2283/life_forge/internal/core/errors"
)

type Repository struct {
	mu         sync.RWMutex
	users      map[string]domain.User
	stats      map[int]domain.Stat
	nextStatID int
}

func NewRepository() *Repository {
	return &Repository{users: make(map[string]domain.User), stats: make(map[int]domain.Stat)}
}

func (r *Repository) GetOrCreateUserByGoogleID(ctx context.Context, googleID, email, name string) (domain.User, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.users[googleID]
	if !ok {
		user = domain.User{ID: len(r.users) + 1, GoogleID: googleID, CreatedAt: time.Now()}
	}
	user.Email, user.Name = email, name
	r.users[googleID] = user
	return user, nil
}

func (r *Repository) GetProfile(ctx context.Context, userID int) (domain.UserProgress, error) {
	if err := ctx.Err(); err != nil {
		return domain.UserProgress{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, user := range r.users {
		if user.ID == userID {
			return domain.UserProgress{UserID: userID, Level: 1, XPForNextLevel: 100}, nil
		}
	}
	return domain.UserProgress{}, core_errors.ErrNotFound
}

func (r *Repository) CreateStat(ctx context.Context, stat domain.Stat) (domain.Stat, error) {
	if err := ctx.Err(); err != nil {
		return domain.Stat{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.stats {
		if existing.UserID == stat.UserID && existing.Title == stat.Title {
			return domain.Stat{}, core_errors.ErrConflict
		}
	}
	r.nextStatID++
	stat.ID = r.nextStatID
	r.stats[stat.ID] = cloneStat(stat)
	return cloneStat(stat), nil
}

func (r *Repository) ListStats(ctx context.Context, userID int) ([]domain.Stat, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	stats := make([]domain.Stat, 0)
	for _, stat := range r.stats {
		if stat.UserID == userID {
			stats = append(stats, cloneStat(stat))
		}
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].ID < stats[j].ID })
	return stats, nil
}

func (r *Repository) DeleteStat(ctx context.Context, userID, statID int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	stat, ok := r.stats[statID]
	if !ok || stat.UserID != userID {
		return core_errors.ErrNotFound
	}
	delete(r.stats, statID)
	return nil
}

func cloneStat(stat domain.Stat) domain.Stat {
	if stat.Description != nil {
		value := *stat.Description
		stat.Description = &value
	}
	return stat
}
