# alces-hunt — Plain Output Specification

This document defines the **exact** machine-readable output formats for all commands that support `--plain`.

These formats are critical for scripting and automation. Implementations must match them.

---

## Overview

- `--plain` produces **tab-separated** output (one record per line).
- No headers.
- Fields are joined with a single tab (`\t`).
- Empty groups are represented as a single pipe: `|`
- For `--plain` on lists, the last column for parsed nodes is the **presets as JSON**.
- Content (which is often multi-line YAML) is emitted **as-is** in `show --plain`.

---

## `list --plain`

**Columns (in order):**

1. `id`
2. `hostname`
3. `ip`
4. Groups: `group1|group2|...` or `|` if no groups
5. `label` (may be empty for buffer nodes)
6. `presets` as compact JSON object (e.g. `{"label":"cnode001"}` or `{}`)

**Example (parsed list):**

```
a1b2c3d4e5f6	node01	10.0.0.41	compute|gpu	cnode041	{"label":"cnode041"}
a1b2c3d4e5f7	node02	10.0.0.42	|	login01	{"label":"login01","prefix":"login"}
```

**Example (buffer list):**

```
a1b2c3d4e5f6	node01	10.0.0.41	|		{}
```

**Field construction:**

```
id, hostname, ip,
groups.any? ? groups.join("|") : "|",
label,
presets as compact JSON
```

joined with a single tab. This column order and logic is **identical** for both buffer and parsed when using `--plain`.

---

## `list --plain --by-group`

**Not supported in plain mode.**

When `--by-group` and `--plain` are both supplied, `--by-group` is **ignored** for plain output and the flat list format above is still emitted.

---

## `show NODE --plain`

**Columns (in order):**

1. `id`
2. `hostname`
3. `ip`
4. Groups: `group1|group2|...` or `|`
5. `content` (the raw collected content — often multi-line YAML)

**Example:**

```
a1b2c3d4e5f6	node01	10.0.0.41	compute|gpu	---
sysuuid: abc123
bootif: 01-00-11-22-33-44-55
nets:
  eno1: 00:11:22:33:44:55
...
```

**Field construction:**

```
id, hostname, ip,
groups.any? ? groups.join("|") : "|",
content
```

joined with a single tab.

Note: After the tab line, the human (non-plain) `show` also prints `content` on following lines. In `--plain` only the single tab line is emitted.

---

## Human-Readable Table Output (non-plain)

When `--plain` is **not** used, output goes through a Unicode box-drawing table renderer.

- Headers differ by context:

  | Context          | Headers                              |
  |------------------|--------------------------------------|
  | Buffer (default) | `ID`, `Hostname`, `IP`, `Groups`, `Presets` |
  | Parsed (default) | `ID`, `Hostname`, `IP`, `Groups`, `Label`   |

- Groups column: comma-separated after sorting.
- Presets column (buffer table): pretty-printed as `key: 'value'` lines (multi-line cells supported).
- The table uses Unicode box drawing with padding.

Implementations that want visual parity should match this structure and use similar table rendering.

---

## Other Commands That Emit Tables (non-plain)

These commands always use the table renderer (unless `--plain` is supported and used):

- `parse` (success and `--dry-run`)
- `remove-node`
- `modify-groups`
- `modify-label` (indirectly via messages + table in some paths)

They use the same header rules as `list` based on whether `--buffer` was supplied.

---

## Important Notes

1. **Groups separator is `|`** (pipe), not comma, in plain mode.
2. **Empty groups is exactly `|`** (single pipe character), not empty string.
3. `presets` JSON in plain list is compact (no extra whitespace).
4. `content` in `show --plain` is emitted raw (may contain tabs, newlines, YAML, etc.). Consumers must handle it as the final field.
5. There is **no header row** in any `--plain` output.
6. Order of nodes in `list --plain` is by `id` (the node list is sorted by `id`).

---

## Recommended Test Cases for Plain Output

- Parsed node with multiple groups
- Buffer node with no groups and no label
- Node with preset label + prefix
- `show --plain` of a node whose `content` contains newlines and colons
- Node with empty presets (`{}`)
- Verify tab count per line (exactly 5 tabs → 6 fields for list plain)
