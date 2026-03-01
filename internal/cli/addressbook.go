package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/internal/cli/render"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// NewAddressbookCmd creates the addressbook command.
func NewAddressbookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "addressbook",
		Aliases: []string{"ab"},
		Short:   "Manage the addressbook",
		Long: `Manage named addresses in the project addressbook (.treb/addressbook.json).

Entries are scoped to the current network's chain ID.
The addressbook is also available to Solidity scripts via Registry.sol.

When run without subcommands, lists all entries for the current chain.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listAddressbook(cmd)
		},
	}

	cmd.AddCommand(NewAddressbookSetCmd())
	cmd.AddCommand(NewAddressbookRemoveCmd())
	cmd.AddCommand(NewAddressbookListCmd())

	// Global flags for namespace/network
	cmd.PersistentFlags().StringP("namespace", "s", "", "Namespace to use")
	cmd.PersistentFlags().StringP("network", "n", "", "Network to run on")

	return cmd
}

// NewAddressbookSetCmd creates the addressbook set subcommand.
func NewAddressbookSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <name> <address>",
		Short: "Set an addressbook entry",
		Long: `Add or update a named address in the addressbook for the current chain.

Examples:
  treb addressbook set WETH 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2
  treb ab set USDC 0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48`,
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			params := usecase.SetAddressbookParams{
				Name:    args[0],
				Address: args[1],
			}

			result, err := app.SetAddressbook.Run(cmd.Context(), params)
			if err != nil {
				return err
			}

			renderer := render.NewAddressbookRenderer(cmd.OutOrStdout())
			return renderer.RenderSet(result)
		},
	}
}

// NewAddressbookRemoveCmd creates the addressbook remove subcommand.
func NewAddressbookRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove an addressbook entry",
		Long: `Remove a named address from the addressbook for the current chain.

Examples:
  treb addressbook remove WETH
  treb ab remove USDC`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			params := usecase.RemoveAddressbookParams{
				Name: args[0],
			}

			result, err := app.RemoveAddressbook.Run(cmd.Context(), params)
			if err != nil {
				return err
			}

			renderer := render.NewAddressbookRenderer(cmd.OutOrStdout())
			return renderer.RenderRemove(result)
		},
	}
}

// NewAddressbookListCmd creates the addressbook list subcommand.
func NewAddressbookListCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List addressbook entries",
		Long: `List all named addresses in the addressbook for the current chain.

Examples:
  treb addressbook list
  treb ab ls
  treb ab ls --json`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonOutput {
				return listAddressbookJSON(cmd)
			}
			return listAddressbook(cmd)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	return cmd
}

func listAddressbook(cmd *cobra.Command) error {
	app, err := getApp(cmd)
	if err != nil {
		return err
	}

	result, err := app.ListAddressbook.Run(cmd.Context())
	if err != nil {
		return err
	}

	renderer := render.NewAddressbookRenderer(cmd.OutOrStdout())
	return renderer.RenderList(result)
}

// addressbookJSONEntry represents a single addressbook entry in JSON output.
type addressbookJSONEntry struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

func listAddressbookJSON(cmd *cobra.Command) error {
	app, err := getApp(cmd)
	if err != nil {
		return err
	}

	result, err := app.ListAddressbook.Run(cmd.Context())
	if err != nil {
		return err
	}

	entries := make([]addressbookJSONEntry, 0, len(result.Entries))
	for _, e := range result.Entries {
		entries = append(entries, addressbookJSONEntry{
			Name:    e.Name,
			Address: e.Address,
		})
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
