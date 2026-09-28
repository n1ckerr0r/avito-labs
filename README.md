# Trip service

HTTP-сервис поездок для лабораторной работы 1. Контракт находится в
`contracts/openapi/trip-service.openapi.yaml`, а сгенерированный из него код —
в `api/api.gen.go`.

## Запуск

1. Поднимите окружение курса и подключитесь к нему: `tripgoctl cluster start`,
   `tripgoctl environment start`, `tripgoctl connect`.
2. Скопируйте `.env.example` в `.env` (или используйте созданный `tripgoctl`),
   экспортируйте переменные: `set -a; source .env; set +a`.
3. Накатите схему: `make migrate`.
4. Запустите сервис: `make run`.

Проверка: `curl -i localhost:8080/health`; `/ready` дополнительно проверяет
соединение с PostgreSQL. Сборка и статические проверки: `make build`,
`make test`, `go vet ./...`.

## Конфигурация

Обязательны `HTTP_ADDR`, `DATABASE_URL`, `LOG_LEVEL`, `SHUTDOWN_TIMEOUT`,
`DATABASE_MAX_CONNS`, `DATABASE_MIN_CONNS`, `DATABASE_CONNECT_TIMEOUT`,
`DATABASE_QUERY_TIMEOUT`, `DATABASE_MAX_CONN_LIFETIME`. Значения и примеры — в
`.env.example`.

Необязательные HTTP-таймауты: `HTTP_READ_TIMEOUT` (по умолчанию `5s`),
`HTTP_READ_HEADER_TIMEOUT` (`5s`), `HTTP_WRITE_TIMEOUT` (`10s`) и
`HTTP_IDLE_TIMEOUT` (`1m`).

## Решения

Создание поездки выполняется в одной транзакции: сначала добавляется запись в
`trips`, затем — начальная запись в `trip_status_history`. Транзакционный
менеджер помещает `pgx.Tx` в `context`; репозиторий сам выбирает транзакцию из
контекста или пул для обычных чтений. Вложенный `Do` использует уже открытую
транзакцию. При ошибке либо панике выполняется rollback.

Используется изоляция PostgreSQL `Read Committed`: для этой задачи она
достаточна, поскольку запрет второй активной поездки закреплён частичным
уникальным индексом `trips_active_driver_idx`, а завершение сделано единым
условным `UPDATE ... WHERE status = 'active'`. Поэтому конкурентные запросы
создания дают один успех и `driver_busy`, а конкурентные finish — один успех и
`trip_completed` без перезаписи времени.

`/health` не обращается к БД и показывает, что процесс жив. `/ready` делает
`Ping` с таймаутом запроса и возвращает `503`, пока БД недоступна. Сервис
останавливается по `SIGINT`/`SIGTERM` через `http.Server.Shutdown`, ожидая
текущие запросы не дольше `SHUTDOWN_TIMEOUT`.

### Идемпотентность

`POST /api/v1/trips` поддерживает необязательный UUID-заголовок
`Idempotency-Key`. Ключ и SHA-256 нормализованного тела резервируются в той же
транзакции, что и поездка. Повтор с тем же телом возвращает исходную поездку с
`200`; другое тело — `409 idempotency_conflict`. Первичный ключ таблицы делает
поведение безопасным при параллельных запросах. Записи хранятся 24 часа и
удаляются при следующем запросе с ключом.

## Docker

`deploy/Dockerfile` — многостадийная сборка; итоговый образ distroless,
запускается пользователем `nonroot` и содержит только статический бинарный
файл. Проверенный локально образ `trip-service-lab:local` занимает **13.1 MB**
(linux/amd64); размер может немного отличаться на другой платформе.
