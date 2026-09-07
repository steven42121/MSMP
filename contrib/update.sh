#!/bin/bash
# MSMP 零停机更新脚本
# 流程：拉取代码 → 构建 → 优雅重启 → 健康验证 → 失败自动回滚
# 用法: ./contrib/update.sh [INSTALL_DIR]

set -uo pipefail

INSTALL_DIR="${1:-/opt/msmp}"
SERVICE_NAME="msmp-server"
HEALTH_URL="http://localhost:8080/api/health"
REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BACKUP_DIR="${INSTALL_DIR}/backup"
NEW_BIN="${INSTALL_DIR}/msmp-server.new"
CURRENT_BIN="${INSTALL_DIR}/msmp-server"
VERSION_FLAG=""

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"; }
info() { log "ℹ $*"; }
ok()   { log "✓ $*"; }
die()  { log "✗ $*"; exit 1; }

# 读取当前运行版本（健康检查返回 version 字段）
get_running_version() {
    curl -sf --max-time 5 "$HEALTH_URL" 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin).get('version','unknown'))" 2>/dev/null || echo "unknown"
}

# 构建新版（传入版本号）
build_new() {
    local version="$1"
    info "构建 ${version} ..."
    cd "$REPO_DIR/server" || die "找不到 server 目录"
    local ldflags="-s -w -X main.Version=${version} -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    go build -ldflags "$ldflags" -o "$NEW_BIN.tmp" . || die "构建失败"
    mv "$NEW_BIN.tmp" "$NEW_BIN"
    cd "$REPO_DIR"
    ok "构建完成"
}

# 备份当前二进制
backup_current() {
    mkdir -p "$BACKUP_DIR"
    if [ -f "$CURRENT_BIN" ]; then
        cp "$CURRENT_BIN" "$BACKUP_DIR/msmp-server.$(date +%s)" || die "备份失败"
        ok "已备份当前二进制"
    fi
}

# 优雅重启（systemd 会先发 SIGTERM 触发 graceful shutdown）
restart_service() {
    info "优雅重启服务 ..."
    systemctl restart "$SERVICE_NAME" || die "重启失败"
}

# 健康验证（最多等 30 秒）
verify_health() {
    local expect_version="$1"
    info "等待服务就绪 ..."
    for i in $(seq 1 30); do
        if curl -sf --max-time 3 "$HEALTH_URL" >/dev/null 2>&1; then
            local running="$(get_running_version)"
            # 版本匹配则判为新版已生效
            if [ -n "$expect_version" ] && [ "$running" = "$expect_version" ]; then
                ok "健康检查通过（version=${running}）"
                return 0
            fi
            # 未配置版本时不强制，仅检查健康
            if [ -z "$expect_version" ] || [ "$expect_version" = "dev" ]; then
                ok "健康检查通过"
                return 0
            fi
        fi
        sleep 1
    done
    return 1
}

# 回滚到上一次备份
rollback() {
    local latest_backup
    latest_backup=$(ls -t "$BACKUP_DIR"/msmp-server.* 2>/dev/null | head -1)
    if [ -z "$latest_backup" ]; then
        die "无可回滚的备份"
    fi
    warn "回滚到 ${latest_backup} ..."
    cp "$latest_backup" "$CURRENT_BIN"
    systemctl restart "$SERVICE_NAME" || die "回滚后重启失败"
    ok "已回滚"
}

warn() { log "⚠ $*"; }

main() {
    log "=== MSMP 零停机更新 ==="

    # 1. 获取最新版本号（从 CHANGELOG 或 git tag）
    local new_version="dev"
    if [ -f "$REPO_DIR/CHANGELOG.md" ]; then
        # 解析最新 version heading
        new_version=$(grep -m1 -oE '\[v?[0-9]+\.[0-9]+\.[0-9]+\]' "$REPO_DIR/CHANGELOG.md" 2>/dev/null | tr -d '[]' | head -1 || true)
    fi
    [ -z "$new_version" ] && new_version="dev"
    info "目标版本: ${new_version}"

    # 2. 更新代码
    info "拉取最新代码 ..."
    cd "$REPO_DIR" || die "仓库不存在"
    git fetch --all --prune 2>/dev/null || info "git fetch 失败（离线模式继续）"
    git pull --ff-only 2>/dev/null || info "git pull 失败（使用当前代码）"

    # 3. 构建新版
    backup_current
    build_new "$new_version" || { rollback; die "构建失败，已回滚"; }

    # 4. 部署新版（原子替换）
    if [ -f "$NEW_BIN" ]; then
        mv "$NEW_BIN" "$CURRENT_BIN" || die "替换二进制失败"
        chmod 755 "$CURRENT_BIN"
    fi

    # 5. 优雅重启
    restart_service

    # 6. 健康验证
    if verify_health "$new_version"; then
        ok "更新完成：${new_version}"
        # 清理旧备份（保留最近 3 个）
        ls -t "$BACKUP_DIR"/msmp-server.* 2>/dev/null | tail -n +4 | xargs -r rm -f
    else
        warn "健康检查失败，执行回滚"
        rollback
        die "更新失败，已回滚到旧版本"
    fi
}

# 需要 systemctl 时检查权限
if ! command -v systemctl >/dev/null 2>&1; then
    die "需要 systemctl 管理服务"
fi
if [ "$(id -u)" -ne 0 ]; then
    die "需要 root 权限执行更新（sudo ./contrib/update.sh）"
fi

main "$@"