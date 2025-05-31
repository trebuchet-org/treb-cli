package integration_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
)

func TestABILoadingWithArtifactPaths(t *testing.T) {
	// Create a test indexer
	projectRoot := fixtureDir
	indexer := contracts.NewIndexer(projectRoot)
	
	// Index contracts
	err := indexer.Index()
	require.NoError(t, err)
	
	// Test cases for GetContractByArtifact
	testCases := []struct {
		name     string
		artifact string
		wantNil  bool
	}{
		// Skipping simple name test because there are multiple Counter contracts
		// {
		// 	name:     "simple contract name",
		// 	artifact: "Counter",
		// 	wantNil:  false,
		// },
		{
			name:     "full artifact path",
			artifact: "src/Counter.sol:Counter",
			wantNil:  false,
		},
		{
			name:     "full artifact path with subdirectory",
			artifact: "src/UpgradeableCounter.sol:UpgradeableCounter",
			wantNil:  false,
		},
		{
			name:     "non-existent contract",
			artifact: "NonExistent",
			wantNil:  true,
		},
		{
			name:     "non-existent artifact path",
			artifact: "src/NonExistent.sol:NonExistent",
			wantNil:  true,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			contract := indexer.GetContractByArtifact(tc.artifact)
			if tc.wantNil {
				require.Nil(t, contract, "expected nil for artifact %s", tc.artifact)
			} else {
				require.NotNil(t, contract, "expected contract for artifact %s", tc.artifact)
				t.Logf("Found contract: %+v", contract)
			}
		})
	}
}