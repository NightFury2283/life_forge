# LifeForge

Go 1.26+, PostgreSQL, Google Calendar и GigaChat. Чат предлагает черновики событий, пользователь подтверждает запись. Есть профиль героя, награды за задачи и две темы оформления. Слои: transport → service → repository.

## Запуск

Без внешних сервисов: `go run ./cmd/life_forge -demo`, затем http://127.0.0.1:8080.

Для обычного режима:
1. Настройте `.env` по `.env.example`: PostgreSQL, случайный `JWT_SECRET_KEY` от 32 байт, Google OAuth credentials и `GIGACHAT_AUTH_KEY`.
2. Поднимите БД: `docker compose -f compose.local.yml up -d postgres`.
3. Примените миграции: `docker compose -f compose.local.yml run --rm migrate`. Для Windows без bind mount есть `scripts/migrate.ps1`.
4. Запустите `go run ./cmd/life_forge`.

Порт локальной БД — 15432. Адрес callback должен точно совпадать в Google Cloud и `GOOGLE_REDIRECT_URL`. Для текущего окружения: `HTTP_ADDR=:5050`, callback `http://localhost:5050/api/v1/auth/callback`, сайт http://localhost:5050. Для существующего Docker-окружения сохраните port-forwarder и тома.

Google-токены хранятся зашифрованными отдельно для каждого пользователя. Не меняйте JWT-ключ без учёта сохранённых токенов. Для HTTPS установите `COOKIE_SECURE=true`. `.env`, credentials и токены не коммитить.

Даты планирования — Europe/Moscow. Повторное подтверждение события или завершение задачи не должно создавать дубликаты и повторные награды.

Проверки и границы тестов: [TESTING.md](TESTING.md).
