# terraform-provider-guacamole

Flexpair fork of [techBeck03/terraform-provider-guacamole](https://github.com/techBeck03/terraform-provider-guacamole) — a [Terraform](https://en.wikipedia.org/wiki/Terraform_(software)) provider for [Apache Guacamole](https://en.wikipedia.org/wiki/Apache_Guacamole).

Upstream: `github.com/techBeck03/terraform-provider-guacamole`
Fork: `github.com/Flexpair/terraform-provider-guacamole`

## Why a Fork?

Flexpair uses HCP Terraform Stacks to orchestrate a multi-component deployment. The Guacamole provider configures users and connections on a gateway VM that is created in a prior component. This creates a chicken-and-egg problem: the provider must be configured before the gateway exists. The upstream provider connects eagerly during `providerConfigure`, which fails immediately.

The fork introduces lazy initialization to solve this, plus several bug fixes discovered during the Stacks migration.

## Flexpair Version History

| Version | Key Changes |
|---------|-------------|
| v2.1.0 | Lazy init + retry + upstream fixes |
| v2.2.0 | Password state preservation (fixes perpetual plan drift) |
| v2.3.0 | Empty URL handling for Stacks (fixes multi-wave plan failures) |
| v2.3.2 | Longer 15-second Guacamole connection polling window (~30 minutes) |
| v2.4.0 | Idempotent CRUD: missing resources recreate instead of erroring (fixes gateway-replacement applies) |

## Commits (oldest → newest)

### 1. `45ac5e0` — feat: lazy provider initialization for HCP Terraform Stacks

**Problem:** The upstream provider calls `Connect()` (which POSTs to `/api/tokens`) during `providerConfigure`. In HCP Terraform Stacks, the Guacamole server doesn't exist yet at planning time — it's created by a different component in a prior wave.

**Changes:**
- **`guacamole/lazy_client.go`** (new file): `LazyClient` wrapper that stores the provider config and defers authentication to the first `Get()` call.
- **`guacamole/provider.go`**: Removed eager `validate()` + `Connect()` from `providerConfigure()`. Returns `NewLazyClient(config)` instead of an authenticated `*guac.Client`.
- **All 16 resource/data source files**: Changed `m.(*Client)` to `m.(*LazyClient).Get()`.
- **`main.go`**: Updated module path from `github.com/techBeck03` to `github.com/Flexpair`.
- **`go.mod`**: Bumped Go to 1.22, updated module path.

**Vendored upstream fixes (in `vendor/github.com/techBeck03/guacamole-api-client/`):**
- **`client.go`**: Replaced deprecated `ioutil.ReadAll` → `io.ReadAll`.
- **`types/connections.go`**: Fixed `xterm-25color` → `xterm-256color` typo (upstream Issue #15).
- **`types/users.go`**: Changed `LastActive` from `int` to `int64` to fix overflow on 32-bit (upstream Issue #12).

---

### 2. `9ef53b0` — feat: add retry with exponential backoff to lazy provider connection

**Problem:** During `terraform apply`, the gateway VM was just created but Docker/Guacamole takes minutes to start. The single-attempt lazy connect fails immediately.

**Changes:**
- **`guacamole/lazy_client.go`**: `Get()` now retries up to 10 times with exponential backoff (15s → 30s → 60s, capped at 60s, ~9 min total). Thread-safe via `sync.Mutex`. Result is cached after first success or final failure.

---

### 3. `c7c6cc0` — fix: preserve password in state when API returns empty

**Problem:** The Guacamole API never returns passwords in its user response (security measure). The upstream provider's `resourceUserRead` unconditionally wrote the empty API response into Terraform state: `d.Set("password", user.Password)`. This overwrote the configured password with `""` on every refresh, causing a perpetual plan diff ("password changed from ... to ...") and preventing Stacks convergence ("too many plans").

**Changes:**
- **`guacamole/resource_user.go`** (line ~552): Added a guard:
  ```go
  // Guacamole API never returns passwords — preserve configured value
  if user.Password != "" {
      d.Set("password", user.Password)
  }
  ```

---

### 4. `3bd87f5` — feat: handle empty provider URL gracefully for Stacks deferred planning

**Problem:** In HCP Terraform Stacks multi-wave deployments, the `early_orbit` component depends on `launch_resources.gateway_ip` for the provider URL. During wave 2 planning, the Terraform SDK may pass an empty string for provider attributes that referenced unknown component outputs. The provider then retried 10 times over ~9 minutes trying to POST to `"/api/tokens"` (no scheme) before failing the plan.

**Changes:**
- **`guacamole/lazy_client.go`**:
  - Added `IsConfigured() bool` — returns false when URL is empty.
  - Added `readWithClient(m, d)` helper — returns `nil, nil` (clears resource ID → "not found") when unconfigured.
  - Added `writeWithClient(m)` helper — returns a clear error when unconfigured.
- **All 16 resource/data source files** (40 CRUD functions total):
  - Read functions: use `readWithClient(m, d)` — plan shows "create" instead of erroring.
  - Create/Update/Delete: use `writeWithClient(m)` — returns clear error if URL is still empty at apply time.

---

### 5. `a9b2371` (tag `v2.3.2`) — extend Guacamole readiness polling

**Problem:** Fresh gateway rebuilds can take longer than the previous ~9-minute connection retry window before Docker, Nginx, and Guacamole are ready to accept `/api/tokens` requests.

**Changes:**
- **`guacamole/lazy_client.go`**: `Get()` now polls every 15 seconds for up to 120 attempts (~30 minutes total). This keeps retry behavior inside the provider instead of adding fixed waits to Terraform components.

---

### 6. Current (tag `v2.4.0`) — idempotent CRUD on missing resources

**Problem:** When the gateway VM is replaced, it comes up with a fresh, empty Guacamole database while Terraform state still holds the old numeric resource IDs. On the next apply the provider's `Read` hit HTTP 404 and returned a hard error (`diag.FromErr`), and `Delete` against a vanished resource also failed — so a gateway replacement broke the `early_orbit` apply instead of cleanly recreating connections and users.

**Changes:**
- **`guacamole/lazy_client.go`**: Added `isNotFoundError(err)` — the vendored API client exposes no typed errors, so an HTTP 404 is detected from the stable `status code 404` substring in the error message.
- **Read functions** (`resource_connection_vnc.go`, `resource_connection_ssh.go`, `resource_connection_rdp.go`, `resource_connection_telnet.go`, `resource_connection_kubernetes.go`, `resource_connection_group.go`, `resource_user.go`, `resource_user_group.go`): on a not-found error, call `d.SetId("")` and return without error so Terraform plans a recreate instead of failing.
- **Delete functions** (same resources): treat a not-found error as success (already deleted) for idempotency.

This pairs with `flexpair-infra` coupling every Guacamole resource in `early-orbit` to the gateway identity via `replace_triggered_by`, so a gateway swap forces a clean destroy-then-create that the idempotent provider can complete.

## Vendored Library Changes

Three files in `vendor/github.com/techBeck03/guacamole-api-client/` differ from upstream:

| File | Change | Upstream Issue |
|------|--------|----------------|
| `client.go` | `ioutil.ReadAll` → `io.ReadAll` | Deprecated since Go 1.16 |
| `types/connections.go` | `xterm-25color` → `xterm-256color` | [#15](https://github.com/techBeck03/terraform-provider-guacamole/issues/15) |
| `types/users.go` | `LastActive int` → `int64` | [#12](https://github.com/techBeck03/terraform-provider-guacamole/issues/12) |

These same three fixes also exist as commit `7e39779` in the [Flexpair/guacamole-api-client](https://github.com/Flexpair/guacamole-api-client) fork (now archived — the vendored copies in this repo are authoritative).
