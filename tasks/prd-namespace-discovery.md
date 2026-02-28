# PRD: Namespace Discovery for `treb list`

## Introduction

When `treb list` is run in a namespace with no deployments, it displays "No deployments found" — a dead end that gives users no indication that deployments exist elsewhere. This feature adds a namespace discovery hint so users can see which namespaces have active deployments and quickly switch context.

## Goals

- Show a helpful hint when the current namespace has no deployments but other namespaces do
- Surface namespace deployment counts so users know where to look
- Include the current network filter context in the hint when applicable
- Keep the happy path (deployments exist) completely unchanged
- Support both human-readable and JSON output modes

## User Stories

### US-001: Add namespace summary to ListDeployments use case
**Description:** As a developer, I need the list use case to return information about other namespaces when the current query returns no results, so the renderer can display a discovery hint.

**Acceptance Criteria:**
- [ ] Add `OtherNamespaces map[string]int` field to `DeploymentListResult` — maps namespace name to deployment count
- [ ] When `result.Deployments` is empty, populate `OtherNamespaces` by calling `repo.GetAllDeployments()` and counting deployments grouped by namespace (filtered by current chain ID if set)
- [ ] `OtherNamespaces` excludes the current namespace (it's already known to be empty)
- [ ] `OtherNamespaces` is `nil` when deployments are found (no extra work on the happy path)
- [ ] Unit tests cover: empty namespace with others available, empty namespace with no others, non-empty namespace (field stays nil)
- [ ] `make lint` passes

### US-002: Render namespace discovery hint in table output
**Description:** As a user running `treb list` in an empty namespace, I want to see which other namespaces have deployments so I can switch context.

**Acceptance Criteria:**
- [ ] When `result.Deployments` is empty and `result.OtherNamespaces` is non-empty, render a hint instead of bare "No deployments found"
- [ ] Hint format:
  ```
  No deployments found in namespace "default" on anvil-31337 (31337).

  Other namespaces with deployments:
    production  5 deployments
    staging     3 deployments

  Use --namespace <name> or `treb config set namespace <name>` to switch.
  ```
- [ ] When no network is set, omit the `on <network>` clause
- [ ] When `OtherNamespaces` is empty (nothing anywhere), keep existing "No deployments found" message
- [ ] Namespace names are sorted alphabetically in the hint
- [ ] Deployment counts are right-aligned for readability
- [ ] `make lint` passes

### US-003: Include namespace discovery in JSON output
**Description:** As a tool consumer, I need the JSON output to include namespace discovery data when the list is empty.

**Acceptance Criteria:**
- [ ] When `--json` is used and deployments list is empty, JSON output includes an `otherNamespaces` field:
  ```json
  {
    "deployments": [],
    "otherNamespaces": {
      "production": 5,
      "staging": 3
    }
  }
  ```
- [ ] When deployments exist, `otherNamespaces` is omitted from JSON output
- [ ] When no other namespaces have deployments, `otherNamespaces` is omitted
- [ ] `make lint` passes

### US-004: Integration tests for namespace discovery
**Description:** As a developer, I need integration tests to verify the namespace discovery feature end-to-end.

**Acceptance Criteria:**
- [ ] Test case: deploy in namespace A, run `treb list` in namespace B — hint shows namespace A with count
- [ ] Test case: deploy in multiple namespaces, list in empty namespace — all populated namespaces shown
- [ ] Test case: deploy in current namespace — no hint shown, normal table output
- [ ] Test case: no deployments anywhere — shows "No deployments found" with no hint
- [ ] Test case: JSON output includes `otherNamespaces` when applicable
- [ ] Golden files created/updated for all test cases
- [ ] `make integration-test` passes

## Functional Requirements

- FR-1: When `treb list` returns zero deployments for the current namespace/network, the system must check for deployments in other namespaces
- FR-2: The other-namespace check must respect the current network/chain ID filter (only count deployments on the same chain)
- FR-3: The discovery hint must list namespace names and their deployment counts
- FR-4: The discovery hint must include instructions for switching namespaces
- FR-5: JSON output must include `otherNamespaces` map when the primary list is empty and others exist
- FR-6: The feature must not add overhead to the happy path (deployments found in current namespace)

## Non-Goals

- No interactive namespace switching (user must run a separate command)
- No cross-chain discovery (only show counts for the current chain filter)
- No changes to `treb show` or other commands — scoped to `treb list` only
- No persistent "last used namespace" tracking
- No changes to the DeploymentRepository interface — use existing `GetAllDeployments()`

## Technical Considerations

- **Performance:** `GetAllDeployments()` loads all deployments into memory. This is only called when the current namespace is empty, so it's a cold path. For projects with very large registries this could be slow, but treb registries are typically small (dozens to hundreds of entries).
- **Filter interaction:** The chain ID filter from `RuntimeConfig.Network` should apply to the other-namespace scan. If a user filters by chain but not namespace, the discovery hint should only count deployments on that chain. If no network is set, count all deployments across all chains.
- **Renderer changes:** The `RenderDeploymentList` method in `render/deployments.go` needs access to the current namespace name and network name for the hint message. These can be added to `DeploymentListResult` or passed as separate parameters.

## Success Metrics

- Users running `treb list` in an empty namespace see actionable guidance instead of a dead end
- All existing integration tests pass without modification
- New integration tests cover the discovery hint in both table and JSON modes
