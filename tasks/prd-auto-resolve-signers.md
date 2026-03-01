# PRD: Auto-resolve Signer and Proposer Account References

## Introduction

When a namespace sender maps to a Safe or OZ Governor account, that account's `signer` or `proposer` field references another account by name. Currently, the referenced account must also be explicitly mapped as a namespace sender — otherwise `BuildSenderScriptConfig()` fails with `"safe signer 'dev-pk' not found in sender configurations"`. This is unintuitive: users shouldn't have to assign a dummy role to an account that's only needed as a signer for another account.

The fix is to auto-resolve the reference chain during config resolution: when building `TrebConfig.Senders`, automatically include any accounts transitively referenced via `signer`/`proposer` fields.

## Problem

Given this config:

```toml
[accounts.dev-pk]
type = "private_key"
private_key = "${DEPLOYER_PRIVATE_KEY}"

[accounts.dev-safe]
type = "safe"
safe = "0x..."
signer = "dev-pk"

[namespace.mainnet.senders]
deployer = "dev-safe"
```

Today this fails at runtime because:
1. `ResolvedNamespaceToTrebConfig()` builds `TrebConfig.Senders` with only `deployer → dev-safe` config
2. `BuildSenderScriptConfig()` reads `dev-safe.Signer = "dev-pk"` and looks for `"dev-pk"` in `TrebConfig.Senders`
3. `"dev-pk"` is not there → error

Users must work around this by adding a pointless role mapping: `signer = "dev-pk"`. This leaks implementation details (which account signs for the Safe) into the namespace role definitions, where it doesn't belong.

## Goals

- Automatically include signer/proposer accounts in `TrebConfig.Senders` when they're referenced by a namespace sender's account
- Handle transitive references (e.g., Governor → proposer account → which is itself a Safe → its signer account)
- Detect circular references and report a clear error
- No changes to the treb.toml config format — this is purely a resolution fix
- No changes to how `SENDER_CONFIGS` is built or how Solidity receives sender data

## User Stories

### US-001: Auto-resolve signer references for Safe accounts
**Description:** As a developer, I want namespace senders that reference Safe accounts to automatically include the Safe's signer account in the resolved config, so I don't have to map it as a namespace sender.

**Acceptance Criteria:**
- [ ] A namespace sender mapping `deployer = "dev-safe"` where `dev-safe` is a Safe with `signer = "dev-pk"` automatically includes `dev-pk` in `TrebConfig.Senders`
- [ ] The auto-included account uses its account name as the key in `TrebConfig.Senders` (e.g., `"dev-pk" → SenderConfig{...}`)
- [ ] `treb run` succeeds without explicitly mapping the signer as a namespace sender
- [ ] Existing configs that DO explicitly map signers continue to work (no regression)
- [ ] When the same account is both explicitly mapped and auto-resolved, no conflict or duplication
- [ ] Lint passes, unit tests pass

### US-002: Auto-resolve proposer references for OZ Governor accounts
**Description:** As a developer, I want namespace senders that reference OZ Governor accounts to automatically include the Governor's proposer account.

**Acceptance Criteria:**
- [ ] A namespace sender mapping `governor = "mento-gov"` where `mento-gov` has `proposer = "dev-pk"` automatically includes `dev-pk` in `TrebConfig.Senders`
- [ ] Works identically to Safe signer auto-resolution
- [ ] Lint passes, unit tests pass

### US-003: Handle transitive and circular references
**Description:** As a developer, I want the resolution to handle chains like Governor → proposer (Safe) → signer (Ledger), and detect circular references.

**Acceptance Criteria:**
- [x] Transitive chains are fully resolved: if a Governor's proposer is a Safe, the Safe's signer is also included
- [x] Circular references (e.g., account A signers with B, B signers with A) produce a clear error: `"circular account reference: dev-safe → dev-pk → dev-safe"`
- [x] Self-references (account signers with itself) produce a clear error
- [x] Lint passes, unit tests pass

### US-004: Integration tests for auto-resolution
**Description:** As a developer, I want integration tests confirming that deployments work with auto-resolved signers.

**Acceptance Criteria:**
- [ ] Test: Safe account with signer NOT in namespace senders → deployment succeeds
- [ ] Test: Governor account with proposer NOT in namespace senders → deployment succeeds (or correctly builds config)
- [ ] Test: transitive chain (Governor → Safe proposer → PK signer) resolves correctly
- [ ] Test: existing configs with explicit signer mappings still work
- [ ] Golden files updated
- [ ] Lint passes

## Functional Requirements

- FR-1: In `ResolvedNamespaceToTrebConfig()` (internal/config/trebfile_v2.go), after building the initial role→sender map, walk all senders and recursively resolve `signer`/`proposer` references from the global accounts map
- FR-2: Auto-resolved accounts are added to `TrebConfig.Senders` keyed by their account name (not a role name)
- FR-3: If an auto-resolved account name collides with an existing role name in the senders map, and the configs are identical, silently deduplicate. If configs differ, emit a warning and prefer the explicitly-mapped version
- FR-4: Resolve references transitively: if account A references B, and B references C, all three must end up in `TrebConfig.Senders`
- FR-5: Detect circular references during resolution and return an error with the full cycle path
- FR-6: The existing validation in `BuildSenderScriptConfig()` (senders.go lines 247-249, 336-338) continues to work as a safety net — the fix is upstream in config resolution, not in sender building
- FR-7: No changes to the `SENDER_CONFIGS` encoding or Solidity-side parsing

## Non-Goals

- No config format changes — `treb.toml` schema stays exactly as-is
- No renaming of fields (`signer`, `proposer`, `accounts`, `senders` all stay)
- No changes to the legacy foundry.toml config path (it already worked because signers were co-located with senders)
- No changes to how `BuildSenderScriptConfig()` collects signers (lines 67-74 in senders.go) — the fix is that they'll already be in `TrebConfig.Senders`

## Technical Considerations

- The fix is localized to `ResolvedNamespaceToTrebConfig()` in `internal/config/trebfile_v2.go` — this is the only place where V2 accounts are converted to the flat `TrebConfig.Senders` map
- The global accounts map (`TrebFileConfigV2.Accounts`) is already available at this point, so looking up referenced accounts is straightforward
- A simple iterative approach works: loop through senders, check for signer/proposer fields, look up referenced accounts, add them, repeat until no new accounts are added (handles transitive chains)
- Use a visited set for circular reference detection
- The `BuildSenderScriptConfig()` code in senders.go that collects `safeSigners` (lines 67-74) will continue to work — it just won't fail anymore because the referenced accounts will already be present

## Success Metrics

- The example config from the problem statement works without modification
- No regressions in existing tests (configs with explicit signer mappings)
- Clear error message for circular references
