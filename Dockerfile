FROM golang:1.26.1-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o history-api ./cmd/api
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o email-worker ./cmd/worker/email
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o storage-worker ./cmd/worker/storage

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Ho_Chi_Minh

WORKDIR /app

COPY --from=builder /app/history-api .
COPY --from=builder /app/email-worker .
COPY --from=builder /app/storage-worker .
COPY data ./data

RUN chmod +x ./history-api ./email-worker ./storage-worker

EXPOSE 3344

CMD ["./history-api"]