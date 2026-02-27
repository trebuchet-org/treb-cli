package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectEnvVar(t *testing.T) {
	tests := []struct {
		name       string
		rawValue   string
		wantEnvVar string
		wantIsVar  bool
	}{
		{
			name:       "simple env var",
			rawValue:   "${SEPOLIA_RPC_URL}",
			wantEnvVar: "SEPOLIA_RPC_URL",
			wantIsVar:  true,
		},
		{
			name:       "env var with underscores",
			rawValue:   "${CELO_SEPOLIA_RPC_URL}",
			wantEnvVar: "CELO_SEPOLIA_RPC_URL",
			wantIsVar:  true,
		},
		{
			name:       "hardcoded URL",
			rawValue:   "https://sepolia.base.org",
			wantEnvVar: "",
			wantIsVar:  false,
		},
		{
			name:       "env var with path suffix",
			rawValue:   "${MY_VAR}/path",
			wantEnvVar: "",
			wantIsVar:  false,
		},
		{
			name:       "empty string",
			rawValue:   "",
			wantEnvVar: "",
			wantIsVar:  false,
		},
		{
			name:       "localhost URL",
			rawValue:   "http://localhost:8545",
			wantEnvVar: "",
			wantIsVar:  false,
		},
		{
			name:       "env var starting with underscore",
			rawValue:   "${_MY_VAR}",
			wantEnvVar: "_MY_VAR",
			wantIsVar:  true,
		},
		{
			name:       "partial env var syntax - missing closing brace",
			rawValue:   "${UNCLOSED",
			wantEnvVar: "",
			wantIsVar:  false,
		},
		{
			name:       "dollar without braces",
			rawValue:   "$MY_VAR",
			wantEnvVar: "",
			wantIsVar:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envVar, isVar := DetectEnvVar(tt.rawValue)
			assert.Equal(t, tt.wantEnvVar, envVar)
			assert.Equal(t, tt.wantIsVar, isVar)
		})
	}
}

func TestGenerateEnvVarName(t *testing.T) {
	tests := []struct {
		name        string
		networkName string
		want        string
	}{
		{
			name:        "simple network",
			networkName: "sepolia",
			want:        "SEPOLIA_RPC_URL",
		},
		{
			name:        "network with dash",
			networkName: "celo-sepolia",
			want:        "CELO_SEPOLIA_RPC_URL",
		},
		{
			name:        "network with number and dash",
			networkName: "anvil-31337",
			want:        "ANVIL_31337_RPC_URL",
		},
		{
			name:        "already uppercase",
			networkName: "MAINNET",
			want:        "MAINNET_RPC_URL",
		},
		{
			name:        "mixed case with dash",
			networkName: "Base-Sepolia",
			want:        "BASE_SEPOLIA_RPC_URL",
		},
		{
			name:        "network with dot",
			networkName: "polygon.zkevm",
			want:        "POLYGON_ZKEVM_RPC_URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateEnvVarName(tt.networkName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoadRawRPCEndpoint(t *testing.T) {
	// Create temp project with foundry.toml
	tmpDir := t.TempDir()
	foundryContent := `[rpc_endpoints]
sepolia = "${SEPOLIA_RPC_URL}"
celo-sepolia = "https://forno.celo-sepolia.celo-testnet.org"
anvil-31337 = "http://localhost:8545"
`
	err := os.WriteFile(filepath.Join(tmpDir, "foundry.toml"), []byte(foundryContent), 0644)
	require.NoError(t, err)

	t.Run("reads env var reference without expanding", func(t *testing.T) {
		raw, err := LoadRawRPCEndpoint(tmpDir, "sepolia")
		require.NoError(t, err)
		assert.Equal(t, "${SEPOLIA_RPC_URL}", raw)
	})

	t.Run("reads hardcoded URL", func(t *testing.T) {
		raw, err := LoadRawRPCEndpoint(tmpDir, "celo-sepolia")
		require.NoError(t, err)
		assert.Equal(t, "https://forno.celo-sepolia.celo-testnet.org", raw)
	})

	t.Run("reads localhost URL", func(t *testing.T) {
		raw, err := LoadRawRPCEndpoint(tmpDir, "anvil-31337")
		require.NoError(t, err)
		assert.Equal(t, "http://localhost:8545", raw)
	})

	t.Run("error on missing network", func(t *testing.T) {
		_, err := LoadRawRPCEndpoint(tmpDir, "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("error on missing foundry.toml", func(t *testing.T) {
		_, err := LoadRawRPCEndpoint(t.TempDir(), "sepolia")
		assert.Error(t, err)
	})
}

func TestMigrateRPCEndpoint(t *testing.T) {
	t.Run("migrates hardcoded URL to env var", func(t *testing.T) {
		tmpDir := t.TempDir()
		foundryContent := `[rpc_endpoints]
celo-sepolia = "https://forno.celo-sepolia.celo-testnet.org"
anvil-31337 = "http://localhost:8545"
`
		err := os.WriteFile(filepath.Join(tmpDir, "foundry.toml"), []byte(foundryContent), 0644)
		require.NoError(t, err)

		err = MigrateRPCEndpoint(tmpDir, "celo-sepolia", "https://forno.celo-sepolia.celo-testnet.org")
		require.NoError(t, err)

		// Check foundry.toml was updated
		data, err := os.ReadFile(filepath.Join(tmpDir, "foundry.toml"))
		require.NoError(t, err)
		content := string(data)
		assert.Contains(t, content, `celo-sepolia = "${CELO_SEPOLIA_RPC_URL}"`)
		assert.NotContains(t, content, "https://forno.celo-sepolia.celo-testnet.org")
		// Other entries preserved
		assert.Contains(t, content, `anvil-31337 = "http://localhost:8545"`)

		// Check .env was created with the var
		envData, err := os.ReadFile(filepath.Join(tmpDir, ".env"))
		require.NoError(t, err)
		assert.Contains(t, string(envData), "CELO_SEPOLIA_RPC_URL=https://forno.celo-sepolia.celo-testnet.org\n")
	})

	t.Run("appends to existing .env", func(t *testing.T) {
		tmpDir := t.TempDir()
		foundryContent := `[rpc_endpoints]
base-sepolia = "https://sepolia.base.org"
`
		err := os.WriteFile(filepath.Join(tmpDir, "foundry.toml"), []byte(foundryContent), 0644)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte("EXISTING_VAR=value\n"), 0644)
		require.NoError(t, err)

		err = MigrateRPCEndpoint(tmpDir, "base-sepolia", "https://sepolia.base.org")
		require.NoError(t, err)

		envData, err := os.ReadFile(filepath.Join(tmpDir, ".env"))
		require.NoError(t, err)
		content := string(envData)
		assert.Contains(t, content, "EXISTING_VAR=value\n")
		assert.Contains(t, content, "BASE_SEPOLIA_RPC_URL=https://sepolia.base.org\n")
	})

	t.Run("adds newline before entry if .env has no trailing newline", func(t *testing.T) {
		tmpDir := t.TempDir()
		foundryContent := `[rpc_endpoints]
base-sepolia = "https://sepolia.base.org"
`
		err := os.WriteFile(filepath.Join(tmpDir, "foundry.toml"), []byte(foundryContent), 0644)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte("NO_NEWLINE=end"), 0644)
		require.NoError(t, err)

		err = MigrateRPCEndpoint(tmpDir, "base-sepolia", "https://sepolia.base.org")
		require.NoError(t, err)

		envData, err := os.ReadFile(filepath.Join(tmpDir, ".env"))
		require.NoError(t, err)
		content := string(envData)
		assert.Equal(t, "NO_NEWLINE=end\nBASE_SEPOLIA_RPC_URL=https://sepolia.base.org\n", content)
	})

	t.Run("skips .env append if var already exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		foundryContent := `[rpc_endpoints]
base-sepolia = "https://sepolia.base.org"
`
		err := os.WriteFile(filepath.Join(tmpDir, "foundry.toml"), []byte(foundryContent), 0644)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte("BASE_SEPOLIA_RPC_URL=https://old.url.org\n"), 0644)
		require.NoError(t, err)

		err = MigrateRPCEndpoint(tmpDir, "base-sepolia", "https://sepolia.base.org")
		require.NoError(t, err)

		envData, err := os.ReadFile(filepath.Join(tmpDir, ".env"))
		require.NoError(t, err)
		// Should not duplicate
		assert.Equal(t, "BASE_SEPOLIA_RPC_URL=https://old.url.org\n", string(envData))
	})

	t.Run("error when URL not found in foundry.toml", func(t *testing.T) {
		tmpDir := t.TempDir()
		foundryContent := `[rpc_endpoints]
sepolia = "${SEPOLIA_RPC_URL}"
`
		err := os.WriteFile(filepath.Join(tmpDir, "foundry.toml"), []byte(foundryContent), 0644)
		require.NoError(t, err)

		err = MigrateRPCEndpoint(tmpDir, "sepolia", "https://some.url.org")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not find entry")
	})
}
