#!/bin/bash
set -euo pipefail

# =============================================================================
# ttl 安装脚本
# 用法：/bin/bash -c "$(curl -fsSL https://your-host/install.sh)"
# 从 GitHub Release 下载预编译二进制
# =============================================================================

BINARY_NAME="ttl"
REPO="your-org/great-tool-go"  # ← 替换为实际仓库：owner/repo
INSTALL_DIR="/usr/local/bin"
FALLBACK_INSTALL_DIR="${HOME}/.local/bin"

# ── 颜色输出 ──────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
RESET='\033[0m'

info()    { echo -e "${BOLD}[ttl]${RESET} $*"; }
success() { echo -e "${GREEN}[ttl]${RESET} $*"; }
warn()    { echo -e "${YELLOW}[ttl]${RESET} $*"; }
error()   { echo -e "${RED}[ttl]${RESET} $*" >&2; exit 1; }

# ── 环境检测 ──────────────────────────────────────────────────────────────────
detect_os() {
    case "$(uname -s)" in
        Darwin) echo "darwin" ;;
        Linux)  echo "linux"  ;;
        *)      error "不支持的操作系统：$(uname -s)（仅支持 macOS / Linux）" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64)          echo "amd64" ;;
        arm64 | aarch64) echo "arm64" ;;
        *)               error "不支持的 CPU 架构：$(uname -m)" ;;
    esac
}

# ── GitHub API 获取最新版本 ───────────────────────────────────────────────────
get_latest_version() {
    local version
    version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | \
        grep '"tag_name"' | \
        sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "${version}" ]; then
        error "无法获取最新版本信息，请检查仓库地址：${REPO}"
    fi
    echo "${version}"
}

# ── 安装目录选择 ───────────────────────────────────────────────────────────────
choose_install_dir() {
    # 优先写 /usr/local/bin（需要写权限）
    if [ -w "${INSTALL_DIR}" ]; then
        echo "${INSTALL_DIR}"
        return
    fi
    # 尝试 sudo
    if command -v sudo &>/dev/null && sudo -n true 2>/dev/null; then
        echo "${INSTALL_DIR}"
        return
    fi
    # 降级到用户目录
    mkdir -p "${FALLBACK_INSTALL_DIR}"
    echo "${FALLBACK_INSTALL_DIR}"
}

install_binary() {
    local src="$1"
    local dest_dir="$2"
    local dest="${dest_dir}/${BINARY_NAME}"

    if [ "${dest_dir}" = "${INSTALL_DIR}" ] && [ ! -w "${INSTALL_DIR}" ]; then
        info "需要管理员权限，正在安装到 ${dest} ..."
        sudo install -m 755 "${src}" "${dest}"
    else
        install -m 755 "${src}" "${dest}"
    fi
}

ensure_in_path() {
    local dir="$1"
    if [[ ":${PATH}:" != *":${dir}:"* ]]; then
        warn "${dir} 不在 \$PATH 中。"
        warn "请将以下内容添加到你的 shell 配置文件（~/.zshrc 或 ~/.bashrc）："
        warn ""
        warn "  export PATH=\"${dir}:\$PATH\""
        warn ""
    fi
}

# ── 主流程 ────────────────────────────────────────────────────────────────────
main() {
    local os arch version dest_dir tmp_dir download_url

    os=$(detect_os)
    arch=$(detect_arch)
    version=$(get_latest_version)
    dest_dir=$(choose_install_dir)

    # 下载文件名格式：ttl-{os}-{arch}
    local filename="${BINARY_NAME}-${os}-${arch}"
    download_url="https://github.com/${REPO}/releases/download/${version}/${filename}"

    info "操作系统：${os} / 架构：${arch}"
    info "最新版本：${version}"
    info "安装目标：${dest_dir}/${BINARY_NAME}"

    # 检查依赖
    if ! command -v curl &>/dev/null && ! command -v wget &>/dev/null; then
        error "需要 curl 或 wget 来下载文件。"
    fi

    # 下载到临时目录
    tmp_dir=$(mktemp -d)
    trap 'rm -rf "${tmp_dir}"' EXIT

    local download_path="${tmp_dir}/${BINARY_NAME}"
    info "正在下载 ${filename}..."

    if command -v curl &>/dev/null; then
        curl -fsSL "${download_url}" -o "${download_path}" || \
            error "下载失败，请检查网络或版本：${version}"
    else
        wget -qO "${download_path}" "${download_url}" || \
            error "下载失败，请检查网络或版本：${version}"
    fi

    # 设置执行权限
    chmod +x "${download_path}"

    # 安装
    install_binary "${download_path}" "${dest_dir}"

    success "安装成功 🎉  ${dest_dir}/${BINARY_NAME}"
    ensure_in_path "${dest_dir}"

    # 验证
    if command -v "${BINARY_NAME}" &>/dev/null; then
        info "验证：$(${BINARY_NAME} version)"
    else
        warn "安装成功，但需要重新打开终端才能使用。"
        warn "或将 ${dest_dir} 添加到 PATH 后执行：source ~/.zshrc"
    fi
}

main "$@"
