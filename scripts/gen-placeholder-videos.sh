#!/usr/bin/env bash
# Generate placeholder FMV clips bằng FFmpeg (màu nền + chữ + beep nhẹ).
# Chạy từ root repo: bash scripts/gen-placeholder-videos.sh
# Output: backend/media/*.mp4 — thay bằng footage thật khi có.
set -euo pipefail

OUT="$(dirname "$0")/../backend/media"
mkdir -p "$OUT"

gen() {
  local name="$1" color="$2" text="$3" dur="$4"
  ffmpeg -y -loglevel error \
    -f lavfi -i "color=c=${color}:s=854x480:d=${dur}" \
    -f lavfi -i "sine=frequency=440:duration=${dur}" \
    -vf "drawtext=text='${text}':fontcolor=white:fontsize=36:x=(w-text_w)/2:y=(h-text_h)/2:font=Sans" \
    -af "volume=0.05" \
    -c:v libx264 -pix_fmt yuv420p -c:a aac -shortest \
    "${OUT}/${name}.mp4"
  echo "  ✓ ${name}.mp4"
}

echo "Generating placeholder clips → ${OUT}"
gen ch1_intro          "#2b1b3d" "Chuong 1 - Mai nha chung"        6
gen ch1_dinner_choice  "#3d1b2b" "Toi nay an gi?"                  5
gen ch1_kitchen        "#1b3d2b" "Vao bep cung Mal-sook"           6
gen ch1_stage          "#1b2b3d" "San khau cua Min-jung"           6
gen ch1_phone          "#3d2b1b" "Tin nhan luc nua dem"            5
gen ch1_end            "#2b2b2b" "Het chuong 1"                    5
gen ch2_morning        "#1f2b4d" "Chuong 2 - Buoi sang"            6
gen ch2_confess_choice "#4d1f2b" "Trai tim chon ai?"               5
gen ch2_malsook_route  "#1f4d2b" "Ben Mal-sook"                    6
gen ch2_minjung_route  "#2b1f4d" "Ben Min-jung"                    6
gen ch2_normal_route   "#3d3d3d" "Giu im lang"                     5
gen ch2_busted         "#4d1f1f" "Bi bat qua tang!"                5
gen end_malsook_good   "#1f4d3d" "ENDING - Mai ben Mal-sook"       6
gen end_minjung_good   "#3d1f4d" "ENDING - Anh den san khau"       6
gen end_normal         "#2f2f2f" "ENDING - Nam nguoi ban"          5
gen end_bad            "#4d0f0f" "BAD ENDING"                      5
gen cg_kitchen         "#205040" "CG - Can bep am ap"              4
gen cg_stage           "#204050" "CG - San khau ruc ro"            4
gen cg_malsook         "#502040" "CG - Nu cuoi Mal-sook"           4
gen cg_minjung         "#402050" "CG - Min-jung sau canh ga"       4
echo "Done: $(ls "$OUT" | wc -l) clips"
