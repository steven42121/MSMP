package controllers

import (
	"log"
	"time"

	"MSMP/server/config"
	"MSMP/server/db"
	"MSMP/server/models"
)

// StartDownsampleCleanup 启动降采样与保留策略清理后台任务，每 15 分钟运行一次。
func StartDownsampleCleanup() {
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			runDownsampleCleanup()
		}
	}()
}

func runDownsampleCleanup() {
	cfg := config.C.Retention
	rawRetentionDays := cfg.RawRetentionDays
	downsampleAtDays := cfg.DownsampleAtDays
	intervalMin := cfg.DownsampleInterval
	if intervalMin <= 0 {
		intervalMin = 5
	}
	if downsampleAtDays <= 0 {
		downsampleAtDays = 7
	}
	if rawRetentionDays <= 0 {
		rawRetentionDays = 90
	}

	cutoffRaw := time.Now().Add(-time.Duration(rawRetentionDays) * 24 * time.Hour)
	cutoffDownsample := time.Now().Add(-time.Duration(downsampleAtDays) * 24 * time.Hour)

	// 1. 对超过 downsampleAtDays 的原始样本做降采样聚合，写入 metric_downsamples
	migrated, err := downsampleOldSamples(cutoffDownsample, intervalMin)
	if err != nil {
		log.Printf("[Retention] downsample error: %v", err)
	} else if migrated > 0 {
		log.Printf("[Retention] downsampled %d raw samples into hourly buckets", migrated)
	}

	// 2. 清理超过 rawRetentionDays 的原始样本（已被降采样过）
	deletedRaw, err := deleteOldRawSamples(cutoffRaw)
	if err != nil {
		log.Printf("[Retention] delete raw samples error: %v", err)
	} else if deletedRaw > 0 {
		log.Printf("[Retention] deleted %d expired raw metric samples", deletedRaw)
	}

	// 3. 清理超过 rawRetentionDays 的降采样数据
	deletedDS, err := deleteOldDownsampled(cutoffRaw)
	if err != nil {
		log.Printf("[Retention] delete downsampled data error: %v", err)
	} else if deletedDS > 0 {
		log.Printf("[Retention] deleted %d expired downsampled records", deletedDS)
	}
}

// downsampleOldSamples 将 cutoffBefore 之前的原始样本按 intervalMin 分钟聚合到 metric_downsamples。
func downsampleOldSamples(cutoffBefore time.Time, intervalMin int) (int64, error) {
	type bucketKey struct {
		tenantID uint
		hostID   uint
		bucket   time.Time
	}
	type bucketValue struct {
		cpuSum    float64
		memPctSum float64
		memUsed   uint64
		memTotal  uint64
		load1     float64
		load5     float64
		load15    float64
		cpuCount  int
		process   int
	}

	// 先找出所有需要聚合的主机（去重）
	var hosts []struct {
		HostID   uint
		TenantID uint
	}
	if err := db.DB.Table("metric_samples").
		Select("DISTINCT host_id, tenant_id").
		Where("timestamp < ?", cutoffBefore).
		Find(&hosts).Error; err != nil {
		return 0, err
	}

	if len(hosts) == 0 {
		return 0, nil
	}

	totalMigrated := int64(0)
	interval := time.Duration(intervalMin) * time.Minute

	for _, h := range hosts {
		// 读取该主机所有待降采样的样本
		var samples []models.MetricSample
		if err := db.DB.Where("host_id = ? AND timestamp < ?", h.HostID, cutoffBefore).
			Order("timestamp ASC").Find(&samples).Error; err != nil {
			log.Printf("[Retention] read samples for host %d: %v", h.HostID, err)
			continue
		}
		if len(samples) == 0 {
			continue
		}

		// 按 bucket 分组聚合
		buckets := make(map[bucketKey]*bucketValue)
		for _, s := range samples {
			bt := s.Timestamp.Truncate(interval)
			k := bucketKey{tenantID: h.TenantID, hostID: h.HostID, bucket: bt}
			v, ok := buckets[k]
			if !ok {
				v = &bucketValue{}
				buckets[k] = v
			}
			v.cpuSum += s.CPUPercent
			v.memPctSum += s.MemPercent
			v.memUsed += s.MemUsed
			v.memTotal += s.MemTotal
			v.load1 += s.Load1
			v.load5 += s.Load5
			v.load15 += s.Load15
			v.cpuCount++
			if s.ProcessCount > v.process {
				v.process = s.ProcessCount
			}
		}

		// 批量插入降采样数据
		downsamples := make([]models.MetricDownsample, 0, len(buckets))
		for k, v := range buckets {
			n := float64(v.cpuCount)
			downsamples = append(downsamples, models.MetricDownsample{
				TenantID:     k.tenantID,
				HostID:       k.hostID,
				Timestamp:    k.bucket,
				CPUPercent:   v.cpuSum / n,
				MemPercent:   v.memPctSum / n,
				MemUsed:      v.memUsed / uint64(v.cpuCount),
				MemTotal:     v.memTotal / uint64(v.cpuCount),
				Load1:        v.load1 / n,
				Load5:        v.load5 / n,
				Load15:       v.load15 / n,
				ProcessCount: v.process,
			})
		}
		if err := db.DB.Create(&downsamples).Error; err != nil {
			log.Printf("[Retention] insert downsample for host %d: %v", h.HostID, err)
			continue
		}
		totalMigrated += int64(len(downsamples))

		// 删除已降采样的原始样本
		if err := db.DB.Where("host_id = ? AND timestamp < ?", h.HostID, cutoffBefore).
			Delete(&models.MetricSample{}).Error; err != nil {
			log.Printf("[Retention] delete raw samples for host %d: %v", h.HostID, err)
		}
	}

	return totalMigrated, nil
}

// deleteOldRawSamples 删除超过保留期的原始指标样本。
func deleteOldRawSamples(before time.Time) (int64, error) {
	result := db.DB.Where("timestamp < ?", before).Delete(&models.MetricSample{})
	return result.RowsAffected, result.Error
}

// deleteOldDownsampled 删除超过保留期的降采样记录。
func deleteOldDownsampled(before time.Time) (int64, error) {
	result := db.DB.Where("timestamp < ?", before).Delete(&models.MetricDownsample{})
	return result.RowsAffected, result.Error
}
