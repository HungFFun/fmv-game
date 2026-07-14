# --- Build stage: biên dịch server + seed (pure Go, không cần cgo) ---
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/server ./cmd/server \
 && CGO_ENABLED=0 go build -trimpath -o /out/seed   ./cmd/seed

# --- Runtime stage: ảnh nhỏ, chứa binary + content + media ---
FROM alpine:3.20
WORKDIR /app
RUN mkdir -p /data
COPY --from=build /out/server /out/seed /app/
COPY content /app/content
COPY media   /app/media
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh
ENV DB_PATH=/data/game.db \
    MEDIA_DIR=/app/media \
    ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/app/entrypoint.sh"]
