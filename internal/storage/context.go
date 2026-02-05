package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"life_forge/internal/models"
	"log"
)

type ContextStorage struct {
	pool *pgxpool.Pool
}

func NewContextStorage(pool *pgxpool.Pool) *ContextStorage {
	return &ContextStorage{
		pool: pool,
	}
}

func (db_ct *ContextStorage) GetContextByID(ctx context.Context, id int) (models.Context, error) {

	op := "internal/storage/context.go GetContextByID"

	sql_query := `
	SELECT id, goals, recent5, progress FROM Context
	WHERE id = $1
	`

	var context models.Context
	var progressJSON []byte

	err := db_ct.pool.QueryRow(ctx, sql_query, id).Scan(
		&context.ID,
		&context.Goals, //pgx TEXT[] -> []string
		&context.Recent5,
		&progressJSON,
	)

	if err != nil {
		log.Println("Error with QueryRow method in ", op, " with error: ", err)
		return models.Context{}, err
	}

	context.Goals = []string{}
	context.Recent5 = []string{}
	context.Progress = map[string]string{}

	if context.Goals == nil {
		context.Goals = []string{}
	}
	if context.Recent5 == nil {
		context.Recent5 = []string{}
	}

	context.Progress = make(map[string]string)

	if len(progressJSON) > 0 && string(progressJSON) != "null" {
		if err := json.Unmarshal(progressJSON, &context.Progress); err != nil {
			log.Println("Failed to unmarshal json (progressJSON) in ", op, "with error: ", err)
		}
	}

	return context, nil
}

func (db_ct *ContextStorage) SaveContext(ctx context.Context, contextData *models.Context) error {
	op := "internal/storage/context.go SaveContext"

	progressJSON, err := json.Marshal(contextData.Progress)
	if err != nil {
		log.Println("Failed to marshal progress in ", op, "with error: ", err)
		return fmt.Errorf("%s: failed to marshal progress: %w", op, err)
	}

	sql_query := `
	INSERT INTO Context (id, goals, recent5, progress) VALUES ($1, $2, $3, $4)
	ON CONFLICT (id) DO UPDATE SET
	goals = EXCLUDED.goals,
	recent5 = EXCLUDED.recent5,
	progress = EXCLUDED.progress
	`

	_, err = db_ct.pool.Exec(ctx, sql_query,
		contextData.ID,
		contextData.Goals, // pgx []string -> TEXT[]
		contextData.Recent5,
		progressJSON,
	)

	if err != nil {
		log.Println("Error with Exec method in ", op, " with error: ", err)
		return fmt.Errorf("%s: failed to save context data: %w", op, err)
	}

	return nil
}
