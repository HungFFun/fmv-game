// Package api — HTTP layer (net/http, Go 1.22+ route patterns).
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"fmv-game/backend/internal/director"
	"fmv-game/backend/internal/media"
	"fmv-game/backend/internal/store"
)

type Server struct {
	St       *store.Store
	Dir      *director.Director
	MediaDir string // thư mục chứa file video dev (backend/media)
}

func New(st *store.Store, mediaDir string) *Server {
	return &Server{St: st, Dir: director.New(st), MediaDir: mediaDir}
}

// ===== helpers =====

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	var de *director.Error
	if errors.As(err, &de) {
		writeJSON(w, de.Status, map[string]any{
			"error": map[string]any{"code": de.Code, "message": de.Msg, "data": de.Data},
		})
		return
	}
	log.Printf("internal error: %v", err)
	writeJSON(w, http.StatusInternalServerError, map[string]any{
		"error": map[string]any{"code": "INTERNAL", "message": "lỗi hệ thống"},
	})
}

func decodeBody[T any](r *http.Request) (T, error) {
	var v T
	if r.Body == nil {
		return v, nil
	}
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&v); err != nil && !errors.Is(err, io.EOF) {
		return v, fmt.Errorf("body JSON không hợp lệ: %w", err)
	}
	return v, nil
}

// ===== dev auth =====
// Cookie "uid" — DEV ONLY. Prod: thay bằng session/JWT (NextAuth, Firebase Auth...).

const uidCookie = "uid"

func (s *Server) currentUser(w http.ResponseWriter, r *http.Request) (int64, error) {
	if c, err := r.Cookie(uidCookie); err == nil {
		if id, err := strconv.ParseInt(c.Value, 10, 64); err == nil {
			if ok, _ := s.St.UserExists(id); ok {
				return id, nil
			}
		}
	}
	email := fmt.Sprintf("player-%06d@dev.local", rand.Intn(1_000_000))
	id, err := s.St.GetOrCreateUser(email)
	if err != nil {
		return 0, err
	}
	http.SetCookie(w, &http.Cookie{
		Name: uidCookie, Value: strconv.FormatInt(id, 10),
		Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode,
		MaxAge: 86400 * 365,
	})
	return id, nil
}

type handlerWithUser func(w http.ResponseWriter, r *http.Request, userID int64)

func (s *Server) auth(h handlerWithUser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := s.currentUser(w, r)
		if err != nil {
			writeErr(w, err)
			return
		}
		h(w, r, userID)
	}
}

// ===== routes =====

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/play/current", s.auth(func(w http.ResponseWriter, r *http.Request, uid int64) {
		resp, err := s.Dir.Current(uid)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	}))

	// Nhảy tới 1 scene cụ thể khi bấm node video trên bản đồ chapter (xem/chơi lại).
	mux.HandleFunc("GET /api/play/scene/{code}", s.auth(func(w http.ResponseWriter, r *http.Request, uid int64) {
		resp, err := s.Dir.JumpToScene(uid, r.PathValue("code"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	}))

	mux.HandleFunc("POST /api/play/advance", s.auth(func(w http.ResponseWriter, r *http.Request, uid int64) {
		body, err := decodeBody[struct {
			ChoiceID *int64 `json:"choiceId"`
		}](r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		resp, err := s.Dir.Advance(uid, body.ChoiceID)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	}))

	mux.HandleFunc("POST /api/play/restart", s.auth(func(w http.ResponseWriter, r *http.Request, uid int64) {
		resp, err := s.Dir.Restart(uid)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	}))

	mux.HandleFunc("GET /api/saves", s.auth(func(w http.ResponseWriter, r *http.Request, uid int64) {
		saves, err := s.Dir.ListSaves(uid)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"saves": saves})
	}))

	mux.HandleFunc("POST /api/saves", s.auth(func(w http.ResponseWriter, r *http.Request, uid int64) {
		body, err := decodeBody[struct {
			Slot int `json:"slot"`
		}](r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if err := s.Dir.SaveToSlot(uid, body.Slot); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}))

	mux.HandleFunc("POST /api/saves/load", s.auth(func(w http.ResponseWriter, r *http.Request, uid int64) {
		body, err := decodeBody[struct {
			Slot int `json:"slot"`
		}](r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		resp, err := s.Dir.LoadFromSlot(uid, body.Slot)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	}))

	mux.HandleFunc("GET /api/gallery", s.auth(func(w http.ResponseWriter, r *http.Request, uid int64) {
		items, err := s.Dir.Gallery(uid)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}))

	mux.HandleFunc("GET /api/store", s.auth(func(w http.ResponseWriter, r *http.Request, uid int64) {
		chapters, err := s.Dir.StoreChapters(uid)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"chapters": chapters})
	}))

	// GIẢ LẬP mua chương. Prod: thay bằng webhook Stripe / App Store
	// (POST /api/iap/webhook) — chỉ cấp entitlement sau khi verify receipt.
	mux.HandleFunc("POST /api/store/purchase", s.auth(func(w http.ResponseWriter, r *http.Request, uid int64) {
		body, err := decodeBody[struct {
			ChapterID int64 `json:"chapterId"`
		}](r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if err := s.Dir.Purchase(uid, body.ChapterID); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}))

	// Chapter browse (data-driven cho màn Chapter select + map).
	mux.HandleFunc("GET /api/chapters", s.auth(func(w http.ResponseWriter, r *http.Request, uid int64) {
		chapters, err := s.Dir.ChaptersOverview(uid)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"chapters": chapters})
	}))

	mux.HandleFunc("GET /api/chapters/{id}/map", s.auth(func(w http.ResponseWriter, r *http.Request, uid int64) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id không hợp lệ"})
			return
		}
		m, err := s.Dir.ChapterMap(uid, id)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, m)
	}))

	// Media: verify chữ ký + exp trước khi serve (signed URL được enforce thật).
	mux.HandleFunc("GET /media/{file}", func(w http.ResponseWriter, r *http.Request) {
		file := r.PathValue("file")
		if strings.Contains(file, "..") || strings.Contains(file, "/") {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		q := r.URL.Query()
		if err := media.Verify("/media/"+file, q.Get("u"), q.Get("exp"), q.Get("sig")); err != nil {
			http.Error(w, "signed URL không hợp lệ: "+err.Error(), http.StatusForbidden)
			return
		}
		path := filepath.Join(s.MediaDir, file)
		if _, err := os.Stat(path); err != nil {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, path)
	})

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Admin CRUD (Phase 2) — bảo vệ bằng admin token (xem admin.go).
	s.registerAdmin(mux)

	return cors(mux)
}

// cors — cho phép FE dev gọi trực tiếp (khi không dùng Vite proxy).
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "http://127.0.0.1") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Admin-Token, Authorization")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
