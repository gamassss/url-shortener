include .env
export

BASE_URL ?= http://localhost:8080
DATABASE_URL = postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-restart:
	docker-compose restart

run:
	go run cmd/api/main.go

test-peak:
	k6 run -e BASE_URL=http://localhost:8080 tests/load/peak.js --out influxdb=http://localhost:8086/k6
