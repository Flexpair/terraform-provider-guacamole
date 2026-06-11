# Copilot Instructions — `terraform-provider-guacamole`

`AGENTS.md` at the repository root is the **single canonical, tool-independent source** of agent
guidance. GitHub Copilot reads `AGENTS.md` natively, so this file stays a thin pointer to avoid
duplicated, drifting rules.

- Read [`AGENTS.md`](../AGENTS.md) before doing repository work. It explains why the fork exists
  (lazy init, password-state preservation, idempotent CRUD), the lazy-client helper pattern, the
  vendored-patch restore step, remote/release conventions, and build/test commands.
- This module is a Git submodule of the Flexpair umbrella repository; cross-cutting rules live in
  the umbrella `AGENTS.md`.

Do **not** copy rules into this file. If a rule should apply to agents, update `AGENTS.md` instead.
