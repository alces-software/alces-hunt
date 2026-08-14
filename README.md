# alces-hunt

Node discovery and inventory for bare-metal HPC and cluster environments.

Machines phone home during early boot or PXE, report identity and hardware
details to a central listener, land in a **buffer**, and are later **parsed**
into a managed inventory with canonical labels and groups.

Copyright (C) 2026 Alces Software Ltd. Distributed under the Eclipse Public
License 2.0. See `LICENSE`, `LICENSE.EPL-2.0`, and `NOTICE`.

## Pre-built binaries

Push a tag `vX.Y.Z` to publish Linux amd64 and arm64 binaries on the
[GitHub Releases](https://github.com/sierra-tango-echo/alces-hunt/releases)
page. Each release includes the raw binary, a tarball (config example,
licenses, systemd units, `bin/start`), and `SHA256SUMS`.

```bash
# Linux x86_64
curl -fsSL -o alces-hunt \
  https://github.com/sierra-tango-echo/alces-hunt/releases/latest/download/alces-hunt-linux-amd64
chmod +x alces-hunt
sudo install -m 0755 alces-hunt /usr/local/bin/alces-hunt
```

For arm64 use `alces-hunt-linux-arm64`. Builds from `main` are also
available as workflow artifacts if you have not tagged a release yet.

The binary needs a data root (`ALCES_HUNT_ROOT`) with `var/buffer` and
`var/parsed`. Send mode still needs `dmidecode` on the node unless you
pass `--label`.

## Install on a clean Linux host

Server and send client (packages, Go, binary, systemd units):

```bash
curl -fsSL https://raw.githubusercontent.com/sierra-tango-echo/alces-hunt/main/install.sh | sudo bash
```

The script that runs is the one curl downloads. After a fix lands on `main`, re-run that command (add `?$(date +%s)` to the URL if a cache serves the old file).

Server only or send only:

```bash
curl -fsSL https://raw.githubusercontent.com/sierra-tango-echo/alces-hunt/main/install.sh | sudo env MODE=server bash
curl -fsSL https://raw.githubusercontent.com/sierra-tango-echo/alces-hunt/main/install.sh | sudo env MODE=send bash
```

From a checkout:

```bash
sudo ./install.sh
sudo MODE=send ./install.sh
```

The script installs upstream packages (`git`, `gcc`, `make`, `curl`; plus
`dmidecode` and `ipmitool` for send mode), builds the Go binary, and installs
to `/opt/alces-hunt`. Override `PREFIX`, `PORT`, `AUTH_KEY`, `TARGET_HOST`,
`REPO`, and `REF` as needed.

## Quick start

```bash
# Server
sudo ALCES_HUNT_ROOT=/opt/alces-hunt alces-hunt hunt --port 2770 --auth secret

# Client (default label is dmidecode system-serial-number)
sudo ALCES_HUNT_ROOT=/opt/alces-hunt alces-hunt send --server 10.0.0.1 --port 2770 --auth secret
sudo ALCES_HUNT_ROOT=/opt/alces-hunt alces-hunt send --broadcast --broadcast-address 10.0.0.255 --port 2770

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
