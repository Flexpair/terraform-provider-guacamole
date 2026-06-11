# Agent Guide — `terraform-provider-guacamole`

Terraform provider for [Apache Guacamole](https://guacamole.apache.org/) — manages users, user
groups, and connections (SSH, RDP, VNC, Telnet, Kubernetes) as Infrastructure-as-Code. This is the
**Flexpair fork** of `techBeck03/terraform-provider-guacamole`, hardened for HCP Terraform Stacks.

## How AI agents load these instructions

This `AGENTS.md` is the **single canonical, tool-independent source** of agent guidance for this
repository. It is read natively by both GitHub Copilot and Claude Code, so keep durable rules here
rather than duplicating them elsewhere.

- `.github/copilot-instructions.md` is a thin pointer to this file. Do not duplicate rules into it.
- This module is a Git submodule of the Flexpair umbrella repository; cross-cutting rules live in
  the umbrella `AGENTS.md`.

## Why this fork exists (the changes that matter)

- **Lazy initialization** — auth is deferred to the first API call (not `providerConfigure`), so
  Terraform can plan before the gateway VM/Guacamole server exists.
- **Password state preservation** — Guacamole's API never returns passwords; `resource_user.go`
  guards `d.Set("password", ...)` so an empty API response no longer overwrites configured state
  and causes perpetual plan drift.
- **Empty-URL + idempotent CRUD** — multi-wave Stacks planning may pass an empty provider URL; Read
  returns "not found" (plan shows create), while Create/Update/Delete error clearly if the URL is
  still empty at apply. Read/Delete treat 404s gracefully so a replaced gateway (fresh DB, stale
  state IDs) reconciles cleanly. Polling window extended (~30 min) for slow gateway rebuilds.

## Tech stack and structure

- Go (see `go.mod`); `hashicorp/terraform-plugin-sdk/v2`; build via GoReleaser (`.goreleaser.yml`)
  with a legacy `Makefile` fallback. Module path is `Flexpair/terraform-provider-guacamole`.
- `guacamole/` holds the provider package: `provider.go`, `lazy_client.go`, `resource_*.go`,
  `data_*.go`, `structure.go`, plus `*_test.go`. `main.go` is the entry point. `vendor/` contains a
  patched copy of `guacamole-api-client`. `docs/` is registry documentation.

## Conventions and gotchas

- **Use the lazy-client helpers, never `m.(*LazyClient)` directly:** `readWithClient(m, d)` for
  Read (returns nil/clears ID when unconfigured) and `writeWithClient(m)` for Create/Update/Delete.
- **Preserve the 404 + password-preservation patterns** when adding or editing resources —
  `isNotFoundError(err)` matches the 404 substring; Read sets `d.SetId("")`, Delete treats 404 as
  success.
- **Re-apply vendored patches after any Dependabot vendor bump:** run
  `./scripts/restore-vendored-guacamole-patches.sh` (fixes `ioutil`→`io`, `LastActive int`→`int64`,
  `xterm-25color`→`xterm-256color`).
- **Remotes:** `origin` is the Flexpair fork (the one to push to); `upstream` is the archived
  original. Do not push to `upstream`.
- **Stacks coupling:** a Guacamole resource in the `early-orbit` component must carry
  `replace_triggered_by = [terraform_data.gateway_anchor]` so it is recreated on gateway swap.

## Build, test, validate

- Build: `go build -o terraform-provider-guacamole` (or `goreleaser build --single-target --clean
  --snapshot`); `make install` drops it into the local Terraform plugin path.
- Test: `go test -v ./...`; acceptance tests need a live server: `TF_ACC=1 go test -v ./...`.
- CI: `.github/workflows/test.yml` runs `go test` on push/PR; `.github/workflows/release.yml` runs
  GoReleaser (GPG-signed) on `v*` tags. Publish to the HCP private registry via
  `./scripts/publish-private-provider.sh`.
