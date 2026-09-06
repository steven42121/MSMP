package controllers

import (
	"errors"
	"path/filepath"
	"strings"
)

var ErrPathInvalid = errors.New("非法路径")

// sanitizeSSHPath 验证目标路径在 baseDir 内，防止路径穿越。
func sanitizeSSHPath(baseDir, target string) (string, error) {
	cleaned := filepath.Clean(target)
	if !strings.HasPrefix(cleaned, "/") {
		return "", ErrPathInvalid
	}
	// 根目录直接放行
	if cleaned == "/" {
		return "/", nil
	}
	bp := filepath.Clean(baseDir) + string(filepath.Separator)
	if !strings.HasPrefix(cleaned, bp) && cleaned != bp {
		return "", ErrPathInvalid
	}
	return cleaned, nil
}

// sanitizeUploadTarget 上传时只保留文件名，丢弃客户端提供的路径部分。
func sanitizeUploadTarget(baseDir, filename string) (string, error) {
	cleanName := filepath.Base(filename)
	if cleanName == "." || cleanName == ".." || cleanName == "" {
		return "", ErrPathInvalid
	}
	return filepath.Join(filepath.Clean(baseDir), cleanName), nil
}

// pathSafe 校验路径不越界（用于 download/delete/mkdir/rename）。
func pathSafe(p string) bool {
	if p == "" {
		return false
	}
	cleaned := filepath.Clean(p)
	return strings.HasPrefix(cleaned, "/")
}
