package contracts

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// ScriptAnalysisResult represents the discovered contract information from a script
type ScriptAnalysisResult struct {
	ContractName string
	ContractPath string // Full path like ./src/Contract.sol:Contract
	SourceHash   string
}

// AnalyzeDeployScript analyzes a deployment script to determine the deployed contract
func AnalyzeDeployScript(scriptPath string) (*ScriptAnalysisResult, error) {
	// Read the script file
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read script file: %w", err)
	}

	scriptContent := string(content)

	// Find type(X).creationCode patterns
	// This regex accounts for whitespace variations
	creationCodeRegex := regexp.MustCompile(`type\s*\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*\)\s*\.creationCode`)
	matches := creationCodeRegex.FindAllStringSubmatch(scriptContent, -1)

	if len(matches) == 0 {
		return nil, fmt.Errorf("no type(X).creationCode pattern found in script")
	}

	// Use the first match (or could be smarter about picking the most likely one)
	contractName := matches[0][1]

	// Now find the import statement for this contract
	importPath, err := findContractImport(scriptContent, contractName)
	if err != nil {
		return nil, fmt.Errorf("failed to find import for contract %s: %w", contractName, err)
	}

	// Resolve the import path to a full contract path
	contractPath, err := resolveContractPath(scriptPath, importPath, contractName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve contract path: %w", err)
	}

	// Calculate source hash using the project root as base
	sourceHash, err := calculateContractSourceHashWithRoot(contractPath, getProjectRoot(filepath.Dir(scriptPath)))
	if err != nil {
		// Non-fatal - verification can still work without source hash
		sourceHash = ""
	}

	return &ScriptAnalysisResult{
		ContractName: contractName,
		ContractPath: contractPath,
		SourceHash:   sourceHash,
	}, nil
}

// findContractImport finds the import statement for a contract
func findContractImport(scriptContent, contractName string) (string, error) {
	// Try different import patterns
	patterns := []string{
		// import {Contract} from "path/to/Contract.sol";
		fmt.Sprintf(`import\s*{[^}]*\b%s\b[^}]*}\s*from\s*["']([^"']+)["']`, contractName),
		// import "path/to/Contract.sol";
		fmt.Sprintf(`import\s*["']([^"']+/%s\.sol)["']`, contractName),
		// import Contract from "path/to/Contract.sol";
		fmt.Sprintf(`import\s+%s\s+from\s*["']([^"']+)["']`, contractName),
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(scriptContent); len(matches) > 1 {
			return matches[1], nil
		}
	}

	// If not found in imports, it might be in the same file
	if strings.Contains(scriptContent, fmt.Sprintf("contract %s", contractName)) {
		return "", nil // Same file
	}

	return "", fmt.Errorf("import not found for contract %s", contractName)
}

// resolveContractPath resolves an import path to a full contract path
func resolveContractPath(scriptPath, importPath, contractName string) (string, error) {
	if importPath == "" {
		// Contract is in the same file as the script
		return fmt.Sprintf("./%s:%s", scriptPath, contractName), nil
	}

	// Get the directory of the script
	scriptDir := filepath.Dir(scriptPath)
	projectRoot := getProjectRoot(scriptDir)

	// Check if it's a remapped import
	resolvedPath := resolveRemapping(importPath, projectRoot)

	// If it starts with ./ or ../, it's relative to the script
	if strings.HasPrefix(resolvedPath, "./") || strings.HasPrefix(resolvedPath, "../") {
		resolvedPath = filepath.Join(scriptDir, resolvedPath)
	} else if !filepath.IsAbs(resolvedPath) {
		// Otherwise, it's relative to the project root
		resolvedPath = filepath.Join(projectRoot, resolvedPath)
	}

	// Clean the path and make it relative to project root
	resolvedPath = filepath.Clean(resolvedPath)
	
	// Validate that the file exists
	if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
		return "", fmt.Errorf("resolved contract file does not exist: %s", resolvedPath)
	}
	
	relPath, err := filepath.Rel(projectRoot, resolvedPath)
	if err != nil {
		relPath = resolvedPath
	}

	// Format as ./path/to/Contract.sol:Contract
	return fmt.Sprintf("./%s:%s", relPath, contractName), nil
}

// getProjectRoot finds the project root by looking for foundry.toml
func getProjectRoot(startDir string) string {
	dir := startDir
	for {
		if _, err := os.Stat(filepath.Join(dir, "foundry.toml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root, return original directory
			return startDir
		}
		dir = parent
	}
}

// resolveRemapping resolves import remappings using forge remappings command
func resolveRemapping(importPath, projectRoot string) string {
	// First try to get remappings from forge command
	remappings := getForgeRemappings(projectRoot)
	
	// Apply remappings
	for from, to := range remappings {
		if strings.HasPrefix(importPath, from) {
			return strings.Replace(importPath, from, to, 1)
		}
	}

	return importPath
}

// getForgeRemappings gets remappings from forge remappings command
func getForgeRemappings(projectRoot string) map[string]string {
	remappings := make(map[string]string)
	
	// Try to run forge remappings command
	cmd := exec.Command("forge", "remappings")
	cmd.Dir = projectRoot
	output, err := cmd.Output()
	
	if err == nil {
		// Parse forge remappings output
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				from := strings.TrimSpace(parts[0])
				to := strings.TrimSpace(parts[1])
				remappings[from] = to
			}
		}
	} else {
		// Fallback to reading remappings.txt if forge command fails
		remappingsFile := filepath.Join(projectRoot, "remappings.txt")
		if content, err := os.ReadFile(remappingsFile); err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					from := strings.TrimSpace(parts[0])
					to := strings.TrimSpace(parts[1])
					remappings[from] = to
				}
			}
		}
	}
	
	return remappings
}

// calculateContractSourceHash calculates the hash of the contract source file
func calculateContractSourceHash(contractPath string) (string, error) {
	return calculateContractSourceHashWithRoot(contractPath, ".")
}

// calculateContractSourceHashWithRoot calculates the hash of the contract source file with a specific project root
func calculateContractSourceHashWithRoot(contractPath, projectRoot string) (string, error) {
	// Extract file path from contract path (format: ./path/to/Contract.sol:Contract)
	parts := strings.Split(contractPath, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid contract path format: %s", contractPath)
	}

	filePath := strings.TrimPrefix(parts[0], "./")
	
	// Make the path relative to the project root
	fullPath := filepath.Join(projectRoot, filePath)
	
	// Read and hash the file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read contract file %s: %w", fullPath, err)
	}

	return calculateHash(content), nil
}

// calculateHash calculates SHA256 hash of content
func calculateHash(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}