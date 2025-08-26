package integration

import (
	"testing"
)

func TestWorkflow(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "complex with libraries",
			TestCmds: [][]string{
				s("config set network anvil-31337"),
				s("gen deploy StringUtilsV2"),
				s("gen deploy src/Counter.sol:Counter"),
				s("gen deploy src/MessageStorageV08"),
				s("run DeployStringUtilsV2"),
				s("run DeployMessageStorageV08"),
				s("ls"),
				s("config set network anvil-31338"),
				s("run DeployCounter"),
				s("run DeployCounter -e LABEL=v2"),
				s("run DeployCounter -e LABEL=v3"),
				s("ls"),
			},
		},
	}

	RunIntegrationTests(t, tests)
}
