# Workmate API

API для управления задачами с асинхронной обработкой и REST интерфейсом.

## Описание проекта

Workmate — это веб-API для управления задачами, построенный на Go с использованием фреймворка Gin. Проект реализует асинхронную обработку задач с возможностью создания, получения, удаления и отслеживания статуса выполнения. API включает Swagger документацию для удобного тестирования и интеграции.

## Архитектура

Проект следует принципам Clean Architecture с четким разделением слоев:

- cmd/ — точка входа приложения
- internal/app/ — конфигурация приложения и dependency injection
- internal/controllers/ — HTTP контроллеры и роутинг
- internal/service/ — бизнес-логика
- internal/repository/ — слой данных (in-memory хранилище)
- internal/models/ — модели данных
- docs/ — Swagger документация
- tests/e2e/ — end-to-end тесты

## Требования

- Go 1.24 или выше
- Docker (опционально)

## Установка и запуск

### Локальный запуск

1. Клонируйте репозиторий:
```bash
git clone https://github.com/nzb3/workmate_test
cd workmate_test
```

2. Установите зависимости:
```bash
go mod download
```

3. Запустите приложение:
```bash
go run cmd/main.go
```

Сервер будет доступен по адресу http://localhost:8080.

### Запуск с Docker

#### Production режим
```bash
docker build --target release -t workmate:latest .
docker run -p 8080:8080 workmate:latest
```

#### Debug режим (с отладчиком)
```bash
docker build --target debug -t workmate:debug .
docker run -p 8080:8080 -p 40000:40000 workmate:debug
```

В debug режиме доступен отладчик Delve на порту 40000.

## API Endpoints

### Задачи

- POST /api/v1/task/create — Создание новой задачи
- GET /api/v1/task/{id} — Получение информации о задаче
- DELETE /api/v1/task/{id} — Удаление задачи
- GET /api/v1/tasks — Получение списка всех задач

### Служебные

- GET /api/v1/health — Проверка работоспособности сервиса
- GET /api/v1/swagger/* — Swagger документация

## Примеры использования

### Создание задачи
```bash
curl -X POST http://localhost:8080/api/v1/task/create \
  -H "Content-Type: application/json" \
  -d '{"name": "Моя задача"}'
```

### Получение информации о задаче
```bash
curl http://localhost:8080/api/v1/task/{task-id}
```

### Получение списка задач
```bash
curl http://localhost:8080/api/v1/tasks
```

## Модель данных

### Task (Задача)
- id (UUID) — уникальный идентификатор
- name (string) — название задачи  
- status (string) — статус: PROCESSING, DONE, FAILED
- created_at (timestamp) — время создания
- processing_time (duration) — время обработки

## Особенности работы

### Асинхронная обработка
Задачи обрабатываются асинхронно с имитацией реальной работы продолжительностью 3-6 минут. Во время обработки можно отслеживать прогресс через API.

### Статусы задач
- PROCESSING — задача выполняется
- DONE — задача успешно завершена 
- FAILED — задача завершилась с ошибкой

### Тайм-аут
Задачи автоматически отменяются через 6 минут если не завершились.

## Swagger документация

После запуска приложения Swagger UI доступен по адресу:
http://localhost:8080/api/v1/swagger/index.html

## Тестирование

### Запуск тестов
```bash
# Unit тесты
go test ./...

# E2E тесты
go test ./tests/e2e/...
```

## Разработка

### Генерация Swagger документации
```bash
swag init -g cmd/main.go -o docs/
```

### Структура проекта
Проект использует принципы dependency injection через DIContainer. Все зависимости инициализируются в internal/app/di.go.

## Конфигурация

Приложение использует следующие настройки по умолчанию:
- Порт: 8080
- Режим Gin: зависит от переменной среды GIN_MODE
- CORS: разрешены все источники

## Зависимости

Основные зависимости проекта:
- gin-gonic/gin — веб-фреймворк
- google/uuid — генерация UUID
- swaggo/gin-swagger — Swagger интеграция
- gin-contrib/cors — CORS middleware
- stretchr/testify — тестирование
