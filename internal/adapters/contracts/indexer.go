package contracts

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// Indexer discovers and indexes contracts and their artifacts
type Indexer struct {
	projectRoot       string
	contracts         map[string]*domain.ContractInfo   // key: "path:contractName" or "contractName" if unique
	contractNames     map[string][]*domain.ContractInfo // key: contract name, value: all contracts with that name
	bytecodeHashIndex map[string]*domain.ContractInfo   // key: bytecodeHash -> ContractInfo
	mu                sync.RWMutex
}

// NewIndexer creates a new contract indexer
func NewIndexer(projectRoot string) *Indexer {
	return &Indexer{
		projectRoot:       projectRoot,
		contracts:         make(map[string]*domain.ContractInfo),
		contractNames:     make(map[string][]*domain.ContractInfo),
		bytecodeHashIndex: make(map[string]*domain.ContractInfo),
	}
}


// Index discovers all contracts and artifacts
func (i *Indexer) Index() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	// Reset indexes
	i.contracts = make(map[string]*domain.ContractInfo)
	i.contractNames = make(map[string][]*domain.ContractInfo)
	i.bytecodeHashIndex = make(map[string]*domain.ContractInfo)

	// Find the 'out' directory
	outDir := filepath.Join(i.projectRoot, "out")
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		// Run forge build to create artifacts
		if err := i.runForgeBuild(); err != nil {
			return fmt.Errorf("failed to build contracts: %w", err)
		}
		// Check again
		if _, err := os.Stat(outDir); os.IsNotExist(err) {
			return fmt.Errorf("out directory not found after forge build")
		}
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
func (i *Indexer) processArtifact(artifactPath string) error {
	// Read artifact file
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		return err
	}

	// Parse Foundry artifact structure
	var artifact struct {
		Bytecode struct {
			Object string `json:"object"`
		} `json:"bytecode"`
		DeployedBytecode struct {
			Object string `json:"object"`
		} `json:"deployedBytecode"`
		Metadata struct {
			Settings struct {
				CompilationTarget map[string]string `json:"compilationTarget"`
			} `json:"settings"`
		} `json:"metadata"`
	}
	
	if err := json.Unmarshal(data, &artifact); err != nil {
		// Skip invalid artifacts
		return nil
	}

	// Skip if no bytecode
	if artifact.Bytecode.Object == "" || artifact.Bytecode.Object == "0x" {
		return nil
	}

	// Extract contract name and source from compilation target
	var contractName, sourceName string
	for source, contract := range artifact.Metadata.Settings.CompilationTarget {
		sourceName = source
		contractName = contract
		break // There should only be one entry
	}

	if contractName == "" || sourceName == "" {
		return nil // Skip if we can't determine the contract
	}

	// Create contract info
	info := &domain.ContractInfo{
		Name:         contractName,
		Path:         sourceName,
		ArtifactPath: artifactPath,
	}

	// Determine if it's a library (simple heuristic)
	if strings.Contains(strings.ToLower(contractName), "lib") || 
	   strings.Contains(sourceName, "/libraries/") ||
	   strings.Contains(sourceName, "/lib/") {
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
func (i *Indexer) GetContract(key string) (*domain.ContractInfo, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if contract, exists := i.contracts[key]; exists {
		return contract, nil
	}

	return nil, fmt.Errorf("contract not found: %s", key)
}

// SearchContracts searches for contracts matching a pattern
func (i *Indexer) SearchContracts(pattern string) []*domain.ContractInfo {
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

// GetContractByArtifact finds a contract by its artifact path
func (i *Indexer) GetContractByArtifact(artifactPath string) *domain.ContractInfo {
	i.mu.RLock()
	defer i.mu.RUnlock()

	for _, info := range i.contracts {
		if info.ArtifactPath == artifactPath {
			return info
		}
	}
	return nil
}

// GetScriptContracts returns all script contracts
func (i *Indexer) GetScriptContracts() []*domain.ContractInfo {
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

// GetAllContracts returns all indexed contracts (for debugging)
func (i *Indexer) GetAllContracts() map[string]*domain.ContractInfo {
	i.mu.RLock()
	defer i.mu.RUnlock()
	
	// Return a copy to avoid race conditions
	result := make(map[string]*domain.ContractInfo)
	for k, v := range i.contracts {
		result[k] = v
	}
	return result
}

// runForgeBuild runs forge build command
func (i *Indexer) runForgeBuild() error {
	cmd := exec.Command("forge", "build")
	cmd.Dir = i.projectRoot
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("forge build failed: %w\nOutput: %s", err, string(output))
	}
	
	return nil
}

// IndexerAdapter adapts the internal indexer to the ContractIndexer interface
type IndexerAdapter struct {
	indexer *Indexer
}

// NewIndexerAdapter creates a new contract indexer adapter
func NewIndexerAdapter(cfg *config.RuntimeConfig) (*IndexerAdapter, error) {
	indexer := NewIndexer(cfg.ProjectRoot)
	
	// Index contracts
	if err := indexer.Index(); err != nil {
		return nil, fmt.Errorf("failed to index contracts: %w", err)
	}

	return &IndexerAdapter{
		indexer: indexer,
	}, nil
}

// GetContract retrieves a contract by key
func (a *IndexerAdapter) GetContract(ctx context.Context, key string) (*domain.ContractInfo, error) {
	return a.indexer.GetContract(key)
}

// SearchContracts searches for contracts matching a pattern
func (a *IndexerAdapter) SearchContracts(ctx context.Context, pattern string) []*domain.ContractInfo {
	return a.indexer.SearchContracts(pattern)
}

// GetContractByArtifact retrieves a contract by its artifact path
func (a *IndexerAdapter) GetContractByArtifact(ctx context.Context, artifact string) *domain.ContractInfo {
	return a.indexer.GetContractByArtifact(artifact)
}

// RefreshIndex refreshes the contract index
func (a *IndexerAdapter) RefreshIndex(ctx context.Context) error {
	return a.indexer.Index()
}