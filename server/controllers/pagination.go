package controllers

import (
	"net/http"
	"strconv"

	"gorm.io/gorm"
)

// pagination 统一列表分页：解析参数、计数、应用偏移并输出标准响应。
type pagination struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

// newPagination 解析 page/page_size 查询参数（默认 1/20，单页上限 100）。
func newPagination(r *http.Request) pagination {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return pagination{Page: page, PageSize: pageSize}
}

// count 统计当前查询条件下的总数。
func (p *pagination) count(query *gorm.DB, model interface{}) {
	query.Model(model).Count(&p.Total)
}

// scope 为已排序的查询应用 Offset/Limit。
func (p pagination) scope(query *gorm.DB) *gorm.DB {
	return query.Offset((p.Page - 1) * p.PageSize).Limit(p.PageSize)
}

// write 输出 {data,total,page,page_size} 标准分页响应。
func (p pagination) write(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": data, "total": p.Total, "page": p.Page, "page_size": p.PageSize,
	})
}
