package forge

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

func newTestForgeAdapter() *ForgeAdapter {
	return NewForgeAdapter("/tmp/test", slog.New(slog.NewTextHandler(os.Stderr, nil)))
}

func baseRunScriptConfig() usecase.RunScriptConfig {
	return usecase.RunScriptConfig{
		Network: &config.Network{
			Name:    "sepolia",
			ChainID: 11155111,
			RPCURL:  "https://sepolia.example.com",
		},
		Namespace:      "default",
		FoundryProfile: "default",
		Script:         &models.Contract{Name: "DeployCounter", Path: "script/deploy/DeployCounter.s.sol"},
		Parameters:     map[string]string{"LABEL": "v1"},
		SenderScriptConfig: config.SenderScriptConfig{
			EncodedConfig: "encoded",
		},
	}
}

func envToMap(envStrings []string) map[string]string {
	m := make(map[string]string)
	for _, s := range envStrings {
		for i := 0; i < len(s); i++ {
			if s[i] == '=' {
				m[s[:i]] = s[i+1:]
				break
			}
		}
	}
	return m
}

func TestBuildEnv_WithForkOverride(t *testing.T) {
	adapter := newTestForgeAdapter()
	cfg := baseRunScriptConfig()
	cfg.ForkEnvOverrides = map[string]string{
		"SEPOLIA_RPC_URL": "http://127.0.0.1:54321",
	}

	env := adapter.buildEnv(cfg)
	envMap := envToMap(env)

	assert.Equal(t, "http://127.0.0.1:54321", envMap["SEPOLIA_RPC_URL"])
	assert.Equal(t, "default", envMap["FOUNDRY_PROFILE"])
	assert.Equal(t, "sepolia", envMap["NETWORK"])
	assert.Equal(t, "v1", envMap["LABEL"])
}

func TestBuildEnv_WithoutForkOverride(t *testing.T) {
	adapter := newTestForgeAdapter()
	cfg := baseRunScriptConfig()

	env := adapter.buildEnv(cfg)
	envMap := envToMap(env)

	_, hasForkVar := envMap["SEPOLIA_RPC_URL"]
	assert.False(t, hasForkVar, "should not have fork RPC override when no fork is active")
	assert.Equal(t, "default", envMap["FOUNDRY_PROFILE"])
	assert.Equal(t, "sepolia", envMap["NETWORK"])
}

func TestBuildEnv_ForkOverrideForDifferentNetwork(t *testing.T) {
	adapter := newTestForgeAdapter()
	cfg := baseRunScriptConfig()
	// Fork is active for mainnet, not for sepolia (which is the current network)
	cfg.ForkEnvOverrides = map[string]string{
		"MAINNET_RPC_URL": "http://127.0.0.1:54321",
	}

	env := adapter.buildEnv(cfg)
	envMap := envToMap(env)

	// The mainnet override IS in the env (it's just data passed through),
	// but crucially the RunScript use case only populates this when the fork
	// matches the current network
	assert.Equal(t, "http://127.0.0.1:54321", envMap["MAINNET_RPC_URL"])
	_, hasSepolia := envMap["SEPOLIA_RPC_URL"]
	assert.False(t, hasSepolia, "sepolia var should not exist")
}

func TestBuildEnv_NilForkOverrides(t *testing.T) {
	adapter := newTestForgeAdapter()
	cfg := baseRunScriptConfig()
	cfg.ForkEnvOverrides = nil

	env := adapter.buildEnv(cfg)
	envMap := envToMap(env)

	assert.Equal(t, "default", envMap["FOUNDRY_PROFILE"])
	assert.Equal(t, "sepolia", envMap["NETWORK"])
}
