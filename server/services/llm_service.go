package services

import (
	"fmt"
	"strings"
	"time"

	"MSMP/server/db"
	"MSMP/server/models"
)

const (
	systemPrompt = `你是 MSMP 智能运维助手，负责服务器监控和运维支持。

你的职责：
1. 分析告警并提供根因诊断和修复建议
2. 回答关于系统运行状态的问题
3. 根据指标数据生成运维报告
4. 提供性能优化建议

回答要求：
- 使用中文回答
- 内容简洁专业，避免冗余
- 给出可操作的修复步骤
- 必要时提供命令或配置建议
- 涉及数据时引用具体数值`
)

// AnalyzeAlert 对指定告警进行 AI 根因分析
func AnalyzeAlert(eventID uint, hostID uint) (string, error) {
	var event models.HostEvent
	if err := db.DB.First(&event, eventID).Error; err != nil {
		return "", fmt.Errorf("告警不存在: %w", err)
	}

	var host models.Host
	if err := db.DB.First(&host, hostID).Error; err != nil {
		return "", fmt.Errorf("主机不存在: %w", err)
	}

	var recentMetrics []models.MetricSample
	db.DB.Where("host_id = ?", hostID).
		Order("timestamp DESC").
		Limit(5).
		Find(&recentMetrics)

	metricsLines := "无近期指标数据"
	if len(recentMetrics) > 0 {
		parts := make([]string, 0, len(recentMetrics))
		for _, m := range recentMetrics {
			diskPct := 0.0
			if m.DiskTotal > 0 {
				diskPct = float64(m.DiskUsed) / float64(m.DiskTotal) * 100
			}
			parts = append(parts, fmt.Sprintf(
				"%s CPU=%.1f%% MEM=%.1f%% DISK=%.1f%% LOAD=%.2f",
				m.Timestamp.Format("15:04:05"), m.CPUPercent, m.MemPercent, diskPct, m.Load1,
			))
		}
		metricsLines = strings.Join(parts, "\n")
	}

	hostInfo := fmt.Sprintf("主机: %s (%s) OS: %s", host.Hostname, host.IP, host.OS)
	prompt := fmt.Sprintf(`请对以下告警进行根因分析并提供修复建议：

告警类型：%s
告警级别：%s
告警消息：%s
%s
最近指标趋势：
%s`, event.Type, event.Level, event.Message, hostInfo, metricsLines)

	client := buildLLMClient()
	if client == nil {
		return "", fmt.Errorf("LLM 未配置，请在系统设置中配置 LLM 服务")
	}

	result, err := client.Chat([]LLMMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return "", err
	}
	return result, nil
}

// ChatQuery 处理用户的自然语言查询
func ChatQuery(query string, tenantID uint) (string, error) {
	var online, offline, pending int64
	db.DB.Model(&models.Host{}).Where("tenant_id = ? AND status = ?", tenantID, "online").Count(&online)
	db.DB.Model(&models.Host{}).Where("tenant_id = ? AND status = ?", tenantID, "offline").Count(&offline)
	db.DB.Model(&models.Host{}).Where("tenant_id = ? AND status = ?", tenantID, "pending").Count(&pending)

	var alertCount int64
	db.DB.Model(&models.HostEvent{}).
		Where("tenant_id = ? AND type = 'alert' AND created_at > ?", tenantID, time.Now().Add(-24*time.Hour)).
		Count(&alertCount)

	var hosts []models.Host
	db.DB.Where("tenant_id = ? AND status = 'online'", tenantID).Limit(8).Find(&hosts)

	hostsInfo := "暂无在线主机"
	if len(hosts) > 0 {
		parts := make([]string, 0, len(hosts))
		for _, h := range hosts {
			parts = append(parts, fmt.Sprintf("- %s (%s)", h.Hostname, h.IP))
		}
		hostsInfo = strings.Join(parts, "\n")
	}

	context := fmt.Sprintf(`当前系统概况：
- 在线主机：%d 台，离线：%d 台，待接入：%d 台
- 近 24 小时告警：%d 条
- 在线主机列表：%s`, online, offline, pending, alertCount, hostsInfo)

	client := buildLLMClient()
	if client == nil {
		return "", fmt.Errorf("LLM 未配置，请在系统设置中配置 LLM 服务")
	}

	result, err := client.Chat([]LLMMessage{
		{Role: "system", Content: systemPrompt + "\n\n" + context},
		{Role: "user", Content: query},
	})
	if err != nil {
		return "", err
	}
	return result, nil
}

// GenerateReport 生成系统健康报告
func GenerateReport(tenantID uint) (string, error) {
	var total, online, offline, pending int64
	db.DB.Model(&models.Host{}).Where("tenant_id = ?", tenantID).Count(&total)
	db.DB.Model(&models.Host{}).Where("tenant_id = ? AND status = ?", tenantID, "online").Count(&online)
	db.DB.Model(&models.Host{}).Where("tenant_id = ? AND status = ?", tenantID, "offline").Count(&offline)
	db.DB.Model(&models.Host{}).Where("tenant_id = ? AND status = ?", tenantID, "pending").Count(&pending)

	var alertCritical, alertWarning int64
	db.DB.Model(&models.HostEvent{}).
		Where("tenant_id = ? AND type = 'alert' AND level = 'critical' AND created_at > ?", tenantID, time.Now().Add(-24*time.Hour)).
		Count(&alertCritical)
	db.DB.Model(&models.HostEvent{}).
		Where("tenant_id = ? AND type = 'alert' AND level = 'warning' AND created_at > ?", tenantID, time.Now().Add(-24*time.Hour)).
		Count(&alertWarning)

	var recentAlerts []models.HostEvent
	db.DB.Where("tenant_id = ? AND type = 'alert' AND created_at > ?", tenantID, time.Now().Add(-24*time.Hour)).
		Order("created_at DESC").Limit(8).Find(&recentAlerts)

	alertsInfo := "近 24 小时无告警"
	if len(recentAlerts) > 0 {
		parts := make([]string, 0, len(recentAlerts))
		for _, a := range recentAlerts {
			parts = append(parts, fmt.Sprintf("[%s] %s", a.Level, a.Message))
		}
		alertsInfo = strings.Join(parts, "\n")
	}

	prompt := fmt.Sprintf(`请根据以下数据生成今日系统运维简报：

系统规模：共 %d 台主机，在线 %d 台，离线 %d 台，待接入 %d 台
近 24 小时告警：严重 %d 条，警告 %d 条

最近告警明细：
%s

请生成简洁的运维简报，包含：
1. 系统整体健康状况评估
2. 需要重点关注的问题
3. 建议的运维操作`, total, online, offline, pending, alertCritical, alertWarning, alertsInfo)

	client := buildLLMClient()
	if client == nil {
		return "", fmt.Errorf("LLM 未配置")
	}

	result, err := client.Chat([]LLMMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return "", err
	}
	return result, nil
}

func buildLLMClient() *LLMClient {
	baseURL, apiKey, model := GetLLMConfig()
	if baseURL == "" || apiKey == "" {
		return nil
	}
	return NewLLMClient(baseURL, apiKey, model)
}
