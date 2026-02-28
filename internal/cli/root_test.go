package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

func TestGetDeprecationWarning(t *testing.T) {
	legacyFoundryConfig := &config.FoundryConfig{
		Profile: map[string]config.ProfileConfig{
			"default": {
				Treb: &config.TrebConfig{
					Senders: map[string]config.SenderConfig{
						"deployer": {Type: config.SenderTypePrivateKey},
					},
				},
			},
		},
	}

	noTrebFoundryConfig := &config.FoundryConfig{
		Profile: map[string]config.ProfileConfig{
			"default": {},
		},
	}

	tests := []struct {
		name     string
		cmdName  string
		cfg      *config.RuntimeConfig
		expected deprecationWarning
	}{
		{
			name:    "shows foundry.toml warning when legacy config detected",
			cmdName: "list",
			cfg: &config.RuntimeConfig{
				ConfigSource:  "foundry.toml",
				FoundryConfig: legacyFoundryConfig,
			},
			expected: foundryTomlWarning,
		},
		{
			name:    "shows v1 treb.toml warning when ns format detected",
			cmdName: "list",
			cfg: &config.RuntimeConfig{
				ConfigSource:  "treb.toml",
				FoundryConfig: legacyFoundryConfig,
			},
			expected: v1TrebTomlWarning,
		},
		{
			name:    "no warning for treb.toml v2 format",
			cmdName: "list",
			cfg: &config.RuntimeConfig{
				ConfigSource:  "treb.toml (v2)",
				FoundryConfig: legacyFoundryConfig,
			},
			expected: noWarning,
		},
		{
			name:    "suppressed for version command",
			cmdName: "version",
			cfg: &config.RuntimeConfig{
				ConfigSource:  "foundry.toml",
				FoundryConfig: legacyFoundryConfig,
			},
			expected: noWarning,
		},
		{
			name:    "suppressed for help command",
			cmdName: "help",
			cfg: &config.RuntimeConfig{
				ConfigSource:  "foundry.toml",
				FoundryConfig: legacyFoundryConfig,
			},
			expected: noWarning,
		},
		{
			name:    "suppressed for completion command",
			cmdName: "completion",
			cfg: &config.RuntimeConfig{
				ConfigSource:  "foundry.toml",
				FoundryConfig: legacyFoundryConfig,
			},
			expected: noWarning,
		},
		{
			name:    "suppressed for init command",
			cmdName: "init",
			cfg: &config.RuntimeConfig{
				ConfigSource:  "foundry.toml",
				FoundryConfig: legacyFoundryConfig,
			},
			expected: noWarning,
		},
		{
			name:    "suppressed for migrate command",
			cmdName: "migrate",
			cfg: &config.RuntimeConfig{
				ConfigSource:  "foundry.toml",
				FoundryConfig: legacyFoundryConfig,
			},
			expected: noWarning,
		},
		{
			name:    "suppressed when json flag is set for foundry.toml",
			cmdName: "list",
			cfg: &config.RuntimeConfig{
				ConfigSource:  "foundry.toml",
				JSON:          true,
				FoundryConfig: legacyFoundryConfig,
			},
			expected: noWarning,
		},
		{
			name:    "suppressed when json flag is set for v1 treb.toml",
			cmdName: "list",
			cfg: &config.RuntimeConfig{
				ConfigSource: "treb.toml",
				JSON:         true,
			},
			expected: noWarning,
		},
		{
			name:    "not shown when foundry.toml has no treb config",
			cmdName: "list",
			cfg: &config.RuntimeConfig{
				ConfigSource:  "foundry.toml",
				FoundryConfig: noTrebFoundryConfig,
			},
			expected: noWarning,
		},
		{
			name:    "not shown when foundry config is nil",
			cmdName: "list",
			cfg: &config.RuntimeConfig{
				ConfigSource: "foundry.toml",
			},
			expected: noWarning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getDeprecationWarning(tt.cmdName, tt.cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasLegacyTrebConfig(t *testing.T) {
	tests := []struct {
		name     string
		fc       *config.FoundryConfig
		expected bool
	}{
		{
			name:     "nil foundry config",
			fc:       nil,
			expected: false,
		},
		{
			name: "no treb config in profiles",
			fc: &config.FoundryConfig{
				Profile: map[string]config.ProfileConfig{
					"default": {},
				},
			},
			expected: false,
		},
		{
			name: "treb config in default profile",
			fc: &config.FoundryConfig{
				Profile: map[string]config.ProfileConfig{
					"default": {
						Treb: &config.TrebConfig{
							Senders: map[string]config.SenderConfig{
								"deployer": {Type: config.SenderTypePrivateKey},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "treb config in non-default profile",
			fc: &config.FoundryConfig{
				Profile: map[string]config.ProfileConfig{
					"default":    {},
					"production": {Treb: &config.TrebConfig{}},
				},
			},
			expected: true,
		},
		{
			name: "empty profiles map",
			fc: &config.FoundryConfig{
				Profile: map[string]config.ProfileConfig{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasLegacyTrebConfig(tt.fc)
			assert.Equal(t, tt.expected, result)
		})
	}
}
