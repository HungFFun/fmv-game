# Năm Trái Tim Dưới Một Mái Nhà — FMV Dating-Sim

Game mô phỏng hẹn hò dạng FMV: kịch bản phân nhánh, nhiều ending, affinity từng
heroine, gallery, bán theo chương. Kiến trúc theo thiết kế 3 hệ con:
**Branching Narrative Engine** + **Video Delivery** + **Save/Entitlement server-authoritative**.

```
fmv-game/
├── backend/          # Go — Director API (server-authoritative)
│   ├── cmd/server/   # entrypoint HTTP :8080
│   ├── cmd/seed/     # nạp kịch bản JSON → DB
│   ├── internal/
│   │   ├── engine/   # Evaluate/Apply — mini-DSL condition/effects (test kỹ nhất)
│   │   ├── store/    # DAL SQLite (dev) — schema mirror Postgres
│   │   ├── director/ # advance/current/save/gallery/store + integration tests
│   │   ├── api/      # net/http routes + dev auth cookie + media signed URL
│   │   └── seed/     # loader JSON authoring → bảng scenes/choices
│   ├── content/demo-story.json   # kịch bản demo (2 chương, 2 heroine, 4 ending)
│   ├── db/postgres-schema.sql    # schema PROD (Postgres, JSONB)
│   └── media/        # ⬅ BỎ FILE .mp4 CỦA BẠN VÀO ĐÂY (xem media/README.md)
├── frontend/         # React + Vite + TS — player UI
│   └── src/
│       ├── api.ts                  # client API (typed)
│       ├── App.tsx                 # GameShell + vòng lặp chơi
│       └── components/
│           ├── PlayerCore.tsx      # VideoSurface + ChoiceOverlay(timer) + BranchPreloader
│           └── Panels.tsx          # AffinityHUD, SaveLoad, Gallery, Store
└── scripts/gen-placeholder-videos.sh  # tự sinh clip test bằng FFmpeg (optional)
```

## Chạy dev

```bash
# 1. Backend (cần Go ≥ 1.23)
cd backend
go mod tidy          # tải modernc.org/sqlite (pure Go, không cần cgo)
go test ./...        # ⚠️ CHẠY ĐẦU TIÊN — engine + director integration tests
go run ./cmd/seed    # nạp demo story → data/game.db
go run ./cmd/server  # :8080

# 2. Bỏ video vào backend/media/ (tên file: xem backend/media/README.md)
#    Thiếu clip không sao — player tự skip, game không kẹt.

# 3. Frontend
cd ../frontend
npm install
npm run dev          # http://localhost:5173 (proxy /api + /media → :8080)
```

> ⚠️ Code Go được viết trong môi trường không có Go toolchain (sandbox bị chặn
> mạng) — **hãy chạy `go test ./...` trước tiên**; frontend đã được build + verify.

## Cách hoạt động (tóm tắt)

- **Server-authoritative**: client chỉ gọi `GET /api/play/current` và
  `POST /api/play/advance {choiceId?}`. Server nạp save → validate choice thuộc
  scene + condition thoả (chống spoof) → apply effects + on_enter → check
  entitlement chương mới (402 `CHAPTER_LOCKED`) → autosave + log `choice_events`
  → trả scene mới kèm **signed video URL** (HMAC + TTL 5 phút, enforce thật ở
  `GET /media/{file}`) và **choices đã lọc** (nhánh chưa đủ điều kiện không bao
  giờ lộ ra client).
- **Mini-DSL** trong JSON (cột `condition`/`effects`/`on_enter`):
  `{"affinity":{"malsook":{">=":30}},"flags":{"saw_secret":true}}` — shorthand
  số = `>=`. Engine `Evaluate`/`Apply` thuần, clamp affinity 0..100.
- **Authoring**: biên kịch sửa `backend/content/demo-story.json` (tham chiếu
  bằng `code`, không đụng id số) → xoá `backend/data/game.db` → `go run ./cmd/seed`.
  Seed validate toàn bộ DSL + dead-end + ref trước khi ghi.

## API

| Method | Path | Ghi chú |
|---|---|---|
| GET | `/api/play/current` | scene hiện tại (tự bắt đầu game mới nếu chưa có save) |
| POST | `/api/play/advance` | `{choiceId?}` — linear không cần choiceId |
| POST | `/api/play/restart` | chơi lại từ đầu (gallery/ending unlock giữ nguyên) |
| GET/POST | `/api/saves`, `/api/saves/load` | slot 1..n (0 = autosave) |
| GET | `/api/gallery` | chỉ item đã unlock có URL |
| GET/POST | `/api/store`, `/api/store/purchase` | purchase = GIẢ LẬP IAP |
| GET | `/media/{file}` | verify chữ ký + exp trước khi serve |

## Lộ trình production (phase 2-4 theo thiết kế)

1. **DB**: SQLite → Postgres (`backend/db/postgres-schema.sql`); DAL trong
   `internal/store` viết SQL chuẩn, đổi driver + placeholder `?`→`$n`.
2. **Auth**: cookie `uid` dev → JWT/session thật.
3. **Payment**: `POST /api/store/purchase` → webhook Stripe / App Store verify
   receipt rồi mới `GrantEntitlement`.
4. **Video**: mp4 tĩnh → HLS adaptive (FFmpeg/Mux/Cloudflare Stream); cột
   `hls_manifest` đã có sẵn trong schema, `internal/media` chỉ cần đổi cách ký.
   Chống lậu nghiêm túc: DRM (Widevine/FairPlay) + forensic watermark.
5. **Editor biên kịch**: React Flow đọc/ghi JSON authoring; hoặc compile
   Ink/Twine → format JSON của `internal/seed`.
6. **Analytics**: bảng `choice_events` đã log đủ để vẽ funnel nhánh.
