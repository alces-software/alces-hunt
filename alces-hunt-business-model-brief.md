# alces-hunt — Business Model Brief

alces-hunt is **not** a standalone revenue product. It is strategic infrastructure tooling whose primary purpose is to lower the barrier to adopting the broader HPC platform and appliance ecosystem.

---

## Value Proposition

Commissioning a rack of bare-metal nodes has historically been extremely painful:

- Manually collect MAC addresses
- Decide on naming schemes
- Hand-enter everything into configuration and provisioning systems
- Repeat painfully for every expansion or refresh

alces-hunt turns this into a repeatable, partially or fully automated workflow. Nodes "phone home", get discovered, get named, get grouped, and (via `auto_apply`) can be automatically profiled.

It is the difference between a multi-day manual, error-prone process and a largely unattended, auditable one.

---

## Target Customers

- University and national laboratory HPC teams
- System integrators building turnkey research and commercial clusters
- Research computing centers and supercomputing facilities
- Organizations that purchase or build appliances containing the platform software stack

These organizations repeatedly face the "new hardware just arrived, now I have to name and register it" problem.

---

## Monetization Model

The tool itself is distributed under open terms:

- Creative Commons Attribution-ShareAlike 4.0
- References to the Eclipse Public License 2.0

The goal is **widespread adoption**, not direct licensing revenue.

Revenue is realized through several channels:

### 1. Platform Pull-Through (Primary)

Once nodes are discovered and parsed, the natural next action is applying profile identities and other platform components.

alces-hunt is the "on-ramp". By the time an organization has a working parsed inventory, they are already inside the operational model.

### 2. Support & Professional Services

Organizations running production clusters at scale want:

- Support contracts
- Professional services for initial deployment and custom integrations
- Training

### 3. Appliances & Pre-Integrated Solutions

Pre-configured hardware + software solutions sold by the vendor or channel partners include alces-hunt as a core, expected component.

### 4. Ecosystem Lock-In via Integration Points

Deep, opinionated integration creates preference for the rest of the stack:

- `auto_apply` rules
- `profile_command`
- `presets` (label / prefix / groups sent from client)
- The overall mental model of buffer → parse → labeled inventory

---

## Strategic Role

alces-hunt is deliberately **low-friction**.

A sysadmin can start using it in minutes during a cluster build. By the time the first nodes have been parsed and profiles applied, the organization has entered the platform way of operating.

This is classic "tooling that sells the platform" economics, very common in infrastructure and developer tooling.

The product is given away (open source) to create a funnel into higher-value offerings (support, services, appliances, the rest of the suite).

---

## Implications for Implementation

The following business/strategic constraints matter:

- The tool must remain **simple and extremely reliable** in early-boot / PXE environments. Complexity here kills adoption.
- Both fully interactive (admin in front of a terminal) and fully automated/scripted paths must be first-class.
- Integration points with profile/identity systems (`auto_apply`, external command execution) are strategically important even if you replace the concrete profile system.
- Behavioral and CLI compatibility (flags, output formats, label generation rules, search syntax) is valuable because users and documentation depend on them.
- Licensing should stay permissive and clearly documented.

---

## Summary

| Aspect                  | Reality |
|-------------------------|---------|
| Direct revenue source   | None (open) |
| Real value              | Dramatically reduces cluster commissioning pain |
| Monetization            | Pull-through into profile + services + appliances |
| Strategic purpose       | On-ramp / adoption tool for the broader platform |
| Success metric          | How easily and quickly new hardware becomes named, grouped, and profiled |

alces-hunt exists to make the rest of the ecosystem feel inevitable once you start bringing up nodes.
