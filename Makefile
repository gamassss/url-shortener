include .env
export

DB_CONTAINER = url-shortener-postgres-1
BASE_URL ?= http://localhost:8080
DATABASE_URL = postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down

build:
	docker-compose build --no-cache

up:
	docker-compose up -d

down:
	docker-compose down

restart:
	docker-compose restart

clean:
	docker-compose down -v
	docker system prune -f

rebuild: down build up

monitor:
	watch -n 1 'docker stats --no-stream'

run:
	go run cmd/api/main.go

run-seed:
	go run tests/load/generate-data/main.go

redis-flushall:
	docker exec -it url-shortener-redis-1 redis-cli flushall

test:
	gotestsum --format testname -- ./... -v

test-integration:
	gotestsum --format testname -- -tags=integration ./tests/integration/... -v

test-load:
	k6 run -e BASE_URL=http://localhost:8080 tests/load/load.js

