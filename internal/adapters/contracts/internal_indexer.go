package contracts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// InternalIndexer discovers and indexes contracts and their artifacts
type InternalIndexer struct {
	projectRoot       string
	contracts         map[string]*domain.ContractInfo   // key: "path:contractName" or "contractName" if unique
	contractNames     map[string][]*domain.ContractInfo // key: contract name, value: all contracts with that name
	bytecodeHashIndex map[string]*domain.ContractInfo   // key: bytecodeHash -> ContractInfo
	mu                sync.RWMutex
}

// NewInternalIndexer creates a new contract indexer
func NewInternalIndexer(projectRoot string) *InternalIndexer {
	return &InternalIndexer{
		projectRoot:       projectRoot,
		contracts:         make(map[string]*domain.ContractInfo),
		contractNames:     make(map[string][]*domain.ContractInfo),
		bytecodeHashIndex: make(map[string]*domain.ContractInfo),
	}
}

// Index discovers all contracts and artifacts
func (i *InternalIndexer) Index() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	// Reset indexes
	i.contracts = make(map[string]*domain.ContractInfo)
	i.contractNames = make(map[string][]*domain.ContractInfo)
	i.bytecodeHashIndex = make(map[string]*domain.ContractInfo)

	// Find the 'out' directory
	outDir := filepath.Join(i.projectRoot, "out")
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		return fmt.Errorf("out directory not found. Run 'forge build' first")
	}

	// Walk through all artifact directories
	err := filepath.Walk(outDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a JSON file
		if filepath.Ext(path) != ".json" || info.IsDir() {
			return nil
		}

		// Skip build info files
		if strings.Contains(path, "build-info") {
			return nil
		}

		// Process the artifact
		return i.processArtifact(path)
	})

	return err
}

// processArtifact processes a single artifact file
func (i *InternalIndexer) processArtifact(artifactPath string) error {
	// Read artifact file
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		return err
	}

	// Parse basic artifact structure
	var artifact struct {
		ContractName string `json:"contractName"`
		SourceName   string `json:"sourceName"`
		Bytecode     struct {
			Object string `json:"object"`
		} `json:"bytecode"`
		DeployedBytecode struct {
			Object string `json:"object"`
		} `json:"deployedBytecode"`
		Metadata string `json:"metadata"`
	}
	
	if err := json.Unmarshal(data, &artifact); err != nil {
		return nil // Skip invalid artifacts
	}

	// Skip if no bytecode
	if artifact.Bytecode.Object == "" || artifact.Bytecode.Object == "0x" {
		return nil
	}

	// Create contract info
	info := &domain.ContractInfo{
		Name:         artifact.ContractName,
		Path:         artifact.SourceName,
		ArtifactPath: artifactPath,
	}

	// Determine if it's a library (simple heuristic)
	if strings.Contains(strings.ToLower(artifact.ContractName), "lib") || 
	   strings.Contains(artifact.SourceName, "/libraries/") ||
	   strings.Contains(artifact.SourceName, "/lib/") {
		info.IsLibrary = true
	}

	// Add to indexes
	// Full key for unique identification
	fullKey := fmt.Sprintf("%s:%s", info.Path, info.Name)
	i.contracts[fullKey] = info

	// Also index by name alone if unique
	if existingList, exists := i.contractNames[info.Name]; exists {
		// Multiple contracts with same name
		i.contractNames[info.Name] = append(existingList, info)
		// Remove simple key if it exists
		delete(i.contracts, info.Name)
	} else {
		// First contract with this name
		i.contractNames[info.Name] = []*domain.ContractInfo{info}
		// Also add simple key
		i.contracts[info.Name] = info
	}

	return nil
}

// GetContract retrieves a contract by key (name or path:name)
func (i *InternalIndexer) GetContract(key string) (*domain.ContractInfo, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if contract, exists := i.contracts[key]; exists {
		return contract, nil
	}

	return nil, fmt.Errorf("contract not found: %s", key)
}

// SearchContracts searches for contracts matching a pattern
func (i *InternalIndexer) SearchContracts(pattern string) []*domain.ContractInfo {
	i.mu.RLock()
	defer i.mu.RUnlock()

	var results []*domain.ContractInfo
	lowPattern := strings.ToLower(pattern)

	// Search through all contracts
	seen := make(map[string]bool)
	for _, info := range i.contracts {
		// Skip duplicates
		key := fmt.Sprintf("%s:%s", info.Path, info.Name)
		if seen[key] {
			continue
		}
		seen[key] = true

		// Match on name or path
		if strings.Contains(strings.ToLower(info.Name), lowPattern) ||
		   strings.Contains(strings.ToLower(info.Path), lowPattern) {
			results = append(results, info)
		}
	}

	return results
}

// GetScriptContracts returns all script contracts
func (i *InternalIndexer) GetScriptContracts() []*domain.ContractInfo {
	i.mu.RLock()
	defer i.mu.RUnlock()

	var scripts []*domain.ContractInfo
	seen := make(map[string]bool)

	for _, info := range i.contracts {
		// Skip duplicates
		key := fmt.Sprintf("%s:%s", info.Path, info.Name)
		if seen[key] {
			continue
		}
		seen[key] = true

		// Check if it's a script based on path
		if strings.HasPrefix(info.Path, "script/") || strings.Contains(info.Path, "/script/") {
			scripts = append(scripts, info)
		}
	}

	return scripts
}

// GetContractByArtifact retrieves a contract by its artifact path
func (i *InternalIndexer) GetContractByArtifact(artifactPath string) *domain.ContractInfo {
	i.mu.RLock()
	defer i.mu.RUnlock()

	for _, info := range i.contracts {
		if info.ArtifactPath == artifactPath {
			return info
		}
	}

	return nil
}