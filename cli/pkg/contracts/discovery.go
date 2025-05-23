package contracts

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ContractInfo represents information about a discovered contract
type ContractDiscovery struct {
	Name         string
	FileName     string
	RelativePath string
	FullPath     string
	Version      string
}

// Discovery handles contract discovery in the src directory
type Discovery struct {
	projectRoot string
}

// NewDiscovery creates a new contract discovery instance
func NewDiscovery(projectRoot string) *Discovery {
	return &Discovery{
		projectRoot: projectRoot,
	}
}

// DiscoverContracts finds all Solidity contracts in the src directory
func (d *Discovery) DiscoverContracts() ([]ContractDiscovery, error) {
	srcDir := filepath.Join(d.projectRoot, "src")
	
	var contracts []ContractDiscovery
	
	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories and non-.sol files
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".sol") {
			return nil
		}
		
		// Extract contract name from filename
		fileName := info.Name()
		contractName := strings.TrimSuffix(fileName, ".sol")
		
		// Get relative path from src directory
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		
		// Parse version from contract file if possible
		version, _ := d.parseVersionFromFile(path)
		
		contracts = append(contracts, ContractDiscovery{
			Name:         contractName,
			FileName:     fileName,
			RelativePath: relPath,
			FullPath:     path,
			Version:      version,
		})
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to discover contracts: %w", err)
	}
	
	// Sort contracts by name for consistent ordering
	sort.Slice(contracts, func(i, j int) bool {
		return contracts[i].Name < contracts[j].Name
	})
	
	return contracts, nil
}

// parseVersionFromFile attempts to extract Solidity version from contract file
func (d *Discovery) parseVersionFromFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "pragma solidity") {
			// Extract version info
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				version := strings.TrimSuffix(parts[2], ";")
				return version, nil
			}
		}
	}
	
	return "unknown", nil
}

// FormatContractOption formats a contract for display in selection lists
func (d *Discovery) FormatContractOption(contract ContractDiscovery) string {
	if contract.RelativePath == contract.FileName {
		// Contract is in root src directory
		return fmt.Sprintf("%s (%s)", contract.Name, contract.Version)
	} else {
		// Contract is in subdirectory
		dir := filepath.Dir(contract.RelativePath)
		return fmt.Sprintf("%s (%s) [%s]", contract.Name, contract.Version, dir)
	}
}

// GetContractNames returns just the contract names for simple selection
func (d *Discovery) GetContractNames(contracts []ContractDiscovery) []string {
	names := make([]string, len(contracts))
	for i, contract := range contracts {
		names[i] = contract.Name
	}
	return names
}

// GetFormattedOptions returns formatted options for display
func (d *Discovery) GetFormattedOptions(contracts []ContractDiscovery) []string {
	options := make([]string, len(contracts))
	for i, contract := range contracts {
		options[i] = d.FormatContractOption(contract)
	}
	return options
}