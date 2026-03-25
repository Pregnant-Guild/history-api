FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Ho_Chi_Minh

WORKDIR /app

COPY build/history-api .
COPY data ./data

EXPOSE 3344

CMD ["./history-api"]