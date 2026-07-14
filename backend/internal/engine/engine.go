// Package engine — Branching Narrative Engine core.
//
// Evaluate(condition, state) & Apply(effects, state) là hàm THUẦN, không I/O.
// Đây là phần quyết định người chơi đi nhánh nào → được unit-test kỹ nhất.
// Server-authoritative: chỉ server gọi 2 hàm này, client không bao giờ.
package engine

import (
	"encoding/json"
	"fmt"
)

const (
	AffinityMin = 0
	AffinityMax = 100
)

// State — trạng thái runtime của một lượt chơi, lưu nguyên cục JSON trong saves.state.
type State struct {
	// Affinity: thiện cảm từng heroine 0..100, key = characters.code.
	Affinity map[string]int `json:"affinity"`
	// Flags: cờ sự kiện; cờ không tồn tại = false.
	Flags map[string]bool `json:"flags"`
	// Chapter: idx chương hiện tại (tham khảo; server vẫn check theo scene).
	Chapter int `json:"chapter,omitempty"`
}

// NewState khởi tạo state rỗng cho game mới.
func NewState() State {
	return State{Affinity: map[string]int{}, Flags: map[string]bool{}}
}

// Clone copy sâu — Apply không bao giờ mutate state gốc.
func (s State) Clone() State {
	c := State{
		Affinity: make(map[string]int, len(s.Affinity)),
		Flags:    make(map[string]bool, len(s.Flags)),
		Chapter:  s.Chapter,
	}
	for k, v := range s.Affinity {
		c.Affinity[k] = v
	}
	for k, v := range s.Flags {
		c.Flags[k] = v
	}
	return c
}

// CmpSet — tập điều kiện so sánh cho MỘT heroine, ví dụ {">=": 10, "<": 20}.
// JSON shorthand: số trần 30 ⇔ {">=": 30}.
type CmpSet map[string]float64

var validOps = map[string]func(a, b float64) bool{
	">=": func(a, b float64) bool { return a >= b },
	">":  func(a, b float64) bool { return a > b },
	"<=": func(a, b float64) bool { return a <= b },
	"<":  func(a, b float64) bool { return a < b },
	"==": func(a, b float64) bool { return a == b },
	"!=": func(a, b float64) bool { return a != b },
}

// UnmarshalJSON hỗ trợ shorthand số: {"malsook": 30} → {"malsook": {">=": 30}}.
func (c *CmpSet) UnmarshalJSON(data []byte) error {
	var n float64
	if err := json.Unmarshal(data, &n); err == nil {
		*c = CmpSet{">=": n}
		return nil
	}
	var m map[string]float64
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("cmp set không hợp lệ: %s", data)
	}
	for op := range m {
		if _, ok := validOps[op]; !ok {
			return fmt.Errorf("toán tử không hợp lệ: %q", op)
		}
	}
	*c = m
	return nil
}

// Condition — mini-DSL điều kiện (AND tất cả các vế). Lưu ở cột JSONB choices.condition.
//
//	{"affinity": {"malsook": {">=": 30}}, "flags": {"saw_secret": true}}
type Condition struct {
	Affinity map[string]CmpSet `json:"affinity,omitempty"`
	Flags    map[string]bool   `json:"flags,omitempty"`
}

// Effects — mini-DSL hiệu ứng. Lưu ở choices.effects và scenes.on_enter.
//
//	{"affinity": {"minjung": 5, "malsook": -2}, "flags": {"confessed": true}}
//
// Affinity là DELTA (clamp 0..100); SetAffinity gán tuyệt đối, chạy SAU delta.
type Effects struct {
	Affinity    map[string]int  `json:"affinity,omitempty"`
	SetAffinity map[string]int  `json:"set_affinity,omitempty"`
	Flags       map[string]bool `json:"flags,omitempty"`
	Chapter     *int            `json:"chapter,omitempty"`
}

// ParseCondition parse JSON từ DB; "" hoặc "{}" → condition rỗng (luôn true).
func ParseCondition(raw string) (Condition, error) {
	var c Condition
	if raw == "" {
		return c, nil
	}
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		return c, fmt.Errorf("condition JSON lỗi: %w", err)
	}
	return c, nil
}

// ParseEffects parse JSON từ DB; "" hoặc "{}" → effects rỗng (no-op).
func ParseEffects(raw string) (Effects, error) {
	var e Effects
	if raw == "" {
		return e, nil
	}
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		return e, fmt.Errorf("effects JSON lỗi: %w", err)
	}
	return e, nil
}

// Evaluate trả về true nếu state thoả TẤT CẢ các vế của condition (AND).
// Affinity chưa từng set = 0; flag chưa từng set = false. Condition rỗng → true.
func Evaluate(c Condition, s State) bool {
	for char, cmps := range c.Affinity {
		value := float64(s.Affinity[char]) // zero-value 0 nếu chưa set
		for op, target := range cmps {
			fn, ok := validOps[op]
			if !ok || !fn(value, target) {
				return false
			}
		}
	}
	for flag, expected := range c.Flags {
		if s.Flags[flag] != expected { // zero-value false nếu chưa set
			return false
		}
	}
	return true
}

func clamp(n int) int {
	if n < AffinityMin {
		return AffinityMin
	}
	if n > AffinityMax {
		return AffinityMax
	}
	return n
}

// Apply áp effects lên state và trả về STATE MỚI (không mutate input).
func Apply(e Effects, s State) State {
	next := s.Clone()
	for char, delta := range e.Affinity {
		next.Affinity[char] = clamp(next.Affinity[char] + delta)
	}
	for char, value := range e.SetAffinity {
		next.Affinity[char] = clamp(value)
	}
	for flag, value := range e.Flags {
		next.Flags[flag] = value
	}
	if e.Chapter != nil {
		next.Chapter = *e.Chapter
	}
	return next
}
