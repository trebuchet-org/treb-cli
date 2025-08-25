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
	output = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z?`).ReplaceAllString(output, "<TIMESTAMP>")

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

// HashNormalizer replaces transaction hashes
type HashNormalizer struct{}

func (n HashNormalizer) Normalize(output string) string {
	output = regexp.MustCompile(`Tx: 0x[a-fA-F0-9]{64}`).ReplaceAllString(output, "Tx: 0x<HASH>")
	output = regexp.MustCompile(`Hash: 0x[a-fA-F0-9]{64}`).ReplaceAllString(output, "Hash: 0x<HASH>")
	output = regexp.MustCompile(`tx 0x[a-fA-F0-9]{64}`).ReplaceAllString(output, "tx 0x<HASH>")
	output = regexp.MustCompile(`transaction 0x[a-fA-F0-9]{64}`).ReplaceAllString(output, "transaction 0x<HASH>")
	return output
}

// ColorNormalizer removes ANSI color codes
type ColorNormalizer struct{}

func (n ColorNormalizer) Normalize(output string) string {
	// Remove ANSI escape sequences (color codes)
	return regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(output, "")
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

	// Git commit in version strings
	output = regexp.MustCompile(`-g[a-f0-9]{7,}`).ReplaceAllString(output, "-g<COMMIT>")

	return output
}

// getDefaultNormalizers returns the default set of normalizers
func GetDefaultNormalizers() []Normalizer {
	return []Normalizer{
		ColorNormalizer{},
		TimestampNormalizer{},
		VersionNormalizer{},
		TargetedGitCommitNormalizer{},
		TargetedHashNormalizer{},
		RepositoryHormalizer{},
		DebugNormalizer{},
		// AddressNormalizer{}, // We don't normalize addresses as they should be deterministic
		// PathNormalizer{},    // Often we want to see actual paths
		// BlockNumberNormalizer{},
		// GasNormalizer{},
	}
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
