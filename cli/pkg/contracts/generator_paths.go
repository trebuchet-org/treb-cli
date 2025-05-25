package contracts

import (
	"fmt"
	"path/filepath"
	"strings"
)

// GetDeployScriptPath returns the path for a deploy script, maintaining directory structure
func (g *Generator) GetDeployScriptPath(contractInfo *ContractInfo) string {
	// Get relative path from src (if applicable)
	relPath := contractInfo.FilePath
	if strings.HasPrefix(relPath, "src/") {
		relPath = strings.TrimPrefix(relPath, "src/")
	}
	
	// Get directory path
	dir := filepath.Dir(relPath)
	
	// Build script path
	if dir == "." {
		// Contract is in root src directory
		return filepath.Join(g.projectRoot, "script", "deploy", fmt.Sprintf("Deploy%s.s.sol", contractInfo.Name))
	} else {
		// Contract is in subdirectory - maintain structure
		return filepath.Join(g.projectRoot, "script", "deploy", dir, fmt.Sprintf("Deploy%s.s.sol", contractInfo.Name))
	}
}

// GetProxyScriptPath returns the path for a proxy deploy script, maintaining directory structure
func (g *Generator) GetProxyScriptPath(contractInfo *ContractInfo) string {
	// Get relative path from src (if applicable)
	relPath := contractInfo.FilePath
	if strings.HasPrefix(relPath, "src/") {
		relPath = strings.TrimPrefix(relPath, "src/")
	}
	
	// Get directory path
	dir := filepath.Dir(relPath)
	
	// Build script path
	if dir == "." {
		// Contract is in root src directory
		return filepath.Join(g.projectRoot, "script", "deploy", fmt.Sprintf("Deploy%sProxy.s.sol", contractInfo.Name))
	} else {
		// Contract is in subdirectory - maintain structure
		return filepath.Join(g.projectRoot, "script", "deploy", dir, fmt.Sprintf("Deploy%sProxy.s.sol", contractInfo.Name))
	}
}

// calculateImportPath calculates the relative import path from a script to a contract
func (g *Generator) calculateImportPath(scriptPath, contractPath string) string {
	// Remove project root from paths to get relative paths
	scriptRel := strings.TrimPrefix(scriptPath, g.projectRoot+string(filepath.Separator))
	contractRel := strings.TrimPrefix(contractPath, g.projectRoot+string(filepath.Separator))
	
	// If contract path doesn't start with src/, add it
	if !strings.HasPrefix(contractRel, "src/") {
		contractRel = "src/" + contractRel
	}
	
	// Get the directory of the script
	scriptDir := filepath.Dir(scriptRel)
	
	// Calculate relative path from script dir to contract
	relPath, err := filepath.Rel(scriptDir, contractRel)
	if err != nil {
		// Fallback to a generic path
		return "../../" + contractRel
	}
	
	// Convert to forward slashes for Solidity imports
	return strings.ReplaceAll(relPath, string(filepath.Separator), "/")
}