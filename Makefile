DB_URL ?= postgres://history:secret@localhost:5432/history_map?sslmode=disable
APP_DIR = cmd/history-api
MAIN_APP = ./cmd/history-api/
MAIN_FILE = $(APP_DIR)/main.go
DOCS_DIR = docs

.PHONY: migrate-up migrate-down migrate-reset swagger sqlc run build dev

migrate-up:
	migrate -path db/migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path db/migrations -database "$(DB_URL)" down 1

migrate-reset:
	migrate -path db/migrations -database "$(DB_URL)" drop -f
	migrate -path db/migrations -database "$(DB_URL)" up

swagger:
	@echo "=> Generating Swagger docs..."
	swag init -g $(MAIN_FILE) -o $(DOCS_DIR) --parseDependency --parseInternal
	@echo "=> Swagger docs generated at $(DOCS_DIR)"

sqlc:
	sqlc generate

run:
	@set GOARCH=amd64& set CGO_ENABLED=0&go run $(MAIN_APP)

build:
	@set GOOS=linux& set GOARCH=amd64& set CGO_ENABLED=0&go build -trimpath -ldflags="-s -w" -o build/history-api $(MAIN_APP)

dev: swagger sqlc migrate-up run