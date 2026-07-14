# 📁 Thư mục video — bỏ file .mp4 của bạn vào đây

Backend serve thư mục này tại `/media/<file>` (có verify signed URL).
Tên file phải khớp `backend/content/demo-story.json`. Danh sách cần có:

| File | Dùng cho |
|---|---|
| `shot_1.mp4` | Chương 1 — A Long Day in Itaewon (mở đầu) |
| `shot_2.mp4` | Moonlit Bar — màn Tương tác (hotspot) |
| `shot_3.mp4` | Nhánh "Ngồi xuống quầy bar" |
| `shot_4.mp4` | Nhánh "Chú ý đến cô gái" (A Quiet Seat) |
| `shot_5.mp4` | Tell Me Your Name First (choice, timer 10s) |
| `shot_6.mp4` | Can I Ask Your Number? (checkpoint) |
| `shot_7.mp4` | Ending — "Đừng vội về nhà" |
| `ch2_morning.mp4` | Chương 2 (ngoài story design 7-shot) — buổi sáng |
| `ch2_confess_choice.mp4` | Màn chọn tỏ tình |
| `ch2_malsook_route.mp4` | Route Mal-sook |
| `ch2_minjung_route.mp4` | Route Min-jung |
| `ch2_normal_route.mp4` | Route im lặng |
| `ch2_busted.mp4` | Route bắt cá hai tay |
| `end_malsook_good.mp4` | Ending good — Mal-sook |
| `end_minjung_good.mp4` | Ending good — Min-jung |
| `end_normal.mp4` | Ending normal |
| `end_bad.mp4` | Ending bad |
| `cg_kitchen.mp4` | Gallery CG |
| `cg_stage.mp4` | Gallery CG |
| `cg_malsook.mp4` | Gallery CG |
| `cg_minjung.mp4` | Gallery CG (bonus) |

Lưu ý:

- **Thiếu file nào cũng không sao** — player tự skip clip lỗi (coi như phát xong), game không kẹt. Test được với vài file trước.
- Muốn đổi tên file / thêm clip: sửa mảng `media` trong `backend/content/demo-story.json`, xoá `backend/data/game.db` rồi chạy lại `go run ./cmd/seed`.
- Format khuyến nghị để test nhanh: H.264 + AAC, mp4. (Prod sẽ chuyển sang HLS — xem README gốc.)
- Nếu vẫn muốn clip placeholder tự sinh: `bash scripts/gen-placeholder-videos.sh` (cần FFmpeg).
