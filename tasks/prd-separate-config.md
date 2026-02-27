# PRD: Separate Treb Config (`treb.toml`)

## Introduction

Move treb-specific sender configuration out of `foundry.toml` into a dedicated `treb.toml` file that lives alongside it. This decouples treb's config from Foundry's, making it clearer what belongs to each tool, and introduces namespaces as a first-class treb concept that can optionally map to a foundry profile. Backwards compatibility is maintained — treb continues to read the old `foundry.toml` locations with a deprecation warning — and a `treb migrate-config` command helps users transition interactively.

## Goals

- Introduce `treb.toml` as the canonical location for treb sender configuration
- Decouple treb namespaces from foundry profiles (namespace can map to a different profile name)
- Maintain full backwards compatibility with existing `[profile.*.treb.*]` config in `foundry.toml`
- Show a clear deprecation warning when legacy config is detected
- Provide an interactive migration command to move config from `foundry.toml` to `treb.toml`

## User Stories

### US-001: Define treb.toml schema and parser
**Description:** As a developer, I need a TOML parser for `treb.toml` so the CLI can read sender configs from the new file.

**Acceptance Criteria:**
- [ ] New domain type `TrebFileConfig` representing the full `treb.toml` schema
- [ ] Parser reads `treb.toml` from project root (same directory as `foundry.toml`)
- [ ] Supports namespace sections: `[ns.<name>.senders.<sender>]` with same sender fields as today
- [ ] Supports optional `profile` field per namespace: `[ns.<name>]` `profile = "<foundry_profile>"`
- [ ] When `profile` is omitted, defaults to the namespace name (preserving current behavior)
- [ ] `[ns.default.senders.*]` is the base config, merged with namespace-specific overrides (same merge semantics as today)
- [ ] Environment variable expansion (`${VAR}`) works in all string fields
- [ ] Unit tests cover parsing, profile mapping, merging, and env var expansion
- [ ] `make lint` passes

### US-002: Integrate treb.toml into config resolution
**Description:** As a developer, I need the config provider to prefer `treb.toml` over `foundry.toml` so users get the new behavior when the file exists.

**Acceptance Criteria:**
- [ ] `config.Provider` checks for `treb.toml` first; if present, loads sender config from it
- [ ] If `treb.toml` exists, `foundry.toml` `[profile.*.treb.*]` sections are ignored entirely
- [ ] If `treb.toml` does not exist, falls back to current `foundry.toml` parsing (no behavior change)
- [ ] `RuntimeConfig.TrebConfig` is populated identically regardless of source
- [ ] The resolved foundry profile name comes from `treb.toml`'s `[ns.<name>]` `profile` field when available
- [ ] `FOUNDRY_PROFILE` env var set during forge execution uses the resolved profile name
- [ ] Unit tests cover both paths (treb.toml present vs absent)
- [ ] Integration tests pass with both config styles
- [ ] `make lint` passes

### US-003: Detect legacy config and show deprecation warning
**Description:** As a user with config in `foundry.toml`, I want to see a clear warning telling me to migrate so I know my setup is outdated.

**Acceptance Criteria:**
- [ ] On every command run (except `version`, `help`, `completion`, `init`, `migrate-config`), check if `foundry.toml` contains `[profile.*.treb.*]` sections
- [ ] If legacy config detected and no `treb.toml` exists, print a yellow warning to stderr:
  ```
  Warning: treb config detected in foundry.toml — this is deprecated.
  Run `treb migrate-config` to move your config to treb.toml.
  ```
- [ ] Warning does not block command execution
- [ ] Warning is suppressed when `--json` flag is set (machine-readable output)
- [ ] Warning is suppressed when `treb.toml` already exists (even if foundry.toml still has stale treb sections)
- [ ] Unit test verifies warning is emitted under correct conditions
- [ ] `make lint` passes

### US-004: Implement `treb migrate-config` command
**Description:** As a user, I want to run `treb migrate-config` to interactively move my sender config from `foundry.toml` to `treb.toml`.

**Acceptance Criteria:**
- [ ] New `migrate-config` CLI command registered under the root command
- [ ] Reads all `[profile.*.treb.*]` sections from `foundry.toml`
- [ ] Maps each foundry profile to a `[ns.<name>]` section in `treb.toml`:
  - Profile name becomes namespace name
  - `profile` field is set to the foundry profile name (explicit even when they match)
  - Sender configs are copied verbatim
- [ ] Shows the user a preview of the generated `treb.toml` content
- [ ] Asks user to confirm writing `treb.toml`
- [ ] Asks user whether to remove the `[profile.*.treb.*]` sections from `foundry.toml`
  - If yes, removes only the `treb` sub-tables (preserves all other profile config)
  - If no, leaves `foundry.toml` untouched and informs user they can clean up manually
- [ ] If `treb.toml` already exists, warns and asks whether to overwrite
- [ ] Supports `--non-interactive` flag: writes `treb.toml` without prompts, does NOT modify `foundry.toml`
- [ ] Prints success message with next steps
- [ ] Integration test covers the full migration flow
- [ ] `make lint` passes

### US-005: Update `treb init` to generate `treb.toml`
**Description:** As a new user running `treb init`, I want the scaffolded project to use `treb.toml` from the start.

**Acceptance Criteria:**
- [ ] `treb init` generates a `treb.toml` with a `[ns.default.senders.deployer]` section (private_key type with env var placeholder)
- [ ] `treb init` no longer adds `[profile.default.treb.*]` sections to `foundry.toml`
- [ ] Generated `treb.toml` includes comments explaining the structure
- [ ] Existing integration tests for `init` updated to expect `treb.toml`
- [ ] `make lint` passes

### US-006: Update documentation and test fixtures
**Description:** As a developer/user, I need docs and test data to reflect the new config format.

**Acceptance Criteria:**
- [ ] Test fixture `test/testdata/project/` updated with a `treb.toml` (and foundry.toml treb sections removed)
- [ ] CLAUDE.md config examples updated to show `treb.toml` format
- [ ] `treb config show` displays which config source is active (`treb.toml` vs `foundry.toml (legacy)`)
- [ ] `make unit-test` passes
- [ ] `make integration-test` passes

## Functional Requirements

- FR-1: The system must parse `treb.toml` from the project root using the `[ns.<namespace>]` section structure
- FR-2: Each namespace section supports an optional `profile` field mapping to a foundry profile (defaults to namespace name)
- FR-3: Sender configs under `[ns.<name>.senders.<sender>]` use the identical schema as today's `SenderConfig`
- FR-4: When `treb.toml` exists, it is the sole source of treb sender config; `foundry.toml` treb sections are ignored
- FR-5: When `treb.toml` does not exist, the system falls back to reading `[profile.*.treb.*]` from `foundry.toml` (full backwards compat)
- FR-6: A deprecation warning is printed to stderr on every command when legacy config is detected without a `treb.toml`
- FR-7: `treb migrate-config` interactively converts `foundry.toml` treb sections to `treb.toml` format
- FR-8: `treb migrate-config` optionally removes migrated sections from `foundry.toml` upon user confirmation
- FR-9: `treb init` generates `treb.toml` instead of adding treb config to `foundry.toml`
- FR-10: `treb config show` indicates the active config source
- FR-11: Sender config merging (default namespace + active namespace) works identically to today's profile merging

## Non-Goals

- No changes to `.treb/config.local.json` — it remains the store for ephemeral local state (namespace, network)
- No migration of `[rpc_endpoints]` or `[etherscan]` out of `foundry.toml` — those are foundry-native
- No new sender types or sender config schema changes
- No forced migration — users can stay on `foundry.toml` indefinitely (with warnings)
- No changes to the Solidity library (`treb-sol/`)

## Technical Considerations

- **TOML library:** Reuse existing `github.com/BurntSushi/toml` for parsing `treb.toml`
- **TOML writing:** For `migrate-config`, use `toml.Marshal` or template-based generation to produce well-formatted, commented output. The `BurntSushi/toml` library supports encoding.
- **Foundry.toml modification:** Removing `[profile.*.treb.*]` sections from `foundry.toml` requires careful TOML manipulation to avoid corrupting the file. Consider using a TOML-aware edit (parse, remove keys, re-encode) or a simpler approach (read lines, identify and remove treb sections). A TOML round-trip (decode + re-encode) may reorder keys — a line-based approach may be safer for preserving user formatting.
- **Config detection:** To detect legacy config, check if any `FoundryConfig.Profile[*].Treb` is non-nil after parsing
- **Profile field default:** When `[ns.<name>]` omits `profile`, the resolved profile name equals the namespace name — this preserves the current `namespace == FOUNDRY_PROFILE` behavior without requiring explicit config

### Example `treb.toml`

```toml
# Treb deployment configuration
# Docs: https://github.com/trebuchet-org/treb-cli

# Default namespace — senders here are inherited by all namespaces
[ns.default]
profile = "default"  # foundry profile to use

[ns.default.senders.anvil]
type = "private_key"
private_key = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

[ns.default.senders.governor]
type = "oz_governor"
governor = "${GOVERNOR_ADDRESS}"
timelock = "${TIMELOCK_ADDRESS}"
proposer = "anvil"

# Production namespace — inherits default senders, can override
[ns.live]
profile = "live"  # maps to [profile.live] in foundry.toml

[ns.live.senders.safe0]
type = "safe"
safe = "0x3D33783D1fd1B6D849d299aD2E711f844fC16d2F"
signer = "signer0"

[ns.live.senders.signer0]
type = "private_key"
private_key = "${BASE_SEPOLIA_SIGNER0_PK}"
```

## Success Metrics

- All existing integration tests pass without modification (backwards compat)
- New integration tests cover both `treb.toml` and legacy `foundry.toml` paths
- `treb migrate-config` correctly converts the test fixture's `foundry.toml` to a valid `treb.toml`
- No user-facing behavior changes for users who haven't migrated (aside from the warning)

## Open Questions

- Should `treb.toml` support top-level settings beyond namespaces in the future (e.g., default timeout, registry path)? For now we scope to senders only, but the `[ns.*]` prefix leaves room for top-level keys later.
- Should the deprecation warning include a version target for removal of legacy support, or keep it open-ended?
- Should `treb migrate-config` handle edge cases where the user has both `treb.toml` and legacy config (merge vs overwrite)?
