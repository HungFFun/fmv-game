# Deploy lên Vultr (Docker Compose)

Kiến trúc: 2 container — `api` (Go Director :8080, nội bộ) + `web` (nginx :80 serve
frontend build + reverse-proxy `/api` và `/media` sang `api`). SQLite nằm trong Docker
volume `db_data` (bền qua restart). Media mp4 đóng gói sẵn trong image `api`.

## 0. Yêu cầu
- VPS Vultr Ubuntu 22.04/24.04, có IP + SSH (user/pass).
- (Tùy chọn) 1 domain trỏ A record về IP nếu muốn HTTPS.

## 1. SSH vào server
```bash
ssh root@SERVER_IP        # hoặc user của bạn; nhập mật khẩu khi được hỏi
```

## 2. Cài Docker + Compose plugin (chạy 1 lần)
```bash
curl -fsSL https://get.docker.com | sh
docker version && docker compose version   # kiểm tra
```

## 3. Lấy code về
```bash
cd /opt
git clone https://github.com/<user>/<repo>.git fmv-game
cd fmv-game
```

## 4. Tạo secret (.env)
```bash
cp .env.example .env
# sinh 2 chuỗi ngẫu nhiên và điền vào .env:
echo "ADMIN_TOKEN=$(openssl rand -hex 32)"          >> .env
echo "MEDIA_SIGNING_SECRET=$(openssl rand -hex 32)" >> .env
# rồi mở .env xoá 2 dòng change-me mẫu ở trên cho gọn:
nano .env
```
> `.env` KHÔNG được commit — chỉ nằm trên server.

## 5. Build & chạy
```bash
docker compose up -d --build
docker compose ps           # cả api + web phải "running"
docker compose logs -f api  # xem "Seeding DB..." rồi "Khởi động server"
```
Lần đầu `api` tự seed DB từ `content/demo-story.json`.

## 6. Mở firewall & truy cập
```bash
ufw allow 80/tcp && ufw allow 22/tcp && ufw --force enable   # nếu dùng ufw
```
Mở trình duyệt: `http://SERVER_IP`

## 7. Cập nhật khi có code mới
```bash
cd /opt/fmv-game
git pull
docker compose up -d --build      # DB giữ nguyên (volume db_data)
```

## 8. Gắn domain + HTTPS (Caddy — tự động)
Compose đã có sẵn service `caddy` (cổng 80/443) tự xin & gia hạn cert Let's Encrypt.

1. **Trỏ DNS**: tại nơi quản lý domain, thêm **A record** cho domain (hoặc subdomain)
   trỏ về IP server. Cloudflare thì tạm để **DNS only** (tắt proxy cam) khi xin cert.
2. **Khai báo domain** trong `.env`:
   ```bash
   echo "DOMAIN=game.tenban.com" >> .env   # thay bằng domain thật
   ```
3. **Mở cổng 443** + chạy lại:
   ```bash
   ufw allow 443/tcp
   docker compose up -d
   docker compose logs -f caddy   # thấy "certificate obtained successfully"
   ```
4. Truy cập `https://game.tenban.com` (Caddy tự chuyển http→https).

> Cert lưu trong volume `caddy_data` — đừng `down -v` kẻo phải xin lại (dễ dính rate-limit
> Let's Encrypt). Chỉ dùng IP không domain? Bỏ service `caddy`, thêm `ports: ["80:80"]` vào `web`.

## Lệnh vận hành hay dùng
```bash
docker compose down            # dừng (giữ DB)
docker compose down -v         # dừng + XOÁ DB volume (reset tiến trình chơi)
docker compose restart api     # restart backend
docker compose logs -f web     # log nginx
```

## Đổi content (story) sau khi deploy
Sửa `backend/content/demo-story.json` → commit → trên server `git pull`. Để nạp lại story:
```bash
docker compose down -v && docker compose up -d --build   # reset + seed lại
```
(hoặc dùng API admin `PUT /api/admin/models/{id}/content` với header `X-Admin-Token`.)
