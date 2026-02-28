package cli

import (
	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/internal/cli/render"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// NewConfigCmd creates the config command using the new architecture
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage treb local config",
		Long: `Manage treb local config stored in .treb/config.local.json

The config defines default values for namespace and network
that are used when these flags are not explicitly provided.

Available subcommands:
  config           Show current config
  config set       Set a config value
  config remove    Remove a config value

When run without subcommands, displays the current config.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default action is to show config
			return showConfig(cmd)
		},
	}

	// Add subcommands
	cmd.AddCommand(NewConfigSetCmd())
	cmd.AddCommand(NewConfigRemoveCmd())

	return cmd
}

// NewConfigSetCmd creates the config set subcommand
func NewConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value",
		Long: `Set a config value in the .treb file.
Available keys: namespace (ns), network

Examples:
  treb config set namespace production
  treb config set network sepolia`,
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get app from context
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			params := usecase.SetConfigParams{
				Key:   args[0],
				Value: args[1],
			}

			result, err := app.SetConfig.Run(cmd.Context(), params)
			if err != nil {
				return err
			}

			// Render result
			renderer := render.NewConfigRenderer(cmd.OutOrStdout())
			return renderer.RenderSet(result)
		},
	}
}

// NewConfigRemoveCmd creates the config remove subcommand
func NewConfigRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <key>",
		Short: "Remove a config value",
		Long: `Remove a config value from the .treb file.
Removing namespace reverts it to 'default'.
Removing network makes it unspecified (required as flags).

Examples:
  treb config remove namespace
  treb config remove network`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get app from context
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			params := usecase.RemoveConfigParams{
				Key: args[0],
			}

			result, err := app.RemoveConfig.Run(cmd.Context(), params)
			if err != nil {
				return err
			}

			// Render result
			renderer := render.NewConfigRenderer(cmd.OutOrStdout())
			return renderer.RenderRemove(result)
		},
	}
}

// showConfig displays the current configuration
func showConfig(cmd *cobra.Command) error {
	// Get app from context
	app, err := getApp(cmd)
	if err != nil {
		return err
	}

	result, err := app.ShowConfig.Run(cmd.Context())
	if err != nil {
		return err
	}

	// Enrich result with config source from runtime config
	result.ConfigSource = app.Config.ConfigSource

	// Render result
	renderer := render.NewConfigRenderer(cmd.OutOrStdout())
	return renderer.RenderConfig(result)
}
