package render

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ConfigRenderer renders config-related output
type ConfigRenderer struct {
	out io.Writer
}

// NewConfigRenderer creates a new config renderer
func NewConfigRenderer(out io.Writer) *ConfigRenderer {
	return &ConfigRenderer{
		out: out,
	}
}

// getRelativePath returns the relative path from current directory
func getRelativePath(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}

	relPath, err := filepath.Rel(cwd, path)
	if err != nil {
		return path
	}

	return relPath
}

// RenderConfig renders the configuration display
func (r *ConfigRenderer) RenderConfig(result *usecase.ShowConfigResult) error {
	if !result.Exists {
		fmt.Fprintf(r.out, "❌ No .treb/config.local.json file found\n")
		fmt.Fprintf(r.out, "⚠️  Without config, commands require explicit --namespace and --network flags\n")
		return nil
	}

	fmt.Fprintln(r.out, "📋 Current config:")

	// Show namespace (always has a value)
	fmt.Fprintf(r.out, "Namespace: %s\n", result.Config.Namespace)

	// Show network (may be empty)
	if result.Config.Network != "" {
		fmt.Fprintf(r.out, "Network:   %s\n", result.Config.Network)
	} else {
		fmt.Fprintf(r.out, "Network:   %s\n", "(not set)")
	}

	// Show config source
	if result.ConfigSource == "treb.toml" {
		fmt.Fprintf(r.out, "\n📦 Config source: treb.toml\n")
	} else if result.ConfigSource == "foundry.toml" {
		fmt.Fprintf(r.out, "\n📦 Config source: foundry.toml (legacy)\n")
	}

	fmt.Fprintf(r.out, "📁 config file: %s\n", getRelativePath(result.ConfigPath))

	return nil
}

// RenderSet renders the result of setting a configuration value
func (r *ConfigRenderer) RenderSet(result *usecase.SetConfigResult) error {
	fmt.Fprintf(r.out, "✅ Set %s to: %s\n", result.Key, result.Value)
	fmt.Fprintf(r.out, "📁 config saved to: %s\n", getRelativePath(result.ConfigPath))
	return nil
}

// RenderRemove renders the result of removing a configuration value
func (r *ConfigRenderer) RenderRemove(result *usecase.RemoveConfigResult) error {
	switch result.Key {
	case config.ConfigKeyNamespace:
		fmt.Fprintf(r.out, "✅ Reset namespace to: default\n")
	case config.ConfigKeyNetwork:
		fmt.Fprintf(r.out, "✅ Removed network from config (will be required as flag)\n")
	}

	fmt.Fprintf(r.out, "📁 config saved to: %s\n", getRelativePath(result.ConfigPath))
	return nil
}
