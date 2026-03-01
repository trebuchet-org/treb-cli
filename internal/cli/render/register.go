package render

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/fatih/color"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// RegisterRenderer renders register command output
type RegisterRenderer struct {
	out io.Writer
}

// NewRegisterRenderer creates a new register renderer
func NewRegisterRenderer(out io.Writer) *RegisterRenderer {
	return &RegisterRenderer{
		out: out,
	}
}

// RenderJSON renders the registration result as JSON
func (r *RegisterRenderer) RenderJSON(result *usecase.RegisterDeploymentResult) error {
	type deploymentJSON struct {
		DeploymentID string `json:"deploymentId"`
		Address      string `json:"address"`
		ContractName string `json:"contractName"`
		Label        string `json:"label"`
	}

	type outputJSON struct {
		Deployments []deploymentJSON `json:"deployments"`
	}

	out := outputJSON{
		Deployments: make([]deploymentJSON, len(result.DeploymentIDs)),
	}
	for i := range result.DeploymentIDs {
		out.Deployments[i] = deploymentJSON{
			DeploymentID: result.DeploymentIDs[i],
			Address:      result.Addresses[i],
			ContractName: result.ContractNames[i],
			Label:        result.Labels[i],
		}
	}

	data, err := json.Marshal(out)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Fprintln(r.out, string(data))
	return nil
}

// RenderSuccess renders the human-readable registration result
func (r *RegisterRenderer) RenderSuccess(result *usecase.RegisterDeploymentResult) {
	fmt.Fprintln(r.out, color.New(color.FgGreen, color.Bold).Sprintf("✓ Successfully registered %d deployment(s)", len(result.DeploymentIDs)))
	fmt.Fprintln(r.out)

	for i := range result.DeploymentIDs {
		fmt.Fprintf(r.out, "  Deployment %d:\n", i+1)
		fmt.Fprintf(r.out, "    Deployment ID: %s\n", cyan.Sprint(result.DeploymentIDs[i]))
		fmt.Fprintf(r.out, "    Address: %s\n", color.New(color.FgGreen).Sprint(result.Addresses[i]))
		fmt.Fprintf(r.out, "    Contract: %s\n", yellow.Sprint(result.ContractNames[i]))
		if result.Labels[i] != "" {
			fmt.Fprintf(r.out, "    Label: %s\n", result.Labels[i])
		}
		if i < len(result.DeploymentIDs)-1 {
			fmt.Fprintln(r.out)
		}
	}
}
