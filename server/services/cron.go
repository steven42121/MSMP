// Package services 提供 Cron 调度器。
package services

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// CronScheduler 管理定时任务调度。
type CronScheduler struct {
	mu      sync.Mutex
	cron    *cron.Cron
	entries map[uint]cron.EntryID // cronJobID -> entryID
}

var globalScheduler *CronScheduler

// GetCronScheduler 获取全局调度器单例。
func GetCronScheduler() *CronScheduler {
	if globalScheduler == nil {
		globalScheduler = &CronScheduler{
			cron:    cron.New(cron.WithSeconds()),
			entries: make(map[uint]cron.EntryID),
		}
		globalScheduler.cron.Start()
	}
	return globalScheduler
}

// Stop 停止调度器。
func (s *CronScheduler) Stop() {
	if s.cron != nil {
		s.cron.Stop()
	}
}

// AddJob 添加一个 Cron 任务。
func (s *CronScheduler) AddJob(jobID uint, expr string, fn func()) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.entries[jobID]; ok {
		s.cron.Remove(existing)
	}

	id, err := s.cron.AddJob(expr, cron.FuncJob(fn))
	if err != nil {
		return err
	}
	s.entries[jobID] = id
	next := s.cron.Entry(id).Next
	log.Printf("[Cron] job %d added: expr=%s next=%s", jobID, expr, next.Format("2006-01-02 15:04:05"))
	return nil
}

// RemoveJob 移除一个 Cron 任务。
func (s *CronScheduler) RemoveJob(jobID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, ok := s.entries[jobID]; ok {
		s.cron.Remove(id)
		delete(s.entries, jobID)
	}
}

// NextRunAt 返回表达式下次运行时间。
func NextRunAt(expr string) (time.Time, error) {
	c := cron.New(cron.WithSeconds())
	defer c.Stop()
	id, err := c.AddFunc(expr, func() {})
	if err != nil {
		return time.Time{}, err
	}
	entry := c.Entry(id)
	return entry.Next, nil
}

// ParseCronExpr 验证 cron 表达式合法性。
func ParseCronExpr(expr string) error {
	c := cron.New(cron.WithSeconds())
	_, err := c.AddFunc(expr, func() {})
	return err
}

// CronWithContext 包装函数，传入 context 超时控制。
func CronWithContext(ctx context.Context, timeout time.Duration, fn func(context.Context)) func() {
	return func() {
		childCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		fn(childCtx)
	}
}
