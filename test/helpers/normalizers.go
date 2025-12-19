package helpers

import (
	"path/filepath"
	"regexp"
	"strings"
)

// Normalizer processes output to remove dynamic content
type Normalizer interface {
	Normalize(output string) string
}

// TimestampNormalizer replaces timestamps with placeholders
type TimestampNormalizer struct{}

func (n TimestampNormalizer) Normalize(output string) string {
	// ISO timestamps: 2024-08-09T14:30:45Z
	output = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z?(\+\d+:\d+)?`).ReplaceAllString(output, "<TIMESTAMP>")

	// Standard timestamps: 2024-08-09 14:30:45
	output = regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`).ReplaceAllString(output, "<TIMESTAMP>")

	// Relative times: "2 minutes ago", "1 hour ago"
	output = regexp.MustCompile(`\d+ \w+ ago`).ReplaceAllString(output, "<TIME_AGO>")

	// Unix timestamps (10-13 digits)
	output = regexp.MustCompile(`\b\d{10,13}\b`).ReplaceAllString(output, "<UNIX_TIME>")

	return output
}

// AddressNormalizer replaces Ethereum addresses
type AddressNormalizer struct{}

func (n AddressNormalizer) Normalize(output string) string {
	return regexp.MustCompile(`0x[a-fA-F0-9]{40}`).ReplaceAllString(output, "0x<ADDRESS>")
}

type LabelNormalizer struct {
	Label string
}

func (n LabelNormalizer) Normalize(output string) string {
	return strings.ReplaceAll(output, n.Label, "<LABEL>")
}

// HashNormalizer replaces transaction hashes
type HashNormalizer struct{}

func (n HashNormalizer) Normalize(output string) string {
	output = regexp.MustCompile(`Tx: 0x[a-fA-F0-9]{64}`).ReplaceAllString(output, "Tx: 0x<HASH>")
	output = regexp.MustCompile(`Hash: 0x[a-fA-F0-9]{64}`).ReplaceAllString(output, "Hash: 0x<HASH>")
	output = regexp.MustCompile(`tx 0x[a-fA-F0-9]{64}`).ReplaceAllString(output, "tx 0x<HASH>")
	output = regexp.MustCompile(`transaction 0x[a-fA-F0-9]{64}`).ReplaceAllString(output, "transaction 0x<HASH>")
	return output
}

// ColorNormalizer removes ANSI color codes and other control sequences
type ColorNormalizer struct{}

func (n ColorNormalizer) Normalize(output string) string {
	// Remove all ANSI escape sequences including:
	// - Color codes: \x1b[0-9;]*m
	// - Line clearing: \x1b[2K
	// - Cursor movement and other control sequences: \x1b[0-9;]*[A-Za-z]
	return regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`).ReplaceAllString(output, "")
}

// FoundryWarningsNormalizer removes noisy Foundry warnings that are not relevant to behavior
// and can vary between environments/toolchain versions.
type FoundryWarningsNormalizer struct{}

func (n FoundryWarningsNormalizer) Normalize(output string) string {
	// Example:
	// Warning: Found unknown `treb` config for profile `default` defined in foundry.toml.
	// Warning: Found unknown `treb` config for profile `live` defined in foundry.toml.
	output = regexp.MustCompile(`(?m)^Warning: Found unknown `+"`"+`treb`+"`"+` config for profile `+"`"+`[^`+"`"+`]+`+"`"+` defined in foundry\.toml\.?\s*$\n?`).ReplaceAllString(output, "")
	return output
}

// LineClearArtifactNormalizer removes occasional line-clear artifacts that can leak into
// captured output depending on how control codes are rendered/escaped.
type LineClearArtifactNormalizer struct{}

func (n LineClearArtifactNormalizer) Normalize(output string) string {
	// Some environments render the escape sequence "\033[2K" as a literal "3[2K".
	// Strip it if it appears.
	output = regexp.MustCompile(`\r?3\[2K`).ReplaceAllString(output, "")
	return output
}

// GitCommitNormalizer replaces git commit hashes
type GitCommitNormalizer struct{}

func (n GitCommitNormalizer) Normalize(output string) string {
	return regexp.MustCompile(`[a-f0-9]{7,40}`).ReplaceAllString(output, "<COMMIT>")
}

// TargetedGitCommitNormalizer only replaces git commits in specific contexts
type TargetedGitCommitNormalizer struct{}

func (n TargetedGitCommitNormalizer) Normalize(output string) string {
	// Only replace git commits in specific contexts
	// Git Commit field in show output (full 40 char hash)
	output = regexp.MustCompile(`Git Commit: [a-f0-9]{40}`).ReplaceAllString(output, "Git Commit: <GIT_COMMIT>")
	// Commit in version output (7 char short hash)
	output = regexp.MustCompile(`commit: [a-f0-9]{7}`).ReplaceAllString(output, "commit: <COMMIT>")
	return output
}

// TargetedHashNormalizer only replaces hashes in specific contexts
type TargetedHashNormalizer struct{}

func (n TargetedHashNormalizer) Normalize(output string) string {
	// Transaction hashes in specific contexts
	output = regexp.MustCompile(`Tx: 0x[a-fA-F0-9]{64}`).ReplaceAllString(output, "Tx: 0x<HASH>")
	output = regexp.MustCompile(`Hash: 0x[a-fA-F0-9]{64}`).ReplaceAllString(output, "Hash: 0x<HASH>")
	// Init code hash and bytecode hash in show output
	output = regexp.MustCompile(`Init Code Hash: 0x[a-fA-F0-9]{64}`).ReplaceAllString(output, "Init Code Hash: 0x<HASH>")
	output = regexp.MustCompile(`Bytecode Hash: 0x[a-fA-F0-9]{64}`).ReplaceAllString(output, "Bytecode Hash: 0x<HASH>")
	return output
}

// BlockNumberNormalizer replaces block numbers
type BlockNumberNormalizer struct{}

func (n BlockNumberNormalizer) Normalize(output string) string {
	output = regexp.MustCompile(`Block: \d+`).ReplaceAllString(output, "Block: <BLOCK>")
	output = regexp.MustCompile(`block \d+`).ReplaceAllString(output, "block <BLOCK>")
	return output
}

// GasNormalizer replaces gas values
type GasNormalizer struct{}

func (n GasNormalizer) Normalize(output string) string {
	output = regexp.MustCompile(`Gas: \d+`).ReplaceAllString(output, "Gas: <GAS>")
	output = regexp.MustCompile(`gas_used: \d+`).ReplaceAllString(output, "gas_used: <GAS>")
	return output
}

// PathNormalizer replaces absolute paths with relative ones
type PathNormalizer struct{}

func (n PathNormalizer) Normalize(output string) string {
	// Get absolute path to fixture dir
	fixtureAbs, _ := filepath.Abs(GetFixtureDir())

	// Replace absolute paths with relative ones
	output = strings.ReplaceAll(output, fixtureAbs, ".")

	// Normalize path separators
	output = strings.ReplaceAll(output, "\\", "/")

	return output
}

type RepositoryHormalizer struct{}

func (n RepositoryHormalizer) Normalize(output string) string {
	output = regexp.MustCompile(`tx-0x[a-fA-F0-9]{64}`).ReplaceAllString(output, `tx-<ID>`)
	output = regexp.MustCompile(`tx-internal-[a-fA-F0-9]{64}`).ReplaceAllString(output, `tx-internal-<ID>`)
	return output
}

type DebugNormalizer struct{}

func (n DebugNormalizer) Normalize(output string) string {
	output = regexp.MustCompile(`(?mi)^level=DEBUG.*\n`).ReplaceAllString(output, "")
	return output
}

// VersionNormalizer replaces version-related strings
type VersionNormalizer struct{}

func (n VersionNormalizer) Normalize(output string) string {
	// Treb version: v1.0.0-beta.1-95-g6a2e70e
	output = regexp.MustCompile(`v\d+\.\d+\.\d+(-[a-zA-Z0-9\.\-]+)?`).ReplaceAllString(output, "v<VERSION>")
	output = regexp.MustCompile(`(?m:^treb [a-z0-9]{7}$)`).ReplaceAllString(output, "treb v<VERSION>")

	// Git commit in version strings
	output = regexp.MustCompile(`-g[a-f0-9]{7,}`).ReplaceAllString(output, "-g<COMMIT>")

	return output
}

// SpinnerNormalizer normalizes spinner animation output
// Spinners can have variable numbers of frames depending on timing, so we collapse
// multiple consecutive spinner lines into a single one
type SpinnerNormalizer struct{}

func (n SpinnerNormalizer) Normalize(output string) string {
	// Match spinner patterns: \r\r[spinner_char] Compiling...\r\r[spinner_char] Compiling...
	// The spinner uses Unicode braille characters in brackets like [⠃], [⠊], [⠒], etc.
	// We'll collapse ALL consecutive spinner frames into a single normalized one

	// Some toolchains print multiple frames on separate lines (no leading \r), e.g.:
	//   \r[⠃] Compiling...\n[⠊] Compiling...\n
	// So we match sequences where each frame may optionally be prefixed with one or more \r.
	spinnerPattern := regexp.MustCompile(`((?:\r)*\[[^\]]+\] Compiling\.\.\.\s*(?:\r?\n|\r))+`)

	// Replace ALL consecutive spinner frames (whether 2, 3, 4, or more) with a SINGLE normalized version
	output = spinnerPattern.ReplaceAllString(output, "\r[⠃] Compiling...\n")

	return output
}

// getDefaultNormalizers returns the default set of normalizers
func GetDefaultNormalizers() []Normalizer {
	return []Normalizer{
		ColorNormalizer{},
		FoundryWarningsNormalizer{},
		LineClearArtifactNormalizer{},
		SpinnerNormalizer{},
		TimestampNormalizer{},
		VersionNormalizer{},
		TargetedGitCommitNormalizer{},
		TargetedHashNormalizer{},
		RepositoryHormalizer{},
		DebugNormalizer{},
	}
}

// LegacySolidityNormalizer handles bytecode differences in legacy Solidity versions
//
// Legacy Solidity versions (< 0.8.0) embed metadata differently than modern versions,
// causing bytecode to differ between environments even with the same compiler.
// This normalizer removes these differences for consistent testing across platforms.
//
// It normalizes:
// - bytecodeHash: Different due to metadata embedding
// - initCodeHash: Different due to constructor bytecode variations
// - Gas costs: Minor variations between environments
type LegacySolidityNormalizer struct{}

func (n LegacySolidityNormalizer) Normalize(output string) string {
	// Normalize bytecode hashes in JSON
	output = regexp.MustCompile(`"bytecodeHash":\s*"0x[a-fA-F0-9]{64}"`).ReplaceAllString(output, `"bytecodeHash": "0x<BYTECODE_HASH>"`)

	// Normalize init code hashes in JSON
	output = regexp.MustCompile(`"initCodeHash":\s*"0x[a-fA-F0-9]{64}"`).ReplaceAllString(output, `"initCodeHash": "0x<INIT_CODE_HASH>"`)

	// Normalize gas costs (matches patterns like "Gas: 616952")
	output = regexp.MustCompile(`Gas:\s*\d+`).ReplaceAllString(output, `Gas: <GAS_AMOUNT>`)

	return output
}

func Normalize(text string, normalizers []Normalizer) string {
	// Apply normalizers
	normalized := text
	for _, n := range normalizers {
		normalized = n.Normalize(normalized)
	}

	// Ensure consistent line endings
	normalized = strings.ReplaceAll(normalized, "\r\n", "\n")
	normalized = strings.TrimSpace(normalized)

	// Always ensure a trailing newline for consistency
	if normalized != "" {
		normalized = normalized + "\n"
	}
	return normalized
}
