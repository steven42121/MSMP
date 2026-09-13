package plugins

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

const notifyTimeout = 15 * time.Second

var notifyHTTPClient = &http.Client{Timeout: notifyTimeout}

func buildText(n Notification) string {
	if n.Time == "" {
		n.Time = time.Now().Format("2006-01-02 15:04:05")
	}
	return fmt.Sprintf("[MSMP 告警] %s\n级别: %s\n主机: %s\n%s\n时间: %s",
		n.Title, levelLabel(n.Level), hostLabel(n.Hostname), n.Message, n.Time)
}

func testNotification() Notification {
	return Notification{
		Title:    "测试通知",
		Message:  "这是一条来自 MSMP 的测试通知，用于验证渠道连通性。",
		Level:    "info",
		Hostname: "-",
		Time:     time.Now().Format("2006-01-02 15:04:05"),
	}
}

func levelLabel(level string) string {
	if level == "" {
		return "info"
	}
	return level
}

func hostLabel(host string) string {
	if host == "" {
		return "-"
	}
	return host
}

func postJSON(ctx context.Context, url string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := notifyHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}

func hmacSHA256B64(secret, data string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// WebhookPlugin 通用 Webhook 通知。
type WebhookPlugin struct{}

func (WebhookPlugin) Meta() PluginMeta {
	return PluginMeta{
		ID:          "notify-webhook",
		Type:        PluginTypeNotifier,
		Name:        "Webhook",
		Description: "向自定义 HTTP 接口 POST JSON 告警",
		ConfigFields: []ConfigField{
			{Key: "url", Label: "Webhook 地址", Type: "string", Required: true, Secret: true},
		},
	}
}

func (WebhookPlugin) Send(ctx context.Context, cfg map[string]string, n Notification) error {
	return postJSON(ctx, cfg["url"], map[string]interface{}{
		"title":     n.Title,
		"message":   n.Message,
		"level":     n.Level,
		"hostname":  n.Hostname,
		"host_uuid": n.HostUUID,
		"time":      n.Time,
	})
}

func (w WebhookPlugin) Test(ctx context.Context, cfg map[string]string) error {
	return w.Send(ctx, cfg, testNotification())
}

// DingTalkPlugin 钉钉群机器人通知。
type DingTalkPlugin struct{}

func (DingTalkPlugin) Meta() PluginMeta {
	return PluginMeta{
		ID:          "notify-dingtalk",
		Type:        PluginTypeNotifier,
		Name:        "钉钉群机器人",
		Description: "向钉钉群自定义机器人发送告警",
		ConfigFields: []ConfigField{
			{Key: "access_token", Label: "Access Token", Type: "secret", Required: true, Secret: true},
			{Key: "secret", Label: "加签密钥（可选）", Type: "secret", Secret: true},
			{Key: "keyword", Label: "关键词（可选）", Type: "string"},
		},
	}
}

func (DingTalkPlugin) Send(ctx context.Context, cfg map[string]string, n Notification) error {
	url := "https://oapi.dingtalk.com/robot/send?access_token=" + cfg["access_token"]
	content := buildText(n)
	if kw := cfg["keyword"]; kw != "" {
		content = kw + "\n" + content
	}
	payload := map[string]interface{}{
		"msgtype": "text",
		"text":    map[string]string{"content": content},
	}
	if secret := cfg["secret"]; secret != "" {
		ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
		sign := hmacSHA256B64(secret, ts+"\n"+secret)
		url += "&timestamp=" + ts + "&sign=" + sign
	}
	return postJSON(ctx, url, payload)
}

func (d DingTalkPlugin) Test(ctx context.Context, cfg map[string]string) error {
	return d.Send(ctx, cfg, testNotification())
}

// FeishuPlugin 飞书群机器人通知。
type FeishuPlugin struct{}

func (FeishuPlugin) Meta() PluginMeta {
	return PluginMeta{
		ID:          "notify-feishu",
		Type:        PluginTypeNotifier,
		Name:        "飞书群机器人",
		Description: "向飞书群自定义机器人发送告警",
		ConfigFields: []ConfigField{
			{Key: "webhook_url", Label: "Webhook 地址", Type: "secret", Required: true, Secret: true},
			{Key: "secret", Label: "签名密钥（可选）", Type: "secret", Secret: true},
		},
	}
}

func (FeishuPlugin) Send(ctx context.Context, cfg map[string]string, n Notification) error {
	payload := map[string]interface{}{
		"msg_type": "text",
		"content":  map[string]string{"text": buildText(n)},
	}
	if secret := cfg["secret"]; secret != "" {
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		payload["timestamp"] = ts
		payload["sign"] = hmacSHA256B64(secret, ts+"\n"+secret)
	}
	return postJSON(ctx, cfg["webhook_url"], payload)
}

func (f FeishuPlugin) Test(ctx context.Context, cfg map[string]string) error {
	return f.Send(ctx, cfg, testNotification())
}

// WeComPlugin 企业微信群机器人通知。
type WeComPlugin struct{}

func (WeComPlugin) Meta() PluginMeta {
	return PluginMeta{
		ID:          "notify-wecom",
		Type:        PluginTypeNotifier,
		Name:        "企业微信群机器人",
		Description: "向企业微信群机器人发送告警",
		ConfigFields: []ConfigField{
			{Key: "webhook_key", Label: "Webhook Key", Type: "secret", Required: true, Secret: true},
		},
	}
}

func (WeComPlugin) Send(ctx context.Context, cfg map[string]string, n Notification) error {
	url := "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=" + cfg["webhook_key"]
	return postJSON(ctx, url, map[string]interface{}{
		"msgtype": "text",
		"text":    map[string]string{"content": buildText(n)},
	})
}

func (w WeComPlugin) Test(ctx context.Context, cfg map[string]string) error {
	return w.Send(ctx, cfg, testNotification())
}

// SlackPlugin Slack Incoming Webhook 通知。
type SlackPlugin struct{}

func (SlackPlugin) Meta() PluginMeta {
	return PluginMeta{
		ID:          "notify-slack",
		Type:        PluginTypeNotifier,
		Name:        "Slack",
		Description: "向 Slack Incoming Webhook 发送告警",
		ConfigFields: []ConfigField{
			{Key: "webhook_url", Label: "Webhook 地址", Type: "secret", Required: true, Secret: true},
		},
	}
}

func (SlackPlugin) Send(ctx context.Context, cfg map[string]string, n Notification) error {
	return postJSON(ctx, cfg["webhook_url"], map[string]interface{}{"text": buildText(n)})
}

func (s SlackPlugin) Test(ctx context.Context, cfg map[string]string) error {
	return s.Send(ctx, cfg, testNotification())
}

// SMTPPlugin 邮件通知。
type SMTPPlugin struct{}

func (SMTPPlugin) Meta() PluginMeta {
	return PluginMeta{
		ID:          "notify-smtp",
		Type:        PluginTypeNotifier,
		Name:        "邮件（SMTP）",
		Description: "通过 SMTP 服务器发送告警邮件",
		ConfigFields: []ConfigField{
			{Key: "host", Label: "SMTP 服务器", Type: "string", Required: true},
			{Key: "port", Label: "端口", Type: "number", Default: "465"},
			{Key: "username", Label: "用户名", Type: "string"},
			{Key: "password", Label: "密码", Type: "secret", Secret: true},
			{Key: "from", Label: "发件人", Type: "string", Required: true},
			{Key: "to", Label: "收件人（逗号分隔）", Type: "string", Required: true},
		},
	}
}

func (SMTPPlugin) Send(ctx context.Context, cfg map[string]string, n Notification) error {
	addr := cfg["host"]
	port := cfg["port"]
	if port == "" {
		port = "465"
	}
	hostport := addr + ":" + port

	subject := fmt.Sprintf("[MSMP] %s - %s", levelLabel(n.Level), n.Title)
	body := buildText(n)

	msg := "From: " + cfg["from"] + "\r\n" +
		"To: " + cfg["to"] + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		body

	recipients := strings.Split(cfg["to"], ",")
	for i := range recipients {
		recipients[i] = strings.TrimSpace(recipients[i])
	}

	auth := smtp.PlainAuth("", cfg["username"], cfg["password"], addr)
	if err := smtpSendMail(ctx, hostport, auth, cfg["from"], recipients, []byte(msg)); err != nil {
		return err
	}
	return nil
}

// smtpSendMail 使用隐式 TLS（465）发送邮件。
func smtpSendMail(ctx context.Context, hostport string, auth smtp.Auth, from string, to []string, msg []byte) error {
	dialer := &net.Dialer{Timeout: notifyTimeout}
	rawConn, err := dialer.DialContext(ctx, "tcp", hostport)
	if err != nil {
		return err
	}
	conn := tls.Client(rawConn, &tls.Config{ServerName: hostFromAddr(hostport)})
	defer conn.Close()
	if err := conn.HandshakeContext(ctx); err != nil {
		return err
	}

	c, err := smtp.NewClient(conn, hostFromAddr(hostport))
	if err != nil {
		return err
	}
	defer c.Quit()

	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return err
		}
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	for _, r := range to {
		if err := c.Rcpt(r); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return nil
}

func hostFromAddr(hostport string) string {
	if i := strings.LastIndex(hostport, ":"); i >= 0 {
		return hostport[:i]
	}
	return hostport
}

func (s SMTPPlugin) Test(ctx context.Context, cfg map[string]string) error {
	return s.Send(ctx, cfg, testNotification())
}