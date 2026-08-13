# alces-hunt

**Specification**

User Stories • Design Brief • Business Model Brief

---

## Project Overview

**alces-hunt** is a command-line tool for **node discovery and inventory management** in bare-metal HPC and cluster environments.

It enables machines to "phone home" during early boot or PXE, reporting identity and hardware details to a central listener. Discovered nodes land in a buffer and are later *parsed* — assigned canonical labels and moved into a managed inventory. The tool integrates with an external profile system for automated post-parsing configuration.

### Core Workflow

- A server runs `alces-hunt hunt` and listens on a port (TCP + UDP).
- New nodes run `alces-hunt send` and transmit hostid, hostname, IP, MACs, disks, BMC info, and optional custom content.
- Nodes arrive in the **buffer** (unprocessed).
- An administrator runs `alces-hunt parse` (interactive TTY menu or `--auto`) to assign labels such as `cnode001` and promote nodes to the **parsed** list.
- Nodes can be organized into groups. Labels and groups can be renamed. Integration with external profile systems is automatic via `auto_apply` rules.

In short: it solves the classic "rack of new machines just booted — now I need to name and register them" problem.

---

## 1. User Stories

These stories are written from the perspective of real users who operate HPC clusters and use alces-hunt to commission and manage hardware.

### Cluster Administrator — Interactive Bring-Up

**As an admin, I want to boot a rack of machines into a PXE environment so that they automatically report themselves to a central listener without me having to manually collect and enter MAC addresses.**

*Benefit:* Eliminates a major source of human error and time waste during initial cluster deployment.

**As an admin, I want to run an interactive `parse` session so that I can review all discovered nodes, edit their proposed labels, and move them from the buffer into the managed inventory in one controlled workflow.**

*Benefit:* Gives the operator full visibility and final say before nodes become part of production naming.

**As an admin, I want to use prefix + start label generation (e.g. `--prefix cnode --start 001`) during parsing so that nodes receive consistent, sequential, zero-padded names without manual typing.**

*Benefit:* Enforces naming conventions at scale.

**As an admin, I want to assign arbitrary groups to nodes (compute, gpu, login, storage, etc.) so that downstream schedulers, configuration management, and monitoring tools can select logical subsets of the cluster.**

*Benefit:* Groups are the primary mechanism for organizing heterogeneous hardware.

### Operator — Automated & Lights-Out Deployment

**As an operator, I want nodes whose hostnames match a regular expression to be automatically parsed on arrival (`auto_parse`) so that known node types never require manual intervention.**

*Benefit:* Enables fully unattended expansion of standard compute or storage pools.

**As an operator, I want `auto_apply` rules so that when a node is assigned a label matching a pattern, a profile identity is automatically applied without further commands.**

*Benefit:* Closes the loop between discovery and configuration.

**As an operator, I want to run `alces-hunt autorun` (in hunt or send mode) from a service wrapper so that the tool participates correctly in Linux boot, restart, and shutdown sequences using a PID file.**

*Benefit:* Supports production deployment as a managed service.

**As an operator, I want `send --broadcast` so nodes can register without knowing a specific target host IP during early boot when DHCP or network configuration may still be settling.**

*Benefit:* Robustness in the most fragile phase of node bring-up.

### Ongoing Cluster Management

**As an admin, I want to list nodes in human or machine-readable form (`--plain`, `--by-group`) and view full details including raw collected content for any node.**

*Benefit:* Supports both day-to-day operations and scripting/auditing.

**As an admin, I want to rename a label or rename a group across many nodes without breaking references in other systems.**

*Benefit:* Handles inevitable naming corrections and reorganizations.

**As an admin, I want to remove nodes by label (or ID when operating on the buffer) or by regex, including the ability to match hostnames.**

*Benefit:* Clean decommissioning and correction of mis-registered hardware.

**As an admin, I want `--allow-existing` (and the config equivalent) so that re-provisioning the same physical node replaces its previous record rather than failing.**

*Benefit:* Real-world clusters are frequently re-imaged.

**As an admin, I want `send` to default the node's label to the machine serial number from `dmidecode -s system-serial-number` so that hardware identity is attached at registration without extra flags.**

*Benefit:* Nodes arrive already labeled with a stable, vendor-assigned identifier.

**As an admin, I want `send` to fail immediately if that default serial-number label is in use and `dmidecode` is not available, so I do not silently register unlabeled or mislabeled hardware.**

*Benefit:* Missing tooling is a hard, visible error instead of a quiet empty label.

**As an admin, I want `send --label` or `send --prefix` from the client side so I can override the serial-number default; the intended name travels with the node and is honored automatically at parse time.**

*Benefit:* Useful when the sending node already knows a different intended identity.

### Scripting & Integration Authors

**As a script author, I want stable `--plain` (tab-separated) output with predictable columns so I can reliably consume node data from shell scripts, Ansible, or other automation.**

*Benefit:* Machine-readable interface is essential for composability.

**As a script author, I want `parse --dry-run` and `parse --auto` so I can validate what labels would be generated or fully automate inventory population in CI/CD-style pipelines.**

*Benefit:* Supports testability and unattended operation.

**As an integrator, I want every configuration value to be overridable via an `ALCES_HUNT_*` environment variable so the tool works inside wrappers, containers, image-based deployments, and service managers.**

*Benefit:* Flexibility for packaging and orchestration layers.

### Resilience & Edge Cases

**As an admin, I want collision handling via `skip_used_index` so that automatic label generation continues incrementing instead of aborting when a candidate label is already taken.**

*Benefit:* Prevents brittle failures during large parallel node arrivals.

**As an admin, I want clear, actionable errors when duplicate preset labels are supplied or when a generated label would collide in the parsed list.**

*Benefit:* Fast feedback instead of silent corruption.

**As an admin, I want a simple shared `auth_key` so that only authorized nodes can register with the alces-hunt server.**

*Benefit:* Basic protection against accidental or malicious registration.

---

## 2. Design Brief — Implementation Specification

This section specifies the complete behavior of alces-hunt. Every important contract, algorithm, and quirk is called out.

### High-Level Architecture

Single-process CLI application using subcommands. Two primary roles exist: **client** (`send`) and **server** (`hunt`).

There are two distinct node lists with different addressing semantics:

- **buffer** (addressed by ID)
- **parsed** (addressed by label)

Persistence is file-based (one YAML file per node). No embedded database. Optional integration with an external "profile" system is performed via subprocess execution.

### Core Data Model

**Node**

- `id` (string, primary key, immutable) — hostid or SYSUUID value; must be unique across both lists
- `hostname` (string)
- `label` (string or null) — only present after parsing; must be unique within the parsed list
- `ip` (string) — injected by the receiving server, never sent by client
- `content` (string) — usually YAML text from the built-in collector, or arbitrary output from a content command
- `groups` (array of strings) — always sorted on display
- `presets` (map) — optional hints from the client: `{ "label": ..., "prefix": ... }`

**Key Invariants**

- A node's `id` is the true primary key across buffer and parsed lists.
- Labels must be unique only inside the parsed list.
- A node exists in exactly one list at any moment.
- When allow-existing is used, the old record (by id) is removed before the new one is written.

### Storage Format (Exact Contract)

Two directories under the installation tree: `var/buffer/` and `var/parsed/`.

Each node is stored as a separate file named `<id>.yaml`.

The YAML document contains exactly these top-level keys (order is not significant):

```
id, hostname, label, ip, content, groups, presets
```

On load, every file in the directory is read.  
On save, every node writes its own file. Implementations may choose atomic write-per-file.

### Wire Protocol & Networking

**JSON Payload (sent by client)**

```json
{
  "hostid": "...",
  "hostname": "...",
  "content": "...",
  "label": "...",        // optional preset
  "prefix": "...",       // optional preset
  "groups": ["..."],
  "auth_key": "..."
}
```

The receiver always injects the source IP address into the record.

**Transports (both required)**

- **TCP**: The client performs a POST-like request. The server performs naive manual HTTP header parsing looking for `Content-Type: application/json` and `Content-Length`. It is **not** a full HTTP server. Transport is plain HTTP (no TLS).
- **UDP**: The client can send a raw JSON datagram (broadcast mode). The server listens with a UDP socket on the same port.
- **Authentication**: Simple shared secret. The payload must contain an `auth_key` that matches the server's configured key. Mismatch results in rejection (logged, 401 response on TCP).

### CLI Command Surface (Complete)

| Command              | Addressing          | Important Options |
|----------------------|---------------------|-------------------|
| `hunt`               | —                   | `--port`, `--allow-existing`, `--include-self`, `--auth`, `--auto-parse` |
| `send`               | —                   | `-s/--server`, `-p/--port`, `--broadcast`, `--broadcast-address`, `--label`, `--prefix`, `--groups`, `-c/--command`, `--auth`, `--retry-interval` |

**Default send label.** When `--label` is omitted, `send` runs `dmidecode -s system-serial-number`, trims the stdout, and uses that value as the payload `label` / `presets["label"]`. This default is the send-label option. If it is in effect and `dmidecode` is not available (not on `PATH`, not executable, or cannot be executed), `send` fails immediately with a clear error and does not transmit. An explicit `--label` overrides the default and does not require `dmidecode`.
| `autorun`            | —                   | Dispatches according to `autorun_mode` (`hunt`\|`send`) |
| `list`               | buffer or label     | `--buffer`, `--plain`, `--by-group` |
| `show NODE`          | ID (buffer) or label| `--buffer`, `--plain` |
| `remove-node NODE[,...]` | ID or label     | genders brackets or `--regex`, `--match-hostname`, `--buffer` |
| `modify-groups NODE` | same                | `--add`, `--remove`, `--buffer`, `--regex` |
| `modify-label OLD NEW` | label only        | — |
| `rename-group OLD NEW` | label or ID       | `--buffer` |
| `parse`              | buffer → parsed     | `--auto`, `--prefix`, `--start`, `--allow-existing`, `--skip-used-index`, `--dry-run`, `--default-label` |
| `dump-buffer`        | —                   | — |

The CLI binary is `alces-hunt`. Commands are invoked as `alces-hunt <command>`.

**Node Selection Syntax**

Commands that accept `NODE[,NODE...]` support two modes:

- By default the list is comma-separated and supports **genders-style bracket expansion**: `node[1-5],login[1-2]` expands to `node1..node5,login1,login2`.
- When `--regex` is supplied, each term is treated as a regular expression against the search field (ID when `--buffer`, label otherwise). Bracket expansion is disabled in regex mode.

### Label Generation Algorithm

This is one of the most important pieces to implement exactly.

1. If the node carries a preset label (from `presets["label"]` or sent `--label`) → use it. The label must be unique in the target list.

2. Else if a prefix is present (presets or CLI `--prefix`):
   - Determine numeric start: `prefix_starts[prefix]` > `--start` > `default_start` ("01").
   - Build name = `prefix` + zero-padded counter.
   - Padding width is derived from the string length of the start value.
   - While the candidate name is already used, increment the counter and retry.

3. Else fall back to hostname, transformed by `default_label`:
   - `long` = full hostname
   - `short` = `hostname.split('.')[0]`
   - `blank` = `""`

4. `skip_used_index` behavior: when `true` and a collision occurs during generation, keep incrementing instead of raising an error.

Implementations must also handle the case where preset labels supplied in a batch are themselves duplicates (error before any node is parsed).

### Parse Flows

**Manual Parse (default)**

Interactive TTY multi-select (ordered). The implementation must preserve the order in which the user selected nodes.

For each selected node the user is prompted for a label; the prompt is pre-filled using the preset or the result of the auto-label algorithm.

Labels are validated for uniqueness against already-parsed nodes and previously chosen labels in the same session.

The multi-select widget has an early-abort path that allows label editing to mutate state before final acceptance.

**Automatic Parse (`--auto`)**

All nodes currently in the buffer are processed. Preset labels are applied first (duplicate detection across the batch). Remaining nodes receive auto-generated labels. The same uniqueness and `skip_used_index` rules apply.

`--dry-run` prints the resulting table without moving any nodes.

### Configuration System

Configuration is loaded from a YAML file (usually `etc/config.yml`) plus XDG user config paths.

Almost every top-level key can be overridden by an environment variable of the form `ALCES_HUNT_<key>` (snake_case preserved).

**Special structured keys include:**

- `auto_apply` — map of regex → identity name. First match wins. Applied only to nodes landing in the parsed list.
- `prefix_starts` — map of prefix string → start string (e.g. `{"cnode": "001"}`).
- `presets` — default label, prefix, and groups that clients may inherit.
- `profile_command` — path (or command + args) to the external profile binary. Defaults to `$ALCES_HUNT_ROOT/bin/alces-hunt profile`. Must be executable.

`default_label` defaults to `"long"`.  
`default_start` defaults to `"01"` (string — important for padding logic).

### External Integration — Profile

When a node is saved to the parsed list and auto-apply rules are active, the implementation looks up the first matching rule for the final label.

It then executes the profile command as a subprocess:

```
<profile_command> apply <label> <identity>
```

The call is best-effort. Failure is logged but does not abort the parse.

### Collector (Default Content)

When no `--command` / `content_command` is supplied, the sending node runs built-in collection:

- Read `/proc/cmdline` → `SYSUUID`, `BOOTIF`
- Enumerate `/sys/class/net/*` (exclude `lo` and dotfiles) → interface name to MAC address
- Enumerate `/sys/class/block/*` (those with a `device/` subdirectory) → name → size
- Run `ipmitool lan print 1` (best effort, may be empty) → `bmcip`, `bmcmac`
- Result is serialized as YAML and placed in the `content` field

Alternatively any shell command can be used; its stdout (chomped) becomes the content string.

### Special Behaviors

- `hunt --include-self` or config `include_self`: after the server starts, it immediately performs a send to localhost (useful for testing and single-node setups). Same default-label / `dmidecode` rules as `send`.
- Default `send` label is the trimmed stdout of `dmidecode -s system-serial-number` unless `--label` is given. If that option is in effect and `dmidecode` is not available, `send` fails and does not transmit.
- **PID file handling**: the hunt server and send client (when used under service wrappers) cooperate with a PID file whose location can be set via `ALCES_HUNT_pidfile`.
- `autorun` command: dispatches to hunt or send based on the `autorun_mode` config value. Used by the `bin/start` wrapper script.
- TCP hunt listener does manual line-by-line header parsing and then reads exactly `Content-Length` bytes. It is intentionally minimal.
- When operating on the buffer list, node selection and display use the **ID** field. When operating on the parsed list, they use the **label**.
- Plain output formats are tab-separated and have slightly different column sets depending on the command and whether `--buffer` is used.

### Non-Functional & Portability Requirements

- Must function in highly constrained early-boot / PXE environments (minimal PATH, limited binaries).
- ID-based operations must be idempotent or safely replaceable when `allow-existing` is enabled.
- Interactive parse must preserve the exact order of user selections for deterministic auto-labeling.
- All error paths that produce distinct messages (duplicate label, auth failure, missing port, collision during auto-parse, etc.) should remain distinguishable.
- Plain output must be stable and script-friendly.

---

## 3. Business Model Brief

alces-hunt is not a revenue product in isolation. It is strategic infrastructure tooling whose purpose is to reduce friction in adopting the broader HPC software and appliance ecosystem.

### Value Proposition

Commissioning a rack of bare-metal nodes has historically been painful: collect MAC addresses, decide on naming, type everything into configuration systems, repeat for every expansion.

alces-hunt turns this into a repeatable, partially or fully automated workflow. It is the difference between a multi-day manual process and a largely unattended one.

### Target Customers

- University and national laboratory HPC teams
- System integrators building turnkey research and commercial clusters
- Research computing centers and supercomputing facilities
- Organizations purchasing or building appliances that include the platform software stack

### Monetization Model

The tool is distributed under open terms (Creative Commons Attribution-ShareAlike with references to the Eclipse Public License). The intent is widespread adoption. Revenue is realized through:

- **Platform pull-through**: Once nodes are discovered and parsed, the natural next step is applying profile identities and other platform components. alces-hunt is the on-ramp.
- **Support & services**: Organizations running production clusters want support contracts, professional services, and training.
- **Appliances & integration**: Pre-configured hardware+software solutions sold by the vendor or channel partners include alces-hunt as a core component.
- **Ecosystem lock-in via integration points**: `auto_apply`, `profile_command`, `presets`, and the overall mental model create preference for the rest of the tooling.

### Strategic Role

alces-hunt is deliberately low-friction. A sysadmin can start using it in minutes during a cluster build. By the time nodes have been parsed and profiles applied, the organization has entered the platform operational model.

This is classic "tooling that sells the platform" economics common in infrastructure software.

### Implications for Implementation

- The tool must remain simple and reliable in early-boot environments.
- Both fully interactive and fully automated paths must be first-class.
- Integration points with profile/identity systems must be preserved (even if the concrete profile system changes).
- Behavioral compatibility (CLI flags, output formats, label generation rules, search syntax) is valuable for users and documentation.
- Licensing should remain permissive and clearly documented.
