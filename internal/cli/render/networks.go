package render

import (
	"fmt"
	"io"

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

	// Render each network
	for _, network := range result.Networks {
		if network.Error != nil {
			fmt.Fprintf(r.out, "  ❌ %s - Error: %v\n", network.Name, network.Error)
		} else {
			fmt.Fprintf(r.out, "  ✅ %s - Chain ID: %d\n", network.Name, network.ChainID)
		}
	}

	return nil
}