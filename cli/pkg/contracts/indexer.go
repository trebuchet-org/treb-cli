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

	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/forge"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

var (
	globalIndexer *Indexer
	indexerMutex  sync.Once
)

type ContractInfo = types.ContractInfo
type Artifact = types.Artifact
type QueryFilter = types.ContractQueryFilter

// Indexer discovers and indexes contracts and their artifacts
type Indexer struct {
	projectRoot       string
	contracts         map[string]*ContractInfo   // key: "path:contractName" or "contractName" if unique
	contractNames     map[string][]*ContractInfo // key: contract name, value: all contracts with that name
	bytecodeHashIndex map[string]*ContractInfo   // key: bytecodeHash -> ContractInfo
	mu                sync.RWMutex
}

// NewIndexer creates a new contract indexer (always includes libraries)
func NewIndexer(projectRoot string) *Indexer {
	return &Indexer{
		projectRoot:       projectRoot,
		contracts:         make(map[string]*ContractInfo),
		contractNames:     make(map[string][]*ContractInfo),
		bytecodeHashIndex: make(map[string]*ContractInfo),
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
		return fmt.Errorf("failed to build contracts: %w", err)
	}

	// First, index all .sol files
	if err := i.indexSolidityFiles(); err != nil {
		return fmt.Errorf("failed to index Solidity files: %w", err)
	}

	// Then, index all compilation artifacts
	if err := i.indexArtifacts(); err != nil {
		return fmt.Errorf("failed to index artifacts: %w", err)
	}

	return nil
}

// indexSolidityFiles discovers all .sol files and parses them for contract definitions
func (i *Indexer) indexSolidityFiles() error {
	// Collect all paths to process
	var paths []string

	// Always include src/ and script/
	srcPath := filepath.Join(i.projectRoot, "src")
	if _, err := os.Stat(srcPath); err == nil {
		paths = append(paths, srcPath)
	}

	scriptPath := filepath.Join(i.projectRoot, "script")
	if _, err := os.Stat(scriptPath); err == nil {
		paths = append(paths, scriptPath)
	}

	// Always include library paths from remappings
	paths = append(paths, i.getLibraryPaths()...)

	// Use a worker pool for parallel processing
	type result struct {
		contracts []*ContractInfo
		err       error
	}

	// Channel for file paths to process
	fileChan := make(chan string, 100)
	resultChan := make(chan result, 10)

	// Start workers
	var wg sync.WaitGroup
	numWorkers := 4
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range fileChan {
				contracts, err := i.parseContractsFromFile(filePath)
				resultChan <- result{contracts: contracts, err: err}
			}
		}()
	}

	// Start result collector
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Walk directories and send files to workers
	go func() {
		defer close(fileChan)
		for _, basePath := range paths {
			_ = filepath.WalkDir(basePath, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil // Skip errors
				}

				// Skip hidden directories and node_modules
				if d.IsDir() && (strings.HasPrefix(d.Name(), ".") || d.Name() == "node_modules") {
					return fs.SkipDir
				}

				// Process .sol files
				if !d.IsDir() && strings.HasSuffix(path, ".sol") {
					fileChan <- path
				}

				return nil
			})
		}
	}()

	// Collect results
	for res := range resultChan {
		if res.err != nil {
			// Log error but continue processing
			fmt.Printf("Warning: %v\n", res.err)
			continue
		}

		i.mu.Lock()
		for _, contract := range res.contracts {
			// Store by full path:name
			key := fmt.Sprintf("%s:%s", contract.Path, contract.Name)
			i.contracts[key] = contract

			// Store by name for lookup
			i.contractNames[contract.Name] = append(i.contractNames[contract.Name], contract)

			// If unique name, also store by name only
			if len(i.contractNames[contract.Name]) == 1 {
				i.contracts[contract.Name] = contract
			} else {
				// Remove simple name key if no longer unique
				delete(i.contracts, contract.Name)
			}
		}
		i.mu.Unlock()
	}

	return nil
}

// parseContractsFromFile parses a Solidity file and extracts contract definitions
func (i *Indexer) parseContractsFromFile(filePath string) ([]*ContractInfo, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Get relative path from project root
	relPath, err := filepath.Rel(i.projectRoot, filePath)
	if err != nil {
		relPath = filePath
	}

	// Parse contracts using regex
	var contracts []*ContractInfo

	// Regex to match contract/library/interface definitions
	// Handles abstract contracts and inheritance
	contractRegex := regexp.MustCompile(`(?m)^\s*(abstract\s+)?(contract|library|interface)\s+(\w+)`)

	// Regex to extract pragma version
	versionRegex := regexp.MustCompile(`pragma\s+solidity\s+([^;]+);`)

	// Extract version
	version := ""
	if matches := versionRegex.FindSubmatch(content); len(matches) > 1 {
		version = strings.TrimSpace(string(matches[1]))
	}

	// Find all contract definitions
	matches := contractRegex.FindAllSubmatch(content, -1)
	for _, match := range matches {
		isAbstract := len(match[1]) > 0
		contractType := string(match[2])
		contractName := string(match[3])

		contract := &ContractInfo{
			Name:        contractName,
			Path:        relPath,
			Version:     version,
			IsLibrary:   contractType == "library",
			IsInterface: contractType == "interface",
			IsAbstract:  isAbstract,
		}

		contracts = append(contracts, contract)
	}

	return contracts, nil
}

// indexArtifacts indexes all compilation artifacts and links them to contracts
func (i *Indexer) indexArtifacts() error {
	outPath := filepath.Join(i.projectRoot, "out")
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		return nil // No artifacts yet
	}

	return filepath.WalkDir(outPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if strings.HasPrefix(path, "out/build-info") {
			return nil
		}

		if strings.HasPrefix(path, "out/.treb-debug") {
			return nil
		}

		// Look for .json files that aren't .dbg.json
		if !d.IsDir() && strings.HasSuffix(path, ".json") && !strings.HasSuffix(path, ".dbg.json") {
			if err := i.processArtifact(path); err != nil {
				// Log error but continue processing
				fmt.Printf("Warning: failed to process artifact %s: %v\n", path, err)
			}
		}

		return nil
	})
}

// processArtifact processes a single artifact file
func (i *Indexer) processArtifact(artifactPath string) error {
	content, err := os.ReadFile(artifactPath)
	if err != nil {
		return err
	}

	var artifact Artifact
	if err := json.Unmarshal(content, &artifact); err != nil {
		return err
	}

	// Extract compilation target
	for sourcePath, contractName := range artifact.Metadata.Settings.CompilationTarget {
		// Find matching contract
		i.mu.Lock()

		// Try exact match first
		key := fmt.Sprintf("%s:%s", sourcePath, contractName)
		if contract, exists := i.contracts[key]; exists {
			relPath, _ := filepath.Rel(i.projectRoot, artifactPath)
			contract.ArtifactPath = relPath
			contract.Artifact = &artifact

			// Calculate and store bytecode hash
			if hash, err := contract.CalculateBytecodeHash(); err == nil {
				i.bytecodeHashIndex[hash] = contract
			}
			i.mu.Unlock()
			continue
		}

		// Try to find by contract name if path doesn't match exactly
		if contracts, exists := i.contractNames[contractName]; exists {
			for _, contract := range contracts {
				// Check if the source paths are related
				if strings.Contains(sourcePath, contract.Path) || strings.Contains(contract.Path, sourcePath) {
					relPath, _ := filepath.Rel(i.projectRoot, artifactPath)
					contract.ArtifactPath = relPath
					contract.Artifact = &artifact

					// Calculate and store bytecode hash
					if hash, err := contract.CalculateBytecodeHash(); err == nil {
						i.bytecodeHashIndex[hash] = contract
					}
					break
				}
			}
		}
		i.mu.Unlock()
	}

	return nil
}

// GetContract returns a contract by key (name or path:name)
func (i *Indexer) GetContract(key string) (*ContractInfo, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if contract, exists := i.contracts[key]; exists {
		return contract, nil
	}

	return nil, fmt.Errorf("contract not found: %s", key)
}

// GetContractByArtifact returns a contract by artifact name
// It handles both simple names (e.g., "Counter") and full artifact paths (e.g., "src/Counter.sol:Counter")
func (i *Indexer) GetContractByArtifact(artifact string) *ContractInfo {
	i.mu.RLock()
	defer i.mu.RUnlock()

	// First try exact match in contracts map (for full artifact paths)
	if contract, exists := i.contracts[artifact]; exists {
		return contract
	}

	var contractPath string
	var contractName string
	// If artifact contains ":", try to find by path:name format
	if strings.Contains(artifact, ":") {
		parts := strings.Split(artifact, ":")
		if len(parts) == 2 {
			contractPath = parts[0]
			contractName = parts[1]
		} else {
			return nil
		}
	} else {
		contractPath = ""
		contractName = artifact
	}

	// TODO: BIG PROBLEM HERE WE NEED TO FIX THIS
	contracts := i.contractNames[contractName]
	for _, contract := range contracts {
		if contract.Artifact != nil && (contractPath == "" || strings.Contains(contract.Path, contractPath)) {
			return contract
		}
	}

	return nil
}

// GetContractsByName returns all contracts with the given name (kept for compatibility)
func (i *Indexer) GetContractsByName(name string) []*ContractInfo {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.contractNames[name]
}

// GetAllContracts returns all discovered contracts
func (i *Indexer) GetAllContracts() []*ContractInfo {
	i.mu.RLock()
	defer i.mu.RUnlock()

	var contracts []*ContractInfo
	seen := make(map[string]bool)

	for _, contract := range i.contracts {
		key := fmt.Sprintf("%s:%s", contract.Path, contract.Name)
		if !seen[key] {
			seen[key] = true
			contracts = append(contracts, contract)
		}
	}

	return contracts
}

// SearchContracts searches for contracts matching a pattern
func (i *Indexer) SearchContracts(pattern string) []*ContractInfo {
	i.mu.RLock()
	defer i.mu.RUnlock()

	pattern = strings.ToLower(pattern)
	var results []*ContractInfo
	seen := make(map[string]bool)

	for _, contract := range i.contracts {
		if strings.Contains(strings.ToLower(contract.Name), pattern) || strings.Contains(strings.ToLower(contract.Path), pattern) {
			key := fmt.Sprintf("%s:%s", contract.Path, contract.Name)
			if !seen[key] {
				seen[key] = true
				results = append(results, contract)
			}
		}
	}

	return results
}

// GetProxyContracts returns all contracts that appear to be proxy contracts
func (i *Indexer) GetProxyContracts() []*ContractInfo {
	i.mu.RLock()
	defer i.mu.RUnlock()

	var proxies []*ContractInfo
	seen := make(map[string]bool)

	for _, contract := range i.contracts {
		if strings.Contains(contract.Name, "Proxy") && !contract.IsLibrary && !contract.IsInterface {
			key := fmt.Sprintf("%s:%s", contract.Path, contract.Name)
			if !seen[key] {
				seen[key] = true
				proxies = append(proxies, contract)
			}
		}
	}

	return proxies
}

// ResolveContractKey returns the appropriate key for a contract
// If the name is unique, returns just the name
// Otherwise returns "path:name"
func (i *Indexer) ResolveContractKey(contract *ContractInfo) string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if contracts := i.contractNames[contract.Name]; len(contracts) == 1 {
		return contract.Name
	}

	return fmt.Sprintf("%s:%s", contract.Path, contract.Name)
}

// QueryContracts returns contracts filtered by the provided filter
func (i *Indexer) QueryContracts(filter QueryFilter) []*ContractInfo {
	i.mu.RLock()
	defer i.mu.RUnlock()

	var results []*ContractInfo
	seen := make(map[string]bool)

	// Compile regex patterns if provided
	var nameRegex, pathRegex *regexp.Regexp
	if filter.NamePattern != "" {
		nameRegex = regexp.MustCompile(filter.NamePattern)
	}
	if filter.PathPattern != "" {
		pathRegex = regexp.MustCompile(filter.PathPattern)
	}

	for _, contract := range i.contracts {
		// Apply filtering
		if !filter.IncludeLibraries && contract.IsLibrary {
			continue
		}
		if !filter.IncludeInterface && contract.IsInterface {
			continue
		}
		if !filter.IncludeAbstract && contract.IsAbstract {
			continue
		}

		// Apply pattern matching
		if nameRegex != nil && !nameRegex.MatchString(contract.Name) {
			continue
		}
		if pathRegex != nil && !pathRegex.MatchString(contract.Path) {
			continue
		}

		// Deduplicate by path:name
		key := fmt.Sprintf("%s:%s", contract.Path, contract.Name)
		if !seen[key] {
			seen[key] = true
			results = append(results, contract)
		}
	}

	return results
}

// FindContractByName finds a contract by exact name match, using filter
func (i *Indexer) FindContractByName(name string, filter QueryFilter) []*ContractInfo {
	filter.NamePattern = regexp.QuoteMeta(name)
	return i.QueryContracts(filter)
}

// GetDeployableContracts returns all deployable contracts (no libs, interfaces, or abstract)
func (i *Indexer) GetDeployableContracts() []*ContractInfo {
	return i.QueryContracts(types.DefaultContractsFilter())
}

// GetProxyContractsFiltered returns proxy contracts using the filter
func (i *Indexer) GetProxyContractsFiltered(filter QueryFilter) []*ContractInfo {
	// Add proxy pattern to name filter
	if filter.NamePattern == "" {
		filter.NamePattern = ".*[Pp]roxy.*"
	} else {
		filter.NamePattern = "(" + filter.NamePattern + ").*[Pp]roxy.*"
	}
	return i.QueryContracts(filter)
}

// GetGlobalIndexer returns a singleton indexer instance for the given project root
// It will be initialized once and reused across the application
func GetGlobalIndexer(projectRoot string) (*Indexer, error) {
	var err error
	indexerMutex.Do(func() {
		globalIndexer = NewIndexer(projectRoot)
		err = globalIndexer.Index()
	})
	return globalIndexer, err
}

// ResetGlobalIndexer resets the global indexer (useful for testing)
func ResetGlobalIndexer() {
	globalIndexer = nil
	indexerMutex = sync.Once{}
}

// GetContractByBytecodeHash returns a contract by its btecode hash
func (i *Indexer) GetContractByBytecodeHash(hash string) *ContractInfo {
	i.mu.RLock()
	defer i.mu.RUnlock()

	// Normalize hash format
	hash = strings.ToLower(strings.TrimPrefix(hash, "0x"))
	if len(hash) != 64 {
		return nil
	}
	hash = "0x" + hash

	return i.bytecodeHashIndex[hash]
}
