#!/bin/bash
set -euo pipefail

BINARY_NAME="ttl"
REPO="ZHANGSHUNLIN/TTL-CLI"
INSTALL_DIR="/usr/local/bin"
FALLBACK_INSTALL_DIR="${HOME}/.local/bin"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
RESET='\033[0m'

info()    { echo -e "${BOLD}[ttl]${RESET} $*"; }
success() { echo -e "${GREEN}[ttl]${RESET} $*"; }
warn()    { echo -e "${YELLOW}[ttl]${RESET} $*"; }
error()   { echo -e "${RED}[ttl]${RESET} $*" >&2; exit 1; }

detect_os() {
    case "$(uname -s)" in
        Darwin) echo "darwin" ;;
        Linux)  echo "linux"  ;;
        *)      error "Unsupported OS: $(uname -s) (only macOS / Linux supported)" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64)          echo "amd64" ;;
        arm64 | aarch64) echo "arm64" ;;
        *)               error "Unsupported CPU architecture: $(uname -m)" ;;
    esac
}

get_latest_version() {
    local version
    version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | \
        grep '"tag_name"' | \
        sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "${version}" ]; then
        error "Failed to get latest version, please check repo: ${REPO}"
    fi
    echo "${version}"
}

choose_install_dir() {
    if [ -w "${INSTALL_DIR}" ]; then
        echo "${INSTALL_DIR}"
        return
    fi
    if command -v sudo &>/dev/null && sudo -n true 2>/dev/null; then
        echo "${INSTALL_DIR}"
        return
    fi
    mkdir -p "${FALLBACK_INSTALL_DIR}"
    echo "${FALLBACK_INSTALL_DIR}"
}

install_binary() {
    local src="$1"
    local dest_dir="$2"
    local dest="${dest_dir}/${BINARY_NAME}"

    if [ "${dest_dir}" = "${INSTALL_DIR}" ] && [ ! -w "${INSTALL_DIR}" ]; then
        info "Sudo required, installing to ${dest} ..."
        sudo install -m 755 "${src}" "${dest}"
    else
        install -m 755 "${src}" "${dest}"
    fi
}

ensure_in_path() {
    local dir="$1"
    if [[ ":${PATH}:" != *":${dir}:"* ]]; then
        warn "${dir} is not in \$PATH."
        warn "Add the following to your shell config (~/.zshrc or ~/.bashrc):"
        warn ""
        warn "  export PATH=\"${dir}:\$PATH\""
        warn ""
    fi
}

main() {
    local os arch version dest_dir tmp_dir download_url filename

    os=$(detect_os)
    arch=$(detect_arch)
    dest_dir=$(choose_install_dir)

    if [ -n "${TTL_DOWNLOAD_URL:-}" ]; then
        download_url="${TTL_DOWNLOAD_URL}"
        filename="${download_url##*/}"
        info "Using custom download URL"
        info "Download file: ${filename}"
    else
        version=$(get_latest_version)
        filename="ttl-cli-${version}-${os}-${arch}"
        download_url="https://github.com/${REPO}/releases/download/${version}/${filename}"
        info "OS: ${os} / Arch: ${arch}"
        info "Latest version: ${version}"
    fi

    info "Install target: ${dest_dir}/${BINARY_NAME}"

    if ! command -v curl &>/dev/null && ! command -v wget &>/dev/null; then
        error "curl or wget is required to download files."
    fi

    tmp_dir=$(mktemp -d)
    trap 'rm -rf "${tmp_dir}"' EXIT

    local download_path="${tmp_dir}/${filename}"
    info "Downloading ${filename}..."

    if command -v curl &>/dev/null; then
        curl -fsSL "${download_url}" -o "${download_path}" || \
            error "Download failed: ${download_url}"
    else
        wget -qO "${download_path}" "${download_url}" || \
            error "Download failed: ${download_url}"
    fi

    chmod +x "${download_path}"

    install_binary "${download_path}" "${dest_dir}"

    local dest="${dest_dir}/${BINARY_NAME}"
    if [ ! -f "${dest}" ]; then
        warn "Renaming ${download_path##*/} to ${BINARY_NAME}..."
        mv "${download_path}" "${dest}"
        chmod +x "${dest}"
    fi

    success "Installation successful 🎉  ${dest}"
    ensure_in_path "${dest_dir}"

    if command -v "${BINARY_NAME}" &>/dev/null; then
        info "Verify: $(${BINARY_NAME} version)"
    else
        warn "Installation successful, but reopen terminal to use."
        warn "Or add ${dest_dir} to PATH and run: source ~/.zshrc"
    fi
}

main "$@"
