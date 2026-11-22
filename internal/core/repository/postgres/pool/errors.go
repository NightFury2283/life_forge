package core_postgres_pool

import "errors"

var (
	ErrNoRows = errors.New("No rows")
	ErrViolatesForeignKey = errors.New("Violates foreign key")
	ErrUnknown = errors.New("Unknown error")
)