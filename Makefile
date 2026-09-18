# mingw32-make file for Go project
include .env
DB_URL=${POSTGRES_DSN}

export

env-up:
	@docker compose up -d life-forge-postgres
env-down:
	@docker compose down life-forge-postgres

env-cleanup:
	@read -p "Очистить все volume файлы окружения? Опасно!!! [y/N]: " ans; \
	if [ "$$ans" = "y" ]; then \
		docker compose down -v life-forge-postgres && \
		echo "Файлы окружения очищены"; \
	else \
		echo "Очистка окружения отменена"; \
	fi

env-volume-clean:
	@read -p "Удалить volume-данные Postgres? Опасно!!! [y/N]: " ans; \
	if [ "$$ans" = "y" ]; then \
		docker compose down life-forge-postgres && \
		rm -rf out/pgdata && \
		docker volume rm life_forge_life_forge_pgdata 2>/dev/null || true && \
		echo "Volume-данные Postgres очищены"; \
	else \
		echo "Очистка volume отменена"; \
	fi

env-port-forward:
	@docker compose up -d port-forwarder

env-port-close:
	@docker compose down port-forwarder

migrate-create:
	@if [ -z "$(seq)" ]; then \
		echo "Отсутствует seq (название миграции). Пример seq=init"; \
		exit 1; \
	fi; \
	docker compose run --rm life-forge-postgres-migrate \
		create \
		-ext sql \
		-dir /migrations \
		-seq "$(seq)"

migrate-up:
	@make migrate-action action=up

migrate-down:
	@make migrate-action action=down

migrate-action:
	@if [ -z "$(action)" ]; then \
		echo "Отсутствует параметр action (действие). Пример action=down 1"; \
		exit 1; \
	fi; \
	echo "Ожидание готовности PostgreSQL..."; \
	until docker compose exec -T life-forge-postgres pg_isready -U ${POSTGRES_USER}; do \
		echo "БД ещё не готова, ждём..."; \
		sleep 2; \
	done; \
	echo "Запуск миграций..."; \
	docker compose run --rm life-forge-postgres-migrate \
		-path /migrations \
		-database postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@life-forge-postgres:5432/${POSTGRES_DB}?sslmode=disable \
		"$(action)"

life-forge-run:
	@export LOGGER_FOLDER=${PROJECT_ROOT}/out/logs && \
	go mod tidy && \
	go run cmd/life_forge/main.go