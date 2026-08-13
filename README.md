# alces-hunt

Node discovery and inventory for bare-metal HPC and cluster environments.

Machines phone home during early boot or PXE, report identity and hardware
details to a central listener, land in a **buffer**, and are later **parsed**
into a managed inventory with canonical labels and groups.

Copyright (C) 2026 Alces Software Ltd. Distributed under the Eclipse Public
License 2.0. See `LICENSE`, `LICENSE.EPL-2.0`, and `NOTICE`.

## Install on a clean Linux host

Server and send client (packages, Go, binary, systemd units):

```bash
curl -fsSL https://raw.githubusercontent.com/sierra-tango-echo/alces-hunt/main/install.sh | sudo bash
```

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
