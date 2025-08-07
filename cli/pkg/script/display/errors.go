package display

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/cli/pkg/forge"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// HandleScriptError processes script execution errors and provides helpful guidance
func HandleScriptError(result *forge.ScriptResult, indexer *contracts.Indexer) {
	if result.Success || result.Error == nil {
		return
	}

	// Check for BytecodeMissing error in the output
	errorOutput := string(result.RawOutput)
	if result.ParsedOutput != nil && result.ParsedOutput.TextOutput != "" {
		errorOutput = result.ParsedOutput.TextOutput
	}

	if missingLibs := parseBytecodeMissingError(errorOutput, indexer); len(missingLibs) > 0 {
		handleMissingLibraries(missingLibs)
		return
	}

	// Default error handling
	PrintErrorMessage("Script execution failed")
}

// parseBytecodeMissingError extracts missing library information from error output
func parseBytecodeMissingError(output string, indexer *contracts.Indexer) []MissingLibrary {
	// Match BytecodeMissing("path/to/Contract.sol:ContractName") pattern
	re := regexp.MustCompile(`BytecodeMissing\("([^"]+)"\)`)
	matches := re.FindAllStringSubmatch(output, -1)

	if len(matches) == 0 {
		return nil
	}

	var missingLibs []MissingLibrary
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		contractRef := match[1]
		if seen[contractRef] {
			continue
		}
		seen[contractRef] = true

		// Parse contract reference (path/to/Contract.sol:ContractName)
		parts := strings.Split(contractRef, ":")
		if len(parts) != 2 {
			continue
		}

		contractPath := parts[0]
		contractName := parts[1]

		// Try to find the contract and extract link references from its artifact
		// First try the full reference, then try just the contract name
		contract, err := indexer.GetContract(contractRef)
		if err != nil || contract == nil || contract.Artifact == nil {
			// Try with just the contract name
			contracts := indexer.GetContractsByName(contractName)
			if len(contracts) > 0 && contracts[0].Artifact != nil {
				contract = contracts[0]
			} else {
				// Still add it to missing libs even without artifact info
				missingLibs = append(missingLibs, MissingLibrary{
					ContractPath: contractPath,
					ContractName: contractName,
					FullRef:      contractRef,
				})
				continue
			}
		}

		// Extract required libraries from link references
		requiredLibs := extractRequiredLibraries(contract.Artifact)
		
		missingLibs = append(missingLibs, MissingLibrary{
			ContractPath:      contractPath,
			ContractName:      contractName,
			FullRef:           contractRef,
			RequiredLibraries: requiredLibs,
		})
	}

	return missingLibs
}

// MissingLibrary represents a contract with missing library dependencies
type MissingLibrary struct {
	ContractPath      string
	ContractName      string
	FullRef           string
	RequiredLibraries []string // e.g., ["src/TestWithNewLib.sol:MathUtils"]
}

// extractRequiredLibraries parses link references from contract artifact
func extractRequiredLibraries(artifact *types.Artifact) []string {
	if artifact == nil {
		return nil
	}

	var libraries []string
	seen := make(map[string]bool)

	// Parse linkReferences from bytecode
	for libPath, libMap := range artifact.Bytecode.LinkReferences {
		for libName := range libMap {
			libRef := fmt.Sprintf("%s:%s", libPath, libName)
			if !seen[libRef] {
				seen[libRef] = true
				libraries = append(libraries, libRef)
			}
		}
	}

	return libraries
}

// handleMissingLibraries displays helpful error message for missing libraries
func handleMissingLibraries(missingLibs []MissingLibrary) {
	PrintErrorMessage("Script execution failed: Missing library bytecode")
	fmt.Println()
	
	fmt.Println("The following contracts are missing library dependencies:")
	
	for _, lib := range missingLibs {
		fmt.Printf("\n  • %s%s%s\n", ColorYellow, lib.FullRef, ColorReset)
		
		if len(lib.RequiredLibraries) > 0 {
			fmt.Printf("    Required libraries:\n")
			for _, reqLib := range lib.RequiredLibraries {
				fmt.Printf("      - %s%s%s\n", ColorBlue, reqLib, ColorReset)
			}
			
			fmt.Printf("\n    To deploy the required libraries:\n")
			for _, reqLib := range lib.RequiredLibraries {
				fmt.Printf("      1. Generate script: %streb gen library %s%s\n", ColorCyan, reqLib, ColorReset)
			}
			fmt.Printf("      2. Deploy: %streb run <library-deploy-script>%s\n", ColorCyan, ColorReset)
		} else {
			fmt.Printf("    %s(Unable to determine required libraries - check contract's link references)%s\n", ColorGray, ColorReset)
		}
	}
	
	fmt.Println()
	fmt.Println("Libraries must be deployed before contracts that depend on them.")
}