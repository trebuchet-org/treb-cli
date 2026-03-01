# PRD: External URL for Fork Enter

## Introduction

Allow `treb fork enter` to accept a `--url` flag pointing to an external Anvil-compatible endpoint instead of spinning up a local Anvil process. This supports workflows where a shared or remote Anvil fork is already running (e.g., a team-shared fork, a CI-managed fork, or a fork started by another tool). All existing fork functionality (run, revert, diff, exit) continues to work identically — the only difference is that treb doesn't own the Anvil process lifecycle.

## Goals

- Support connecting to an external Anvil fork via `--url` flag on `fork enter`
- Maintain full snapshot/revert/diff/exit functionality by requiring an Anvil-compatible endpoint
- Keep the existing local fork flow unchanged (no regressions)
- Minimize code changes by reusing the existing fork infrastructure

## User Stories

### US-001: Enter fork with external URL
**Description:** As a developer, I want to run `treb fork enter sepolia --url http://remote:8545` so that I can use an already-running Anvil fork instead of treb starting one for me.

**Acceptance Criteria:**
- [ ] `treb fork enter <network> --url <url>` connects to the provided URL
- [ ] Network name is still required for config resolution (chain ID, env var name, RPC endpoint lookup)
- [ ] The endpoint is validated: `evm_snapshot` is called on enter and must succeed
- [ ] If validation fails, the command exits with a clear error explaining the endpoint must be Anvil-compatible
- [ ] Registry files are backed up (same as local fork flow)
- [ ] An initial EVM snapshot is taken and stored in fork state
- [ ] Fork state is persisted to `.treb/priv/fork-state.json`
- [ ] The `--url` flag and local anvil startup are mutually exclusive
- [ ] Unit tests pass, lint passes

### US-002: Run scripts against external fork
**Description:** As a developer, I want `treb run` to work transparently against the external fork, just like it does with a local fork.

**Acceptance Criteria:**
- [ ] `treb run <script>` routes to the external fork URL via env var override (existing mechanism)
- [ ] Pre-run health check verifies the external endpoint is still reachable
- [ ] Pre-run EVM snapshot is taken (same as local fork flow)
- [ ] Registry files are backed up before each run
- [ ] `TREB_FORK_MODE=true` is set in the forge environment
- [ ] No behavioral difference from local fork mode during script execution
- [ ] Unit tests pass, lint passes

### US-003: Revert and diff against external fork
**Description:** As a developer, I want `treb fork revert` and `treb fork diff` to work against external forks so I can undo runs and inspect changes.

**Acceptance Criteria:**
- [ ] `treb fork revert` calls `evm_revert` on the external endpoint and restores registry files
- [ ] `treb fork revert --all` reverts to initial snapshot
- [ ] `treb fork diff` compares current registry state against initial backup
- [ ] All commands work identically to local fork mode
- [ ] Unit tests pass, lint passes

### US-004: Exit and restart external fork
**Description:** As a developer, I want `treb fork exit` to clean up fork state without trying to stop a process I don't own, and `treb fork restart` to handle the external case gracefully.

**Acceptance Criteria:**
- [ ] `treb fork exit` restores registry files to pre-fork state (same as local)
- [ ] `treb fork exit` does NOT attempt to stop/kill any process
- [ ] Fork state is cleaned up from `.treb/priv/fork-state.json`
- [ ] Snapshot directories are cleaned up
- [ ] `treb fork restart` either re-validates the external endpoint or returns a clear error explaining that restart is not supported for external forks (since treb doesn't own the process)
- [ ] `treb fork status` shows the fork as external (no PID, shows URL)
- [ ] Unit tests pass, lint passes

### US-005: Integration tests for external fork flow
**Description:** As a developer, I want integration tests covering the external URL fork workflow.

**Acceptance Criteria:**
- [ ] Test: `fork enter <network> --url <anvil-url>` succeeds with a running Anvil
- [ ] Test: `fork enter <network> --url <dead-url>` fails with clear error
- [ ] Test: run a script against external fork, verify deployment registered
- [ ] Test: revert after run, verify state restored
- [ ] Test: exit external fork, verify registry files restored and state cleaned up
- [ ] Test: `fork status` shows external fork info
- [ ] Golden files updated

## Functional Requirements

- FR-1: Add `--url` string flag to the `fork enter` command
- FR-2: When `--url` is provided, skip local Anvil startup (port allocation, process spawn, PID file creation, log file creation)
- FR-3: When `--url` is provided, validate the endpoint by calling `evm_snapshot` — if it fails, exit with error: "endpoint does not support Anvil snapshot operations; --url requires an Anvil-compatible endpoint"
- FR-4: Store a boolean `External` field (or equivalent) on `ForkEntry` to distinguish external forks from locally-managed forks
- FR-5: Store the provided URL as the `ForkURL` in `ForkEntry`; `OriginalRPC` remains the resolved network RPC (for reference)
- FR-6: `AnvilPID` should be 0 and `PidFile`/`LogFile` should be empty for external forks
- FR-7: All env var override logic in `run_script.go` works unchanged (it already uses `ForkURL` from fork state)
- FR-8: `exit_fork.go` must skip process termination when `External` is true (or PID is 0)
- FR-9: `restart_fork.go` must return a clear error for external forks ("cannot restart external fork — treb does not manage this process")
- FR-10: `fork_status.go` should indicate external forks in its output (e.g., "external" label, no PID/uptime)
- FR-11: `fork enter` must still reject duplicate forks for the same network (same as today)
- FR-12: The `--url` flag must not interfere with the existing env var migration flow — when `--url` is provided, skip the hardcoded-URL-in-foundry.toml check since the user is explicitly providing the endpoint

## Non-Goals

- No support for non-Anvil endpoints (e.g., Hardhat, Tenderly) — we require `evm_snapshot`/`evm_revert`
- No automatic discovery of remote Anvil instances
- No changes to the `treb dev anvil` commands
- No persistent `--url` configuration in `treb.toml` (it's a per-session flag)
- No WebSocket URL support (HTTP only, matching existing Anvil RPC pattern)

## Technical Considerations

- The `EnterFork` use case already receives `RPCURL` as a parameter — the main change is conditionally skipping Anvil startup and adding validation
- `ForkEntry` domain model needs a new `External bool` field; this is a backwards-compatible addition to the JSON state file
- The health check in `run_script.go` already calls `AnvilManager.GetStatus()` which does an RPC health check — this should work for external URLs if the instance is constructed correctly
- The `--slow` flag is already added for all fork mode runs, which is correct for external forks too
- CreateX deployment should be attempted on external forks (via `anvil_setCode`) since the remote Anvil may not have it — this is already part of the startup flow

## Success Metrics

- External fork enter/run/revert/exit works end-to-end without errors
- No regressions in existing local fork tests
- Clear error messages when endpoint is unreachable or incompatible

## Open Questions

- Should `fork enter --url` attempt CreateX deployment on the external endpoint? The current flow deploys CreateX on every new Anvil instance. For an external fork that may already have CreateX (or where another user is managing it), should we still force-deploy via `anvil_setCode`, or check first and skip if present?
