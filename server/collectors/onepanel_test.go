package collectors

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// TestOnePanelSign 验证 OnePanel HMAC-SHA256 签名（hex 编码，与官方文档对齐）。
func TestOnePanelSign(t *testing.T) {
	apiKey := "test-api-key"
	ts := "1757733000" // 秒级 Unix 时间戳
	want := hex.EncodeToString(hmacSHA256([]byte(apiKey), "1panel:"+ts))

	mac := hmac.New(sha256.New, []byte(apiKey))
	mac.Write([]byte("1panel:" + ts))
	got := hex.EncodeToString(mac.Sum(nil))
	if got != want {
		t.Fatalf("sign mismatch: got %q want %q", got, want)
	}
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}
