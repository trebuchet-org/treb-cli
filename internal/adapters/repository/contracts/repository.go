package contracts

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// FoundryProfile represents a Foundry build profile name (maps to namespace).
type FoundryProfile string

// Repository discovers and indexes contracts and their artifacts
type Repository struct {
	projectRoot       string
	profile           FoundryProfile
	contracts         map[string]*models.Contract   // key: "path:contractName" or "contractName" if unique
	contractNames     map[string][]*models.Contract // key: contract name, value: all contracts with that name
	bytecodeHashIndex map[string]*models.Contract   // key: bytecodeHash -> Contract
	log               *slog.Logger
	mu                sync.RWMutex
	indexed           bool
	built             bool // tracks if we've run forge build this session
}

// NewRepository creates a new contract indexer
func NewRepository(projectRoot string, profile FoundryProfile, log *slog.Logger) *Repository {
	return &Repository{
		projectRoot:       projectRoot,
		profile:           profile,
		log:               log,
		contracts:         make(map[string]*models.Contract),
		contractNames:     make(map[string][]*models.Contract),
		bytecodeHashIndex: make(map[string]*models.Contract),
	}
}

// Index discovers all contracts and artifacts by scanning existing build output.
// It does NOT run forge build — see buildAndReindex for on-demand compilation.
func (i *Repository) Index() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.indexed {
		return nil
	}

	return i.indexArtifacts()
}

// indexArtifacts scans the out/ directory for compiled artifacts.
// Must be called with i.mu held.
func (i *Repository) indexArtifacts() error {
	start := time.Now()
	i.log.Debug("starting contract indexing")

	// Reset indexes
	i.contracts = make(map[string]*models.Contract)
	i.contractNames = make(map[string][]*models.Contract)
	i.bytecodeHashIndex = make(map[string]*models.Contract)

	outDir := filepath.Join(i.projectRoot, "out")
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		// No out/ directory yet — mark indexed with empty maps so build-on-miss can trigger later
		i.indexed = true
		i.log.Debug("out directory does not exist, indexed with zero contracts")
		return nil
	}

	err := filepath.Walk(outDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) != ".json" || info.IsDir() {
			return nil
		}

		if strings.Contains(path, "build-info") {
			return nil
		}

		if procErr := i.processArtifact(path); procErr != nil {
			return procErr
		}
		return nil
	})

	if err == nil {
		i.indexed = true
	}
	duration := time.Since(start)
	i.log.Debug("contract indexing completed", "duration", duration, "contracts_found", len(i.contracts))
	return err
}

// runForgeBuild runs forge build command with the configured FOUNDRY_PROFILE.
func (i *Repository) runForgeBuild() error {
	start := time.Now()
	i.log.Debug("running forge build for contract indexing", "dir", i.projectRoot, "profile", i.profile)

	cmd := exec.Command("forge", "build")
	cmd.Dir = i.projectRoot
	if i.profile != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("FOUNDRY_PROFILE=%s", i.profile))
	}

	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	if err != nil {
		i.log.Error("forge build failed", "error", err, "duration", duration)
		return fmt.Errorf("forge build failed: %w\nOutput: %s", err, string(output))
	}

	i.log.Debug("forge build completed", "duration", duration)
	return nil
}

// buildAndReindex runs forge build (once per session) and re-scans artifacts.
// Must be called with i.mu held for writing.
func (i *Repository) buildAndReindex() error {
	if i.built {
		return nil
	}
	i.built = true

	if err := i.runForgeBuild(); err != nil {
		return err
	}

	i.indexed = false
	return i.indexArtifacts()
}

// processArtifact processes a single artifact file
func (i *Repository) processArtifact(artifactPath string) error {
	// Read artifact file
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		return err
	}

	// Parse Foundry artifact structure
	var artifact models.Artifact
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

	// Make artifact path relative to project root
	relArtifactPath, _ := filepath.Rel(i.projectRoot, artifactPath)

	// Create contract info
	info := &models.Contract{
		Name:         contractName,
		Path:         sourceName,
		ArtifactPath: relArtifactPath,
		Artifact:     &artifact,
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
	} else {
		// First contract with this name
		i.contractNames[info.Name] = []*models.Contract{info}
	}

	return nil
}

// GetContract retrieves a contract by key (name or path:name)
func (i *Repository) GetContract(ctx context.Context, key string) (*models.Contract, error) {
	if err := i.Index(); err != nil {
		return nil, err
	}
	i.mu.RLock()
	if contract, exists := i.contracts[key]; exists {
		i.mu.RUnlock()
		return contract, nil
	}
	i.mu.RUnlock()

	// Not found — try building and re-indexing
	i.mu.Lock()
	if err := i.buildAndReindex(); err != nil {
		i.mu.Unlock()
		return nil, err
	}
	i.mu.Unlock()

	i.mu.RLock()
	defer i.mu.RUnlock()
	if contract, exists := i.contracts[key]; exists {
		return contract, nil
	}
	return nil, fmt.Errorf("contract not found: %s", key)
}

// SearchContracts searches for contracts matching a pattern, triggering a
// forge build if no results are found (build-on-miss). Use this during
// pre-execution phases where compilation may not have happened yet.
func (i *Repository) SearchContracts(ctx context.Context, query domain.ContractQuery) ([]*models.Contract, error) {
	if err := i.Index(); err != nil {
		return nil, err
	}

	results := i.searchContractsLocked(query)

	// No results — try building and re-indexing
	if len(results) == 0 {
		i.mu.Lock()
		if err := i.buildAndReindex(); err != nil {
			i.mu.Unlock()
			return nil, err
		}
		i.mu.Unlock()
		results = i.searchContractsLocked(query)
	}

	return results, nil
}

// FindContracts searches the existing artifact index without triggering a build.
// Use this for best-effort lookups where compilation has already happened
// (e.g., ABI resolution during output rendering).
func (i *Repository) FindContracts(ctx context.Context, query domain.ContractQuery) ([]*models.Contract, error) {
	if err := i.Index(); err != nil {
		return nil, err
	}

	return i.searchContractsLocked(query), nil
}

func (i *Repository) searchContractsLocked(query domain.ContractQuery) []*models.Contract {
	i.mu.RLock()
	defer i.mu.RUnlock()

	var results []*models.Contract
	var artifactQuery = ""
	var pathRegex *regexp.Regexp
	if query.Query != nil {
		artifactQuery = strings.ToLower(*query.Query)
	}
	if query.PathPattern != nil {
		pathRegex = regexp.MustCompile(*query.PathPattern)
	}

	for key, contract := range i.contracts {
		if artifactQuery != "" {
			if !strings.Contains(strings.ToLower(key), artifactQuery) {
				continue
			}
		}
		if pathRegex != nil {
			if !pathRegex.MatchString(contract.Path) {
				continue
			}
		}
		results = append(results, contract)
	}

	return results
}

// GetContractByArtifact finds a contract by its artifact path.
// No build-on-miss: this is called during hydration after forge script
// already compiled everything. If the contract isn't indexed, rebuilding won't help.
func (i *Repository) GetContractByArtifact(ctx context.Context, artifact string) (*models.Contract, error) {
	if err := i.Index(); err != nil {
		return nil, err
	}

	return i.getContractByArtifactLocked(artifact), nil
}

func (i *Repository) getContractByArtifactLocked(artifact string) *models.Contract {
	i.mu.RLock()
	defer i.mu.RUnlock()

	for _, contract := range i.contracts {
		fullArtifact := fmt.Sprintf("%s:%s", contract.Path, contract.Name)
		if fullArtifact == artifact || contract.Name == artifact {
			return contract
		}
	}
	return nil
}

// GetScriptContracts returns all script contracts
func (i *Repository) GetScriptContracts() ([]*models.Contract, error) {
	if err := i.Index(); err != nil {
		return nil, err
	}
	i.mu.RLock()
	defer i.mu.RUnlock()

	var scripts []*models.Contract
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

	return scripts, nil
}

// GetAllContracts returns all indexed contracts (for debugging)
func (i *Repository) GetAllContracts() (map[string]*models.Contract, error) {
	if err := i.Index(); err != nil {
		return nil, err
	}
	i.mu.RLock()
	defer i.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]*models.Contract)
	maps.Copy(result, i.contracts)
	return result, nil
}

// Ensure the adapter implements both interfaces
var _ usecase.ContractRepository = (*Repository)(nil)
