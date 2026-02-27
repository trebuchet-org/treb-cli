package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

func TestShouldShowDeprecationWarning(t *testing.T) {
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
		expected bool
	}{
		{
			name:    "shows warning when legacy config detected",
			cmdName: "list",
			cfg: &config.RuntimeConfig{
				ConfigSource:  "foundry.toml",
				FoundryConfig: legacyFoundryConfig,
			},
			expected: true,
		},
		{
			name:    "suppressed when treb.toml exists",
			cmdName: "list",
			cfg: &config.RuntimeConfig{
				ConfigSource:  "treb.toml",
				FoundryConfig: legacyFoundryConfig,
			},
			expected: false,
		},
		{
			name:    "suppressed for version command",
			cmdName: "version",
			cfg: &config.RuntimeConfig{
				ConfigSource:  "foundry.toml",
				FoundryConfig: legacyFoundryConfig,
			},
			expected: false,
		},
		{
			name:    "suppressed for help command",
			cmdName: "help",
			cfg: &config.RuntimeConfig{
				ConfigSource:  "foundry.toml",
				FoundryConfig: legacyFoundryConfig,
			},
			expected: false,
		},
		{
			name:    "suppressed for completion command",
			cmdName: "completion",
			cfg: &config.RuntimeConfig{
				ConfigSource:  "foundry.toml",
				FoundryConfig: legacyFoundryConfig,
			},
			expected: false,
		},
		{
			name:    "suppressed for init command",
			cmdName: "init",
			cfg: &config.RuntimeConfig{
				ConfigSource:  "foundry.toml",
				FoundryConfig: legacyFoundryConfig,
			},
			expected: false,
		},
		{
			name:    "suppressed for migrate-config command",
			cmdName: "migrate-config",
			cfg: &config.RuntimeConfig{
				ConfigSource:  "foundry.toml",
				FoundryConfig: legacyFoundryConfig,
			},
			expected: false,
		},
		{
			name:    "suppressed when json flag is set",
			cmdName: "list",
			cfg: &config.RuntimeConfig{
				ConfigSource:  "foundry.toml",
				JSON:          true,
				FoundryConfig: legacyFoundryConfig,
			},
			expected: false,
		},
		{
			name:    "not shown when foundry.toml has no treb config",
			cmdName: "list",
			cfg: &config.RuntimeConfig{
				ConfigSource:  "foundry.toml",
				FoundryConfig: noTrebFoundryConfig,
			},
			expected: false,
		},
		{
			name:    "not shown when foundry config is nil",
			cmdName: "list",
			cfg: &config.RuntimeConfig{
				ConfigSource: "foundry.toml",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldShowDeprecationWarning(tt.cmdName, tt.cfg)
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
