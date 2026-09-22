# RSS Aggregator

Сервис собирает публикации из RSS-лент в PostgreSQL и отдаёт объединённую ленту через HTTP API.

## Возможности

- добавление и получение RSS-источников;
- фоновая загрузка публикаций сразу после запуска и каждые 10 минут;
- ограниченный пул из 10 воркеров;
- защита от повторного сохранения публикаций;
- таймауты исходящих запросов и graceful shutdown;
- блокировка исходящих запросов в приватные и локальные сети.

## Запуск

Требования: Go 1.25 или новее и Docker Compose.

```bash
docker compose up -d
cp .env.example .env
go run ./scraper
```

Сервер будет доступен на `http://localhost:8080`.

## API

Добавить RSS-источник:

```bash
curl -X POST http://localhost:8080/api/feeds \
  -H 'Content-Type: application/json' \
  -d '{"name":"Hacker News","url":"https://hnrss.org/frontpage"}'
```

Получить источники:

```bash
curl http://localhost:8080/api/feeds
```

Получить последние публикации:

```bash
curl 'http://localhost:8080/api/posts?limit=20'
```

`limit` по умолчанию равен 50. Допустимое максимальное значение — 100.

## Проверка

```bash
go test ./...
go test -race ./...
go vet ./...
```

## Конфигурация

Приложение читает строку подключения из обязательной переменной `DATABASE_URL`. Локально переменную можно хранить в `.env`; этот файл исключён из Git.
