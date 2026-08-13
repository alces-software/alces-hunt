# alces-hunt — User Stories

A comprehensive set of user stories for alces-hunt, written from the perspective of real HPC cluster administrators, operators, and integrators.

These stories capture the full range of use cases needed to drive the implementation.

---

## Cluster Administrator — Interactive Bring-Up

**As an admin, I want to boot a rack of machines into a PXE environment so that they automatically report themselves to a central listener without me having to manually collect and enter MAC addresses.**

*Benefit:* Eliminates a major source of human error and time waste during initial cluster deployment.

**As an admin, I want to run an interactive `parse` session so that I can review all discovered nodes, edit their proposed labels, and move them from the buffer into the managed inventory in one controlled workflow.**

*Benefit:* Gives the operator full visibility and final say before nodes become part of production naming.

**As an admin, I want to use prefix + start label generation (e.g. `--prefix cnode --start 001`) during parsing so that nodes receive consistent, sequential, zero-padded names without manual typing.**

*Benefit:* Enforces naming conventions at scale.

**As an admin, I want to assign arbitrary groups to nodes (compute, gpu, login, storage, etc.) so that downstream schedulers, configuration management, and monitoring tools can select logical subsets of the cluster.**

*Benefit:* Groups are the primary mechanism for organizing heterogeneous hardware.

---

## Operator — Automated & Lights-Out Deployment

**As an operator, I want nodes whose hostnames match a regular expression to be automatically parsed on arrival (`auto_parse`) so that known node types never require manual intervention.**

*Benefit:* Enables fully unattended expansion of standard compute or storage pools.

**As an operator, I want `auto_apply` rules so that when a node is assigned a label matching a pattern, a profile identity is automatically applied without further commands.**

*Benefit:* Closes the loop between discovery and configuration.

**As an operator, I want to run `alces-hunt autorun` (in hunt or send mode) from a service wrapper so that the tool participates correctly in Linux boot, restart, and shutdown sequences using a PID file.**

*Benefit:* Supports production deployment as a managed service.

**As an operator, I want `send --broadcast` so nodes can register without knowing a specific target host IP during early boot when DHCP or network configuration may still be settling.**

*Benefit:* Robustness in the most fragile phase of node bring-up.

---

## Ongoing Cluster Management

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

---

## Scripting & Integration Authors

**As a script author, I want stable `--plain` (tab-separated) output with predictable columns so I can reliably consume node data from shell scripts, Ansible, or other automation.**

*Benefit:* Machine-readable interface is essential for composability.

**As a script author, I want `parse --dry-run` and `parse --auto` so I can validate what labels would be generated or fully automate inventory population in CI/CD-style pipelines.**

*Benefit:* Supports testability and unattended operation.

**As an integrator, I want every configuration value to be overridable via an `ALCES_HUNT_*` environment variable so the tool works inside wrappers, containers, image-based deployments, and service managers.**

*Benefit:* Flexibility for packaging and orchestration layers.

---

## Resilience & Edge Cases

**As an admin, I want collision handling via `skip_used_index` so that automatic label generation continues incrementing instead of aborting when a candidate label is already taken.**

*Benefit:* Prevents brittle failures during large parallel node arrivals.

**As an admin, I want clear, actionable errors when duplicate preset labels are supplied or when a generated label would collide in the parsed list.**

*Benefit:* Fast feedback instead of silent corruption.

**As an admin, I want a simple shared `auth_key` so that only authorized nodes can register with the alces-hunt server.**

*Benefit:* Basic protection against accidental or malicious registration.

---

*These stories should be treated as acceptance criteria for the implementation.*
