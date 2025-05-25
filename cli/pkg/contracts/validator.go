package contracts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)


// Validator handles contract validation and discovery
type Validator struct {
	projectRoot string
}

// NewValidator creates a new contract validator
func NewValidator(projectRoot string) *Validator {
	return &Validator{
		projectRoot: projectRoot,
	}
}

// ValidateContract checks if a contract exists in the src directory
func (v *Validator) ValidateContract(contractName string) (*ContractInfo, error) {
	srcDir := filepath.Join(v.projectRoot, "src")
	
	// Check if src directory exists
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("src directory not found at %s", srcDir)
	}

	// Look for contract file
	var contractInfo *ContractInfo
	var found bool

	// Try common patterns: ContractName.sol, contractName.sol
	patterns := []string{
		fmt.Sprintf("%s.sol", contractName),
		fmt.Sprintf("%s.sol", strings.ToLower(contractName)),
	}

	for _, pattern := range patterns {
		filePath := filepath.Join(srcDir, pattern)
		if _, err := os.Stat(filePath); err == nil {
			contractInfo = &ContractInfo{
				Name: contractName,
				Path: filePath,
			}
			found = true
			break
		}
	}

	// Search in subdirectories if not found
	if !found {
		err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".sol") {
				// Read file and check for contract declaration
				content, readErr := os.ReadFile(path)
				if readErr != nil {
					return nil // Continue searching
				}
				
				// Simple check for contract declaration
				if strings.Contains(string(content), fmt.Sprintf("contract %s", contractName)) {
					contractInfo = &ContractInfo{
						Name: contractName,
						Path: path,
					}
					found = true
					return filepath.SkipAll // Found it, stop searching
				}
			}
			
			return nil
		})
		
		if err != nil {
			return nil, fmt.Errorf("error searching for contract: %w", err)
		}
	}

	return contractInfo, nil
}

// GetDeployScriptPath returns the expected path for a deploy script
func (v *Validator) GetDeployScriptPath(contractName string) string {
	return filepath.Join(v.projectRoot, "script", "deploy", fmt.Sprintf("Deploy%s.s.sol", contractName))
}

// DeployScriptExists checks if a deploy script already exists
func (v *Validator) DeployScriptExists(contractName string) bool {
	scriptPath := v.GetDeployScriptPath(contractName)
	_, err := os.Stat(scriptPath)
	return err == nil
}

// IsLibrary checks if a contract is a library
func (v *Validator) IsLibrary(contractName string) bool {
	contractInfo, err := v.ValidateContract(contractName)
	if err != nil || contractInfo == nil {
		return false
	}

	// Read the contract file
	content, err := os.ReadFile(contractInfo.Path)
	if err != nil {
		return false
	}

	// Check for library declaration
	return strings.Contains(string(content), fmt.Sprintf("library %s", contractName))
}