package engine

import (
	"encoding/json"
	"testing"
)

func st(aff map[string]int, flags map[string]bool) State {
	s := NewState()
	for k, v := range aff {
		s.Affinity[k] = v
	}
	for k, v := range flags {
		s.Flags[k] = v
	}
	return s
}

func mustCond(t *testing.T, raw string) Condition {
	t.Helper()
	c, err := ParseCondition(raw)
	if err != nil {
		t.Fatalf("ParseCondition(%s): %v", raw, err)
	}
	return c
}

func mustEff(t *testing.T, raw string) Effects {
	t.Helper()
	e, err := ParseEffects(raw)
	if err != nil {
		t.Fatalf("ParseEffects(%s): %v", raw, err)
	}
	return e
}

func TestEvaluate(t *testing.T) {
	cases := []struct {
		name string
		cond string
		s    State
		want bool
	}{
		{"condition rỗng luôn true", `{}`, NewState(), true},
		{"shorthand số = >= (đủ)", `{"affinity":{"malsook":30}}`, st(map[string]int{"malsook": 30}, nil), true},
		{"shorthand số = >= (thiếu)", `{"affinity":{"malsook":30}}`, st(map[string]int{"malsook": 29}, nil), false},
		{"toán tử >= đúng", `{"affinity":{"a":{">=":10}}}`, st(map[string]int{"a": 10}, nil), true},
		{"toán tử > sai khi bằng", `{"affinity":{"a":{">":10}}}`, st(map[string]int{"a": 10}, nil), false},
		{"toán tử <= đúng", `{"affinity":{"a":{"<=":10}}}`, st(map[string]int{"a": 10}, nil), true},
		{"toán tử < sai khi bằng", `{"affinity":{"a":{"<":10}}}`, st(map[string]int{"a": 10}, nil), false},
		{"toán tử == đúng", `{"affinity":{"a":{"==":10}}}`, st(map[string]int{"a": 10}, nil), true},
		{"toán tử != sai khi bằng", `{"affinity":{"a":{"!=":10}}}`, st(map[string]int{"a": 10}, nil), false},
		{"range trong khoảng", `{"affinity":{"a":{">=":10,"<":20}}}`, st(map[string]int{"a": 15}, nil), true},
		{"range chạm trần", `{"affinity":{"a":{">=":10,"<":20}}}`, st(map[string]int{"a": 20}, nil), false},
		{"affinity chưa set = 0", `{"affinity":{"ghost":{"==":0}}}`, NewState(), true},
		{"flag chưa set = false", `{"flags":{"unseen":false}}`, NewState(), true},
		{"flag chưa set != true", `{"flags":{"unseen":true}}`, NewState(), false},
		{
			"AND affinity + flags: đủ cả hai",
			`{"affinity":{"malsook":{">=":30}},"flags":{"saw_secret":true}}`,
			st(map[string]int{"malsook": 42}, map[string]bool{"saw_secret": true}),
			true,
		},
		{
			"AND affinity + flags: thiếu flag",
			`{"affinity":{"malsook":{">=":30}},"flags":{"saw_secret":true}}`,
			st(map[string]int{"malsook": 42}, nil),
			false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Evaluate(mustCond(t, tc.cond), tc.s); got != tc.want {
				t.Errorf("Evaluate(%s) = %v, want %v", tc.cond, got, tc.want)
			}
		})
	}
}

func TestParseConditionRejectsBadOperator(t *testing.T) {
	if _, err := ParseCondition(`{"affinity":{"a":{"~=":1}}}`); err == nil {
		t.Error("muốn lỗi với toán tử lạ (bắt lỗi data biên kịch), nhận nil")
	}
}

func TestApply(t *testing.T) {
	t.Run("effects rỗng trả copy, không mutate", func(t *testing.T) {
		s := st(map[string]int{"a": 5}, map[string]bool{"f": true})
		out := Apply(Effects{}, s)
		if out.Affinity["a"] != 5 || !out.Flags["f"] {
			t.Errorf("copy sai: %+v", out)
		}
		out.Affinity["a"] = 99
		if s.Affinity["a"] != 5 {
			t.Error("Apply mutate state gốc")
		}
	})

	t.Run("affinity là delta, cộng từ 0 nếu chưa có", func(t *testing.T) {
		out := Apply(mustEff(t, `{"affinity":{"minjung":5,"malsook":-2}}`), st(map[string]int{"malsook": 10}, nil))
		if out.Affinity["malsook"] != 8 || out.Affinity["minjung"] != 5 {
			t.Errorf("delta sai: %+v", out.Affinity)
		}
	})

	t.Run("clamp 0..100", func(t *testing.T) {
		if got := Apply(mustEff(t, `{"affinity":{"a":-50}}`), st(map[string]int{"a": 10}, nil)).Affinity["a"]; got != 0 {
			t.Errorf("clamp dưới: %d", got)
		}
		if got := Apply(mustEff(t, `{"affinity":{"a":999}}`), st(map[string]int{"a": 10}, nil)).Affinity["a"]; got != 100 {
			t.Errorf("clamp trên: %d", got)
		}
		if got := Apply(mustEff(t, `{"set_affinity":{"a":150}}`), NewState()).Affinity["a"]; got != 100 {
			t.Errorf("set_affinity clamp: %d", got)
		}
	})

	t.Run("set_affinity chạy SAU delta", func(t *testing.T) {
		out := Apply(mustEff(t, `{"affinity":{"a":5},"set_affinity":{"a":0}}`), st(map[string]int{"a": 50}, nil))
		if out.Affinity["a"] != 0 {
			t.Errorf("muốn 0, nhận %d", out.Affinity["a"])
		}
	})

	t.Run("flags set/clear + chapter", func(t *testing.T) {
		out := Apply(mustEff(t, `{"flags":{"confessed":true,"secret":false},"chapter":2}`),
			st(nil, map[string]bool{"secret": true}))
		if !out.Flags["confessed"] || out.Flags["secret"] || out.Chapter != 2 {
			t.Errorf("flags/chapter sai: %+v", out)
		}
	})
}

func TestStateJSONRoundtrip(t *testing.T) {
	s := st(map[string]int{"malsook": 42}, map[string]bool{"x": true})
	s.Chapter = 3
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var s2 State
	if err := json.Unmarshal(b, &s2); err != nil {
		t.Fatal(err)
	}
	if s2.Affinity["malsook"] != 42 || !s2.Flags["x"] || s2.Chapter != 3 {
		t.Errorf("roundtrip sai: %+v", s2)
	}
}

// Tích hợp: gate tỏ tình kiểu "Five Hearts".
func TestConfessGateIntegration(t *testing.T) {
	gate := mustCond(t, `{"affinity":{"malsook":{">=":30}}}`)
	plusFive := mustEff(t, `{"affinity":{"malsook":5}}`)

	s := NewState()
	for i := 0; i < 6; i++ {
		s = Apply(plusFive, s)
	}
	if s.Affinity["malsook"] != 30 {
		t.Fatalf("tích luỹ sai: %d", s.Affinity["malsook"])
	}
	if !Evaluate(gate, s) {
		t.Error("đủ 30 affinity phải mở được lựa chọn Tỏ tình")
	}

	s2 := NewState()
	for i := 0; i < 3; i++ {
		s2 = Apply(plusFive, s2)
	}
	if Evaluate(gate, s2) {
		t.Error("15 affinity không được mở Tỏ tình → normal end")
	}
}
