# alces-hunt — Verification Checklist

This checklist defines the **minimum set of scenarios** an implementation must pass.

Treat each item as an acceptance test. Prioritized roughly by importance.

---

## P0 — Core Contracts (Must Work First)

| # | Scenario | Expected Behavior | Verification |
|---|----------|-------------------|--------------|
| 1 | `hunt` starts a dual listener (TCP + UDP) on the configured port | Both sockets accept data | `ss -tlnp`, `ss -ulnp`, or equivalent |
| 2 | `send` to a running `hunt` via TCP | Node appears in buffer with correct id, hostname, ip (injected), content | `alces-hunt list --buffer --plain` |
| 3 | `send --broadcast` | Node is received via UDP path | Same as above |
| 4 | `auth_key` mismatch | Server rejects (logs "Unauthorised..."), no node stored | Check logs + buffer empty |
| 5 | ID uniqueness | Second send with same hostid does not create duplicate (unless `--allow-existing`) | Count of nodes stays 1 |
| 6 | `parse --auto --prefix cnode --start 001` on 3 buffer nodes | Nodes get `cnode001`, `cnode002`, `cnode003` (in selection / arrival order) | `alces-hunt list --plain` |
| 7 | Preset label from client | `send --label foobar01` results in that exact label after parse | Label appears as sent |
| 7a | Default send label | `send` with no `--label` sets preset label to trimmed stdout of `dmidecode -s system-serial-number` | Buffer `presets.label` matches `dmidecode` |
| 7b | Default send label, `dmidecode` missing | `send` with no `--label` fails immediately; no payload transmitted; no node stored | Error + buffer unchanged |
| 7c | Explicit `--label` without `dmidecode` | `send --label foobar01` succeeds even if `dmidecode` is not available | Node stored with that label |
| 8 | Label uniqueness enforcement | Attempt to parse two nodes to the same label fails with clear error | No nodes moved, error message |
| 9 | `skip_used_index` | When a generated label collides, higher number is used instead of error | Correct next available label |
| 10 | Buffer vs parsed addressing | `remove-node` with `--buffer` uses ID; without uses label | Correct node removed only when right identifier given |

---

## P1 — Label Generation

| # | Scenario | Expected Behavior |
|---|----------|-------------------|
| 11 | No prefix, default_label=long | Uses full hostname |
| 12 | No prefix, default_label=short | Uses hostname up to first `.` |
| 13 | Prefix with custom start length | Padding matches start string length (e.g. start="001" → `cnode001`) |
| 14 | `prefix_starts` override | Specific prefix uses the configured start value |
| 15 | Preset label takes precedence over prefix | Even if `--prefix` is given, preset label wins |
| 16 | Duplicate preset labels in one `--auto` batch | Fails early before any node is written |
| 17 | `default_label=blank` + no prefix | Produces empty label (and should be rejected or handled per current rules) |

---

## P2 — Search & Selection Syntax

| # | Scenario | Expected Behavior |
|---|----------|-------------------|
| 18 | `node[1-3]` expansion | Expands to node1,node2,node3 |
| 19 | Mixed: `c[01-02],login[1-1]` | Expands correctly |
| 20 | Invalid range `[5-1]` | Raises `InvalidRangeError` with message |
| 21 | `--regex` mode | Each term treated as regex, no bracket expansion |
| 22 | `--match-hostname` on remove | Matches against hostname even when targeting parsed list |
| 23 | Comma list with spaces | Handled gracefully (trim surrounding whitespace) |

---

## P3 — Buffer vs Parsed Distinction

| # | Scenario | Expected Behavior |
|---|----------|-------------------|
| 24 | `list --buffer` | Shows ID, Hostname, IP, Groups, Presets |
| 25 | `list` (no buffer) | Shows ID, Hostname, IP, Groups, Label |
| 26 | `show` on buffer node | Must be addressed by ID |
| 27 | `show` on parsed node | Must be addressed by label |
| 28 | `modify-groups --buffer` | Operates on buffer nodes using IDs |
| 29 | `rename-group --buffer` | Renames group in buffer list |

---

## P4 — Plain Output Fidelity

| # | Scenario | Exact Output Requirement |
|---|----------|--------------------------|
| 30 | `list --plain` | 6 tab-separated fields: id, hostname, ip, groups\|or\|, label, compact presets JSON |
| 31 | Empty groups in plain | Single `\|` character |
| 32 | `show --plain` | 5 tab-separated fields, last field = raw `content` (may contain newlines) |
| 33 | No header rows in any `--plain` output | Must be pure data lines |
| 34 | `presets` JSON is compact | No pretty-print whitespace |

See also: `alces-hunt-plain-output-spec.md`

---

## P5 — Auto-Parse & Auto-Apply

| # | Scenario | Expected Behavior |
|---|----------|-------------------|
| 35 | `hunt --auto-parse '/^cnode/'` | Nodes whose hostname matches are immediately placed in parsed list with auto-generated label |
| 36 | `auto_apply` rule match on parse | When node lands in parsed and label matches a rule, `<profile_command> apply <label> <identity>` is executed |
| 37 | Profile command not found / fails | Logged but does not abort the parse (best effort) |

---

## P6 — Special Behaviors

| # | Scenario | Expected Behavior |
|---|----------|-------------------|
| 38 | `hunt --include-self` | Server starts, then immediately sends to itself |
| 39 | `autorun` with `autorun_mode: hunt` | Behaves like `hunt` |
| 40 | `autorun` with `autorun_mode: send` | Behaves like `send` |
| 41 | PID file | When `ALCES_HUNT_pidfile` is set, the process creates/uses it |
| 42 | `send --retry-interval 3` | Warning is printed, value is raised to 5.0 |
| 43 | `send --retry-interval 10` | Uses 10s retry loop on failure |

---

## P7 — Error Conditions (Clear & Distinguishable)

| # | Error Condition | Expected Behavior (should be recognizable) |
|---|------------------|-------------------------------------------|
| 44 | No port configured for hunt/send | "No port provided!" |
| 45 | Port busy on hunt | "Provided port X is busy" |
| 46 | Duplicate label on modify-label | "Label 'X' already exists..." |
| 47 | Node not found on show/remove | "No ... 'X' found in list 'Y'" |
| 48 | Group does not exist on rename-group | "Group 'X' does not exist..." |
| 49 | Auth failure | "Unauthorised node attempted to connect" |
| 50 | Malformed packet | "Malformed packet received from ..." |
| 50a | Default send label, `dmidecode` not available | Clear error that `dmidecode` is required / not available; non-zero exit; nothing sent |

---

## P8 — Collector & Content

| # | Scenario | Expected Behavior |
|---|----------|-------------------|
| 51 | Default collector | Produces YAML containing at least `nets`, `sysuuid`/`hostid` fallback |
| 52 | Custom `-c 'echo hello'` | The string "hello" becomes the `content` |
| 53 | Content with newlines/tabs | Survives round-trip in storage and `show --plain` |

---

## P9 — Configuration & Environment

| # | Scenario | Expected Behavior |
|---|----------|-------------------|
| 54 | `ALCES_HUNT_port=1234` | Overrides config file |
| 55 | `ALCES_HUNT_auto_apply` as JSON-like hash | Parsed and validated as map of regex → identity |
| 56 | Invalid regex in `auto_parse` or `auto_apply` | Clear error at startup / use time |
| 57 | `profile_command` points to non-executable | Error when attempting to use it |

---

## P10 — Edge Cases & Robustness

| # | Scenario | Expected Behavior |
|---|----------|-------------------|
| 58 | Re-sending same ID with `--allow-existing` | Old record replaced (by id) |
| 59 | `parse --dry-run` | Shows table of would-be labels, no nodes moved |
| 60 | `dump-buffer` | Buffer directory emptied, nodes gone |
| 61 | Very long content | Survives YAML serialization and storage |
| 62 | Node with no groups | Displays cleanly in both table and plain modes |

---

## Recommended Verification Approach

1. Start with a clean `var/buffer` and `var/parsed`.
2. Run the commands against the implementation.
3. Check:
   - Exit codes
   - Plain output (exact byte match where possible)
   - Resulting YAML files in buffer/parsed
   - Log messages for errors
4. For interactive parse, script the TTY if possible or manually verify the happy paths + collision cases.

---

## Suggested Test Data Set

Create a small set of synthetic nodes with:

- Different hostnames (some with dots, some matching auto-parse regex)
- Varying numbers of groups
- One node that will collide on label generation
- One node sent with `--label`
- One node sent with `--prefix`
- One node sent with no `--label` (default `dmidecode` serial)

Use these same nodes for most of the checklist items.
