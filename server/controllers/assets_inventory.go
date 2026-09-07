// Package controllers 提供主机资产清单 API（进程/端口/软件包）。
package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"MSMP/server/db"
	"MSMP/server/models"
)

// HostProcessesHandler GET /api/hosts/{uuid}/assets/processes
func HostProcessesHandler(w http.ResponseWriter, r *http.Request) {
	uuid := strings.TrimPrefix(r.URL.Path, "/api/hosts/")
	uuid = strings.SplitN(uuid, "/", 2)[0]
	if uuid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "uuid required"})
		return
	}

	var host models.Host
	if err := db.DB.Where("uuid = ?", uuid).First(&host).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "host not found"})
		return
	}

	limit := 200
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}

	var procs []models.HostProcess
	q := db.DB.Where("host_id = ?", host.ID).Order("mem_percent DESC, cpu_percent DESC")
	if limit > 0 {
		q = q.Limit(limit).Offset(offset)
	}
	q.Find(&procs)

	var total int64
	db.DB.Model(&models.HostProcess{}).Where("host_id = ?", host.ID).Count(&total)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"host_uuid": uuid,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
		"processes": procs,
	})
}

// HostPortsHandler GET /api/hosts/{uuid}/assets/ports
func HostPortsHandler(w http.ResponseWriter, r *http.Request) {
	uuid := strings.TrimPrefix(r.URL.Path, "/api/hosts/")
	uuid = strings.SplitN(uuid, "/", 2)[0]
	if uuid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "uuid required"})
		return
	}

	var host models.Host
	if err := db.DB.Where("uuid = ?", uuid).First(&host).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "host not found"})
		return
	}

	limit := 500
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 2000 {
			limit = n
		}
	}

	var ports []models.HostPort
	db.DB.Where("host_id = ?", host.ID).
		Order("local_port ASC").
		Limit(limit).
		Find(&ports)

	var total int64
	db.DB.Model(&models.HostPort{}).Where("host_id = ?", host.ID).Count(&total)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"host_uuid": uuid,
		"total":     total,
		"limit":     limit,
		"ports":     ports,
	})
}

// HostPackagesHandler GET /api/hosts/{uuid}/assets/packages
func HostPackagesHandler(w http.ResponseWriter, r *http.Request) {
	uuid := strings.TrimPrefix(r.URL.Path, "/api/hosts/")
	uuid = strings.SplitN(uuid, "/", 2)[0]
	if uuid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "uuid required"})
		return
	}

	var host models.Host
	if err := db.DB.Where("uuid = ?", uuid).First(&host).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "host not found"})
		return
	}

	limit := 500
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 2000 {
			limit = n
		}
	}
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}
	keyword := r.URL.Query().Get("keyword")

	q := db.DB.Where("host_id = ?", host.ID)
	if keyword != "" {
		q = q.Where("name LIKE ? OR version LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	q = q.Order("name ASC").Limit(limit).Offset(offset)
	var pkgs []models.HostPackage
	q.Find(&pkgs)

	var total int64
	baseQ := db.DB.Model(&models.HostPackage{}).Where("host_id = ?", host.ID)
	if keyword != "" {
		baseQ = baseQ.Where("name LIKE ? OR version LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	baseQ.Count(&total)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"host_uuid": uuid,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
		"packages":  pkgs,
	})
}
