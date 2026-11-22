# LifeForge

Планировщик с игровой механикой на Go. Направление развития: Google Calendar, GigaChat, подтверждение планов человеком и система прогресса. Сейчас слоистая версия позволяет войти, открыть профиль, создать, посмотреть и удалить собственные характеристики.

## Быстро попробовать

Нужен **Go 1.26+**. Из корня проекта:

```powershell
go run ./cmd/life_forge -demo
```

Откройте **http://127.0.0.1:8080** и нажмите «Попробовать демо». Можно создавать характеристики, проверять дубликаты, удалять их и выходить. Если порт занят: `go run ./cmd/life_forge -demo -addr 127.0.0.1:8081`.

Демо работает только на loopback-адресе, без PostgreSQL, ключей и внешних сервисов. Это общий локальный аккаунт, **не проверка личности через Google**. JWT и защита API работают; данные и случайный ключ JWT сбрасываются при перезапуске. Обычный режим не регистрирует `/auth/demo`.

## Обычный запуск с Google и PostgreSQL

1. Настройте `.env` по `.env.example`. Если `.env` уже существует, обновите нужные поля, сохранив свои значения. Задайте случайный `JWT_SECRET_KEY` длиной минимум 32 байта. В PowerShell 7: `[Convert]::ToHexString([Security.Cryptography.RandomNumberGenerator]::GetBytes(32))`.
2. Включите Docker Desktop, затем поднимите отдельное окружение и примените миграции:

   ```powershell
   docker compose -f compose.local.yml up -d postgres
   docker compose -f compose.local.yml run --rm migrate
   ```

   Используется отдельный именованный том; прежний `out/pgdata` не затрагивается. Порт — `15432`. Если старое окружение уже занимает порт, сначала остановите его.
3. Поместите OAuth credentials типа **Web application** в `credentials.json`. В Google Cloud добавьте точный Authorized redirect URI `http://localhost:8080/auth/callback`, тот же адрес укажите в `GOOGLE_REDIRECT_URL`. Для consent screen в режиме Testing добавьте свой аккаунт в test users. Используются scopes calendar, userinfo.email и userinfo.profile.
4. Запустите `go run ./cmd/life_forge`, откройте **http://localhost:8080** и выберите «Войти через Google». Используйте один hostname на старте и callback: cookie `localhost` и `127.0.0.1` различаются.

`.env` загружается автоматически, заданные переменные окружения имеют приоритет. Для HTTPS задайте `COOKIE_SECURE=true`. JWT живёт 24 часа. Выход удаляет cookie браузера; серверный отзыв уже выданного JWT пока не реализован.

Если `GOOGLE_REDIRECT_URL` не задан, используется адрес из `credentials.json`. Поддерживаются также прежние `/api/v1/auth/google` и `/api/v1/auth/callback`. Для ранее зарегистрированного `http://localhost:5050/api/v1/auth/callback` задайте `HTTP_ADDR=:5050`, такой же `GOOGLE_REDIRECT_URL` и открывайте сайт по `http://localhost:5050`.

## Аутентификация и авторизация

Google подтверждает личность, backend создаёт или обновляет пользователя по Google ID и выдаёт JWT в `HttpOnly`, `SameSite=Lax` cookie. Случайный OAuth `state` связывает callback с браузером и удаляется при обработке callback. JWT принимается только с HS256, правильным issuer, сроком действия и согласованным положительным ID пользователя. Запросы браузера с другого origin не могут менять данные.

Защищённые маршруты берут `userID` из проверенного JWT. Передать владельца в JSON нельзя. Список характеристик фильтруется по владельцу, удаление использует ID характеристики и владельца. Чужая характеристика возвращает 404. Ролей администратора пока нет.

## API текущей версии

| Метод | Маршрут | Назначение |
| --- | --- | --- |
| GET | `/auth/google` | Начать вход через Google |
| GET | `/auth/callback` | Проверить state и завершить вход |
| POST | `/auth/logout` | Удалить cookie с JWT |
| GET | `/api/status` | Режим приложения и доступность входа |
| GET | `/api/v1/gamification/profile` | Профиль текущего пользователя |
| GET | `/api/v1/gamification/stats` | Его характеристики |
| POST | `/api/v1/gamification/stats` | Создать характеристику |
| DELETE | `/api/v1/gamification/stats/{id}` | Удалить свою характеристику |

Пример создания: `{"title":"Дисциплина","description":"Заниматься регулярно","value":0}`. Описание необязательно. Название — 1–100 Unicode-символов, описание при наличии — 1–1000, значение — неотрицательное число в пределах PostgreSQL INTEGER. Ошибки: 400, 401, 404, 409, 500.

## Тесты

```powershell
go test ./...
go vet ./...
go test ./internal/app -run TestIntegration -v
```

Тестам не нужны `.env`, PostgreSQL, Google, GigaChat или Docker. Сценарии и команды покрытия описаны в [TESTING.md](TESTING.md).

## Структура и границы версии

`internal/features/{auth,users,gamification}`: transport → service → repository. `internal/app` собирает приложение и используется интеграционными тестами. `internal/core` содержит JWT, HTTP, ошибки, логирование и доменные типы. `internal/demo` — хранилище в памяти. `static/app.*` — текущий интерфейс, встроенный в Go-бинарник. `internal/usecases` сохраняет протестированный разбор ответов ИИ.

Удалены старые монолитные обработчики/хранилища, дубли миграций, незавершённые копии пакетов и старый exe. История сохранена в Git. Прежние HTML-макеты оставлены в `static` для дальнейшего переноса; приложение отдаёт только новый интерфейс. Календарь, чат, начисление XP, задания, магазин и досье ещё нужно подключить к новой архитектуре. Они не имитируются в текущем интерфейсе.
