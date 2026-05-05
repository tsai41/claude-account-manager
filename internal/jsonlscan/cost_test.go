package jsonlscan

import (
	"math"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestPricingCost(t *testing.T) {
	p := Pricing{InputPerM: 5, OutputPerM: 25, CacheCreate5mMult: 1.25, CacheCreate1hMult: 2.0, CacheReadMult: 0.1}
	tk := Tokens{Input: 1_000_000, Output: 1_000_000, CacheCreate5m: 1_000_000, CacheCreate1h: 1_000_000, CacheRead: 1_000_000}
	got := p.Cost(tk)
	want := 5 + 25 + 5*1.25 + 5*2.0 + 5*0.1
	if !approx(got, want) {
		t.Fatalf("Cost = %v, want %v", got, want)
	}
}

func TestFamilyOf(t *testing.T) {
	cases := []struct {
		model string
		fam   string
		ok    bool
	}{
		{"claude-opus-4-7", "Opus", true},
		{"claude-sonnet-4-6", "Sonnet", true},
		{"claude-haiku-4-5-20251001", "Haiku", true},
		{"unknown-model", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		t.Run(c.model, func(t *testing.T) {
			fam, _, ok := familyOf(c.model)
			if fam != c.fam || ok != c.ok {
				t.Fatalf("familyOf(%q) = (%q,%v), want (%q,%v)", c.model, fam, ok, c.fam, c.ok)
			}
		})
	}
}
