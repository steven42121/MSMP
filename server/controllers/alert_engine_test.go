package controllers

import "testing"

func TestEscalateLevel(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"warning", "critical"},
		{"info", "warning"},
		{"critical", "critical"},
		{"", ""},
	}
	for _, c := range cases {
		if got := escalateLevel(c.in); got != c.want {
			t.Errorf("escalateLevel(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestMatchOperator(t *testing.T) {
	cases := []struct {
		op        string
		value     float64
		threshold float64
		want      bool
	}{
		{"gt", 91, 90, true},
		{"gt", 90, 90, false},
		{"gte", 90, 90, true},
		{"lt", 5, 10, true},
		{"lt", 10, 10, false},
		{"lte", 10, 10, true},
		{"unknown", 1, 1, false},
	}
	for _, c := range cases {
		if got := matchOperator(c.op, c.value, c.threshold); got != c.want {
			t.Errorf("matchOperator(%q, %v, %v) = %v, want %v", c.op, c.value, c.threshold, got, c.want)
		}
	}
}