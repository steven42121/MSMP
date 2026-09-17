package controllers

import "testing"

func TestIsOpenAddr(t *testing.T) {
	cases := map[string]bool{
		"0.0.0.0": true,
		"::":      true,
		"":        true,
		"127.0.0.1": false,
		"*":       true,
		"::1":     false,
	}
	for in, want := range cases {
		if got := isOpenAddr(in); got != want {
			t.Errorf("isOpenAddr(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestIsLoopbackAddr(t *testing.T) {
	cases := map[string]bool{
		"127.0.0.1": true,
		"127.0.0.2": true,
		"::1":       true,
		"localhost": true,
		"0.0.0.0":   false,
		"10.0.0.1":  false,
		"::":        false,
	}
	for in, want := range cases {
		if got := isLoopbackAddr(in); got != want {
			t.Errorf("isLoopbackAddr(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestRiskPortTable(t *testing.T) {
	// 至少覆盖常见危险端口
	expected := []int{22, 3306, 5432, 6379, 27017, 9200, 2375, 3389}
	for _, p := range expected {
		def, ok := riskPorts[p]
		if !ok {
			t.Errorf("risk port %d missing from table", p)
			continue
		}
		if def.Service == "" || def.Suggestion == "" {
			t.Errorf("risk port %d missing service/suggestion", p)
		}
	}
}

func TestEvaluateBaseline(t *testing.T) {
	cases := []struct {
		section string
		detail  string
		want    string
	}{
		{"permit_root", "PermitRootLogin no", "pass"},
		{"permit_root", "PermitRootLogin yes", "fail"},
		{"password_auth", "PasswordAuthentication no", "pass"},
		{"password_auth", "PasswordAuthentication yes", "fail"},
		{"firewall", "firewalld", "pass"},
		{"firewall", "none", "fail"},
		{"firewall_active", "active", "pass"},
		{"firewall_active", "inactive", "fail"},
		{"selinux", "Enforcing", "pass"},
		{"selinux", "Permissive", "warn"},
		{"selinux", "n/a", "na"},
		{"empty_password", "", "pass"},
		{"empty_password", "root", "fail"},
	}
	for _, c := range cases {
		got := evaluateBaseline(c.section, c.detail).Status
		if got != c.want {
			t.Errorf("evaluateBaseline(%q, %q) = %q, want %q", c.section, c.detail, got, c.want)
		}
	}
}