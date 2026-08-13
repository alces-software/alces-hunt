# alces-hunt — Design Brief

**Implementation Specification**

This document provides everything needed to build alces-hunt.

---

## High-Level Architecture

Single-process CLI application using subcommands.

Two primary roles:

- **Client**: `send`
- **Server**: `hunt`

Two distinct node lists with different addressing semantics:

- **buffer** — addressed by `id`
- **parsed** — addressed by `label`

Persistence is strictly file-based (one YAML file per node). No database.

Optional integration with an external "profile" system is performed via subprocess.

---

## Core Data Model

### Node

| Field     | Type          | Description |
|-----------|---------------|-------------|
| `id`      | string        | Primary key (immutable). Usually hostid or SYSUUID. Unique across both lists. |
| `hostname`| string        | The node's hostname at send time. |
| `label`   | string \| null| Only set after parsing. Must be unique within the **parsed** list only. |
| `ip`      | string        | Injected by the receiver (never sent by client). |
| `content` | string        | Usually YAML (from built-in collector) or arbitrary text from a content command. |
| `groups`  | array<string> | Always sorted on output. |
| `presets` | object        | `{ "label"?, "prefix"? }` — hints sent by the client to influence label generation. |

### Key Invariants

- `id` is the true primary key across **both** buffer and parsed lists.
- Labels are only required to be unique inside the parsed list.
- A node exists in exactly one list at a time.
- When `--allow-existing` / `allow_existing` is true, the old record (matched by `id`) is deleted before the new one is inserted.

---

## Storage Format (Exact Contract)

Two directories:

- `var/buffer/`
- `var/parsed/`

Each node is persisted as its own file named `<id>.yaml`.

The YAML contains exactly these top-level keys:

```yaml
id: "..."
hostname: "..."
label: "..."          # may be absent/null in buffer
ip: "..."
content: "..."        # often a multi-line YAML string
groups: [...]
presets:
  label: "..."
  prefix: "..."
```

- Loading a node list reads every file in the directory.
- Save writes every node as its own file (atomic-per-file is acceptable).

---

## Wire Protocol & Networking

### JSON Payload (sent by client)

```json
{
  "hostid": "...",
  "hostname": "...",
  "content": "...",
  "label": "...",        // optional
  "prefix": "...",       // optional
  "groups": ["..."],     // optional
  "auth_key": "..."
}
```

The receiver always adds `"ip"` using the source address.

### Transports (both must be supported)

**TCP**
- Client sends a POST-like request.
- Server does naive manual header parsing looking for:
  - `Content-Type: application/json`
  - `Content-Length: N`
- Then reads exactly N bytes of JSON body.
- Not a real HTTP server.
- Transport is plain HTTP (no TLS).

**UDP**
- Client can send raw JSON as a UDP datagram (used for `--broadcast`).
- Server binds a UDP socket on the same port.

**Authentication**
- Simple shared secret.
- `auth_key` in the payload must match the server's configured key.
- Mismatch → reject (log message + 401 on TCP).

---

## CLI Command Surface

| Command              | Addressing (buffer vs parsed)      | Key Flags |
|----------------------|------------------------------------|-----------|
| `hunt`               | —                                  | `--port`, `--allow-existing`, `--include-self`, `--auth`, `--auto-parse` |
| `send`               | —                                  | `-s/--server`, `-p/--port`, `--broadcast`, `--broadcast-address`, `--label`, `--prefix`, `--groups`, `-c/--command`, `--auth`, `--retry-interval` |

When `--label` is omitted, `send` sets the preset label from `dmidecode -s system-serial-number` (see Default Send Label).
| `autorun`            | —                                  | Uses `autorun_mode` from config |
| `list`               | `--buffer` uses ID, else label     | `--plain`, `--by-group`, `--buffer` |
| `show NODE`          | ID (buffer) or label               | `--buffer`, `--plain` |
| `remove-node`        | ID or label                        | comma list + genders brackets or `--regex`, `--match-hostname`, `--buffer` |
| `modify-groups`      | same as remove                     | `--add`, `--remove`, `--buffer`, `--regex` |
| `modify-label`       | label only                         | `OLD NEW` |
| `rename-group`       | label or ID                        | `--buffer` |
| `parse`              | buffer → parsed                    | `--auto`, `--prefix`, `--start`, `--allow-existing`, `--skip-used-index`, `--dry-run`, `--default-label` |
| `dump-buffer`        | —                                  | — |

The CLI binary is `alces-hunt`. Commands are invoked as `alces-hunt <command>`.

### Default Send Label

Unless `--label` is supplied, `send` obtains the preset label by running:

```
dmidecode -s system-serial-number
```

- Use the command's stdout, stripped of surrounding whitespace, as `presets["label"]` (and as the payload `label` field).
- This is the default send-label option. It is in effect whenever `--label` is not specified.
- If this option is in effect and `dmidecode` is not available (not found on `PATH`, not executable, or cannot be executed), `send` **must fail** immediately with a clear error and **must not** transmit a payload.
- An explicit `--label` overrides the default and does **not** require `dmidecode`.
- `hunt --include-self` uses the same `send` path, so the same rule applies.

### Node Selection Syntax

- Default: comma-separated list supporting genders-style expansion: `node[1-5],login[1-2]`
- `--regex`: each term is a regex. No bracket expansion.

When `--buffer` is used, matching and display use the `id` field.  
Otherwise they use `label`.

---

## Label Generation Algorithm (Critical)

This logic must be reproduced exactly.

1. **Preset label wins**  
   If the node has `presets["label"]` (or was sent with `--label`), use it. Must be unique in target list.

2. **Prefix-based generation**  
   If a prefix exists (`presets["prefix"]` or `--prefix`):
   - Start value priority: `prefix_starts[prefix]` > `--start` > `default_start` ("01")
   - Name = `prefix` + zero-padded number
   - Padding width = `max(0, start_string.length - current_number_string.length)`
   - Increment while name is already used (respecting `skip_used_index`)

3. **Hostname fallback**  
   If no prefix:
   - `default_label == "long"` → full hostname
   - `default_label == "short"` → hostname.split('.')[0]
   - `default_label == "blank"` → ""

4. `skip_used_index`  
   When true, on collision during generation the algorithm keeps incrementing instead of failing.

Additional rule: In `--auto` mode, the implementation must first detect if multiple nodes in the current buffer batch have the **same** preset label and error before writing anything.

---

## Parse Flows

### Manual Parse (no `--auto`)

- Ordered interactive multi-select (TTY).
- Selection order is preserved and affects auto-label sequencing.
- For each selected node, prompt for label (prefilled using preset or auto-label logic).
- Uniqueness validation runs against both already-parsed nodes and labels chosen earlier in the same session.
- The multi-select widget has an early-return path that allows label mutation before final commit.

### Automatic Parse (`--auto`)

- Process entire buffer.
- Apply preset labels first (with duplicate check across the batch).
- Generate labels for remaining nodes.
- Respect `--allow-existing`, `skip_used_index`, `--dry-run`, etc.
- Write results only if validation passes.

---

## Configuration System

- Primary source: YAML config file (normally `etc/config.yml`) + XDG user paths.
- **Every scalar key** can be overridden by an `ALCES_HUNT_<key>` environment variable.
- Structured maps:
  - `auto_apply`: `{ regex: "identity", ... }`
  - `prefix_starts`: `{ "cnode": "001", ... }`
  - `presets`: `{ label, prefix, groups }`

Special values:

- `default_label`: `"long"` | `"short"` | `"blank"` (default `"long"`)
- `default_start`: string (default `"01"`) — the string length controls padding
- `profile_command`: array or string. Must resolve to an executable. Defaults to `$ALCES_HUNT_ROOT/bin/alces-hunt profile`

---

## External Integration (Profile)

When saving a node to the parsed list and `auto_apply` rules exist:

1. Find the first regex in `auto_apply` that matches the final `label`.
2. Execute:

   ```
   <profile_command> apply <label> <identity>
   ```

This is best-effort (log on failure; do not abort the parse).

---

## Collector (Default Content)

When no custom command is supplied, the client runs:

- `/proc/cmdline` → `SYSUUID`, `BOOTIF`
- `/sys/class/net/*` (skip `lo` and dot-prefixed) → name → MAC
- `/sys/class/block/*` (only those with `device/` subdir) → name → size
- `ipmitool lan print 1` (rescue on error) → `bmcip`, `bmcmac`
- Serialized as YAML into the `content` field

Override path: any shell command whose stdout becomes `content`.

---

## Special Behaviors

- `hunt --include-self` / `include_self`: immediately after starting the listener, the server sends a payload to itself (same default-label / `dmidecode` rules as `send`).
- Default `send` label is `dmidecode -s system-serial-number` unless `--label` is given; missing `dmidecode` is a hard failure when that default is in effect.
- PID file cooperation via `ALCES_HUNT_pidfile` (used by `bin/start` wrapper).
- `autorun` command dispatches based on `autorun_mode`.
- TCP hunt listener is intentionally minimal (manual header parsing + exact byte read).
- Buffer operations use `id` for addressing; parsed operations use `label`.
- Plain output is tab-separated with slightly different columns depending on context.

---

## Non-Functional Requirements

- Must work in early-boot / PXE environments with very limited tooling.
- Must support both fully interactive and fully scripted operation.
- Label generation and search syntax must be stable.
- Error messages for common failure modes (duplicate labels, auth failure, collisions, missing config) should remain distinguishable.
- Plain output formats must be stable for machine consumption.
