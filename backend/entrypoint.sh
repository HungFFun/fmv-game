#!/bin/sh
# Seed DB lần đầu (nếu chưa có), rồi chạy server. DB nằm ở volume /data nên bền qua restart.
set -e
if [ ! -f "$DB_PATH" ]; then
  echo "[entrypoint] Seeding DB tại $DB_PATH ..."
  /app/seed
fi
echo "[entrypoint] Khởi động server ($ADDR) ..."
exec /app/server
