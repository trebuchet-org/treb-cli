# PRD: Config v2 — Senders sub-table & Interactive Migration

## Problem

The current v2 treb.toml has three issues:

### 1. Role names mixed with namespace config

```toml
# Current: "profile" is a magic reserved key among role mappings
[namespace.default]
profile = "default"
anvil = "anvil"         # Is this a role? A config field?
governor = "governor"
```

Role mappings and namespace-level settings share the same flat key space. Adding new namespace-level settings (e.g. `slow`, `rpc_url`) means more reserved keys competing with user-defined role names.

### 2. Migration produces auto-generated names nobody understands

The current `treb migrate` auto-names accounts (e.g. `deployer-key`, `safe-32cb58`) and writes the result without user input. Users get a treb.toml with cryptic names and no opportunity to choose meaningful ones.

### 3. Empty namespaces carried forward silently

Legacy foundry.toml may have profiles that were used once or are no longer active. Migration carries them all forward without asking if they're still needed.

## Solution

### 1. Move role mappings into `[namespace.*.senders]` sub-table

```toml
[namespace.default]
profile = "default"

[namespace.default.senders]
deployer = "anvil"
governance = "governor"
```

This cleanly separates namespace config (`profile`, future settings) from the role→account mapping. The `senders` sub-table is a TOML table, so each key is `role = "account-name"`.

Since the flat v2.0 format was never shipped, there is no backwards compatibility concern — the parser is updated in-place to use the new struct.

For hierarchical namespaces with dots in the name, the TOML is:

```toml
[namespace."production.ntt".senders]
deployer = "ntt-deployer"
```

### 2. Interactive account naming during migration

Instead of auto-generating names silently, the migration flow becomes:

1. Parse all unique sender configs from foundry.toml
2. For each unique sender, display its config and prompt the user to name it:
   ```
   Found account: type=private_key, private_key="${DEPLOYER_PRIVATE_KEY}"
   Name [deployer-key]: deployer
   ```
   The auto-generated name is the default (pre-filled), user can accept with Enter or type a new name.
3. Validate names: no duplicates, valid TOML key characters.
4. Build the namespace→senders mappings using the user-chosen names.

In `--non-interactive` mode, fall back to auto-generated names (current behavior).

### 3. Prune empty namespaces during migration

After building the namespace map, check the deployment registry for each namespace:

1. Load `.treb/deployments.json`
2. Count deployments per namespace
3. For namespaces with 0 deployments, prompt:
   ```
   Namespace "staging" has no deployments. Keep it? [y/N]
   ```
4. Remove namespaces the user declines to keep.

In `--non-interactive` mode, keep all namespaces (safe default).

## TOML Format Details

The `senders` key becomes a sub-table under each namespace. The raw TOML struct changes from:

```go
// Before: everything is flat strings
Namespace map[string]map[string]string `toml:"namespace"`

// After: namespace has typed fields
type namespaceRaw struct {
    Profile string            `toml:"profile"`
    Senders map[string]string `toml:"senders"`
}
Namespace map[string]namespaceRaw `toml:"namespace"`
```

This means `profile` is no longer a "reserved key" in the flat map — it's a proper struct field.

## Migration Output Example

Given this foundry.toml:

```toml
[profile.default.treb.senders.deployer]
type = "private_key"
private_key = "${DEPLOYER_PRIVATE_KEY}"

[profile.production.treb.senders.safe]
type = "safe"
safe = "0x32CB58b145d3f7e28c45cE4B2Cc31fa94248b23F"
signer = "proposer"

[profile.production.treb.senders.proposer]
type = "ledger"
derivation_path = "${PROD_PROPOSER_DERIVATION_PATH}"
```

Interactive session:

```
Found 3 unique accounts to name:

  1. type=private_key  private_key="${DEPLOYER_PRIVATE_KEY}"
     Name [deployer-key]: deployer

  2. type=safe  safe="0x32CB58b..."  signer=→(account 3)
     Name [safe-32cb58]: treasury

  3. type=ledger  derivation_path="${PROD_PROPOSER_DERIVATION_PATH}"
     Name [ledger-prod-proposer]: hw-signer

Namespace "production" has 12 deployments.
Namespace "default" has 3 deployments.

Generated treb.toml:
───────────────────
[accounts.deployer]
type = "private_key"
private_key = "${DEPLOYER_PRIVATE_KEY}"

[accounts.hw-signer]
type = "ledger"
derivation_path = "${PROD_PROPOSER_DERIVATION_PATH}"

[accounts.treasury]
type = "safe"
safe = "0x32CB58b145d3f7e28c45cE4B2Cc31fa94248b23F"
signer = "hw-signer"

[namespace.default]
profile = "default"

[namespace.default.senders]
deployer = "deployer"

[namespace.production]
profile = "production"

[namespace.production.senders]
proposer = "hw-signer"
safe = "treasury"
───────────────────

Write this to treb.toml? [Y/n]
```

### 4. Graceful handling of unknown account references

Currently, if a namespace role references an account that doesn't exist (e.g., production namespace has `deployer = "prod-deployer"` but `[accounts.prod-deployer]` is not defined), the CLI crashes with a hard error:

```
Error: failed to initialize app: failed to resolve namespace "production":
  namespace "production" role "deployer" references unknown account "prod-deployer"
```

This blocks all CLI commands for that namespace. Instead, print a warning to stderr and skip the broken sender:

```
Warning: namespace "production" role "deployer" references unknown account "prod-deployer" — skipping
```

Other valid senders in the same namespace continue to work. The error only surfaces at runtime if a script actually tries to use the missing sender.

## No Backwards Compatibility with v2.0

The flat v2.0 format (`[namespace.default]` with role mappings as sibling keys to `profile`) was never released. The parser is updated in-place to expect the `senders` sub-table. No dual-format support is needed.

## Scope

### In scope
- Move role mappings to `[namespace.*.senders]` sub-table (replace existing parser, not dual-support)
- Graceful handling of unknown account references (warn + skip instead of hard error)
- Interactive account naming in migrate (with pre-filled defaults)
- Deployment stats check + prune prompt for empty namespaces
- Update all tests, golden files, test fixtures
- Update `config show` output to reflect new structure
- Update `treb init` generated treb.toml template

### Out of scope
- XDG directory restructure
- New namespace-level settings beyond `profile`
- Changes to how senders are resolved at runtime (internal `TrebConfig.Senders` map unchanged)
