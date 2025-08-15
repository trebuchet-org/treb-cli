# CRUSH.md

Guidance for agentic coding in this repository.

Build/lint/test commands
- Build CLI: make build (outputs bin/treb)
- Build v2 CLI: make build-v2 (outputs bin/treb-v2)
- Build Foundry lib: cd treb-sol && forge build
- Generate Go bindings: make bindings (runs forge build + abigen)
- Run all Go tests: go test -v ./...
- Run a single Go package: go test -v ./cli/pkg/abi
- Run a single Go test file: go test -v ./cli/pkg/abi -run TestName
- Run integration tests: make integration-test (builds binaries, prepares fixtures)
- Run integration tests with coverage: make integration-test-coverage
- Lint (golangci-lint): make lint (install via make dev-setup or make lint-install)
- Format Go: make fmt
- Foundry tests (treb-sol): cd treb-sol && forge test -vvv

Code style guidelines
- Language: Go (CLI/orchestration) + Solidity (treb-sol submodule). All chain I/O is via Foundry scripts; Go must not perform direct chain RPC writes.
- Imports: group std, third-party, internal; use module path github.com/trebuchet-org/treb-cli for internal; avoid dot imports; keep aliases minimal and descriptive.
- Formatting: gofmt -s; keep lines concise; prefer small functions; no commented-out code; logs minimal and actionable.
- Types and errors: return (T, error); wrap with fmt.Errorf("context: %w", err); sentinel errors in internal/domain/errors.go; avoid panics except in main startup failures.
- Naming: CamelCase for types; lowerCamel for vars/funcs; Error values use ErrX; interfaces end with -er if behavior (e.g., Resolver); keep package names short, lowercase, no underscores.
- Context: pass context.Context as first param for I/O, network, subprocess; respect timeouts; no global state.
- CLI: add commands under cli/cmd following cobra patterns present; keep user-facing messages clear and stable for tests.
- Testing: prefer table-driven tests; use test/helpers for shared utilities; for fixtures, keep deterministic outputs to satisfy golden tests.
- Security: never log secrets; .env values only loaded where needed; do not commit keys; respect .gitignore.
- Solidity: follow foundry defaults; run solhint if present; keep deterministic deployment patterns; use CreateX; tests under treb-sol/test; avoid stateful global singletons.

Assistant rules
- Also consult CLAUDE.md (project architecture and workflows). No .cursor or Copilot rules present.
