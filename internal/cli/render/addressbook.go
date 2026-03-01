package render

import (
	"fmt"
	"io"

	"github.com/fatih/color"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

var abNameStyle = color.New(color.FgYellow, color.Bold)

// AddressbookRenderer renders addressbook-related output.
type AddressbookRenderer struct {
	out io.Writer
}

// NewAddressbookRenderer creates a new addressbook renderer.
func NewAddressbookRenderer(out io.Writer) *AddressbookRenderer {
	return &AddressbookRenderer{out: out}
}

// RenderSet renders the result of setting an addressbook entry.
func (r *AddressbookRenderer) RenderSet(result *usecase.SetAddressbookResult) error {
	fmt.Fprintf(r.out, "Set %s = %s (chain %d)\n", result.Name, result.Address, result.ChainID)
	return nil
}

// RenderRemove renders the result of removing an addressbook entry.
func (r *AddressbookRenderer) RenderRemove(result *usecase.RemoveAddressbookResult) error {
	fmt.Fprintf(r.out, "Removed %s (chain %d)\n", result.Name, result.ChainID)
	return nil
}

// RenderList renders the addressbook listing.
func (r *AddressbookRenderer) RenderList(result *usecase.ListAddressbookResult) error {
	if len(result.Entries) == 0 {
		fmt.Fprintln(r.out, "No addressbook entries found")
		return nil
	}

	for _, entry := range result.Entries {
		fmt.Fprintf(r.out, "  %s  %s\n", abNameStyle.Sprintf("%-24s", entry.Name), addressStyle.Sprint(entry.Address))
	}

	return nil
}
