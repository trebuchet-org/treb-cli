# PRD: CLI Output Cosmetic Polish

## Introduction

A cosmetic polish pass across all treb CLI commands to fix alignment bugs, improve visual consistency, and bring newer commands (fork, config, register, reset) up to the standard set by `run`, `show`, and `list`. The `run`, `show`, and `list` commands are the gold standard and should not be modified (except for the list column alignment bug).

## Goals

- Fix the `list` command column alignment bug where verification status columns drift
- Polish fork commands: add colors, fix io.Writer usage, improve spacing
- Expand `config` to show resolved sender/account information from treb.toml
- Audit and fix inconsistencies across all other commands (reset, register, prune, sync, verify, networks, tag, init, gen, dev anvil, compose)

## Audit Findings

### Commands That Are Good (Do Not Touch)
- **`treb run`**: Gold standard. Rich banner, colored transaction trees, deployment summary, script logs. Well-structured with section headers and `─` separators.
- **`treb show`**: Gold standard. Clean section-based layout with colored values, `=` separator header, proper indentation.
- **`treb list`**: Gold standard layout/design. Has one alignment bug (see US-001).

### Commands That Need Work

#### `treb list` — Column Alignment Bug
The verification status column (`e[✔︎] s[✔︎] b[✔︎]` vs `e[-] s[-] b[-]`) causes downstream columns (timestamp) to drift. The `✔︎` character has a different visual width than `-`, and the padding compensation in `getVerifierStatuses()` adds a trailing space after wide characters — but this space is counted differently by the `go-pretty` table library vs the terminal. The `displayWidth()` function uses `runewidth.StringWidth()` on the stripped string, but the padding space added inside the colored string isn't accounted for correctly in column width calculation.

**Root cause**: `formatStatus()` adds padding _inside_ the formatted string conditionally, then `calculateTableColumnWidths` measures the stripped string width. The go-pretty library does its own width calculation that may disagree. The fix should normalize the verification cell to a fixed visual width regardless of content.

#### `treb fork` — Barebone Output
All fork subcommands (`enter`, `exit`, `status`, `revert`, `restart`, `history`, `diff`) use raw `fmt.Printf`/`fmt.Println` instead of the renderer's `io.Writer`. No colors are used anywhere. The output is functional but visually flat compared to other commands.

**Issues:**
- Uses `fmt.Println`/`fmt.Printf` directly instead of writing to `io.Writer` (breaks testability, piping)
- No colors on any values (URLs, PIDs, network names, statuses)
- No section headers or visual hierarchy for `fork status` (which shows multiple forks)
- `fork diff` uses `%-20s` fixed width which may truncate long contract names
- `fork history` marker `→` placement is awkward with double-space prefix
- No success/info icons consistent with other commands

#### `treb config` — Minimal and Missing Sender Info
Config show is bare: just namespace, network, source, and file path. Doesn't show the resolved senders from treb.toml which is the most useful information for debugging deployment issues.

**Issues:**
- Doesn't show sender configuration from treb.toml (the most asked-for debug info)
- No colors on values
- Emoji usage (`📋`, `📦`, `📁`) is inconsistent with other commands that use colored text headers
- The "Config source" line is somewhat redundant with the file path

#### `treb reset` — Inline Rendering
Reset has no dedicated renderer; all output is inline in the cobra command. Uses no colors, no icons, plain `fmt.Fprintf`. The confirmation prompt is functional but plain.

#### `treb register` — Inline Rendering, Inconsistent Style
Register also renders inline in the cobra command. Uses `color` in some places but inconsistently. The success output with numbered deployments is useful but unpolished. JSON output mode uses raw `fmt.Printf` instead of proper JSON marshaling.

#### `treb networks` — Fine but Could Match
Uses emoji (`🌐`, `✅`, `❌`). Output is simple enough that it works, but network names aren't colored and there's no alignment of chain IDs.

#### `treb verify` — Mostly Good
Well-structured with spinners, progress indicators, and colored status. One minor issue: mixes `✓` (plain checkmark) with `✗` (plain X) in result lines vs emoji elsewhere.

#### `treb sync` — Mostly Good
Clean bullet-point style output. Minor: uses `•` bullets which look good. The "Syncing registry..." header could have a spinner in interactive mode.

#### `treb prune` — Mostly Good
Clean output with emoji headers. Minor: could benefit from colored addresses/IDs.

#### `treb tag` — Mostly Good
Has colors and good formatting. Minor: uses `fmt.Println`/`fmt.Print` directly instead of `io.Writer` in some places.

#### `treb init` — Good
Well-structured with step indicators and "Next steps" guide. Uses colors appropriately.

#### `treb gen deploy` — Minimal but Fine
Single success line plus instructions. Appropriate for the simplicity of the command.

#### `treb dev anvil` — Good
Has colors and status indicators. Minor inconsistency: mixes emoji icons with colored text labels.

#### `treb compose` — Good
Uses `═` double-line separator for summary which is distinct but intentional (heavier weight for orchestration). Fine as-is.

## User Stories

### US-001: Fix list command column alignment
**Description:** As a user, I want the `list` output columns to be perfectly aligned regardless of verification status content so that the output is easy to scan.

**Acceptance Criteria:**
- [ ] Verification status column has consistent visual width whether showing `✔︎`, `-`, or `⏳`
- [ ] Address column starts at the same position for all rows
- [ ] Timestamp column starts at the same position for all rows
- [ ] Existing golden tests are updated to reflect the fix
- [ ] Unit tests pass
- [ ] Integration tests pass

### US-002: Polish fork command output — io.Writer and colors
**Description:** As a user, I want fork commands to have colored output and use the standard io.Writer pattern so that the output is visually consistent with other commands and testable.

**Acceptance Criteria:**
- [ ] All `fmt.Println`/`fmt.Printf` calls in `fork.go` renderer replaced with `fmt.Fprintln`/`fmt.Fprintf` using the renderer's `io.Writer`
- [ ] `ForkRenderer` struct updated to accept and store an `io.Writer`
- [ ] Network names colored (cyan or blue, matching list/run style)
- [ ] URLs colored (faint/gray for fork URLs)
- [ ] Status values colored (green for healthy, red for unhealthy)
- [ ] PID numbers colored (faint)
- [ ] Success messages use `✓` with green color (matching run/sync pattern)
- [ ] `fork status` header "Active Forks" is bold
- [ ] `fork diff` uses dynamic width for contract names instead of fixed `%-20s`
- [ ] `fork history` current marker `→` is colored (green or cyan)
- [ ] Keep the simple key-value format — do not add section headers/separators

### US-003: Expand config command to show sender information
**Description:** As a user running `treb config`, I want to see the resolved sender configuration from treb.toml for the current namespace so that I can debug deployment configuration issues without opening the TOML file.

**Acceptance Criteria:**
- [ ] `treb config` shows the senders defined for the current namespace
- [ ] Each sender shows: name, type (private_key/ledger/safe), and relevant details
- [ ] For `private_key` type: show that a key is configured (not the key itself — mask it)
- [ ] For `safe` type: show the safe address
- [ ] For `ledger` type: show the derivation path
- [ ] Sender info section uses colors: sender names in cyan, types in faint
- [ ] If no treb.toml exists, show a hint about creating one
- [ ] Values that reference env vars show the env var name (not resolved value for secrets)
- [ ] Namespace and network values are colored
- [ ] Remove redundant emoji icons, use consistent colored text labels
- [ ] Unit tests pass
- [ ] Integration tests pass (if config tests exist)

### US-004: Move reset command output to dedicated renderer
**Description:** As a developer, I want the reset command to use a dedicated renderer so that output is consistent and testable.

**Acceptance Criteria:**
- [ ] Create `render/reset.go` with a `ResetRenderer` struct
- [ ] Move all output formatting from `reset.go` cobra command into the renderer
- [ ] Add colors: item counts in bold, namespace/network in cyan
- [ ] Confirmation prompt stays in the cobra command (interactive concern)
- [ ] Success message uses `✓` green pattern
- [ ] Unit tests pass

### US-005: Clean up register command output
**Description:** As a user, I want the register command's success output to be cleanly formatted with colors so that registered deployments are easy to review.

**Acceptance Criteria:**
- [ ] Create `render/register.go` with a `RegisterRenderer` struct
- [ ] Move success output formatting from `register.go` cobra command into the renderer
- [ ] Deployment IDs colored cyan, addresses colored green, contract names colored yellow
- [ ] JSON output uses proper `json.Marshal` instead of raw `fmt.Printf`
- [ ] Interactive prompts stay in the cobra command
- [ ] Unit tests pass

### US-006: Minor polish for networks command
**Description:** As a user, I want the networks list to have aligned chain IDs and colored network names.

**Acceptance Criteria:**
- [ ] Network names are colored (cyan or bold)
- [ ] Chain IDs are right-aligned for visual consistency
- [ ] Error messages are colored red
- [ ] Keep emoji icons (`✅`/`❌`) — they work well here
- [ ] Unit tests pass

### US-007: Fix tag renderer io.Writer usage
**Description:** As a developer, I want the tag renderer to consistently use `io.Writer` instead of mixing `fmt.Println` with `io.Writer` so that output is fully testable.

**Acceptance Criteria:**
- [ ] All `fmt.Println`/`fmt.Print`/`fmt.Printf` calls in `tag.go` renderer replaced with `fmt.Fprintln`/`fmt.Fprint`/`fmt.Fprintf` using an `io.Writer`
- [ ] `TagRenderer` struct updated to accept and store an `io.Writer`
- [ ] No functional changes to output content
- [ ] Unit tests pass

### US-008: Normalize icon usage across commands
**Description:** As a user, I want consistent success/error/warning indicators across all commands so the CLI feels cohesive.

**Acceptance Criteria:**
- [ ] Success indicators: use `✓` (green) for inline results, `✅` only for major milestones (init, gen)
- [ ] Error indicators: use `✗` (red) for inline errors, `❌` for fatal/blocking errors
- [ ] Warning indicators: use `⚠️` (yellow) consistently
- [ ] Document the icon convention in a comment in `helpers.go`
- [ ] Audit all renderers and update any outliers
- [ ] No changes to run/show/list output (except the list alignment fix in US-001)

## Functional Requirements

- FR-1: The verification status cell in list output must render at a fixed visual width (e.g., always pad to the width of the widest possible content)
- FR-2: Fork renderer must accept `io.Writer` as constructor parameter and use it for all output
- FR-3: Config command must resolve and display sender configuration from treb.toml for the current namespace
- FR-4: Config command must never display raw secret values (private keys); show env var names or masked placeholders
- FR-5: Reset and register commands must use dedicated renderers in the render package
- FR-6: All renderers must use `io.Writer` for output (no direct `fmt.Println` to stdout)
- FR-7: Colors must degrade gracefully when output is piped (the `color` library handles this automatically via `NO_COLOR` / terminal detection)

## Non-Goals

- No changes to the `run` command output
- No changes to the `show` command output
- No changes to the `list` command output beyond the column alignment fix
- No changes to command behavior or business logic
- No new commands or flags
- No changes to JSON output format (except register's raw printf → json.Marshal)
- No interactive/TUI enhancements (spinners, progress bars, etc.)
- No changes to the compose command output

## Technical Considerations

- The `go-pretty` table library does its own ANSI-aware width calculation; the fix for US-001 should work _with_ the library rather than fighting it
- The `fatih/color` library auto-detects terminal capabilities; no special handling needed for piped output
- Fork renderer currently has no `io.Writer` field — adding one is a breaking change to `NewForkRenderer()` signature; callers must be updated
- Config sender resolution requires reading treb.toml from the config package; the `ShowConfigResult` struct needs new fields
- Tag renderer uses `config.RuntimeConfig` in constructor but doesn't need it for rendering; consider simplifying

## Success Metrics

- All columns in `list` output are perfectly aligned for any combination of verification statuses
- All commands use colored output that is visually scannable
- `treb config` shows enough information to debug sender issues without opening treb.toml
- No raw `fmt.Println` calls remain in any renderer (all use `io.Writer`)
- All existing tests continue to pass (golden files updated where needed)

## Open Questions

- Should `fork status` show a summary line (e.g., "2 active forks") or is the current format sufficient?
- For `config` sender display, should we show senders from all namespaces or only the current one? (Recommendation: current namespace only, with a note about `--namespace` flag)
- Should we add `--no-color` as a global flag, or rely on the `NO_COLOR` env var convention? (Recommendation: rely on `NO_COLOR` — it's the standard)
