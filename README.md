# alces-hunt

Node discovery and inventory for bare-metal HPC and cluster environments.

Machines phone home during early boot or PXE, report identity and hardware
details to a central listener, land in a **buffer**, and are later **parsed**
into a managed inventory with canonical labels and groups.

Copyright (C) 2026 Alces Software Ltd. Distributed under the Eclipse Public
License 2.0. See `LICENSE`, `LICENSE.EPL-2.0`, and `NOTICE`.

## Install

`install.sh` downloads the pre-built Linux binary from GitHub Releases. It
does not compile anything.

**System install** — `/opt/alces-hunt` when that location is writable
(usually `sudo`):

```bash
curl -fsSL https://raw.githubusercontent.com/alces-software/alces-hunt/main/install.sh | sudo bash
```

**Local install** — `~/.local/alces-hunt` when `/opt` is not writable:

```bash
curl -fsSL https://raw.githubusercontent.com/alces-software/alces-hunt/main/install.sh | bash
```

Config, inventory, and logs stay under the install root:

| Path | Role |
|---|---|
| `$PREFIX/bin/alces-hunt` | binary |
| `$PREFIX/etc/config.yml` | config |
| `$PREFIX/var/buffer` | unparsed nodes |
| `$PREFIX/var/parsed` | named inventory |
| `$PREFIX/var/log` | logs |

The binary locates that root from its own path (or `ALCES_HUNT_ROOT`). A
symlink is also placed in `/usr/local/bin` or `~/.local/bin`.

Useful environment variables:

| Variable | Default |
|---|---|
| `VERSION` | `latest` (or pin e.g. `v0.2`) |
| `PREFIX` | `/opt/alces-hunt` if writable, else `~/.local/alces-hunt` |
| `MODE` | `both` (`server` / `send`; send may install `dmidecode` if root) |
| `PORT` / `AUTH_KEY` | written into a new `etc/config.yml` |
| `ENABLE_SERVICE=1` | enable the systemd unit (system install) |
| `PUBLISH` | directory to copy `install.sh`, the tarball, and a client `config.yml` into. Unset skips publishing |
| `DIST_URL` | unset uses GitHub. `MODE=send` defaults to `http://10.178.0.1/repo/alces-hunt/` |
| `TARGET_HOST` | client server address published in `config.yml` (default: `hostname -i`) |
| `BROADCAST_ADDRESS` | UDP broadcast address written into `config.yml`. Unset leaves broadcast off |

### Nodes without internet

Install once on a host that can reach GitHub, and publish the script, tarball, and client config. Pass the same `AUTH_KEY` and `PORT` the server should use:

```bash
curl -fsSL https://raw.githubusercontent.com/alces-software/alces-hunt/main/install.sh \
  | sudo env PUBLISH=/var/www/hunter AUTH_KEY=secret PORT=2770 bash
```

`PUBLISH` is the directory that receives `install.sh`, `alces-hunt-linux-<arch>.tar.gz`, and `config.yml`. The published config carries the install-time settings (including `auth_key` and `port`), sets `autorun_mode` to `send`, and sets `target_host` to this machine's address from `hostname -i`. Set `TARGET_HOST` when that address is the wrong interface. Serve that directory over HTTP yourself.

On each cluster node, install in send mode. `MODE=send` defaults `DIST_URL` to `http://10.178.0.1/repo/alces-hunt/` and copies the published `config.yml` into place:

```bash
curl -fsSL "http://10.178.0.1/repo/alces-hunt/install.sh" \
  | sudo env MODE=send bash
```

Set `DIST_URL` to another directory to use a different mirror, or set it empty to download that send install from GitHub. A pinned `VERSION` is ignored while `DIST_URL` is set, because the mirror already holds the staged release. An existing `$PREFIX/etc/config.yml` is left unchanged.

## Pre-built binaries

Each [GitHub Release](https://github.com/alces-software/alces-hunt/releases)
attaches `alces-hunt-linux-amd64`, `alces-hunt-linux-arm64`, matching
tarballs, and `SHA256SUMS`.

```bash
curl -fsSL -o alces-hunt \
  https://github.com/alces-software/alces-hunt/releases/latest/download/alces-hunt-linux-amd64
chmod +x alces-hunt
```

If you copy only the binary, set `ALCES_HUNT_ROOT` to a directory that
contains `etc/` and `var/`. Send mode still needs `dmidecode` unless you
pass `--label`.

## Quick start

```bash
# Server
alces-hunt hunt --port 2770 --auth secret

# Client (default label is dmidecode system-serial-number)
alces-hunt send --server 10.0.0.1 --port 2770 --auth secret
alces-hunt send --broadcast --broadcast-address 10.0.0.255 --port 2770

# Review and name nodes
alces-hunt list --buffer --plain
alces-hunt parse --auto --prefix cnode --start 001
alces-hunt list --plain
```

`send` without `--label` requires `dmidecode`. Pass `--label` to skip it.

## Commands

| Command | Role |
|---|---|
| `hunt` | Dual TCP+UDP listener |
| `send` | Client phone-home |
| `autorun` | Dispatches to hunt or send from `autorun_mode` |
| `parse` | Buffer → parsed (`--auto` or interactive TTY) |
| `list` / `show` | Inventory (`--plain`, `--buffer`, `--by-group`) |
| `remove-node` | Delete by label, ID (`--buffer`), regex, or hostname |
| `modify-groups` / `modify-label` / `rename-group` | Inventory edits |
| `dump-buffer` | Empty the buffer |

Configuration lives in `$ALCES_HUNT_ROOT/etc/config.yml`. Every scalar key
can be overridden by `ALCES_HUNT_<key>` (snake_case preserved), for example
`ALCES_HUNT_port`, `ALCES_HUNT_auth_key`, `ALCES_HUNT_pidfile`.

## Build from source

Requires Go 1.22+.

```bash
make
make test
ALCES_HUNT_ROOT=$PWD ./bin/alces-hunt --help
```

## License

Eclipse Public License 2.0. Copyright (C) 2026 Alces Software Ltd.
