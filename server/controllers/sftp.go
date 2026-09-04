package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"sort"
	"time"

	"MSMP/server/db"
	"MSMP/server/models"

	"github.com/pkg/sftp"
)

// FileInfo 文件信息（序列化给前端）
type FileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	IsDir   bool   `json:"is_dir"`
	Mode    string `json:"mode"`
	ModTime string `json:"mod_time"`
}

// FileListHandler GET /api/hosts/{uuid}/files?path=/xxx
func FileListHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {
	if getRole(r) != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "仅管理员可访问文件管理"})
		return
	}

	dirPath := r.URL.Query().Get("path")
	if dirPath == "" {
		dirPath = "/"
	}

	client, err := dialHostSSH(tenantID, host.ID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": fmt.Sprintf("SFTP 连接失败: %v", err)})
		return
	}
	defer sftpClient.Close()

	entries, err := sftpClient.ReadDir(dirPath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("读取目录失败: %v", err)})
		return
	}

	files := make([]FileInfo, 0, len(entries))
	for _, e := range entries {
		files = append(files, FileInfo{
			Name:    e.Name(),
			Path:    path.Join(dirPath, e.Name()),
			Size:    e.Size(),
			IsDir:   e.IsDir(),
			Mode:    e.Mode().String(),
			ModTime: e.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	// 目录在前，按名称排序
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return files[i].Name < files[j].Name
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"path":  dirPath,
		"files": files,
		"count": len(files),
	})
}

// FileDownloadHandler GET /api/hosts/{uuid}/files/download?path=/xxx
func FileDownloadHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {
	if getRole(r) != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "仅管理员可访问文件管理"})
		return
	}

	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path required"})
		return
	}

	client, err := dialHostSSH(tenantID, host.ID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": fmt.Sprintf("SFTP 连接失败: %v", err)})
		return
	}
	defer sftpClient.Close()

	stat, err := sftpClient.Stat(filePath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("文件不存在: %v", err)})
		return
	}
	if stat.IsDir() {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "不能下载目录"})
		return
	}

	f, err := sftpClient.Open(filePath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("打开文件失败: %v", err)})
		return
	}
	defer f.Close()

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", path.Base(filePath)))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))

	io.Copy(w, f)

	db.DB.Create(&models.AuditLog{
		TenantID: tenantID,
		UserID:   userID,
		Action:   "file_download",
		Resource: fmt.Sprintf("host:%d:%s", host.ID, filePath),
		Status:   200,
	})
}

// FileUploadHandler POST /api/hosts/{uuid}/files/upload (multipart: file, path)
func FileUploadHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {
	if getRole(r) != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "仅管理员可访问文件管理"})
		return
	}

	r.ParseMultipartForm(64 << 20) // 64MB max

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "缺少 file 字段"})
		return
	}
	defer file.Close()

	targetDir := r.FormValue("path")
	if targetDir == "" {
		targetDir = "/tmp"
	}
	targetPath := path.Join(targetDir, header.Filename)

	client, err := dialHostSSH(tenantID, host.ID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": fmt.Sprintf("SFTP 连接失败: %v", err)})
		return
	}
	defer sftpClient.Close()

	dst, err := sftpClient.Create(targetPath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("创建文件失败: %v", err)})
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("上传失败: %v", err)})
		return
	}

	db.DB.Create(&models.AuditLog{
		TenantID: tenantID,
		UserID:   userID,
		Action:   "file_upload",
		Resource: fmt.Sprintf("host:%d:%s", host.ID, targetPath),
		Status:   200,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"uploaded": true,
		"path":     targetPath,
		"size":     header.Size,
	})
}

// FileMkdirHandler POST /api/hosts/{uuid}/files/mkdir {path}
func FileMkdirHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {
	if getRole(r) != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "仅管理员可访问文件管理"})
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path required"})
		return
	}

	client, err := dialHostSSH(tenantID, host.ID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": fmt.Sprintf("SFTP 连接失败: %v", err)})
		return
	}
	defer sftpClient.Close()

	if err := sftpClient.MkdirAll(req.Path); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("创建目录失败: %v", err)})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"created": true, "path": req.Path})
}

// FileDeleteHandler DELETE /api/hosts/{uuid}/files?path=/xxx
func FileDeleteHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {
	if getRole(r) != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "仅管理员可访问文件管理"})
		return
	}

	targetPath := r.URL.Query().Get("path")
	if targetPath == "" || targetPath == "/" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "非法路径"})
		return
	}

	client, err := dialHostSSH(tenantID, host.ID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": fmt.Sprintf("SFTP 连接失败: %v", err)})
		return
	}
	defer sftpClient.Close()

	stat, err := sftpClient.Stat(targetPath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("路径不存在: %v", err)})
		return
	}

	if stat.IsDir() {
		if err := sftpClient.RemoveDirectory(targetPath); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("删除目录失败（需为空目录）: %v", err)})
			return
		}
	} else {
		if err := sftpClient.Remove(targetPath); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("删除文件失败: %v", err)})
			return
		}
	}

	db.DB.Create(&models.AuditLog{
		TenantID: tenantID,
		UserID:   userID,
		Action:   "file_delete",
		Resource: fmt.Sprintf("host:%d:%s", host.ID, targetPath),
		Status:   200,
	})

	log.Printf("[SFTP] user=%d tenant=%d host=%d deleted %s", userID, tenantID, host.ID, targetPath)
	writeJSON(w, http.StatusOK, map[string]interface{}{"deleted": true, "path": targetPath, "time": time.Now().Unix()})
}

// FileRenameHandler POST /api/hosts/{uuid}/files/rename {old_path, new_path}
func FileRenameHandler(w http.ResponseWriter, r *http.Request, host *models.Host, tenantID, userID uint) {
	if getRole(r) != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "仅管理员可访问文件管理"})
		return
	}

	var req struct {
		OldPath string `json:"old_path"`
		NewPath string `json:"new_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OldPath == "" || req.NewPath == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "old_path 和 new_path 必填"})
		return
	}

	client, err := dialHostSSH(tenantID, host.ID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": fmt.Sprintf("SFTP 连接失败: %v", err)})
		return
	}
	defer sftpClient.Close()

	if err := sftpClient.Rename(req.OldPath, req.NewPath); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("重命名失败: %v", err)})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"renamed": true, "old_path": req.OldPath, "new_path": req.NewPath})
}