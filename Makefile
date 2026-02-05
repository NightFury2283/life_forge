# mingw32-make file for Go project
include .env
DB_URL=${POSTGRES_DSN}

migrate-up:
	migrate -path ./migrations -database "$(DB_URL)" up
go:
	@go run cmd/life_forge/main.go

#new migration: migrate create -ext sql -dir migrations -seq название_миграции
