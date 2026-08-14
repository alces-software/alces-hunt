#!/bin/bash
# Copyright (C) 2026 Alces Software Ltd.
# SPDX-License-Identifier: EPL-2.0
#
# Download a pre-built alces-hunt binary and install it.
#
# System install (when /opt is writable, typically via sudo):
#   curl -fsSL https://raw.githubusercontent.com/alces-software/alces-hunt/main/install.sh | sudo bash
#
# Local install (no root — uses ~/.local/alces-hunt):
#   curl -fsSL https://raw.githubusercontent.com/alces-software/alces-hunt/main/install.sh | bash
#
# Optional environment:
#   VERSION=latest|v0.2       release tag to download (default: latest)
#   PREFIX=/opt/alces-hunt    override install root
#   MODE=both|server|send     send mode may install dmidecode/ipmitool if root
#   PORT=2770
#   AUTH_KEY=
#   TARGET_HOST=
#   AUTORUN_MODE=hunt|send
#   ENABLE_SERVICE=1          enable systemd unit (system install only)
#   REPO_SLUG=alces-software/alces-hunt

set -euo pipefail

MODE="${MODE:-both}"
VERSION="${VERSION:-latest}"
PORT="${PORT:-2770}"
AUTH_KEY="${AUTH_KEY:-}"
TARGET_HOST="${TARGET_HOST:-}"
ENABLE_SERVICE="${ENABLE_SERVICE:-0}"
REPO_SLUG="${REPO_SLUG:-alces-software/alces-hunt}"
RELEASE_BASE="https://github.com/${REPO_SLUG}/releases"

log()  { printf '==> %s\n' "$*" >&2; }
warn() { printf 'warning: %s\n' "$*" >&2; }
die()  { printf 'error: %s\n' "$*" >&2; exit 1; }

writable_dir() {
  local dir="$1"
  if [ -d "$dir" ]; then
    [ -w "$dir" ]
  else
    local parent
    parent="$(dirname "$dir")"
    [ -d "$parent" ] && [ -w "$parent" ]
  fi
}

choose_prefix() {
  if [ -n "${PREFIX:-}" ]; then
    return 0
  fi
  if writable_dir /opt/alces-hunt || writable_dir /opt; then
    PREFIX=/opt/alces-hunt
  else
    PREFIX="${HOME}/.local/alces-hunt"
  fi
}

choose_bindir() {
  if [ -n "${BINDIR:-}" ]; then
    return 0
  fi
  if writable_dir /usr/local/bin; then
    BINDIR=/usr/local/bin
  else
    BINDIR="${HOME}/.local/bin"
  fi
}

detect_arch() {
  case "$(uname -s | tr '[:upper:]' '[:lower:]')" in
    linux) ;;
    *) die "pre-built binaries are Linux-only (this host is $(uname -s))" ;;
  esac
  case "$(uname -m)" in
    x86_64|amd64)  ARCH=amd64 ;;
    aarch64|arm64) ARCH=arm64 ;;
    *) die "unsupported architecture $(uname -m); expected x86_64 or aarch64" ;;
  esac
}

need_curl() {
  if command -v curl >/dev/null 2>&1; then
    return 0
  fi
  die "curl is required to download the release binary"
}

asset_url() {
  local file="$1"
  if [ "$VERSION" = "latest" ]; then
    printf '%s/latest/download/%s' "$RELEASE_BASE" "$file"
  else
    printf '%s/download/%s/%s' "$RELEASE_BASE" "$VERSION" "$file"
  fi
}

download_release() {
  local name tarball url dest
  name="alces-hunt-linux-${ARCH}"
  tarball="${name}.tar.gz"
  dest="$(mktemp -d /tmp/alces-hunt-dl.XXXXXX)"
  STAGING="$dest"
  url="$(asset_url "$tarball")"
  log "downloading ${url}"
  if curl -fL --retry 3 --retry-delay 1 -o "${dest}/${tarball}" "$url"; then
    tar -C "$dest" -xzf "${dest}/${tarball}"
    if [ -x "${dest}/${name}/bin/alces-hunt" ]; then
      EXTRACTED="${dest}/${name}"
      return 0
    fi
    die "tarball did not contain ${name}/bin/alces-hunt"
  fi

  warn "tarball not available; falling back to raw binary"
  url="$(asset_url "$name")"
  log "downloading ${url}"
  mkdir -p "${dest}/bin"
  curl -fL --retry 3 --retry-delay 1 -o "${dest}/bin/alces-hunt" "$url" \
    || die "failed to download ${name} from ${url}"
  chmod 0755 "${dest}/bin/alces-hunt"
  EXTRACTED="$dest"
}

write_start_script() {
  local dest="$1"
  cat >"$dest" <<'EOF'
#!/bin/bash
# Copyright (C) 2026 Alces Software Ltd.
# SPDX-License-Identifier: EPL-2.0
set -euo pipefail
pid_file="${1:-}"
if [ -z "$pid_file" ]; then
  echo "No pid_file provided!" >&2
  exit 1
fi
install_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export ALCES_HUNT_ROOT="${ALCES_HUNT_ROOT:-$install_dir}"
mkdir -p "${ALCES_HUNT_ROOT}/var/log"
exec env ALCES_HUNT_pidfile="$pid_file" \
  "${install_dir}/bin/alces-hunt" autorun \
  2>&1 | tee -a "${ALCES_HUNT_ROOT}/var/log/log.txt"
EOF
  chmod 0755 "$dest"
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

write_unit() {
  local dest="$1"
  local mode="$2"
  local desc exec_type extra
  if [ "$mode" = hunt ]; then
    desc="alces-hunt discovery server (hunt mode)"
    exec_type="Type=simple"
    extra=$'Restart=on-failure\nRestartSec=5'
  else
    desc="alces-hunt client (send mode)"
    exec_type=$'Type=oneshot\nRemainAfterExit=no'
    extra=""
  fi
  cat >"$dest" <<EOF
[Unit]
Description=${desc}
Documentation=https://github.com/${REPO_SLUG}
After=network-online.target
Wants=network-online.target

[Service]
${exec_type}
Environment=ALCES_HUNT_ROOT=${PREFIX}
Environment=ALCES_HUNT_autorun_mode=${mode}
Environment=ALCES_HUNT_pidfile=${PREFIX}/var/alces-hunt-${mode}.pid
ExecStart=${PREFIX}/bin/alces-hunt autorun
WorkingDirectory=${PREFIX}
${extra}

[Install]
WantedBy=multi-user.target
EOF
}

install_runtime_packages() {
  case "$MODE" in
    send|both) ;;
    *) return 0 ;;
  esac
  if [ "$(id -u)" -ne 0 ]; then
    warn "not root; install dmidecode (and ipmitool if you want BMC fields) with your package manager"
    return 0
  fi
  local pm=""
  if command -v dnf >/dev/null 2>&1; then
    pm=dnf
  elif command -v yum >/dev/null 2>&1; then
    pm=yum
  elif command -v apt-get >/dev/null 2>&1; then
    pm=apt
  elif command -v zypper >/dev/null 2>&1; then
    pm=zypper
  elif command -v apk >/dev/null 2>&1; then
    pm=apk
  elif command -v pacman >/dev/null 2>&1; then
    pm=pacman
  fi
  [ -n "$pm" ] || { warn "no package manager; skip dmidecode/ipmitool"; return 0; }
  log "installing send-mode packages via ${pm}"
  case "$pm" in
    apt)
      export DEBIAN_FRONTEND=noninteractive
      apt-get update -y
      apt-get install -y --no-install-recommends dmidecode ipmitool iproute2
      ;;
    dnf)    dnf install -y dmidecode ipmitool iproute ;;
    yum)    yum install -y dmidecode ipmitool iproute ;;
    zypper) zypper --non-interactive install -y dmidecode ipmitool iproute2 ;;
    apk)    apk add --no-cache dmidecode ipmitool iproute2 ;;
    pacman) pacman -Sy --noconfirm dmidecode ipmitool iproute2 ;;
  esac
}

install_tree() {
  local src="$1"
  log "installing to ${PREFIX}"
  mkdir -p "${PREFIX}/bin" "${PREFIX}/etc" "${PREFIX}/var/buffer" "${PREFIX}/var/parsed" "${PREFIX}/var/log" "${PREFIX}/share/licenses" "${PREFIX}/systemd"

  install -m 0755 "${src}/bin/alces-hunt" "${PREFIX}/bin/alces-hunt"
  if [ -f "${src}/bin/start" ]; then
    install -m 0755 "${src}/bin/start" "${PREFIX}/bin/start"
  else
    write_start_script "${PREFIX}/bin/start"
  fi

  write_config "${PREFIX}/etc/config.yml"
  if [ -f "${src}/etc/config.yml.ex" ]; then
    install -m 0644 "${src}/etc/config.yml.ex" "${PREFIX}/etc/config.yml.ex"
  fi
  for lic in LICENSE LICENSE.EPL-2.0 NOTICE; do
    if [ -f "${src}/${lic}" ]; then
      install -m 0644 "${src}/${lic}" "${PREFIX}/share/licenses/${lic}"
    fi
  done

  write_unit "${PREFIX}/systemd/alces-hunt-server.service" hunt
  write_unit "${PREFIX}/systemd/alces-hunt-send.service" send
  if [ -d /etc/systemd/system ] && writable_dir /etc/systemd/system; then
    install -m 0644 "${PREFIX}/systemd/alces-hunt-server.service" /etc/systemd/system/alces-hunt-server.service
    install -m 0644 "${PREFIX}/systemd/alces-hunt-send.service" /etc/systemd/system/alces-hunt-send.service
    systemctl daemon-reload || true
    SYSTEM_UNITS=1
  else
    SYSTEM_UNITS=0
  fi

  mkdir -p "$BINDIR"
  ln -sfn "${PREFIX}/bin/alces-hunt" "${BINDIR}/alces-hunt"
}

maybe_enable_service() {
  [ "$ENABLE_SERVICE" = "1" ] || return 0
  if [ "${SYSTEM_UNITS:-0}" != "1" ]; then
    warn "systemd system units were not installed; start ${PREFIX}/bin/alces-hunt by hand"
    return 0
  fi
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

  binary : ${PREFIX}/bin/alces-hunt  (also ${BINDIR}/alces-hunt)
  config : ${PREFIX}/etc/config.yml
  data   : ${PREFIX}/var/{buffer,parsed}

The binary finds config and var under the install root automatically
(when invoked as ${PREFIX}/bin/alces-hunt or via the symlink).
Override with ALCES_HUNT_ROOT=${PREFIX} if you copy the binary elsewhere.

EOF
  case ":${PATH}:" in
    *":${BINDIR}:"*) ;;
    *)
      printf 'Add %s to PATH, for example:\n  export PATH="%s:$PATH"\n\n' "$BINDIR" "$BINDIR"
      ;;
  esac
  cat <<EOF
Edit ${PREFIX}/etc/config.yml (port, auth_key, target_host) before first use.

Server:
  ${PREFIX}/bin/alces-hunt hunt
Send (needs dmidecode unless you pass --label):
  ${PREFIX}/bin/alces-hunt send --server <hunt-host>

Every config key can be overridden with ALCES_HUNT_<key>.
EOF
}

cleanup() {
  if [ -n "${STAGING:-}" ] && [ -d "${STAGING:-}" ]; then
    rm -rf "$STAGING"
  fi
}

main() {
  case "$MODE" in
    server|send|both) ;;
    *) die "MODE must be server, send, or both (got '${MODE}')" ;;
  esac
  trap cleanup EXIT
  choose_prefix
  choose_bindir
  detect_arch
  need_curl
  log "prefix=${PREFIX} bindir=${BINDIR} arch=${ARCH} version=${VERSION}"
  install_runtime_packages
  STAGING=""
  EXTRACTED=""
  download_release
  install_tree "$EXTRACTED"
  maybe_enable_service
  print_next_steps
}

main "$@"
