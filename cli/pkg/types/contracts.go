package types

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/sha3"
)

// ContractInfo represents information about a discovered contract
type ContractInfo struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	ArtifactPath string    `json:"artifactPath,omitempty"`
	Version      string    `json:"version,omitempty"`
	IsLibrary    bool      `json:"isLibrary"`
	IsInterface  bool      `json:"isInterface"`
	IsAbstract   bool      `json:"isAbstract"`
	Artifact     *Artifact `json:"artifact,omitempty"`
}

// ArtifactMetadata represents the metadata section of a Foundry artifact
type ArtifactMetadata struct {
	Compiler struct {
		Version string `json:"version"`
	} `json:"compiler"`
	Language string `json:"language"`
	Output   struct {
		ABI      json.RawMessage `json:"abi"`
		DevDoc   json.RawMessage `json:"devdoc"`
		UserDoc  json.RawMessage `json:"userdoc"`
		Metadata string          `json:"metadata"`
	} `json:"output"`
	Settings struct {
		CompilationTarget map[string]string `json:"compilationTarget"`
	} `json:"settings"`
}

// BytecodeObject represents the bytecode section of an artifact
type BytecodeObject struct {
	Object         string                          `json:"object"`
	LinkReferences map[string]map[string][]LinkRef `json:"linkReferences"`
	SourceMap      string                          `json:"sourceMap"`
}

// LinkRef represents a library link reference
type LinkRef struct {
	Start  int `json:"start"`
	Length int `json:"length"`
}

// Artifact represents a Foundry compilation artifact
type Artifact struct {
	ABI               json.RawMessage   `json:"abi"`
	Bytecode          BytecodeObject    `json:"bytecode"`
	DeployedBytecode  BytecodeObject    `json:"deployedBytecode"`
	MethodIdentifiers map[string]string `json:"methodIdentifiers"`
	RawMetadata       string            `json:"rawMetadata"`
	Metadata          ArtifactMetadata  `json:"metadata"`
}

// QueryFilter defines filtering options for contract queries
type ContractQueryFilter struct {
	IncludeLibraries bool
	IncludeAbstract  bool
	IncludeInterface bool
	NamePattern      string // regex pattern for name matching
	PathPattern      string // regex pattern for path matching
}

// DefaultContractsFilter returns a filter that includes only deployable contracts (no libs, interfaces, or abstract)
func DefaultContractsFilter() ContractQueryFilter {
	return ContractQueryFilter{
		IncludeLibraries: false,
		IncludeAbstract:  false,
		IncludeInterface: false,
	}
}

// ProjectContractsFilter returns a filter that includes only deployable contracts (no libs, interfaces, or abstract)
func ProjectContractsFilter() ContractQueryFilter {
	return ContractQueryFilter{
		IncludeLibraries: false,
		IncludeAbstract:  false,
		IncludeInterface: false,
		PathPattern:      "^src/.*$",
	}
}

// AllFilter returns a filter that includes everything
func AllContractsFilter() ContractQueryFilter {
	return ContractQueryFilter{
		IncludeLibraries: true,
		IncludeAbstract:  true,
		IncludeInterface: true,
	}
}

// ScriptContractFilter returns a filter that includes only deployable contracts (no libs, interfaces, or abstract)
func ScriptContractFilter() ContractQueryFilter {
	return ContractQueryFilter{
		IncludeLibraries: false,
		IncludeAbstract:  false,
		IncludeInterface: false,
		PathPattern:      "^script/.*$",
	}
}

// LibraryRequirement represents a library dependency
type LibraryRequirement struct {
	Path string
	Name string
}

// GetArtifactPath returns the artifact path for this contract
func (c *ContractInfo) GetArtifactPath() string {
	return c.ArtifactPath
}

func (c *ContractInfo) SourcePreview() string {
	content, err := os.ReadFile(c.Path)
	if err != nil {
		return fmt.Sprintf("Error reading file: %v", err)
	}
	return string(content)
}

// calculateBytecodeHash computes the keccak256 hash of the contract's bytecode
func (c *ContractInfo) CalculateBytecodeHash() (string, error) {
	if c.Artifact == nil {
		return "", fmt.Errorf("no artifact found for contract %s", c.Name)
	}

	// Get the creation bytecode
	bytecode := c.Artifact.Bytecode.Object
	if bytecode == "" || bytecode == "0x" {
		return "", fmt.Errorf("no bytecode found for contract %s", c.Name)
	}

	// Remove 0x prefix if present
	bytecode = strings.TrimPrefix(bytecode, "0x")

	// Decode hex string to bytes
	bytecodeBytes, err := hex.DecodeString(bytecode)
	if err != nil {
		return "", fmt.Errorf("failed to decode bytecode: %w", err)
	}

	// Calculate keccak256 hash
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(bytecodeBytes)
	hash := hasher.Sum(nil)

	// Return as hex string with 0x prefix
	return "0x" + hex.EncodeToString(hash), nil
}

// GetRequiredLibraries returns the libraries required by this contract
func (c *ContractInfo) GetRequiredLibraries() []LibraryRequirement {
	if c.Artifact == nil {
		return nil
	}

	var libs []LibraryRequirement
	seen := make(map[string]bool)

	// Check bytecode link references
	for path, libMap := range c.Artifact.Bytecode.LinkReferences {
		for libName := range libMap {
			key := fmt.Sprintf("%s:%s", path, libName)
			if !seen[key] {
				seen[key] = true
				libs = append(libs, LibraryRequirement{
					Path: path,
					Name: libName,
				})
			}
		}
	}

	// Also check deployed bytecode link references
	for path, libMap := range c.Artifact.DeployedBytecode.LinkReferences {
		for libName := range libMap {
			key := fmt.Sprintf("%s:%s", path, libName)
			if !seen[key] {
				seen[key] = true
				libs = append(libs, LibraryRequirement{
					Path: path,
					Name: libName,
				})
			}
		}
	}

	return libs
}
