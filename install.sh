#!/bin/bash
# Copyright (C) 2026 Alces Software Ltd.
# SPDX-License-Identifier: EPL-2.0
#
# Install alces-hunt on a clean Linux distribution.
#
# Curl-pipe (server + send):
#   curl -fsSL https://raw.githubusercontent.com/sierra-tango-echo/alces-hunt/main/install.sh | sudo bash
#
# Server only or send only:
#   curl -fsSL .../install.sh | sudo env MODE=server bash
#   curl -fsSL .../install.sh | sudo env MODE=send bash
#
# From a git checkout:
#   sudo ./install.sh
#
# Optional environment:
#   MODE=both|server|send     (default: both)
#   PREFIX=/opt/alces-hunt
#   REPO=https://github.com/sierra-tango-echo/alces-hunt.git
#   REF=main
#   PORT=2770
#   AUTH_KEY=
#   TARGET_HOST=
#   AUTORUN_MODE=hunt|send
#   ENABLE_SERVICE=1          enable and start the matching systemd unit

set -euo pipefail

MODE="${MODE:-both}"
PREFIX="${PREFIX:-/opt/alces-hunt}"
REPO="${REPO:-https://github.com/sierra-tango-echo/alces-hunt.git}"
REF="${REF:-main}"
PORT="${PORT:-2770}"
AUTH_KEY="${AUTH_KEY:-}"
TARGET_HOST="${TARGET_HOST:-}"
ENABLE_SERVICE="${ENABLE_SERVICE:-0}"
GO_VERSION="${GO_VERSION:-1.22.10}"

log()  { printf '==> %s\n' "$*" >&2; }
warn() { printf 'warning: %s\n' "$*" >&2; }
die()  { printf 'error: %s\n' "$*" >&2; exit 1; }

need_root() {
  if [ "$(id -u)" -ne 0 ]; then
    die "run as root (sudo) so packages and ${PREFIX} can be installed"
  fi
}

detect_pm() {
  if command -v apt-get >/dev/null 2>&1; then
    echo apt
  elif command -v dnf >/dev/null 2>&1; then
    echo dnf
  elif command -v yum >/dev/null 2>&1; then
    echo yum
  elif command -v zypper >/dev/null 2>&1; then
    echo zypper
  elif command -v apk >/dev/null 2>&1; then
    echo apk
  elif command -v pacman >/dev/null 2>&1; then
    echo pacman
  else
    echo none
  fi
}

pkg_install() {
  local pm="$1"
  shift
  [ "$#" -eq 0 ] && return 0
  case "$pm" in
    apt)
      export DEBIAN_FRONTEND=noninteractive
      apt-get update -y
      apt-get install -y --no-install-recommends "$@"
      ;;
    dnf)    dnf install -y "$@" ;;
    yum)    yum install -y "$@" ;;
    zypper) zypper --non-interactive install -y "$@" ;;
    apk)    apk add --no-cache "$@" ;;
    pacman) pacman -Sy --noconfirm "$@" ;;
    none)   warn "no package manager detected; install these by hand: $*" ;;
  esac
}

install_packages() {
  local pm
  pm="$(detect_pm)"
  log "package manager: ${pm}"

  local build_pkgs=()
  local send_pkgs=()
  case "$pm" in
    apt)
      build_pkgs=(ca-certificates curl git gcc make)
      send_pkgs=(dmidecode ipmitool iproute2)
      ;;
    dnf|yum)
      build_pkgs=(ca-certificates curl git gcc make)
      send_pkgs=(dmidecode ipmitool iproute)
      ;;
    zypper)
      build_pkgs=(ca-certificates curl git gcc make)
      send_pkgs=(dmidecode ipmitool iproute2)
      ;;
    apk)
      build_pkgs=(ca-certificates curl git gcc make musl-dev)
      send_pkgs=(dmidecode ipmitool iproute2)
      ;;
    pacman)
      build_pkgs=(ca-certificates curl git gcc make)
      send_pkgs=(dmidecode ipmitool iproute2)
      ;;
    none)
      build_pkgs=()
      send_pkgs=()
      ;;
  esac

  pkg_install "$pm" "${build_pkgs[@]}"
  case "$MODE" in
    send|both)
      pkg_install "$pm" "${send_pkgs[@]}"
      ;;
  esac
}

have_go() {
  command -v go >/dev/null 2>&1 || return 1
  local ver maj min
  ver="$(go env GOVERSION 2>/dev/null || go version | awk '{print $3}')"
  ver="${ver#go}"
  maj="${ver%%.*}"
  min="${ver#*.}"
  min="${min%%.*}"
  [ "${maj:-0}" -gt 1 ] || { [ "${maj:-0}" -eq 1 ] && [ "${min:-0}" -ge 22 ]; }
}

install_go() {
  if have_go; then
    log "using existing Go $(go env GOVERSION 2>/dev/null || go version)"
    return 0
  fi
  local arch os tarball
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$(uname -m)" in
    x86_64|amd64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    armv7l) arch=armv6l ;;
    *) die "unsupported architecture $(uname -m) — install Go 1.22+ manually" ;;
  esac
  tarball="go${GO_VERSION}.${os}-${arch}.tar.gz"
  log "installing Go ${GO_VERSION} to /usr/local/go"
  curl -fsSL "https://go.dev/dl/${tarball}" -o "/tmp/${tarball}"
  rm -rf /usr/local/go
  tar -C /usr/local -xzf "/tmp/${tarball}"
  rm -f "/tmp/${tarball}"
  export PATH="/usr/local/go/bin:${PATH}"
  have_go || die "Go ${GO_VERSION} installed but not usable"
}

script_dir() {
  local src="${BASH_SOURCE[0]:-}"
  if [ -n "$src" ] && [ -f "$src" ]; then
    cd "$(dirname "$src")" >/dev/null && pwd
  else
    printf '%s' ""
  fi
}

# Sets SOURCE_DIR. Must not print the path on stdout — callers do not
# capture this function (curl|bash + logging used to mix into $src).
obtain_source() {
  local here dest
  here="$(script_dir)"
  if [ -n "$here" ] && [ -f "${here}/go.mod" ] && [ -f "${here}/cmd/alces-hunt/main.go" ]; then
    SOURCE_DIR="$here"
    log "using local source ${SOURCE_DIR}"
    return 0
  fi
  dest="$(mktemp -d /tmp/alces-hunt-src.XXXXXX)"
  log "cloning ${REPO} (${REF}) into ${dest}"
  git clone --depth 1 --branch "$REF" "$REPO" "$dest/alces-hunt" >&2
  SOURCE_DIR="${dest}/alces-hunt"
  if [ ! -f "${SOURCE_DIR}/go.mod" ]; then
    die "clone did not produce a source tree at ${SOURCE_DIR}"
  fi
}

write_config() {
  local dest="$1"
  if [ -f "$dest" ]; then
    log "keeping existing ${dest}"
    return 0
  fi
  local autorun="hunt"
  case "$MODE" in
    send) autorun="send" ;;
  esac
  if [ -n "${AUTORUN_MODE:-}" ]; then
    autorun="$AUTORUN_MODE"
  fi
  cat >"$dest" <<EOF
---
port: ${PORT}
autorun_mode: ${autorun}
include_self: false
allow_existing: false
auth_key: "${AUTH_KEY}"
broadcast_address: 255.255.255.255
default_label: long
default_start: "01"
skip_used_index: true
retry_interval: 5
EOF
  if [ -n "$TARGET_HOST" ]; then
    printf 'target_host: %s\n' "$TARGET_HOST" >>"$dest"
  fi
  chmod 0644 "$dest"
}

install_tree() {
  local src="$1"
  if [ ! -d "$src" ] || [ ! -f "${src}/go.mod" ]; then
    die "source path is not an alces-hunt checkout: ${src}"
  fi
  log "building alces-hunt"
  (
    cd "$src"
    export PATH="/usr/local/go/bin:${PATH}"
    mkdir -p bin
    go build -trimpath -ldflags "-s -w" -o bin/alces-hunt ./cmd/alces-hunt
  )

  log "installing to ${PREFIX}"
  mkdir -p "${PREFIX}/bin" "${PREFIX}/etc" "${PREFIX}/var/buffer" "${PREFIX}/var/parsed" "${PREFIX}/var/log" "${PREFIX}/share/doc" "${PREFIX}/share/licenses"
  install -m 0755 "${src}/bin/alces-hunt" "${PREFIX}/bin/alces-hunt"
  install -m 0755 "${src}/bin/start" "${PREFIX}/bin/start"
  write_config "${PREFIX}/etc/config.yml"
  if [ -f "${src}/etc/config.yml.ex" ]; then
    install -m 0644 "${src}/etc/config.yml.ex" "${PREFIX}/etc/config.yml.ex"
  fi
  install -m 0644 "${src}/LICENSE" "${PREFIX}/share/licenses/LICENSE"
  install -m 0644 "${src}/NOTICE" "${PREFIX}/share/licenses/NOTICE"
  install -m 0644 "${src}/LICENSE.EPL-2.0" "${PREFIX}/share/licenses/LICENSE.EPL-2.0"
  if [ -d /etc/systemd/system ] && [ -d "${src}/systemd" ]; then
    install -m 0644 "${src}/systemd/alces-hunt-server.service" /etc/systemd/system/alces-hunt-server.service
    install -m 0644 "${src}/systemd/alces-hunt-send.service" /etc/systemd/system/alces-hunt-send.service
    # Point units at PREFIX if it is not the default.
    if [ "$PREFIX" != /opt/alces-hunt ]; then
      sed -i.bak "s|/opt/alces-hunt|${PREFIX}|g" \
        /etc/systemd/system/alces-hunt-server.service \
        /etc/systemd/system/alces-hunt-send.service
      rm -f /etc/systemd/system/alces-hunt-server.service.bak /etc/systemd/system/alces-hunt-send.service.bak
    fi
    systemctl daemon-reload || true
  fi
  mkdir -p /usr/local/bin
  ln -sfn "${PREFIX}/bin/alces-hunt" /usr/local/bin/alces-hunt
}

maybe_enable_service() {
  [ "$ENABLE_SERVICE" = "1" ] || return 0
  command -v systemctl >/dev/null 2>&1 || { warn "systemctl not found; skip service enable"; return 0; }
  case "$MODE" in
    server|both)
      systemctl enable --now alces-hunt-server.service
      ;;
    send)
      systemctl enable alces-hunt-send.service
      systemctl start alces-hunt-send.service || true
      ;;
  esac
}

print_next_steps() {
  cat <<EOF

alces-hunt installed to ${PREFIX}

  binary : ${PREFIX}/bin/alces-hunt  (also /usr/local/bin/alces-hunt)
  config : ${PREFIX}/etc/config.yml
  data   : ${PREFIX}/var/{buffer,parsed}

Edit the config (port, auth_key, target_host) before first use.

Server (listener):
  sudo ALCES_HUNT_ROOT=${PREFIX} alces-hunt hunt
  sudo systemctl enable --now alces-hunt-server

Send (client, needs dmidecode unless you pass --label):
  sudo ALCES_HUNT_ROOT=${PREFIX} alces-hunt send --server <hunt-host>
  sudo ALCES_HUNT_ROOT=${PREFIX} alces-hunt send --broadcast --broadcast-address 255.255.255.255

Every config key can be overridden with ALCES_HUNT_<key>, for example:
  ALCES_HUNT_port=2770 ALCES_HUNT_auth_key=secret ALCES_HUNT_target_host=10.0.0.1

EOF
}

main() {
  case "$MODE" in
    server|send|both) ;;
    *) die "MODE must be server, send, or both (got '${MODE}')" ;;
  esac
  need_root
  install_packages
  install_go
  SOURCE_DIR=""
  obtain_source
  install_tree "$SOURCE_DIR"
  maybe_enable_service
  print_next_steps
}

main "$@"
