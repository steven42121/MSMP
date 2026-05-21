package common

// CollectAssetInfoFull 由各平台实现（win/collect.go 等）
var CollectAssetInfoFull func() AgentInfo

// CollectMetrics 由各平台实现
var CollectMetrics func() MetricData