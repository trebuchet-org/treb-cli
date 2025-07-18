package parameters

import (
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/trebuchet-org/treb-cli/cli/pkg/interactive"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// ParameterPrompter handles interactive prompting for missing parameters
type ParameterPrompter struct {
	resolver *ParameterResolver
}

// NewParameterPrompter creates a new parameter prompter
func NewParameterPrompter(resolver *ParameterResolver) *ParameterPrompter {
	return &ParameterPrompter{
		resolver: resolver,
	}
}

// PromptForMissingParameters prompts for any missing required parameters
func (p *ParameterPrompter) PromptForMissingParameters(params []Parameter, envVars map[string]string) (map[string]string, error) {
	result := make(map[string]string)

	// Copy existing values
	for k, v := range envVars {
		result[k] = v
	}

	for _, param := range params {
		// Skip if already provided or optional
		if _, exists := result[param.Name]; exists && result[param.Name] != "" {
			continue
		}
		if param.Optional {
			continue
		}

		// Prompt based on type
		value, err := p.promptForParameter(param)
		if err != nil {
			return nil, fmt.Errorf("failed to prompt for %s: %w", param.Name, err)
		}

		result[param.Name] = value
	}

	return result, nil
}

// promptForParameter prompts for a single parameter based on its type
func (p *ParameterPrompter) promptForParameter(param Parameter) (string, error) {
	promptMsg := fmt.Sprintf("%s (%s): %s", param.Name, param.Type, param.Description)

	switch param.Type {
	case TypeString:
		return p.promptString(promptMsg)

	case TypeAddress:
		return p.promptAddress(promptMsg)

	case TypeUint256, TypeInt256:
		return p.promptNumber(promptMsg, param.Type)

	case TypeBytes32, TypeBytes:
		return p.promptBytes(promptMsg, param.Type)

	case TypeSender:
		return p.promptSender(promptMsg)

	case TypeDeployment:
		return p.promptDeployment(promptMsg)

	case TypeArtifact:
		return p.promptArtifact(promptMsg)

	default:
		return "", fmt.Errorf("unsupported parameter type: %s", param.Type)
	}
}

// promptString prompts for a string value
func (p *ParameterPrompter) promptString(message string) (string, error) {
	prompt := promptui.Prompt{
		Label: message,
	}
	return prompt.Run()
}

// promptAddress prompts for an address value with validation
func (p *ParameterPrompter) promptAddress(message string) (string, error) {
	validate := func(input string) error {
		if input == "" {
			return fmt.Errorf("address cannot be empty")
		}
		parser := NewParameterParser()
		return parser.ValidateValue(Parameter{Type: TypeAddress}, input)
	}

	prompt := promptui.Prompt{
		Label:    message,
		Validate: validate,
	}
	return prompt.Run()
}

// promptNumber prompts for a numeric value
func (p *ParameterPrompter) promptNumber(message string, paramType ParameterType) (string, error) {
	validate := func(input string) error {
		if input == "" {
			return fmt.Errorf("number cannot be empty")
		}
		parser := NewParameterParser()
		return parser.ValidateValue(Parameter{Type: paramType}, input)
	}

	prompt := promptui.Prompt{
		Label:    message + " (decimal or hex)",
		Validate: validate,
	}
	return prompt.Run()
}

// promptBytes prompts for a bytes value
func (p *ParameterPrompter) promptBytes(message string, paramType ParameterType) (string, error) {
	labelSuffix := " (hex starting with 0x)"
	if paramType == TypeBytes32 {
		labelSuffix = " (0x + 64 hex chars)"
	}

	validate := func(input string) error {
		if input == "" {
			return fmt.Errorf("bytes value cannot be empty")
		}
		parser := NewParameterParser()
		return parser.ValidateValue(Parameter{Type: paramType}, input)
	}

	prompt := promptui.Prompt{
		Label:    message + labelSuffix,
		Validate: validate,
	}
	return prompt.Run()
}

// promptSender prompts for a sender selection
func (p *ParameterPrompter) promptSender(message string) (string, error) {
	if p.resolver.trebConfig == nil {
		return "", fmt.Errorf("no treb configuration found")
	}

	// Get available senders
	var senderIDs []string
	for id := range p.resolver.trebConfig.Senders {
		senderIDs = append(senderIDs, id)
	}

	if len(senderIDs) == 0 {
		return "", fmt.Errorf("no senders configured")
	}

	// If only one sender, use it
	if len(senderIDs) == 1 {
		fmt.Printf("%s: %s (only available sender)\n", message, senderIDs[0])
		return senderIDs[0], nil
	}

	// Prompt for selection
	prompt := promptui.Select{
		Label: message,
		Items: senderIDs,
	}

	_, result, err := prompt.Run()
	return result, err
}

// promptDeployment prompts for a deployment selection
func (p *ParameterPrompter) promptDeployment(message string) (string, error) {
	// Get all deployments for the namespace and chain
	deployments := p.resolver.registryManager.GetAllDeployments()

	// Filter by namespace and chain
	var matches []*types.Deployment
	for _, d := range deployments {
		if d.Namespace == p.resolver.namespace && d.ChainID == p.resolver.chainID {
			matches = append(matches, d)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no deployments found for namespace %s on chain %d",
			p.resolver.namespace, p.resolver.chainID)
	}

	deployment, err := interactive.PickDeployment(matches, "Select deployment for: "+message)
	if err != nil {
		return "", err
	}

	// Return the deployment reference (contract:label or just contract)
	ref := deployment.ContractName
	if deployment.Label != "" {
		ref = fmt.Sprintf("%s:%s", deployment.ContractName, deployment.Label)
	}
	return ref, nil
}

// promptArtifact prompts for an artifact selection
func (p *ParameterPrompter) promptArtifact(message string) (string, error) {
	// Get all contracts using the indexer
	allContracts := p.resolver.lookup.contracts.QueryContracts(types.AllContractsFilter())
	if len(allContracts) == 0 {
		return "", fmt.Errorf("no contracts found")
	}

	contract, err := interactive.SelectContract(allContracts, "Select artifact for: "+message)
	if err != nil {
		return "", err
	}

	return contract.Name, nil
}
