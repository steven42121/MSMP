package controllers

import (
	"fmt"
	"time"

	"MSMP/server/services"

	"golang.org/x/crypto/ssh"
)

// dialHostSSH 建立到指定主机的 SSH 连接（复用 SSHBinding 解析逻辑）
func dialHostSSH(tenantID, hostID uint) (*ssh.Client, error) {
	credSvc := services.GlobalCredSvc
	if credSvc == nil {
		return nil, fmt.Errorf("凭证服务未初始化")
	}

	binding, authMethod, hostAddr, err := resolveSSHBinding(tenantID, hostID, credSvc)
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		User:            binding.Username,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	return ssh.Dial("tcp", hostAddr, config)
}