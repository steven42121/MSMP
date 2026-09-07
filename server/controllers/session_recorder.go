// Package controllers 提供 WebSSH 会话录制能力。
package controllers

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// SessionRecorder 记录 SSH 会话输出到文件。
type SessionRecorder struct {
	mu         sync.Mutex
	recordings map[string]*recordingFile
	dataDir    string
}

type recordingFile struct {
	path   string
	writer *bufio.Writer
	f      *os.File
	mu     sync.Mutex
}

var globalRecorder *SessionRecorder

// InitSessionRecorder 初始化会话录制器。
func InitSessionRecorder(dataDir string) {
	if globalRecorder != nil {
		globalRecorder.CloseAll()
	}
	sessionsDir := filepath.Join(dataDir, "sessions")
	os.MkdirAll(sessionsDir, 0755)
	globalRecorder = &SessionRecorder{
		recordings: make(map[string]*recordingFile),
		dataDir:    sessionsDir,
	}
}

func getRecorder() *SessionRecorder {
	if globalRecorder == nil {
		InitSessionRecorder("./data")
	}
	return globalRecorder
}

// StartRecording 开始录制会话，返回 sessionID。
func (r *SessionRecorder) StartRecording(userID, hostID uint, hostname string) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s_user%d_host%d_%s.log",
		timestamp, userID, hostID, randomStr(6))
	path := filepath.Join(r.dataDir, filename)

	f, err := os.Create(path)
	if err != nil {
		return ""
	}
	rec := &recordingFile{path: path, writer: bufio.NewWriter(f), f: f}
	r.recordings[filename] = rec
	return filename
}

// GetWriter 获取指定 session 的写入器。
func (r *SessionRecorder) GetWriter(filename string) io.Writer {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.recordings[filename]
	if !ok {
		return io.Discard
	}
	return &lockedWriter{rec: rec}
}

// StopRecording 停止录制并关闭文件。
func (r *SessionRecorder) StopRecording(filename string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.recordings[filename]
	if !ok {
		return
	}
	rec.writer.Flush()
	rec.f.Close()
	delete(r.recordings, filename)
}

// CloseAll 关闭所有录制文件。
func (r *SessionRecorder) CloseAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rec := range r.recordings {
		rec.writer.Flush()
		rec.f.Close()
	}
	r.recordings = make(map[string]*recordingFile)
}

// ListRecordings 列出所有会话录制文件。
func (r *SessionRecorder) ListRecordings() ([]RecordingInfo, error) {
	entries, err := os.ReadDir(r.dataDir)
	if err != nil {
		return nil, err
	}

	var infos []RecordingInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		infos = append(infos, RecordingInfo{
			Filename:  e.Name(),
			SizeBytes: fi.Size(),
			Modified:  fi.ModTime(),
		})
	}
	return infos, nil
}

// DownloadRecording 读取录制文件内容。
func (r *SessionRecorder) DownloadRecording(filename string) ([]byte, error) {
	return os.ReadFile(filepath.Join(r.dataDir, filename))
}

// RecordingInfo 单个录制文件信息。
type RecordingInfo struct {
	Filename  string    `json:"filename"`
	SizeBytes int64     `json:"size_bytes"`
	Modified  time.Time `json:"modified_at"`
}

// lockedWriter 线程安全的 io.Writer。
type lockedWriter struct {
	rec *recordingFile
}

func (w *lockedWriter) Write(p []byte) (int, error) {
	w.rec.mu.Lock()
	defer w.rec.mu.Unlock()
	return w.rec.writer.Write(p)
}

func randomStr(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[int(time.Now().UnixNano())%len(chars)]
	}
	return string(b)
}
