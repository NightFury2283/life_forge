package storage

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"life_forge/internal/models"
	"time"
)

type JournalStorage struct {
	pool *pgxpool.Pool
}

func NewJournalStorage(pool *pgxpool.Pool) *JournalStorage {
	return &JournalStorage{
		pool: pool,
	}
}

func (db_js *JournalStorage) CreateEntry(ctx context.Context, entry *models.JournalEntry) error {
	op := "internal/storage/journal.go CreateEntry"

	entry.CreatedAt = time.Now()

	sql_query := `
	INSERT INTO journal_entries
	(entry_text, mood_score, created_at)
	VALUES ($1, $2, $3);
	`

	_, err := db_js.pool.Exec(
		ctx,
		sql_query,
		entry.EntryText,
		entry.MoodScore,
		entry.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("Failure to create entry in %s: %w", op, err)
	}

	return nil
}

func (db_js *JournalStorage) GetEntries(ctx context.Context, limit int) ([]models.JournalEntry, error) {
	op := "internal/storage/journal.go GetEntries"

	sql_query := `
	SELECT * FROM journal_entries
	ORDER BY created_at DESC
	LIMIT $1;
	`

	rows, err := db_js.pool.Query(ctx, sql_query, limit)

	if err != nil {
		return nil, fmt.Errorf("Failure to get entries in %s: %w", op, err)
	}
	defer rows.Close()
	entries := []models.JournalEntry{}

	for rows.Next() {
		entry := models.JournalEntry{}

		err := rows.Scan(
			&entry.ID,
			&entry.EntryText,
			&entry.MoodScore,
			&entry.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("Failure to Scan entries in %s: %w", op, err)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}
