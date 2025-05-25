package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// FoundryConfig represents the full foundry.toml configuration
type FoundryConfig struct {
	Profile map[string]ProfileFoundryConfig `toml:"profile"`
}

// ProfileFoundryConfig represents a profile's foundry configuration
type ProfileFoundryConfig struct {
	Deployer  DeployerConfig `toml:"deployer,omitempty"`
	Libraries []string       `toml:"libraries,omitempty"`
	// Other foundry settings
	SrcPath       string   `toml:"src,omitempty"`
	OutPath       string   `toml:"out,omitempty"`
	LibPaths      []string `toml:"libs,omitempty"`
	TestPath      string   `toml:"test,omitempty"`
	ScriptPath    string   `toml:"script,omitempty"`
	Remappings    []string `toml:"remappings,omitempty"`
	SolcVersion   string   `toml:"solc_version,omitempty"`
	Optimizer     bool     `toml:"optimizer,omitempty"`
	OptimizerRuns int      `toml:"optimizer_runs,omitempty"`
}

// FoundryManager handles foundry.toml file operations
type FoundryManager struct {
	projectRoot string
	configPath  string
}

// NewFoundryManager creates a new foundry configuration manager
func NewFoundryManager(projectRoot string) *FoundryManager {
	return &FoundryManager{
		projectRoot: projectRoot,
		configPath:  filepath.Join(projectRoot, "foundry.toml"),
	}
}

// Load reads the foundry configuration
func (fm *FoundryManager) Load() (*FoundryConfig, error) {
	// Check if foundry.toml exists
	if _, err := os.Stat(fm.configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("foundry.toml not found at %s", fm.configPath)
	}

	// Read the file content
	data, err := os.ReadFile(fm.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read foundry.toml: %w", err)
	}

	// Parse TOML
	var config FoundryConfig
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse foundry.toml: %w", err)
	}

	// Initialize profile map if needed
	if config.Profile == nil {
		config.Profile = make(map[string]ProfileFoundryConfig)
	}

	return &config, nil
}

// Save writes the foundry configuration back to file
// DEPRECATED: This method should not be used as it reformats the entire file
func (fm *FoundryManager) Save(config *FoundryConfig) error {
	return fmt.Errorf("Save method is deprecated - use surgical update methods instead")
}

// AddLibrary adds a library mapping to the specified profile using surgical file update
func (fm *FoundryManager) AddLibrary(profile, libraryPath, libraryName, address string) error {
	// Read the file content
	content, err := os.ReadFile(fm.configPath)
	if err != nil {
		return fmt.Errorf("failed to read foundry.toml: %w", err)
	}

	// Format library entry: "path/to/Library.sol:LibraryName:0xAddress"
	libraryEntry := fmt.Sprintf("\"%s:%s:%s\"", libraryPath, libraryName, address)

	// Convert content to string for manipulation
	fileContent := string(content)

	// Look for the profile section
	profilePattern := fmt.Sprintf(`\[profile\.%s\]`, regexp.QuoteMeta(profile))
	profileRegex := regexp.MustCompile(profilePattern)
	
	profileMatch := profileRegex.FindStringIndex(fileContent)
	if profileMatch == nil {
		// Profile doesn't exist, need to add it at the end
		if !strings.HasSuffix(fileContent, "\n") {
			fileContent += "\n"
		}
		fileContent += fmt.Sprintf("\n[profile.%s]\nlibraries = [%s]\n", profile, libraryEntry)
		return os.WriteFile(fm.configPath, []byte(fileContent), 0644)
	}

	// Find the libraries array within this profile
	// Start searching from the profile position
	searchStart := profileMatch[1]
	
	// Find the next profile section or end of file
	nextProfileRegex := regexp.MustCompile(`\n\[`)
	nextProfileMatch := nextProfileRegex.FindStringIndex(fileContent[searchStart:])
	searchEnd := len(fileContent)
	if nextProfileMatch != nil {
		searchEnd = searchStart + nextProfileMatch[0]
	}
	
	profileSection := fileContent[searchStart:searchEnd]
	
	// Look for existing libraries array
	librariesRegex := regexp.MustCompile(`(?m)^libraries\s*=\s*\[([^\]]*)\]`)
	librariesMatch := librariesRegex.FindStringSubmatchIndex(profileSection)
	
	if librariesMatch != nil {
		// Libraries array exists, update it
		arrayStart := searchStart + librariesMatch[2]
		arrayEnd := searchStart + librariesMatch[3]
		currentLibraries := strings.TrimSpace(fileContent[arrayStart:arrayEnd])
		
		// Check if library already exists
		if strings.Contains(currentLibraries, ":"+libraryName+":") {
			// Replace existing library
			libraryPattern := fmt.Sprintf(`"[^"]*:%s:[^"]*"`, regexp.QuoteMeta(libraryName))
			libraryRegex := regexp.MustCompile(libraryPattern)
			updatedLibraries := libraryRegex.ReplaceAllString(currentLibraries, libraryEntry)
			
			fileContent = fileContent[:arrayStart] + updatedLibraries + fileContent[arrayEnd:]
		} else {
			// Add new library
			if currentLibraries == "" {
				// Empty array
				fileContent = fileContent[:arrayStart] + "\n    " + libraryEntry + "\n" + fileContent[arrayEnd:]
			} else {
				// Non-empty array, add comma and new entry
				fileContent = fileContent[:arrayEnd] + ",\n    " + libraryEntry + fileContent[arrayEnd:]
			}
		}
	} else {
		// No libraries array, need to add it after the profile header
		insertPos := profileMatch[1]
		// Skip any whitespace after the profile header
		for insertPos < len(fileContent) && fileContent[insertPos] == '\n' {
			insertPos++
		}
		
		librariesArray := fmt.Sprintf("\nlibraries = [\n    %s\n]\n", libraryEntry)
		fileContent = fileContent[:insertPos] + librariesArray + fileContent[insertPos:]
	}

	// Write back to file
	return os.WriteFile(fm.configPath, []byte(fileContent), 0644)
}

// UpdateLibraryAddress updates the address of an existing library using surgical file update
func (fm *FoundryManager) UpdateLibraryAddress(profile, libraryName, newAddress string) error {
	// Read the file content
	content, err := os.ReadFile(fm.configPath)
	if err != nil {
		return fmt.Errorf("failed to read foundry.toml: %w", err)
	}

	fileContent := string(content)
	
	// Pattern to find the library entry
	libraryPattern := fmt.Sprintf(`"([^"]*:%s:)[^"]*"`, regexp.QuoteMeta(libraryName))
	libraryRegex := regexp.MustCompile(libraryPattern)
	
	// Check if library exists
	if !libraryRegex.MatchString(fileContent) {
		return fmt.Errorf("library '%s' not found", libraryName)
	}
	
	// Replace with new address
	replacement := fmt.Sprintf("\"${1}%s\"", newAddress)
	updatedContent := libraryRegex.ReplaceAllString(fileContent, replacement)
	
	// Write back to file
	return os.WriteFile(fm.configPath, []byte(updatedContent), 0644)
}

// GetRemappings returns all remappings for a profile
func (fm *FoundryManager) GetRemappings(profile string) ([]string, error) {
	config, err := fm.Load()
	if err != nil {
		return nil, err
	}

	if profileConfig, exists := config.Profile[profile]; exists {
		return profileConfig.Remappings, nil
	}

	// If profile doesn't exist, check default
	if profile != "default" {
		if defaultProfile, exists := config.Profile["default"]; exists {
			return defaultProfile.Remappings, nil
		}
	}

	return nil, nil
}

// ParseRemapping parses a remapping string and returns the mapping name and path
// Example: "@openzeppelin/=lib/openzeppelin-contracts/" returns ("@openzeppelin/", "lib/openzeppelin-contracts/")
func ParseRemapping(remapping string) (string, string, error) {
	parts := strings.SplitN(remapping, "=", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid remapping format: %s", remapping)
	}
	return parts[0], parts[1], nil
}

// GetLibraries returns all libraries for a profile
func (fm *FoundryManager) GetLibraries(profile string) ([]string, error) {
	config, err := fm.Load()
	if err != nil {
		return nil, err
	}

	profileConfig, exists := config.Profile[profile]
	if !exists {
		return []string{}, nil
	}

	return profileConfig.Libraries, nil
}

// RemoveLibrary removes a library from a profile using surgical file update
func (fm *FoundryManager) RemoveLibrary(profile, libraryName string) error {
	// Read the file content
	content, err := os.ReadFile(fm.configPath)
	if err != nil {
		return fmt.Errorf("failed to read foundry.toml: %w", err)
	}

	fileContent := string(content)
	
	// Pattern to find the library entry (including potential comma and whitespace)
	libraryPattern := fmt.Sprintf(`\s*,?\s*"[^"]*:%s:[^"]*",?\s*`, regexp.QuoteMeta(libraryName))
	libraryRegex := regexp.MustCompile(libraryPattern)
	
	// Check if library exists
	if !libraryRegex.MatchString(fileContent) {
		return fmt.Errorf("library '%s' not found", libraryName)
	}
	
	// Remove the library entry
	updatedContent := libraryRegex.ReplaceAllString(fileContent, "")
	
	// Clean up any double commas or trailing commas
	updatedContent = regexp.MustCompile(`,\s*,`).ReplaceAllString(updatedContent, ",")
	updatedContent = regexp.MustCompile(`\[\s*,`).ReplaceAllString(updatedContent, "[")
	updatedContent = regexp.MustCompile(`,\s*\]`).ReplaceAllString(updatedContent, "]")
	
	// Write back to file
	return os.WriteFile(fm.configPath, []byte(updatedContent), 0644)
}

// ParseLibraryEntry parses a library entry into its components
func ParseLibraryEntry(entry string) (path, name, address string, err error) {
	parts := strings.Split(entry, ":")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid library entry format: %s", entry)
	}
	return parts[0], parts[1], parts[2], nil
}

// findLibrarySourcePath finds the source path for a library
func findLibrarySourcePath(projectRoot, libraryName string) (string, error) {
	// Common locations to check
	searchPaths := []string{
		"src",
		"contracts",
		"lib",
	}

	// Use regex to find library files
	libraryFilePattern := regexp.MustCompile(fmt.Sprintf(`%s\.sol$`, libraryName))

	for _, searchPath := range searchPaths {
		basePath := filepath.Join(projectRoot, searchPath)
		if _, err := os.Stat(basePath); os.IsNotExist(err) {
			continue
		}

		// Walk the directory
		var foundPath string
		err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors
			}
			
			if !info.IsDir() && libraryFilePattern.MatchString(info.Name()) {
				// Make path relative to project root
				relPath, err := filepath.Rel(projectRoot, path)
				if err != nil {
					return err
				}
				foundPath = relPath
				return filepath.SkipDir // Stop walking
			}
			return nil
		})

		if err == nil && foundPath != "" {
			return foundPath, nil
		}
	}

	// If not found in common locations, check if full path was provided
	if strings.HasSuffix(libraryName, ".sol") {
		if _, err := os.Stat(filepath.Join(projectRoot, libraryName)); err == nil {
			return libraryName, nil
		}
	}

	return "", fmt.Errorf("could not find source file for library %s", libraryName)
}

// AddLibraryAuto automatically finds the library source path and adds it
func (fm *FoundryManager) AddLibraryAuto(profile, libraryName, address string) error {
	// Find the library source path
	sourcePath, err := findLibrarySourcePath(fm.projectRoot, libraryName)
	if err != nil {
		// Fallback to a generic path
		sourcePath = fmt.Sprintf("src/%s.sol", libraryName)
	}

	return fm.AddLibrary(profile, sourcePath, libraryName, address)
}