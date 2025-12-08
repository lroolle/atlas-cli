#!/usr/bin/env bash
set -euo pipefail

REPO="lroolle/atlas-cli"
SKILL_NAME="atl-cli"
BIN_NAME="atl"

# Defaults
BIN_DIR="$HOME/.local/bin"
SKILL_DIR="$HOME/.claude/skills"
INSTALL_BIN=true
INSTALL_SKILL=true
USE_SUDO=false

# Temp dir with proper cleanup - set trap BEFORE mktemp
TMP_DIR=""
cleanup() { [[ -n "$TMP_DIR" && -d "$TMP_DIR" ]] && rm -rf "$TMP_DIR"; }
trap cleanup EXIT INT TERM HUP

usage() {
    cat <<EOF
Usage: $0 [OPTIONS]

Install atlas-cli binary and Claude Code skill.

Options:
    --system        Install binary to /usr/local/bin (requires sudo)
    --bin-only      Install only the binary
    --skill-only    Install only the skill
    --bin-dir DIR   Custom binary directory (default: ~/.local/bin)
    --skill-dir DIR Custom skill directory (default: ~/.claude/skills)
    -h, --help      Show this help

Examples:
    $0                    # Install both to user directories
    $0 --system           # Install binary system-wide
    $0 --skill-only       # Only install the Claude Code skill
EOF
    exit 0
}

log() { echo "==> $*"; }
warn() { echo "WARN: $*" >&2; }
err() { echo "ERR: $*" >&2; exit 1; }

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --system)
                BIN_DIR="/usr/local/bin"
                USE_SUDO=true
                shift ;;
            --bin-only)
                INSTALL_SKILL=false
                shift ;;
            --skill-only)
                INSTALL_BIN=false
                shift ;;
            --bin-dir)
                BIN_DIR="$2"
                shift 2 ;;
            --skill-dir)
                SKILL_DIR="$2"
                shift 2 ;;
            -h|--help)
                usage ;;
            *)
                err "Unknown option: $1" ;;
        esac
    done
}

detect_platform() {
    local os arch
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    arch="$(uname -m)"

    case "$arch" in
        x86_64)  arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *)       err "Unsupported architecture: $arch" ;;
    esac

    case "$os" in
        linux|darwin) ;;
        *)            err "Unsupported OS: $os" ;;
    esac

    echo "${os}_${arch}"
}

download_and_verify() {
    local url="$1" dest="$2"
    curl -fsSL "$url" -o "$dest" || return 1
    [[ -s "$dest" ]] || return 1
}

install_binary() {
    log "Installing binary to $BIN_DIR"

    local platform version download_url
    platform="$(detect_platform)"
    log "Platform: $platform"

    # Get latest release version
    log "Checking GitHub releases..."
    version=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null \
        | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/') || true

    if [[ -z "$version" ]]; then
        warn "No GitHub releases found"
        log "Fallback: go install"
        install_via_go
        return
    fi
    log "Latest release: $version"

    download_url="https://github.com/$REPO/releases/download/$version/${BIN_NAME}_${platform}.tar.gz"
    log "Downloading: $download_url"

    if ! download_and_verify "$download_url" "$TMP_DIR/archive.tar.gz"; then
        warn "Download failed (no prebuilt binary for $platform)"
        log "Fallback: go install"
        install_via_go
        return
    fi

    # Extract with safety flags
    tar -xzf "$TMP_DIR/archive.tar.gz" -C "$TMP_DIR" --no-same-owner 2>/dev/null \
        || tar -xzf "$TMP_DIR/archive.tar.gz" -C "$TMP_DIR"

    # Validate binary exists
    if [[ ! -f "$TMP_DIR/$BIN_NAME" ]]; then
        warn "Archive missing expected binary, trying go install..."
        install_via_go
        return
    fi

    mkdir -p "$BIN_DIR"
    if $USE_SUDO; then
        sudo install -m 755 -- "$TMP_DIR/$BIN_NAME" "$BIN_DIR/$BIN_NAME"
    else
        install -m 755 -- "$TMP_DIR/$BIN_NAME" "$BIN_DIR/$BIN_NAME"
    fi

    log "Installed $BIN_NAME $version to $BIN_DIR/$BIN_NAME"
}

install_via_go() {
    if ! command -v go >/dev/null 2>&1; then
        err "No prebuilt binary and Go not installed. Install Go first: https://go.dev/dl/"
    fi

    local go_pkg="github.com/$REPO/cmd/$BIN_NAME@latest"
    log "Running: GOBIN=$BIN_DIR go install $go_pkg"
    mkdir -p "$BIN_DIR"
    export GOBIN="$BIN_DIR"

    if ! go install "$go_pkg"; then
        err "go install failed"
    fi

    # Verify binary was actually installed
    if [[ ! -f "$BIN_DIR/$BIN_NAME" ]]; then
        local default_gobin="${GOPATH:-$HOME/go}/bin"
        if [[ -f "$default_gobin/$BIN_NAME" ]]; then
            log "Moving binary from $default_gobin to $BIN_DIR"
            mv "$default_gobin/$BIN_NAME" "$BIN_DIR/$BIN_NAME"
        else
            err "go install completed but binary not found"
        fi
    fi

    log "Installed: $BIN_DIR/$BIN_NAME"
}

install_skill() {
    log "Installing skill to $SKILL_DIR/$SKILL_NAME"

    local script_dir skill_src skill_dest staging
    skill_dest="$SKILL_DIR/$SKILL_NAME"
    staging="$TMP_DIR/skill_staging"

    mkdir -p "$staging/references"

    # Try local copy first (running from repo checkout)
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" 2>/dev/null && pwd)" || script_dir=""
    skill_src="$script_dir/skills/$SKILL_NAME"

    if [[ -n "$script_dir" && -d "$skill_src" && -f "$skill_src/SKILL.md" ]]; then
        log "Source: local repo ($skill_src)"
        cp -r "$skill_src/"* "$staging/"
    else
        log "Source: GitHub (raw.githubusercontent.com)"
        local base_url="https://raw.githubusercontent.com/$REPO/main/skills/$SKILL_NAME"

        log "Downloading: SKILL.md"
        download_and_verify "$base_url/SKILL.md" "$staging/SKILL.md" \
            || err "Failed to download SKILL.md"

        log "Downloading: references/confluence-guidelines.md"
        download_and_verify "$base_url/references/confluence-guidelines.md" \
            "$staging/references/confluence-guidelines.md" \
            || err "Failed to download confluence-guidelines.md"
    fi

    # Validate skill structure
    [[ -f "$staging/SKILL.md" ]] || err "Invalid skill: missing SKILL.md"

    # Atomic install
    rm -rf "$skill_dest"
    mkdir -p "$SKILL_DIR"
    mv "$staging" "$skill_dest"

    log "Installed: $skill_dest"
}

check_path() {
    if $INSTALL_BIN; then
        # Check if binary is actually callable
        if ! command -v "$BIN_NAME" >/dev/null 2>&1; then
            echo ""
            echo "NOTE: $BIN_NAME not found in PATH after install"
            echo "Add to your shell config:"
            echo "  export PATH=\"\$PATH:$BIN_DIR\""
        fi
    fi
}

main() {
    parse_args "$@"

    # Create temp dir after trap is set
    TMP_DIR=$(mktemp -d)

    if $INSTALL_BIN; then
        install_binary
    fi

    if $INSTALL_SKILL; then
        install_skill
    fi

    check_path

    echo ""
    log "Done!"
    $INSTALL_BIN && echo "  Binary: $BIN_DIR/$BIN_NAME"
    $INSTALL_SKILL && echo "  Skill:  $SKILL_DIR/$SKILL_NAME"
}

main "$@"
