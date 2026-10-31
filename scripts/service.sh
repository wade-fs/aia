#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# AIA Background Service Management Script (systemd user service)
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
GO_DIR="${REPO_ROOT}/go"

CONFIG_DIR="${HOME}/.config/aia"
ENV_FILE="${CONFIG_DIR}/service.env"
BIN_DIR="${HOME}/.local/bin"
TARGET_BIN="${BIN_DIR}/aia"
SYSTEMD_USER_DIR="${HOME}/.config/systemd/user"
UNIT_FILE="${SYSTEMD_USER_DIR}/aia.service"
SERVICE_NAME="aia.service"

DEFAULT_FRED_API_KEY="61b261dba7294222d315d0336a8be46f"
DEFAULT_FINMIND_TOKEN="eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2lkIjoid2FkZS5mc0BnbWFpbC5jb20iLCJlbWFpbCI6IndhZGUuZnNAZ21haWwuY29tIiwidG9rZW5fdmVyc2lvbiI6MX0.T9avd2YujjYPoGVJYiSPi6_Tbp1rAQF41-Eyiiiw5ds"
DEFAULT_PORT="8080"
DEFAULT_HOST="127.0.0.1"

# Colors for terminal output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

ensure_dirs() {
    mkdir -p "${CONFIG_DIR}"
    mkdir -p "${BIN_DIR}"
    mkdir -p "${SYSTEMD_USER_DIR}"
    mkdir -p "${HOME}/.local/share/aia"
}

build_binary() {
    echo -e "${BLUE}🔨 編譯最新 AIA 二進位執行檔...${NC}"
    (cd "${GO_DIR}" && go build -o bin/aia ./cmd/aia)
    cp -f "${GO_DIR}/bin/aia" "${TARGET_BIN}"
    chmod +x "${TARGET_BIN}"
    echo -e "${GREEN}✓ 二進位執行檔已就緒: ${TARGET_BIN}${NC}"
}

write_env_file() {
    local fred_key="${FRED_API_KEY:-${DEFAULT_FRED_API_KEY}}"
    local finmind_token="${FINMIND_TOKEN:-${DEFAULT_FINMIND_TOKEN}}"
    local port="${AIA_PORT:-${DEFAULT_PORT}}"
    local host="${AIA_HOST:-${DEFAULT_HOST}}"

    # If env file exists, preserve any user-added keys while ensuring FRED and FINMIND exist
    if [[ -f "${ENV_FILE}" ]]; then
        echo -e "${YELLOW}⚙️  偵測到既有環境設定檔: ${ENV_FILE}，安全更新金鑰...${NC}"
        # Update or append keys
        grep -q "^FRED_API_KEY=" "${ENV_FILE}" 2>/dev/null && \
            sed -i "s|^FRED_API_KEY=.*|FRED_API_KEY=${fred_key}|" "${ENV_FILE}" || \
            echo "FRED_API_KEY=${fred_key}" >> "${ENV_FILE}"

        grep -q "^FINMIND_TOKEN=" "${ENV_FILE}" 2>/dev/null && \
            sed -i "s|^FINMIND_TOKEN=.*|FINMIND_TOKEN=${finmind_token}|" "${ENV_FILE}" || \
            echo "FINMIND_TOKEN=${finmind_token}" >> "${ENV_FILE}"

        grep -q "^AIA_PORT=" "${ENV_FILE}" 2>/dev/null || echo "AIA_PORT=${port}" >> "${ENV_FILE}"
        grep -q "^AIA_HOST=" "${ENV_FILE}" 2>/dev/null || echo "AIA_HOST=${host}" >> "${ENV_FILE}"
        grep -q "^PATH=" "${ENV_FILE}" 2>/dev/null || echo "PATH=${BIN_DIR}:/usr/local/bin:/usr/bin:/bin" >> "${ENV_FILE}"
    else
        echo -e "${BLUE}📝 建立安全環境變數檔: ${ENV_FILE}${NC}"
        cat <<EOF > "${ENV_FILE}"
# ==============================================================================
# AIA Background Service Environment Configuration
# Permissions: 0600 (Confidential)
# ==============================================================================
FRED_API_KEY=${fred_key}
FINMIND_TOKEN=${finmind_token}
AIA_PORT=${port}
AIA_HOST=${host}
PATH=${BIN_DIR}:/usr/local/bin:/usr/bin:/bin
EOF
    fi

    # Set strict permission 0600 to protect secrets
    chmod 600 "${ENV_FILE}"
    echo -e "${GREEN}✓ 環境變數設定完成（已設定 0600 機密保護權限）${NC}"
}

write_unit_file() {
    echo -e "${BLUE}📝 設定 systemd user unit: ${UNIT_FILE}${NC}"
    cat <<EOF > "${UNIT_FILE}"
[Unit]
Description=AIA - Personal Autonomous Intelligence Architecture Daemon
Documentation=https://github.com/wade/aia
After=network.target

# Anti-Flapping / Crash Loop Protection (避免持續性錯誤導致反覆重啟風暴)
StartLimitIntervalSec=60s
StartLimitBurst=4

[Service]
Type=simple
Environment="PATH=%h/.local/bin:/usr/local/bin:/usr/bin:/bin"
EnvironmentFile=-%h/.config/aia/service.env
ExecStart=%h/.local/bin/aia server start
Restart=on-failure
RestartSec=5s
KillMode=control-group
StandardOutput=journal
StandardError=journal
TimeoutStopSec=15

# Security Hardening
NoNewPrivileges=true

[Install]
WantedBy=default.target
EOF
    chmod 644 "${UNIT_FILE}"
}

is_service_installed() {
    [[ -f "${UNIT_FILE}" ]]
}

is_service_active() {
    systemctl --user is-active --quiet "${SERVICE_NAME}" 2>/dev/null
}

install_service() {
    ensure_dirs
    build_binary
    write_env_file

    local already_installed=false
    if is_service_installed; then
        already_installed=true
    fi

    write_unit_file

    echo -e "${BLUE}🔄 重新載入 systemd user daemon...${NC}"
    systemctl --user daemon-reload

    # Enable lingering so user services stay running in background after logout
    loginctl enable-linger "${USER}" 2>/dev/null || true

    if [[ "${already_installed}" == "true" ]]; then
        echo -e "${YELLOW}⚡ AIA 背景服務已存在（冪等檢測通過）。更新執行檔與設定，正在平滑重啟...${NC}"
        systemctl --user reset-failed "${SERVICE_NAME}" 2>/dev/null || true
        systemctl --user restart "${SERVICE_NAME}"
        echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${GREEN}✓ AIA 背景服務已成功更新並重啟！未產生重複服務。${NC}"
        echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    else
        echo -e "${BLUE}🚀 啟用並啟動 AIA 服務...${NC}"
        systemctl --user reset-failed "${SERVICE_NAME}" 2>/dev/null || true
        systemctl --user enable --now "${SERVICE_NAME}"
        echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${GREEN}✓ AIA 背景服務已成功安裝並啟動！${NC}"
        echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    fi

    sleep 1
    show_status
}

show_status() {
    if ! is_service_installed; then
        echo -e "${YELLOW}ℹ️  AIA 背景服務尚未安裝。${NC}"
        echo -e "   可執行 'make install_service' 進行安裝。"
        return 0
    fi

    echo -e "\n${BLUE}📊 AIA 背景服務運行狀態：${NC}"
    systemctl --user status "${SERVICE_NAME}" --no-pager || true

    echo -e "\n${BLUE}🌐 連線資訊：${NC}"
    local port="${AIA_PORT:-${DEFAULT_PORT}}"
    echo -e "   Web 儀表板位址 : ${GREEN}http://127.0.0.1:${port}/${NC}"
    echo -e "   金鑰配置檔案   : ${ENV_FILE} (chmod 600)"
    echo -e "   取得登入 Token : ${YELLOW}make token${NC}"
    echo -e "   即時查看日誌   : ${YELLOW}make logs_service${NC}"
}

get_token() {
    if ! is_service_installed; then
        echo -e "${RED}錯誤: AIA 服務未安裝。${NC}"
        exit 1
    fi

    echo -e "${BLUE}🔍 正在從 systemd journal 提取最新的 One-Time Login Token...${NC}"
    local token_line
    token_line=$(journalctl --user -u "${SERVICE_NAME}" -n 100 -o cat 2>/dev/null | grep -A 1 "One-Time Login Token:" | tail -n 1 | tr -d ' ' || true)

    if [[ -n "${token_line}" && "${token_line}" != *"One-Time"* ]]; then
        echo -e "${GREEN}=======================================================================${NC}"
        echo -e "🔑 最新 One-Time Login Token:"
        echo -e "   ${YELLOW}${token_line}${NC}"
        echo -e "${GREEN}=======================================================================${NC}"
        echo -e "👉 請在瀏覽器登入頁 (http://127.0.0.1:8080/) 輸入此 Token 進行登入。"
    else
        echo -e "${YELLOW}未能立即解析出 Token，顯示最近 25 行日誌：${NC}"
        journalctl --user -u "${SERVICE_NAME}" -n 25 --no-pager
    fi
}

view_logs() {
    if ! is_service_installed; then
        echo -e "${RED}錯誤: AIA 服務未安裝。${NC}"
        exit 1
    fi
    journalctl --user -u "${SERVICE_NAME}" -n 50 --no-pager
}

restart_service() {
    if ! is_service_installed; then
        echo -e "${RED}錯誤: AIA 服務尚未安裝。請先執行 'make install_service'。${NC}"
        exit 1
    fi
    echo -e "${BLUE}🔄 重啟 AIA 背景服務...${NC}"
    systemctl --user reset-failed "${SERVICE_NAME}" 2>/dev/null || true
    systemctl --user restart "${SERVICE_NAME}"
    echo -e "${GREEN}✓ AIA 服務已重啟。${NC}"
    sleep 1
    show_status
}

uninstall_service() {
    if ! is_service_installed; then
        echo -e "${YELLOW}ℹ️  AIA 背景服務尚未安裝，無需移除。${NC}"
        return 0
    fi

    echo -e "${YELLOW}🛑 正在停止並解除安裝 AIA 背景服務...${NC}"
    systemctl --user stop "${SERVICE_NAME}" 2>/dev/null || true
    systemctl --user disable "${SERVICE_NAME}" 2>/dev/null || true
    rm -f "${UNIT_FILE}"
    systemctl --user daemon-reload

    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}✓ AIA 背景服務已完全停止並解除註冊！${NC}"
    echo -e "   註：您的資料庫及設定檔 (${CONFIG_DIR}) 依然安全保留。"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

case "${1:-install}" in
    install)
        install_service
        ;;
    uninstall)
        uninstall_service
        ;;
    status)
        show_status
        ;;
    restart)
        restart_service
        ;;
    logs)
        view_logs
        ;;
    token)
        get_token
        ;;
    *)
        echo "使用方式: $0 {install|uninstall|status|restart|logs|token}"
        exit 1
        ;;
esac
