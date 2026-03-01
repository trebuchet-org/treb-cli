package render

import (
	"fmt"
	"io"
	"strconv"

	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// NetworksRenderer renders network lists
type NetworksRenderer struct {
	out   io.Writer
	color bool
}

// NewNetworksRenderer creates a new networks renderer
func NewNetworksRenderer(out io.Writer, color bool) *NetworksRenderer {
	return &NetworksRenderer{
		out:   out,
		color: color,
	}
}

// RenderNetworksList renders the list of networks in the same format as v1
func (r *NetworksRenderer) RenderNetworksList(result *usecase.ListNetworksResult) error {
	if len(result.Networks) == 0 {
		fmt.Fprintln(r.out, "No networks configured in foundry.toml [rpc_endpoints]")
		return nil
	}

	fmt.Fprintln(r.out, "🌐 Available Networks:")
	fmt.Fprintln(r.out)

	// Calculate max widths for alignment
	maxNameWidth := 0
	maxChainIDWidth := 0
	for _, network := range result.Networks {
		if len(network.Name) > maxNameWidth {
			maxNameWidth = len(network.Name)
		}
		if network.Error == nil {
			w := len(strconv.FormatUint(network.ChainID, 10))
			if w > maxChainIDWidth {
				maxChainIDWidth = w
			}
		}
	}

	// Render each network
	for _, network := range result.Networks {
		if network.Error != nil {
			fmt.Fprintf(r.out, "  ❌ %-*s - ", maxNameWidth, network.Name)
			red.Fprintf(r.out, "Error: %v", network.Error)
			fmt.Fprintln(r.out)
		} else {
			fmt.Fprintf(r.out, "  ✅ ")
			cyan.Fprintf(r.out, "%-*s", maxNameWidth, network.Name)
			fmt.Fprintf(r.out, " - Chain ID: %*d\n", maxChainIDWidth, network.ChainID)
		}
	}

	return nil
}
