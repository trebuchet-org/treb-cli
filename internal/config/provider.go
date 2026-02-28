package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

// Provider creates RuntimeConfig for Wire dependency injection
func Provider(v *viper.Viper) (*config.RuntimeConfig, error) {
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

	cfg := &config.RuntimeConfig{
		ProjectRoot:    projectRoot,
		DataDir:        filepath.Join(projectRoot, ".treb"),
		Namespace:      v.GetString("namespace"),
		Debug:          v.GetBool("debug"),
		NonInteractive: v.GetBool("non_interactive"),
		JSON:           v.GetBool("json"),
		Timeout:        v.GetDuration("timeout"),
		DryRun:         v.GetBool("dry_run"),
	}

	// Load foundry config (always needed for network resolution etc.)
	foundryConfig, err := loadFoundryConfig(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to load foundry config: %w", err)
	}
	cfg.FoundryConfig = foundryConfig

	// Detect treb.toml format version and load accordingly
	trebFormat, err := detectTrebConfigFormat(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to detect treb config format: %w", err)
	}

	switch trebFormat {
	case TrebConfigFormatV2:
		v2Config, err := loadTrebConfigV2(projectRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to load treb config v2: %w", err)
		}
		resolved, err := ResolveNamespace(v2Config, cfg.Namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve namespace %q: %w", cfg.Namespace, err)
		}
		trebConfig, err := ResolvedNamespaceToTrebConfig(resolved, v2Config.Accounts)
		if err != nil {
			return nil, fmt.Errorf("failed to convert resolved namespace: %w", err)
		}
		cfg.ConfigSource = "treb.toml (v2)"
		cfg.TrebConfig = trebConfig
		cfg.FoundryProfile = resolved.Profile
		if cfg.FoundryProfile == "" {
			cfg.FoundryProfile = cfg.Namespace
		}
		cfg.ForkSetup = v2Config.Fork.Setup

	case TrebConfigFormatV1:
		trebFileConfig, err := loadTrebConfig(projectRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to load treb config: %w", err)
		}
		if trebFileConfig != nil {
			cfg.ConfigSource = "treb.toml"
			cfg.TrebConfig, cfg.FoundryProfile = mergeTrebFileConfig(trebFileConfig, cfg.Namespace)
		} else {
			cfg.ConfigSource = "foundry.toml"
			cfg.FoundryProfile = cfg.Namespace
			cfg.TrebConfig = mergeFoundryTrebConfig(foundryConfig, cfg.Namespace)
		}
		// Backwards compat: read forkSetup from config.local.json via viper
		cfg.ForkSetup = v.GetString("forksetup")

	default:
		cfg.ConfigSource = "foundry.toml"
		cfg.FoundryProfile = cfg.Namespace
		cfg.TrebConfig = mergeFoundryTrebConfig(foundryConfig, cfg.Namespace)
		// Backwards compat: read forkSetup from config.local.json via viper
		cfg.ForkSetup = v.GetString("forksetup")
	}

	// Resolve slow mode: default to true if not configured
	cfg.Slow = true
	if cfg.TrebConfig != nil && cfg.TrebConfig.Slow != nil {
		cfg.Slow = *cfg.TrebConfig.Slow
	}

	if os.Getenv("TREB_DEBUG") != "" {
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

	// Resolve network if specified
	if networkName := v.GetString("network"); networkName != "" {
		networkResolver := NewNetworkResolver(projectRoot, foundryConfig)
		network, err := networkResolver.ResolveNetwork(context.Background(), networkName)
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
	nameFormatter := strings.NewReplacer("-", "_", ".", "_")

	// Set up config file
	v.SetConfigName("config.local")
	v.SetConfigType("json")
	v.AddConfigPath(filepath.Join(projectRoot, ".treb"))

	// Set up environment variables
	v.SetEnvPrefix("TREB")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(nameFormatter)

	// Set defaults
	v.SetDefault("namespace", "default")
	v.SetDefault("timeout", "5m")
	v.SetDefault("debug", false)
	v.SetDefault("non_interactive", false)
	v.SetDefault("project_root", projectRoot)

	// Try to read config file (ignore error if not found)
	_ = v.ReadInConfig()

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		name := nameFormatter.Replace(f.Name)
		err := v.BindPFlag(name, f)
		if err != nil {
			panic(err)
		}
	})

	return v
}

// mergeTrebFileConfig builds a merged TrebConfig from treb.toml namespaces.
// It starts with ns.default senders, then overlays ns.<namespace> senders.
// Returns the merged TrebConfig and the resolved Foundry profile name.
func mergeTrebFileConfig(trebFile *config.TrebFileConfig, namespace string) (*config.TrebConfig, string) {
	merged := &config.TrebConfig{
		Senders: make(map[string]config.SenderConfig),
	}

	// Start with top-level slow setting
	merged.Slow = trebFile.Slow

	// Start with default namespace senders
	if defaultNs, ok := trebFile.Ns["default"]; ok {
		for k, v := range defaultNs.Senders {
			merged.Senders[k] = v
		}
	}

	// Resolve foundry profile: default to namespace name
	foundryProfile := namespace

	// Overlay active namespace senders (if not "default")
	if namespace != "default" {
		if activeNs, ok := trebFile.Ns[namespace]; ok {
			for k, v := range activeNs.Senders {
				merged.Senders[k] = v
			}
			foundryProfile = activeNs.Profile
			// Per-namespace slow overrides top-level
			if activeNs.Slow != nil {
				merged.Slow = activeNs.Slow
			}
		}
	} else if defaultNs, ok := trebFile.Ns["default"]; ok {
		foundryProfile = defaultNs.Profile
		// Per-namespace slow overrides top-level
		if defaultNs.Slow != nil {
			merged.Slow = defaultNs.Slow
		}
	}

	return merged, foundryProfile
}

// mergeFoundryTrebConfig builds a merged TrebConfig from foundry.toml profiles.
// This preserves the legacy behavior: start with profile.default.treb, overlay profile.<namespace>.treb.
func mergeFoundryTrebConfig(foundryConfig *config.FoundryConfig, namespace string) *config.TrebConfig {
	var merged *config.TrebConfig

	// Start with default profile if it exists
	if defaultProfile, ok := foundryConfig.Profile["default"]; ok {
		merged = &config.TrebConfig{
			Senders: make(map[string]config.SenderConfig),
		}
		if defaultProfile.Treb != nil {
			merged.Slow = defaultProfile.Treb.Slow
			if defaultProfile.Treb.Senders != nil {
				for k, v := range defaultProfile.Treb.Senders {
					merged.Senders[k] = v
				}
			}
		}
	}

	// If requesting a specific profile, merge it with default
	if namespace != "default" {
		if profile, ok := foundryConfig.Profile[namespace]; ok {
			if merged == nil {
				merged = profile.Treb
			} else if profile.Treb != nil {
				// Per-namespace slow overrides default
				if profile.Treb.Slow != nil {
					merged.Slow = profile.Treb.Slow
				}
				if profile.Treb.Senders != nil {
					for k, v := range profile.Treb.Senders {
						merged.Senders[k] = v
					}
				}
			}
		}
	}

	return merged
}

// ProvideNetworkResolver creates a NetworkResolver for Wire dependency injection
func ProvideNetworkResolver(cfg *config.RuntimeConfig) *NetworkResolver {
	return NewNetworkResolver(cfg.ProjectRoot, cfg.FoundryConfig)
}
