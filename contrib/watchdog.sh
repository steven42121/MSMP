#!/bin/bash
# MSMP Watchdog — 监控后端进程，崩溃后自动重启
# 用法: ./watchdog.sh [PID_FILE] [HEALTH_URL] [CHECK_INTERVAL]

set -u

PID_FILE="${1:-/var/run/msmp-server.pid}"
HEALTH_URL="${2:-http://localhost:8080/api/health}"
CHECK_INTERVAL="${3:-30}"
LOG_FILE="${4:-/var/log/msmp-server-watchdog.log}"
MAX_RESTARTS="${5:-10}"
RESTART_WINDOW="${6:-300}"  # 5 分钟内重启超过 MAX_RESTARTS 次则放弃

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SERVER_BIN="${SCRIPT_DIR}/msmp-server"
SERVER_DIR="${SCRIPT_DIR}/.."

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

check_health() {
    if curl -sf --max-time 5 "$HEALTH_URL" >/dev/null 2>&1; then
        return 0
    fi
    return 1
}

is_running() {
    if [ -f "$PID_FILE" ]; then
        local pid
        pid=$(cat "$PID_FILE" 2>/dev/null)
        if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
            return 0
        fi
    fi
    return 1
}

start_server() {
    log "Starting msmp-server..."
    cd "$SERVER_DIR"
    nohup "$SERVER_BIN" >> "$LOG_FILE" 2>&1 &
    local pid=$!
    echo "$pid" > "$PID_FILE"
    log "Started msmp-server with PID $pid"
}

stop_server() {
    if is_running; then
        local pid
        pid=$(cat "$PID_FILE")
        log "Stopping msmp-server (PID $pid)..."
        kill "$pid" 2>/dev/null
        sleep 2
        kill -9 "$pid" 2>/dev/null
        rm -f "$PID_FILE"
        log "Stopped msmp-server"
    fi
}

# 统计重启次数
count_restarts() {
    local restart_log="/tmp/msmp-restart-count"
    local now window_start count
    now=$(date +%s)
    window_start=$((now - RESTART_WINDOW))
    
    # 清理过期记录
    if [ -f "$restart_log" ]; then
        awk -v ws="$window_start" '$1 > ws' "$restart_log" > "${restart_log}.tmp"
        mv "${restart_log}.tmp" "$restart_log"
    fi
    
    count=$(wc -l < "$restart_log" 2>/dev/null || echo 0)
    echo "$count"
}

record_restart() {
    date +%s >> /tmp/msmp-restart-count
}

# 主循环
log "Watchdog started, checking every ${CHECK_INTERVAL}s"
log "Health URL: $HEALTH_URL"

while true; do
    if ! is_running; then
        count=$(count_restarts)
        if [ "$count" -ge "$MAX_RESTARTS" ]; then
            log "ERROR: Restart limit ($MAX_RESTARTS in ${RESTART_WINDOW}s) exceeded. Aborting."
            exit 1
        fi
        start_server
        record_restart
    elif ! check_health; then
        count=$(count_restarts)
        if [ "$count" -ge "$MAX_RESTARTS" ]; then
            log "ERROR: Health check failing repeatedly. Aborting."
            exit 1
        fi
        log "Health check failed, restarting..."
        stop_server
        sleep 2
        start_server
        record_restart
    fi
    sleep "$CHECK_INTERVAL"
done
