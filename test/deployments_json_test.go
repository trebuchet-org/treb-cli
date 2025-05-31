package integration_test

import (
	"testing"
)

// TestDeploymentsJSONIntegrity was removed because it tests the old v1 deployments.json structure
// which has been replaced with the v2 .treb directory structure
func TestDeploymentsJSONIntegrity(t *testing.T) {
	t.Skip("This test is for the old v1 deployments.json format which has been replaced with the v2 .treb directory structure")
}