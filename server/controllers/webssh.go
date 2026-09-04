package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"MSMP/server/db"
	"MSMP/server/models"
	"MSMP/server/services"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// SSHRequest WebSocket 升级时客户端发送的首条消息
type SSHRequest struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// WebSSHHandler 处理 /api/hosts/{uuid}/ssh 的 WebSocket 升级并代理到 SSH
// 由 HostDetailHandler 在解析出 host 后调用
func WebSSHHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {
	// 升级为 WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WebSSH] upgrade failed: %v", err)
		return
	}
	defer ws.Close()

	// 读取客户端初始消息（终端尺寸）
	_, msgBytes, err := ws.ReadMessage()
	if err != nil {
		return
	}
	var req SSHRequest
	if err := json.Unmarshal(msgBytes, &req); err != nil {
		req.Width = 80
		req.Height = 24
	}
	if req.Width <= 0 {
		req.Width = 80
	}
	if req.Height <= 0 {
		req.Height = 24
	}

	// 查找 SSH 凭证
	credSvc := services.GlobalCredSvc
	if credSvc == nil {
		sendWSClose(ws, "凭证服务未初始化")
		return
	}

	binding, authMethod, hostAddr, err := resolveSSHBinding(tenantID, host.ID, credSvc)
	if err != nil {
		sendWSClose(ws, err.Error())
		return
	}

	// 建立 SSH 连接
	sshConfig := &ssh.ClientConfig{
		User:            binding.Username,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	sshClient, err := ssh.Dial("tcp", hostAddr, sshConfig)
	if err != nil {
		sendWSClose(ws, fmt.Sprintf("SSH 连接失败: %v", err))
		return
	}
	defer sshClient.Close()

	// 创建会话
	session, err := sshClient.NewSession()
	if err != nil {
		sendWSClose(ws, fmt.Sprintf("SSH 会话创建失败: %v", err))
		return
	}
	defer session.Close()

	// 请求 PTY
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", req.Height, req.Width, modes); err != nil {
		sendWSClose(ws, fmt.Sprintf("PTY 请求失败: %v", err))
		return
	}

	// 获取管道
	stdin, err := session.StdinPipe()
	if err != nil {
		sendWSClose(ws, fmt.Sprintf("stdin 管道失败: %v", err))
		return
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		sendWSClose(ws, fmt.Sprintf("stdout 管道失败: %v", err))
		return
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		sendWSClose(ws, fmt.Sprintf("stderr 管道失败: %v", err))
		return
	}

	// 启动 shell
	if err := session.Shell(); err != nil {
		sendWSClose(ws, fmt.Sprintf("shell 启动失败: %v", err))
		return
	}

	// 审计日志
	db.DB.Create(&models.AuditLog{
		TenantID: tenantID,
		UserID:   userID,
		Action:   "webssh_connect",
		Resource: fmt.Sprintf("host:%d", host.ID),
		Status:   200,
	})
	log.Printf("[WebSSH] user=%d tenant=%d host=%d (%s) connected", userID, tenantID, host.ID, host.Hostname)

	// 双向管道
	done := make(chan struct{})
	var once sync.Once
	closeDone := func() { once.Do(func() { close(done) }) }

	// SSH 输出 → WebSocket
	copyOutput := func(pipe io.Reader) {
		defer closeDone()
		buf := make([]byte, 4096)
		for {
			n, err := pipe.Read(buf)
			if n > 0 {
				ws.SetWriteDeadline(time.Now().Add(30 * time.Second))
				if werr := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}
	go copyOutput(stdout)
	go copyOutput(stderr)

	// WebSocket → SSH stdin
	go func() {
		defer closeDone()
		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				return
			}
			// 检测 resize 消息
			if len(msg) > 0 && msg[0] == '{' {
				var resize struct {
					Type   string `json:"type"`
					Width  int    `json:"width"`
					Height int    `json:"height"`
				}
				if json.Unmarshal(msg, &resize) == nil && resize.Type == "resize" && resize.Width > 0 && resize.Height > 0 {
					session.WindowChange(resize.Height, resize.Width)
					continue
				}
			}
			if _, err := stdin.Write(msg); err != nil {
				return
			}
		}
	}()

	<-done

	ws.SetWriteDeadline(time.Now().Add(2 * time.Second))
	ws.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "session ended"))

	log.Printf("[WebSSH] user=%d tenant=%d host=%d disconnected", userID, tenantID, host.ID)
}

// resolveSSHBinding 根据 hostID 查找 SSH 渠道配置并解密凭证
func resolveSSHBinding(tenantID, hostID uint, credSvc *services.CredentialService) (*models.ChannelBinding, ssh.AuthMethod, string, error) {
	var binding models.ChannelBinding
	if err := db.DB.Where("host_id = ? AND tenant_id = ? AND type = 'ssh' AND enabled = true", hostID, tenantID).
		Order("priority ASC").First(&binding).Error; err != nil {
		return nil, nil, "", fmt.Errorf("主机未配置 SSH 渠道")
	}

	credential, err := credSvc.Decrypt(binding.Credential)
	if err != nil {
		return nil, nil, "", fmt.Errorf("凭证解密失败: %v", err)
	}

	var authMethod ssh.AuthMethod
	switch binding.AuthMode {
	case "password":
		authMethod = ssh.Password(credential)
	case "private_key", "generated_key":
		signer, err := ssh.ParsePrivateKey([]byte(credential))
		if err != nil {
			return nil, nil, "", fmt.Errorf("私钥解析失败: %v", err)
		}
		authMethod = ssh.PublicKeys(signer)
	default:
		return nil, nil, "", fmt.Errorf("不支持的认证模式: %s", binding.AuthMode)
	}

	return &binding, authMethod, binding.Address, nil
}

// sendWSClose 发送 WebSocket close frame
func sendWSClose(ws *websocket.Conn, reason string) {
	ws.SetWriteDeadline(time.Now().Add(2 * time.Second))
	ws.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.ClosePolicyViolation, reason))
}