package storage

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CalendarStorage struct {
	pool *pgxpool.Pool
}

func NewCalendarStorage(pool *pgxpool.Pool) *CalendarStorage {
	return &CalendarStorage{
		pool: pool,
	}
}

func (db_ct *ContextStorage) GetCalendarPreview(ctx context.Context) string {
	return "пока пусто"
}
