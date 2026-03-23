DB_URL ?= postgres://history:secret@localhost:5432/history_map?sslmode=disable
APP = cmd/history-api/

.PHONY: postgres createdb dropdb migrate-up migrate-down migrate-reset sqlc run build dev

migrate-up:
	migrate -path db/migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path db/migrations -database "$(DB_URL)" down 1

migrate-reset:
	migrate -path db/migrations -database "$(DB_URL)" drop -f
	migrate -path db/migrations -database "$(DB_URL)" up

sqlc:
	sqlc generate

run:
	go run $(APP)

build:
	go build -o app $(APP)

dev: sqlc migrate-up run