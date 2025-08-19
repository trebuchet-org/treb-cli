package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Provider creates RuntimeConfig for Wire dependency injection
func Provider(v *viper.Viper) (*RuntimeConfig, error) {
	// Get project root from viper
	projectRoot := v.GetString("project_root")
	if projectRoot == "" {
		// Try to find project root
		var err error
		projectRoot, err = FindProjectRoot()
		if err != nil {
			return nil, fmt.Errorf("failed to find project root: %w", err)
		}
	}

	cfg := &RuntimeConfig{
		ProjectRoot:    projectRoot,
		DataDir:        filepath.Join(projectRoot, ".treb"),
		Namespace:      v.GetString("namespace"),
		Debug:          v.GetBool("debug"),
		NonInteractive: v.GetBool("non_interactive"),
		JSON:           v.GetBool("json"),
		Timeout:        v.GetDuration("timeout"),
		DryRun:         v.GetBool("dry_run"),
	}

	// Load foundry config
	foundryConfig, err := loadFoundryConfig(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to load foundry config: %w", err)
	}
	cfg.FoundryConfig = foundryConfig

	// Load profile-specific treb config (namespace = profile)
	profile, ok := foundryConfig.Profile[cfg.Namespace]
	if ok {
		cfg.TrebConfig = profile.Treb
		if os.Getenv("TREB_TEST_DEBUG") != "" {
			fmt.Printf("DEBUG: Loaded TrebConfig for profile %s\n", cfg.Namespace)
			if cfg.TrebConfig != nil && cfg.TrebConfig.Senders != nil {
				fmt.Printf("DEBUG: Found %d senders\n", len(cfg.TrebConfig.Senders))
				for name, sender := range cfg.TrebConfig.Senders {
					fmt.Printf("DEBUG: Sender %s: type=%s\n", name, sender.Type)
				}
			} else {
				fmt.Printf("DEBUG: TrebConfig or Senders is nil\n")
			}
		}
	} else {
		if os.Getenv("TREB_TEST_DEBUG") != "" {
			fmt.Printf("DEBUG: Profile %s not found in foundry config\n", cfg.Namespace)
			fmt.Printf("DEBUG: Available profiles: %v\n", getMapKeys(foundryConfig.Profile))
		}
	}

	// Resolve network if specified
	if networkName := v.GetString("network"); networkName != "" {
		networkResolver := NewNetworkResolver(projectRoot, foundryConfig)
		network, err := networkResolver.Resolve(networkName)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve network %s: %w", networkName, err)
		}
		cfg.Network = network
	}

	return cfg, nil
}

// FindProjectRoot walks up from current directory to find foundry.toml
func FindProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		foundryToml := filepath.Join(dir, "foundry.toml")
		if _, err := os.Stat(foundryToml); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding foundry.toml
			return "", fmt.Errorf("not in a Foundry project (foundry.toml not found)")
		}
		dir = parent
	}
}

// SetupViper creates and configures a viper instance
func SetupViper(projectRoot string, cmd *cobra.Command) *viper.Viper {
	v := viper.New()

	// Set up config file
	v.SetConfigName("config.local")
	v.SetConfigType("json")
	v.AddConfigPath(filepath.Join(projectRoot, ".treb"))

	// Set up environment variables
	v.SetEnvPrefix("TREB")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	// Set defaults
	v.SetDefault("namespace", "default")
	v.SetDefault("timeout", "5m")
	v.SetDefault("debug", false)
	v.SetDefault("non_interactive", false)
	v.SetDefault("project_root", projectRoot)

	// Try to read config file (ignore error if not found)
	_ = v.ReadInConfig()

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		err := v.BindPFlag(f.Name, f)
		if err != nil {
			panic(err)
		}
	})

	return v
}

// // Only bind flags that exist and have been changed
// if f := cmd.Flag("debug"); f != nil && f.Changed {
// 	v.Set("debug", f.Value.String())
// }
// if f := cmd.Flag("non-interactive"); f != nil && f.Changed {
// 	v.Set("non_interactive", f.Value.String())
// }
// // Intentionally omit --json to preserve v1 compatibility in usage output
// if f := cmd.Flag("namespace"); f != nil && f.Changed {
// 	v.Set("namespace", f.Value.String())
// }
// if f := cmd.Flag("network"); f != nil && f.Changed {
// 	v.Set("network", f.Value.String())
// }

// ProvideNetworkResolver creates a NetworkResolver for Wire dependency injection
func ProvideNetworkResolver(cfg *RuntimeConfig) *NetworkResolver {
	return NewNetworkResolver(cfg.ProjectRoot, cfg.FoundryConfig)
}

// getMapKeys returns the keys of a map as a slice
func getMapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
