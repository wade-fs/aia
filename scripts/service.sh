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
SYS_ENV_FILE="/etc/aia/service.env"
BIN_DIR="${HOME}/.local/bin"
TARGET_BIN="${BIN_DIR}/aia"
SYSTEMD_USER_DIR="${HOME}/.config/systemd/user"
UNIT_FILE="${SYSTEMD_USER_DIR}/aia.service"
SERVICE_NAME="aia.service"

DEFAULT_PORT="7777"
DEFAULT_HOST="127.0.0.1"

# Colors for terminal output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Helper to read a KEY=value safely from an env file without sourcing
get_key_from_file() {
    local key="$1"
    local file="$2"
    if [[ -f "$file" ]]; then
        grep -E "^${key}=" "$file" 2>/dev/null | tail -n 1 | cut -d'=' -f2- | tr -d '"' | tr -d "'" || true
    fi
}

# Resolve secret/config variable from:
# 1. Current shell environment variable (e.g. $FRED_API_KEY)
# 2. Runtime user config file (~/.config/aia/service.env)
# 3. Global system config file (/etc/aia/service.env)
resolve_var() {
    local var_name="$1"
    # 1. Environment variable
    if [[ -n "${!var_name:-}" ]]; then
        echo "${!var_name}"
        return
    fi
    # 2. User ~/.config/aia/service.env
    local val
    val="$(get_key_from_file "$var_name" "${ENV_FILE}")"
    if [[ -n "$val" ]]; then
        echo "$val"
        return
    fi
    # 3. Global /etc/aia/service.env
    val="$(get_key_from_file "$var_name" "${SYS_ENV_FILE}")"
    if [[ -n "$val" ]]; then
        echo "$val"
        return
    fi
    echo ""
}

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
    local fred_key
    fred_key="$(resolve_var "FRED_API_KEY")"
    local finmind_token
    finmind_token="$(resolve_var "FINMIND_TOKEN")"
    local port="${AIA_PORT:-${DEFAULT_PORT}}"
    local host="${AIA_HOST:-${DEFAULT_HOST}}"

    # Remind the user if external API keys are not set
    local missing_keys=()
    [[ -z "${fred_key}" ]] && missing_keys+=("FRED_API_KEY (影響美股/總經與殖利率抓取)")
    [[ -z "${finmind_token}" ]] && missing_keys+=("FINMIND_TOKEN (影響進階台股個股資訊，未設定時使用 TWSE OpenAPI 基礎行情)")

    if [[ ${#missing_keys[@]} -gt 0 ]]; then
        echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${YELLOW}⚠️  【配置提醒】尚未偵測到以下資料來源金鑰設定：${NC}"
        for k in "${missing_keys[@]}"; do
            echo -e "   ${YELLOW}• ${k}${NC}"
        done
        echo -e "\n   💡 您可以隨時透過以下任一方式注入金鑰："
        echo -e "      1. 寫入使用者金鑰檔: ${GREEN}${ENV_FILE}${NC}"
        echo -e "      2. 或是寫入系統全域檔: ${GREEN}${SYS_ENV_FILE}${NC}"
        echo -e "      3. 或是執行時帶入: ${GREEN}export FRED_API_KEY=... && make install_service${NC}"
        echo -e "      設定完成後執行 ${YELLOW}make restart_service${NC} 即可生效。"
        echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
    fi

    # Update or create user ~/.config/aia/service.env
    if [[ -f "${ENV_FILE}" ]]; then
        echo -e "${BLUE}⚙️  檢視既有環境設定檔: ${ENV_FILE}...${NC}"
        if [[ -n "${fred_key}" ]]; then
            if grep -q "^FRED_API_KEY=" "${ENV_FILE}" 2>/dev/null; then
                sed -i "s|^FRED_API_KEY=.*|FRED_API_KEY=${fred_key}|" "${ENV_FILE}"
            else
                echo "FRED_API_KEY=${fred_key}" >> "${ENV_FILE}"
            fi
        else
            grep -q "^FRED_API_KEY=" "${ENV_FILE}" 2>/dev/null || echo "FRED_API_KEY=" >> "${ENV_FILE}"
        fi

        if [[ -n "${finmind_token}" ]]; then
            if grep -q "^FINMIND_TOKEN=" "${ENV_FILE}" 2>/dev/null; then
                sed -i "s|^FINMIND_TOKEN=.*|FINMIND_TOKEN=${finmind_token}|" "${ENV_FILE}"
            else
                echo "FINMIND_TOKEN=${finmind_token}" >> "${ENV_FILE}"
            fi
        else
            grep -q "^FINMIND_TOKEN=" "${ENV_FILE}" 2>/dev/null || echo "FINMIND_TOKEN=" >> "${ENV_FILE}"
        fi

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
    echo -e "${GREEN}✓ 環境變數設定完成（已確保 0600 機密保護權限）${NC}"
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
    local port
    port="$(resolve_var "AIA_PORT")"
    port="${port:-${DEFAULT_PORT}}"
    local host
    host="$(resolve_var "AIA_HOST")"
    host="${host:-${DEFAULT_HOST}}"
    echo -e "   Web 儀表板位址 : ${GREEN}http://${host}:${port}/${NC}"
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
        echo -e "👉 請在瀏覽器登入頁 (http://127.0.0.1:7777/) 輸入此 Token 進行登入。"
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
