package deployment

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// parseProxyArtifactPath extracts the proxy contract artifact path from a proxy deployment script
// It looks for the ProxyDeployment constructor and extracts the first argument
func parseProxyArtifactPath(scriptPath string) (string, error) {
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read script file: %w", err)
	}

	// Convert content to string
	scriptContent := string(content)

	// Look for ProxyDeployment constructor
	// Pattern matches: ProxyDeployment("artifact/path:ContractName", ...)
	// This regex captures the string literal inside the first argument
	pattern := `ProxyDeployment\s*\(\s*"([^"]+)"\s*,`

	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(scriptContent)

	if len(matches) < 2 {
		// Try single quotes as well
		pattern = `ProxyDeployment\s*\(\s*'([^']+)'\s*,`
		re = regexp.MustCompile(pattern)
		matches = re.FindStringSubmatch(scriptContent)

		if len(matches) < 2 {
			return "", fmt.Errorf("could not find ProxyDeployment constructor with artifact path in script")
		}
	}

	artifactPath := matches[1]

	// Validate that it looks like a valid artifact path (contains ':')
	if !strings.Contains(artifactPath, ":") {
		return "", fmt.Errorf("invalid artifact path format: %s (expected format: path/to/Contract.sol:ContractName)", artifactPath)
	}

	return artifactPath, nil
}
