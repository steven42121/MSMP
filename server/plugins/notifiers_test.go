package plugins

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

// TestDingTalkSign 验证钉钉 HmacSHA256 签名。
func TestDingTalkSign(t *testing.T) {
	secret := "SEC0123456789abcdef"
	ts := "1694567890000"
	input := ts + "\n" + secret
	want := hmacSHA256B64(secret, input)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(input))
	got := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if got != want {
		t.Fatalf("dingtalk sign mismatch: got %q want %q", got, want)
	}
}

// TestFeishuSign 验证飞书 HmacSHA256 签名。
func TestFeishuSign(t *testing.T) {
	secret := "my-secret"
	ts := "1694567890"
	input := ts + "\n" + secret
	want := hmacSHA256B64(secret, input)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(input))
	got := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if got != want {
		t.Fatalf("feishu sign mismatch: got %q want %q", got, want)
	}
}

// TestBuildText 验证通知文本格式。
func TestBuildText(t *testing.T) {
	n := Notification{
		Title:    "CPU 过高",
		Message:  "主机 cpu 使用率 95%",
		Level:    "critical",
		Hostname: "prod-01",
		Time:     "2026-09-13 10:30:00",
	}
	out := buildText(n)
	if out == "" {
		t.Fatal("empty text")
	}
	for _, want := range []string{"CPU 过高", "prod-01", "critical", "2026-09-13 10:30:00"} {
		if !contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

// TestTestNotification 验证测试消息字段完整。
func TestTestNotification(t *testing.T) {
	n := testNotification()
	if n.Title == "" || n.Message == "" {
		t.Fatalf("testNotification fields empty: %+v", n)
	}
}

// TestRegistry 验证注册中心基本行为。
func TestRegistry(t *testing.T) {
	r := NewRegistry()
	p := WebhookPlugin{}
	r.RegisterNotifier(p)
	meta, ok := r.Meta("notify-webhook")
	if !ok {
		t.Fatal("meta not found")
	}
	if meta.Name != p.Meta().Name {
		t.Fatalf("name mismatch: got %q", meta.Name)
	}
	list := r.List()
	if len(list) == 0 {
		t.Fatal("empty list")
	}
	if _, ok := r.GetNotifier("notify-webhook"); !ok {
		t.Fatal("notifier not found")
	}
	r.RegisterMeta(PluginMeta{ID: "p1", Type: PluginTypeProbe, Name: "P"})
	if len(r.List()) != 2 {
		t.Fatalf("expected 2 metas, got %d", len(r.List()))
	}
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}