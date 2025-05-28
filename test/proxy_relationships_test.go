package integration_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestProxyDeploymentRelationships(t *testing.T) {
	// Skip this test in CI for now - needs infrastructure improvements
	if os.Getenv("CI") != "" {
		t.Skip("Skipping proxy relationships test in CI - needs infrastructure improvements")
	}
	
	t.Run("deploy_implementation_and_proxy", func(t *testing.T) {
		// Clean slate
		removeDeploymentsFile()

		// Generate the implementation deployment script if it doesn't exist
		cmd := exec.Command(trebBin, "gen", "deploy", "UpgradeableCounter", "--strategy", "CREATE3", "--non-interactive")
		cmd.Dir = fixtureDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("Script generation output: %s", output)
			// Script might already exist, continue with deployment
		}

		// Deploy the implementation contract first
		cmd = exec.Command(trebBin, "deploy", "UpgradeableCounter", "--network", "local")
		cmd.Dir = fixtureDir
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to deploy implementation: %v\nOutput: %s", err, output)
		}

		// Deploy the proxy
		cmd = exec.Command(trebBin, "deploy", "proxy", "UpgradeableCounter", "--network", "local")
		cmd.Dir = fixtureDir
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to deploy proxy: %v\nOutput: %s", err, output)
		}

		// Verify deployments.json structure
		deployments := readDeploymentsFile(t)
		
		var implementationDeployment, proxyDeployment *DeploymentEntry
		for _, deployment := range deployments.Networks["31337"].Deployments {
			if deployment.ContractName == "UpgradeableCounter" {
				implementationDeployment = deployment
			} else if deployment.ContractName == "ERC1967Proxy" {
				proxyDeployment = deployment
			}
		}

		// Verify both deployments exist
		if implementationDeployment == nil {
			t.Fatal("Implementation deployment not found")
		}
		if proxyDeployment == nil {
			t.Fatal("Proxy deployment not found")
		}

		// Verify deployment types
		if implementationDeployment.Type != "SINGLETON" {
			t.Errorf("Expected implementation type to be SINGLETON, got %s", implementationDeployment.Type)
		}
		if proxyDeployment.Type != "PROXY" {
			t.Errorf("Expected proxy type to be PROXY, got %s", proxyDeployment.Type)
		}

		// Verify proxy has target_deployment_fqid pointing to implementation
		if proxyDeployment.TargetDeploymentFQID == "" {
			t.Error("Proxy deployment missing target_deployment_fqid")
		}
		if !strings.Contains(proxyDeployment.TargetDeploymentFQID, "UpgradeableCounter") {
			t.Errorf("Expected proxy to target UpgradeableCounter, got %s", proxyDeployment.TargetDeploymentFQID)
		}
	})

	t.Run("list_shows_proxy_relationships", func(t *testing.T) {
		// Run list command and check output
		cmd := exec.Command(trebBin, "list", "--filter-network", "local")
		cmd.Dir = fixtureDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to run list command: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		t.Logf("List output:\n%s", outputStr)

		// Verify the proxy is shown under PROXIES section, not SINGLETONS
		if !strings.Contains(outputStr, "PROXIES") {
			t.Error("Expected to see PROXIES section in list output")
		}

		// Verify the implementation is shown under SINGLETONS
		if !strings.Contains(outputStr, "SINGLETONS") {
			t.Error("Expected to see SINGLETONS section in list output")
		}

		// Verify proxy shows implementation relationship
		lines := strings.Split(outputStr, "\n")
		var inProxiesSection bool
		var foundProxyRelationship bool
		
		for _, line := range lines {
			if strings.Contains(line, "PROXIES") {
				inProxiesSection = true
				continue
			}
			if strings.Contains(line, "SINGLETONS") {
				inProxiesSection = false
				continue
			}
			
			if inProxiesSection && strings.Contains(line, "ERC1967Proxy") {
				// Should show the proxy with implementation reference
				if strings.Contains(line, "UpgradeableCounter") {
					foundProxyRelationship = true
				}
			}
		}

		if !foundProxyRelationship {
			t.Error("Expected to see proxy relationship to UpgradeableCounter in PROXIES section")
		}
	})
}

// Helper to remove deployments file
func removeDeploymentsFile() {
	os.Remove(fixtureDir + "/deployments.json")
}

// Helper to read deployments file
func readDeploymentsFile(t *testing.T) *DeploymentsRegistry {
	data, err := os.ReadFile(fixtureDir + "/deployments.json")
	if err != nil {
		t.Fatalf("Failed to read deployments.json: %v", err)
	}

	var deployments DeploymentsRegistry
	if err := json.Unmarshal(data, &deployments); err != nil {
		t.Fatalf("Failed to parse deployments.json: %v", err)
	}

	return &deployments
}

// Types for parsing deployments.json
type DeploymentsRegistry struct {
	Networks map[string]NetworkEntry `json:"networks"`
}

type NetworkEntry struct {
	Name        string                         `json:"name"`
	Deployments map[string]*DeploymentEntry    `json:"deployments"`
}

type DeploymentEntry struct {
	FQID                 string `json:"fqid"`
	SID                  string `json:"sid"`
	Address              string `json:"address"`
	ContractName         string `json:"contract_name"`
	Namespace            string `json:"namespace"`
	Type                 string `json:"type"`
	TargetDeploymentFQID string `json:"target_deployment_fqid,omitempty"`
	// ... other fields omitted for brevity
}