// Package plugins 提供 MSMP 通用插件框架。
// 插件采用内置注册编译方式：定义统一元数据与分类型接口，编译进主程序，
// 保证跨平台分发稳定，后续可平滑升级为动态加载。
package plugins

import (
	"context"
	"sync"
)

type PluginType string

const (
	PluginTypeNotifier  PluginType = "notifier"
	PluginTypeCollector PluginType = "collector"
	PluginTypeProbe     PluginType = "probe"
)

// ConfigField 描述插件实例的一个配置字段，供前端动态渲染。
type ConfigField struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Type     string `json:"type"` // string | number | bool | secret | textarea
	Required bool   `json:"required"`
	Secret   bool   `json:"secret"`
	Default  string `json:"default,omitempty"`
}

// PluginMeta 是插件的静态元数据。
type PluginMeta struct {
	ID           string        `json:"id"`
	Type         PluginType    `json:"type"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	ConfigFields []ConfigField `json:"config_fields"`
}

// Notification 是发送给通知插件的一条事件消息。
type Notification struct {
	Title    string `json:"title"`
	Message  string `json:"message"`
	Level    string `json:"level"`
	Hostname string `json:"hostname"`
	HostUUID string `json:"host_uuid"`
	Time     string `json:"time"`
}

// NotifierPlugin 是通知渠道插件的运行接口。
type NotifierPlugin interface {
	Meta() PluginMeta
	Send(ctx context.Context, cfg map[string]string, n Notification) error
	Test(ctx context.Context, cfg map[string]string) error
}

// Registry 是分类型插件注册中心。
type Registry struct {
	mu        sync.RWMutex
	notifiers map[string]NotifierPlugin
	metas     []PluginMeta
}

func NewRegistry() *Registry {
	return &Registry{notifiers: map[string]NotifierPlugin{}}
}

// RegisterNotifier 登记一个通知插件，并收录其元数据。
func (r *Registry) RegisterNotifier(p NotifierPlugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notifiers[p.Meta().ID] = p
	r.metas = append(r.metas, p.Meta())
}

// RegisterMeta 登记一个仅用于展示的插件元数据（如 collector/probe）。
func (r *Registry) RegisterMeta(m PluginMeta) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.metas = append(r.metas, m)
}

// List 返回全部已登记插件的元数据。
func (r *Registry) List() []PluginMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]PluginMeta, len(r.metas))
	copy(out, r.metas)
	return out
}

// GetNotifier 返回指定 ID 的通知插件。
func (r *Registry) GetNotifier(id string) (NotifierPlugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.notifiers[id]
	return p, ok
}

// Meta 返回指定 ID 的插件元数据。
func (r *Registry) Meta(id string) (PluginMeta, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, m := range r.metas {
		if m.ID == id {
			return m, true
		}
	}
	return PluginMeta{}, false
}