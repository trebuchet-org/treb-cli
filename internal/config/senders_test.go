package config

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
)

func TestSendersManager_BuildSenderInitConfigs(t *testing.T) {
	tests := []struct {
		name           string
		trebConfig     *config.TrebConfig
		senders        []string
		envVars        map[string]string
		expectedError  string
		validateConfig func(t *testing.T, configs []config.SenderInitConfig)
	}{
		{
			name: "private key sender with env var",
			trebConfig: &config.TrebConfig{
				Senders: map[string]config.SenderConfig{
					"signer0": {
						Type:       "private_key",
						PrivateKey: "${TEST_PRIVATE_KEY}",
					},
				},
			},
			senders: []string{"signer0"},
			envVars: map[string]string{
				"TEST_PRIVATE_KEY": "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
			},
			validateConfig: func(t *testing.T, configs []config.SenderInitConfig) {
				require.Len(t, configs, 1)
				assert.Equal(t, "signer0", configs[0].Name)
				assert.Equal(t, SENDER_TYPE_IN_MEMORY, configs[0].SenderType)
				assert.True(t, configs[0].CanBroadcast)
				
				// Decode the config to check the private key
				configHex := hex.EncodeToString(configs[0].Config)
				assert.NotContains(t, configHex, strings.Repeat("00", 32), "Private key should not be all zeros")
			},
		},
		{
			name: "safe sender with signer",
			trebConfig: &config.TrebConfig{
				Senders: map[string]config.SenderConfig{
					"safe0": {
						Type:   "safe",
						Safe:   "0x3D33783D1fd1B6D849d299aD2E711f844fC16d2F",
						Signer: "signer0",
					},
					"signer0": {
						Type:       "private_key", 
						PrivateKey: "${TEST_PRIVATE_KEY}",
					},
				},
			},
			senders: []string{"safe0"},
			envVars: map[string]string{
				"TEST_PRIVATE_KEY": "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
			},
			validateConfig: func(t *testing.T, configs []config.SenderInitConfig) {
				require.Len(t, configs, 2) // Should include both safe and signer
				
				// Find the safe config
				var safeConfig, signerConfig *config.SenderInitConfig
				for i := range configs {
					if configs[i].Name == "safe0" {
						safeConfig = &configs[i]
					} else if configs[i].Name == "signer0" {
						signerConfig = &configs[i]
					}
				}
				
				require.NotNil(t, safeConfig, "Safe config not found")
				require.NotNil(t, signerConfig, "Signer config not found")
				
				// Validate safe config
				assert.Equal(t, SENDER_TYPE_GNOSIS_SAFE, safeConfig.SenderType)
				assert.Equal(t, common.HexToAddress("0x3D33783D1fd1B6D849d299aD2E711f844fC16d2F"), safeConfig.Account)
				
				// Validate signer config
				assert.Equal(t, SENDER_TYPE_IN_MEMORY, signerConfig.SenderType)
				assert.Equal(t, common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"), signerConfig.Account)
				
				// Check that the private key is not zero
				configHex := hex.EncodeToString(signerConfig.Config)
				assert.NotContains(t, configHex, strings.Repeat("00", 32), "Private key should not be all zeros")
			},
		},
		{
			name: "safe sender with missing signer",
			trebConfig: &config.TrebConfig{
				Senders: map[string]config.SenderConfig{
					"safe0": {
						Type:   "safe",
						Safe:   "0x3D33783D1fd1B6D849d299aD2E711f844fC16d2F",
						Signer: "missing_signer",
					},
				},
			},
			senders:       []string{"safe0"},
			expectedError: "safe signer 'missing_signer' not found in sender configurations",
		},
		{
			name: "private key with invalid hex",
			trebConfig: &config.TrebConfig{
				Senders: map[string]config.SenderConfig{
					"signer0": {
						Type:       "private_key",
						PrivateKey: "invalid_hex",
					},
				},
			},
			senders:       []string{"signer0"},
			expectedError: "invalid private key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}
			
			// Create runtime config with expanded env vars
			runtimeConfig := &config.RuntimeConfig{
				TrebConfig: tt.trebConfig,
			}
			
			// Expand environment variables in sender configs
			if runtimeConfig.TrebConfig != nil && runtimeConfig.TrebConfig.Senders != nil {
				for name, sender := range runtimeConfig.TrebConfig.Senders {
					sender.PrivateKey = os.ExpandEnv(sender.PrivateKey)
					sender.Safe = os.ExpandEnv(sender.Safe)
					sender.Address = os.ExpandEnv(sender.Address)
					sender.Signer = os.ExpandEnv(sender.Signer)
					sender.DerivationPath = os.ExpandEnv(sender.DerivationPath)
					runtimeConfig.TrebConfig.Senders[name] = sender
				}
			}
			
			// Create sender manager
			manager := NewSendersManager(runtimeConfig)
			
			var configs []config.SenderInitConfig
			var err error
			
			// For safe senders test, we need to test the full BuildSenderScriptConfig
			// because buildSenderInitConfigs is called with allSenders internally
			if tt.name == "safe sender with signer" {
				// Create a mock artifact
				artifact := &models.Artifact{
					Metadata: models.ArtifactMetadata{
						Output: struct {
							ABI      json.RawMessage `json:"abi"`
							DevDoc   json.RawMessage `json:"devdoc"`
							UserDoc  json.RawMessage `json:"userdoc"`
							Metadata string          `json:"metadata"`
						}{
							DevDoc: json.RawMessage(`{"methods":{"run()":{"custom:senders":"safe0"}}}`),
						},
					},
				}
				
				scriptConfig, err := manager.BuildSenderScriptConfig(artifact)
				if err != nil {
					t.Fatalf("BuildSenderScriptConfig failed: %v", err)
				}
				configs = scriptConfig.SenderInitConfigs
			} else {
				// Build sender init configs
				configs, err = manager.buildSenderInitConfigs(tt.senders)
			}
			
			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
				if tt.validateConfig != nil {
					tt.validateConfig(t, configs)
				}
			}
		})
	}
}

func TestSendersManager_BuildSenderScriptConfig(t *testing.T) {
	// Create a test artifact with custom:senders tag
	artifact := &models.Artifact{
		Metadata: models.ArtifactMetadata{
			Output: struct {
				ABI      json.RawMessage `json:"abi"`
				DevDoc   json.RawMessage `json:"devdoc"`
				UserDoc  json.RawMessage `json:"userdoc"`
				Metadata string          `json:"metadata"`
			}{
				DevDoc: json.RawMessage(`{"methods":{"run()":{"custom:senders":"safe0"}}}`),
			},
		},
	}
	
	// Set up test environment
	os.Setenv("TEST_PRIVATE_KEY", "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	defer os.Unsetenv("TEST_PRIVATE_KEY")
	
	trebConfig := &config.TrebConfig{
		Senders: map[string]config.SenderConfig{
			"safe0": {
				Type:   "safe",
				Safe:   "0x3D33783D1fd1B6D849d299aD2E711f844fC16d2F",
				Signer: "signer0",
			},
			"signer0": {
				Type:       "private_key",
				PrivateKey: "${TEST_PRIVATE_KEY}",
			},
		},
	}
	
	// Expand env vars
	for name, sender := range trebConfig.Senders {
		sender.PrivateKey = os.ExpandEnv(sender.PrivateKey)
		trebConfig.Senders[name] = sender
	}
	
	runtimeConfig := &config.RuntimeConfig{
		TrebConfig: trebConfig,
	}
	
	manager := NewSendersManager(runtimeConfig)
	
	// Build sender script config
	scriptConfig, err := manager.BuildSenderScriptConfig(artifact)
	require.NoError(t, err)
	
	// Verify the config
	assert.Len(t, scriptConfig.Senders, 1)
	assert.Equal(t, "safe0", scriptConfig.Senders[0])
	assert.Len(t, scriptConfig.SenderInitConfigs, 2) // safe0 and signer0
	assert.NotEmpty(t, scriptConfig.EncodedConfig)
	
	// Decode and verify the encoded config
	encodedBytes, err := hex.DecodeString(strings.TrimPrefix(scriptConfig.EncodedConfig, "0x"))
	require.NoError(t, err)
	assert.NotEmpty(t, encodedBytes)
	
	// Check that we have both safe0 and signer0 in the configs
	var foundSafe, foundSigner bool
	for _, config := range scriptConfig.SenderInitConfigs {
		if config.Name == "safe0" {
			foundSafe = true
			assert.Equal(t, SENDER_TYPE_GNOSIS_SAFE, config.SenderType)
		} else if config.Name == "signer0" {
			foundSigner = true
			assert.Equal(t, SENDER_TYPE_IN_MEMORY, config.SenderType)
			// Verify private key is not zeros
			configHex := hex.EncodeToString(config.Config)
			assert.NotContains(t, configHex, strings.Repeat("00", 32), "Private key should not be all zeros")
		}
	}
	assert.True(t, foundSafe, "Safe sender not found")
	assert.True(t, foundSigner, "Signer not found")
}

func TestParsePrivateKey(t *testing.T) {
	tests := []struct {
		name          string
		privateKeyHex string
		expectedAddr  string
		expectError   bool
	}{
		{
			name:          "valid private key with 0x prefix",
			privateKeyHex: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
			expectedAddr:  "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
			expectError:   false,
		},
		{
			name:          "valid private key without 0x prefix",
			privateKeyHex: "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
			expectedAddr:  "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
			expectError:   false,
		},
		{
			name:          "invalid hex",
			privateKeyHex: "invalid_hex",
			expectError:   true,
		},
		{
			name:          "empty string",
			privateKeyHex: "",
			expectError:   true,
		},
		{
			name:          "all zeros",
			privateKeyHex: "0x0000000000000000000000000000000000000000000000000000000000000000",
			expectError:   true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsePrivateKey(tt.privateKeyHex)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedAddr, result.Address.Hex())
			}
		})
	}
}