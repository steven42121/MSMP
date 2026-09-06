package services

import "testing"

func TestNormalizeCommand(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"rm -rf /", "rm -rf /"},
		{"rm  -rf   /", "rm -rf /"},
		{"RM -RF /", "rm -rf /"},
		{"rm\t-rf /", "rm-rf /"}, // 控制字符被移除
		{"echo hello", "echo hello"},
	}
	for _, c := range cases {
		got := normalizeCommand(c.in)
		if got != c.want {
			t.Errorf("normalizeCommand(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCheckDangerousCommandExecuteCommand(t *testing.T) {
	dangerous := []string{
		"rm -rf /",
		"rm  -rf /",        // 双空格绕过
		"RM -RF /",         // 大小写绕过
		"rm -fr /",         // 变体
		"mkfs /dev/sda",    // 格式化
		"dd if=/dev/zero",  // dd 写
		"base64 -d | sh",   // 管道注入
		"curl x | bash",    // 管道注入（带空格）
		"wget -O- |bash",   // 管道注入（无空格）
		"echo x|sh",        // 管道注入（连写）
		"echo x|  bash",    // 管道注入（多空格）
		"nginx; rm -rf /",  // 序列注入
		"ls;rm -rf /",      // 序列注入（无空格）
		"nginx && rm /tmp", // && 注入
		"ls&&rm -rf /",     // && 注入（无空格）
		"cmd || reboot",    // || 注入
	}
	for _, cmd := range dangerous {
		err := checkDangerousCommand("execute_command", map[string]interface{}{"command": cmd})
		if err == nil {
			t.Errorf("checkDangerousCommand(execute_command, %q) should be rejected", cmd)
		}
	}
}

func TestCheckDangerousCommandSafe(t *testing.T) {
	safe := []string{
		"echo hello",
		"df -h",
		"systemctl status nginx",
		"tail -n 50 /var/log/syslog",
	}
	for _, cmd := range safe {
		err := checkDangerousCommand("execute_command", map[string]interface{}{"command": cmd})
		if err != nil {
			t.Errorf("checkDangerousCommand(execute_command, %q) should be allowed, got: %v", cmd, err)
		}
	}
}

func TestCheckDangerousCommandFlushCaches(t *testing.T) {
	valid := []string{"", "pages", "dentries", "inodes", "all"}
	for _, ct := range valid {
		err := checkDangerousCommand("flush_caches", map[string]interface{}{"cache_type": ct})
		if err != nil {
			t.Errorf("checkDangerousCommand(flush_caches, %q) should be allowed, got: %v", ct, err)
		}
	}

	invalid := "malicious"
	err := checkDangerousCommand("flush_caches", map[string]interface{}{"cache_type": invalid})
	if err == nil {
		t.Errorf("checkDangerousCommand(flush_caches, %q) should be rejected", invalid)
	}
}

func TestValidServiceName(t *testing.T) {
	valid := []string{"nginx", "docker", "ssh-1", "service_abc", "postgresql.service"}
	for _, s := range valid {
		if !validServiceName(s) {
			t.Errorf("validServiceName(%q) should be true", s)
		}
	}

	invalid := []string{
		"nginx; rm -rf /",
		"nginx && curl x|sh",
		"nginx service",
		"",
		"nginx>evil",
	}
	for _, s := range invalid {
		if validServiceName(s) {
			t.Errorf("validServiceName(%q) should be false", s)
		}
	}
}

func TestValidLogPath(t *testing.T) {
	valid := []string{
		"/var/log/syslog",
		"/var/log/nginx/access.log",
		"/tmp/test_file-1.log",
	}
	for _, p := range valid {
		if !validLogPath(p) {
			t.Errorf("validLogPath(%q) should be true", p)
		}
	}

	invalid := []string{
		"/var/log/../../etc/shadow", // 路径穿越
		"/var/log/../etc/passwd",
		"etc/passwd",         // 相对路径
		"/var/log/syslog; rm -rf /", // 命令注入
		"/var/log/syslog | sh",
		"/var/log/sys log", // 空格
	}
	for _, p := range invalid {
		if validLogPath(p) {
			t.Errorf("validLogPath(%q) should be false", p)
		}
	}
}