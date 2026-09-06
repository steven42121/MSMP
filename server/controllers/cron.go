// Package controllers 提供 Cron 调度 API。
package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"MSMP/server/db"
	"MSMP/server/models"
	"MSMP/server/services"
)

// CronJobsHandler GET /api/cron-jobs 列表，POST /api/cron-jobs 创建
func CronJobsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	switch r.Method {
	case http.MethodGet:
		var jobs []models.CronJob
		db.DB.Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&jobs)
		writeJSON(w, http.StatusOK, jobs)

	case http.MethodPost:
		var job models.CronJob
		if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		job.TenantID = tenantID
		job.Enabled = true

		// 验证 cron 表达式
		if _, err := services.NextRunAt(job.Expression); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid cron expression: " + err.Error()})
			return
		}

		next, _ := services.NextRunAt(job.Expression)
		job.NextRunAt = &next
		db.DB.Create(&job)

		// 注册到调度器
		sched := services.GetCronScheduler()
		sched.AddJob(job.ID, job.Expression, services.CronWithContext(r.Context(), 30*time.Second, func(ctx context.Context) {
			executeCronJob(ctx, &job)
		}))
		writeJSON(w, http.StatusCreated, job)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// CronJobDetailHandler PUT/DELETE /api/cron-jobs/{id}
func CronJobDetailHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/api/cron-jobs/"):]
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var job models.CronJob
	if err := db.DB.Where("id = ? AND tenant_id = ?", id, getTenantID(r)).First(&job).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
		return
	}

	switch r.Method {
	case http.MethodPut:
		var updates map[string]interface{}
		json.NewDecoder(r.Body).Decode(&updates)
		db.DB.Model(&job).Updates(updates)
		writeJSON(w, http.StatusOK, job)

	case http.MethodDelete:
		services.GetCronScheduler().RemoveJob(job.ID)
		db.DB.Delete(&job)
		writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// CronJobRunHandler POST /api/cron-jobs/{id}/run 手动执行一次
func CronJobRunHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/api/cron-jobs/"):]
	idStr = idStr[len("/run"):]
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var job models.CronJob
	if err := db.DB.Where("id = ? AND tenant_id = ?", id, getTenantID(r)).First(&job).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
		return
	}

 executeCronJob(r.Context(), &job)
 writeJSON(w, http.StatusOK, map[string]string{"message": "executed"})
}

// ExecuteCronJob 公开接口：执行 Cron 任务。
func ExecuteCronJob(ctx context.Context, job *models.CronJob) {
	executeCronJob(ctx, job)
}

// executeCronJob 根据 target_type 执行对应的任务。
func executeCronJob(ctx context.Context, job *models.CronJob) {
	switch job.TargetType {
	case "host":
		// TODO: 触发主机相关任务（如探活、数据采集）
	case "probe":
		// TODO: 触发可用性探测
	case "shell":
		// TODO: 在指定主机上执行 shell 命令
	default:
		// 默认：记录日志
	}

	now := time.Now()
	db.DB.Model(job).Updates(map[string]interface{}{
		"last_run_at": now,
		"next_run_at": computeNextRun(job.Expression),
	})
}

func computeNextRun(expr string) *time.Time {
	next, err := services.NextRunAt(expr)
	if err != nil {
		return nil
	}
	return &next
}
