package contracts

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/trebuchet-org/treb-cli/internal/adapters/config"
	"github.com/trebuchet-org/treb-cli/internal/adapters/forge"
	"github.com/trebuchet-org/treb-cli/internal/domain"
)

var (
	globalIndexer *Indexer
	indexerMutex  sync.Once
)

// Indexer discovers and indexes contracts and their artifacts
type Indexer struct {
	projectRoot       string
	contracts         map[string]*domain.ContractInfo   // key: "path:contractName" or "contractName" if unique
	contractNames     map[string][]*domain.ContractInfo // key: contract name, value: all contracts with that name
	bytecodeHashIndex map[string]*domain.ContractInfo   // key: bytecodeHash -> ContractInfo
	mu                sync.RWMutex
}

// NewIndexer creates a new contract indexer (always includes libraries)
func NewIndexer(projectRoot string) *Indexer {
	return &Indexer{
		projectRoot:       projectRoot,
		contracts:         make(map[string]*domain.ContractInfo),
		contractNames:     make(map[string][]*domain.ContractInfo),
		bytecodeHashIndex: make(map[string]*domain.ContractInfo),
	}
}

// getLibraryPaths returns all library paths to index based on remappings
func (i *Indexer) getLibraryPaths() []string {
	var paths []string
	seen := make(map[string]bool)

	// Load foundry config to get remappings
	foundryManager := config.NewFoundryManager(i.projectRoot)
	remappings := foundryManager.GetRemappings()
	// Parse remappings to find library paths
	for _, path := range remappings {
		// Convert to absolute path
		absPath := filepath.Join(i.projectRoot, path)

		// Remove trailing slash if present
		absPath = strings.TrimSuffix(absPath, "/")

		// Check if path exists
		if info, err := os.Stat(absPath); err == nil && info.IsDir() {
			// Avoid duplicates
			if !seen[absPath] {
				seen[absPath] = true
				paths = append(paths, absPath)
			}
		}
	}

	normalizedPaths := make([]string, 0)
	for _, path := range paths {
		skip := false
		for i, normalizedPath := range normalizedPaths {
			if strings.HasPrefix(path, normalizedPath) {
				skip = true
			} else if strings.HasPrefix(normalizedPath, path) {
				normalizedPaths[i] = path
				skip = true
			}

			if skip {
				break
			}
		}
		if !skip {
			normalizedPaths = append(normalizedPaths, path)
		}
	}

	return normalizedPaths
}

// Index discovers all contracts and artifacts
func (i *Indexer) Index() error {
	// Trigger a forge build to ensure artifacts are up to date
	forgeExecutor := forge.NewForge(i.projectRoot)
	if err := forgeExecutor.Build(); err != nil {
		// Show the build output and fail
		return fmt.Errorf("forge build failed: %w", err)
	}

	// Clear existing indices
	i.mu.Lock()
	i.contracts = make(map[string]*domain.ContractInfo)
	i.contractNames = make(map[string][]*domain.ContractInfo)
	i.bytecodeHashIndex = make(map[string]*domain.ContractInfo)
	i.mu.Unlock()

	// Index all contracts
	srcPath := filepath.Join(i.projectRoot, "src")
	if err := i.indexDirectory(srcPath); err != nil {
		return fmt.Errorf("failed to index src: %w", err)
	}

	// Also index libraries
	for _, libPath := range i.getLibraryPaths() {
		if err := i.indexDirectory(libPath); err != nil {
			// Don't fail on library indexing errors, just log them
			fmt.Fprintf(os.Stderr, "Warning: failed to index library %s: %v\n", libPath, err)
		}
	}

	return nil
}

// indexDirectory recursively indexes contracts in a directory
func (i *Indexer) indexDirectory(dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Solidity files
		if d.IsDir() || !strings.HasSuffix(path, ".sol") {
			return nil
		}

		// Skip test files
		if strings.Contains(path, "/test/") || strings.HasSuffix(path, ".t.sol") {
			return nil
		}

		// Index this file
		return i.indexFile(path)
	})
}

// indexFile indexes all contracts in a Solidity file
func (i *Indexer) indexFile(path string) error {
	// Read the file
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Extract contract names using regex
	// This regex matches: contract, library, interface, abstract contract
	contractRegex := regexp.MustCompile(`(?m)^\s*(?:abstract\s+)?(?:contract|library|interface)\s+(\w+)`)
	matches := contractRegex.FindAllSubmatch(content, -1)

	relPath, _ := filepath.Rel(i.projectRoot, path)

	for _, match := range matches {
		contractName := string(match[1])
		
		// Check if it's a library
		isLibrary := strings.Contains(string(match[0]), "library")
		isInterface := strings.Contains(string(match[0]), "interface")
		isAbstract := strings.Contains(string(match[0]), "abstract")

		// Create contract info
		info := &domain.ContractInfo{
			Name:        contractName,
			Path:        relPath,
			IsLibrary:   isLibrary,
			IsInterface: isInterface,
			IsAbstract:  isAbstract,
		}

		// Try to load the artifact to get more details
		artifactPath := i.findArtifact(relPath, contractName)
		if artifactPath != "" {
			info.ArtifactPath = artifactPath
			// Could load artifact here to get bytecode hash, version etc
		}

		i.mu.Lock()
		// Add to indices
		key := fmt.Sprintf("%s:%s", relPath, contractName)
		i.contracts[key] = info
		
		// Also index by just contract name if unique
		if existing, exists := i.contractNames[contractName]; !exists || len(existing) == 0 {
			i.contracts[contractName] = info
		}
		
		i.contractNames[contractName] = append(i.contractNames[contractName], info)
		i.mu.Unlock()
	}

	return nil
}

// findArtifact finds the artifact path for a contract
func (i *Indexer) findArtifact(solPath, contractName string) string {
	// Standard artifact path
	artifactBase := strings.TrimSuffix(solPath, ".sol")
	artifactPath := filepath.Join(i.projectRoot, "out", artifactBase+".sol", contractName+".json")
	
	if _, err := os.Stat(artifactPath); err == nil {
		relPath, _ := filepath.Rel(i.projectRoot, artifactPath)
		return relPath
	}

	return ""
}

// GetContract finds a contract by key (path:name or just name)
func (i *Indexer) GetContract(key string) *domain.ContractInfo {
	i.mu.RLock()
	defer i.mu.RUnlock()
	
	return i.contracts[key]
}

// SearchContracts searches for contracts by pattern
func (i *Indexer) SearchContracts(pattern string) []*domain.ContractInfo {
	i.mu.RLock()
	defer i.mu.RUnlock()

	var results []*domain.ContractInfo
	seen := make(map[*domain.ContractInfo]bool)

	// Convert pattern to regex
	regex, err := regexp.Compile(pattern)
	if err != nil {
		// If invalid regex, treat as literal string
		pattern = strings.ToLower(pattern)
		for _, info := range i.contracts {
			if strings.Contains(strings.ToLower(info.Name), pattern) && !seen[info] {
				results = append(results, info)
				seen[info] = true
			}
		}
		return results
	}

	// Search by regex
	for _, info := range i.contracts {
		if regex.MatchString(info.Name) && !seen[info] {
			results = append(results, info)
			seen[info] = true
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

// LoadArtifact loads and parses a contract artifact
func (i *Indexer) LoadArtifact(artifactPath string) (*domain.Artifact, error) {
	fullPath := filepath.Join(i.projectRoot, artifactPath)
	
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read artifact: %w", err)
	}

	var artifact domain.Artifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		return nil, fmt.Errorf("failed to parse artifact: %w", err)
	}

	return &artifact, nil
}

// GetSingleton returns the global singleton indexer
func GetSingleton(projectRoot string) *Indexer {
	indexerMutex.Do(func() {
		globalIndexer = NewIndexer(projectRoot)
	})
	return globalIndexer
}