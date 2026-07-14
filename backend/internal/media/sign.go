// Package media — ký & verify URL video (TTL ngắn, theo user).
//
// Dev: file tĩnh dưới backend/media, ký HMAC + exp; handler /media verify
// trước khi serve → signed URL được ENFORCE thật chứ không chỉ trang trí.
// Prod: thay bằng S3/CloudFront signed URL, Mux playback token, hoặc DRM
// license (xem README mục Video Delivery) — interface không đổi.
package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"time"
)

const ttl = 5 * time.Minute // TTL ngắn theo thiết kế

func secret() []byte {
	if s := os.Getenv("MEDIA_SIGNING_SECRET"); s != "" {
		return []byte(s)
	}
	return []byte("dev-secret-change-me")
}

func sign(path string, userID int64, exp int64) string {
	mac := hmac.New(sha256.New, secret())
	fmt.Fprintf(mac, "%s:%d:%d", path, userID, exp)
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

// SignURL tạo URL có chữ ký cho file media (path dạng "/media/clip.mp4").
func SignURL(path string, userID int64) string {
	exp := time.Now().Add(ttl).Unix()
	return fmt.Sprintf("%s?u=%d&exp=%d&sig=%s", path, userID, exp, sign(path, userID, exp))
}

// Verify kiểm tra chữ ký + hạn của request media. Trả về nil nếu hợp lệ.
func Verify(path, uStr, expStr, sig string) error {
	userID, err := strconv.ParseInt(uStr, 10, 64)
	if err != nil {
		return fmt.Errorf("u không hợp lệ")
	}
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return fmt.Errorf("exp không hợp lệ")
	}
	if time.Now().Unix() > exp {
		return fmt.Errorf("URL hết hạn")
	}
	expected := sign(path, userID, exp)
	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return fmt.Errorf("chữ ký sai")
	}
	return nil
}
