package template

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ScriptGeneratorAdapter generates deployment scripts using Go templates
type ScriptGeneratorAdapter struct {
	abiParser usecase.ABIParser
}

// NewScriptGeneratorAdapter creates a new script generator adapter
func NewScriptGeneratorAdapter(cfg *config.RuntimeConfig, abiParser usecase.ABIParser) (*ScriptGeneratorAdapter, error) {
	return &ScriptGeneratorAdapter{
		abiParser: abiParser,
	}, nil
}

// GenerateScript generates a deployment script from a template
func (g *ScriptGeneratorAdapter) GenerateScript(ctx context.Context, tmpl *domain.ScriptTemplate) (string, error) {
	switch tmpl.Type {
	case domain.ScriptTypeLibrary:
		return g.generateLibraryScript(tmpl)
	case domain.ScriptTypeProxy:
		return g.generateProxyScript(tmpl)
	case domain.ScriptTypeContract:
		return g.generateContractScript(tmpl)
	default:
		return "", fmt.Errorf("unknown script type: %s", tmpl.Type)
	}
}

// generateLibraryScript generates a library deployment script
func (g *ScriptGeneratorAdapter) generateLibraryScript(tmpl *domain.ScriptTemplate) (string, error) {
	const libraryTemplate = `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/src/TrebScript.sol";
import {Senders} from "treb-sol/src/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/src/internal/sender/Deployer.sol";

contract Deploy{{.ContractName}} is TrebScript {
    using Senders for Senders.Sender;
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    /**
     * @custom:env {sender:optional} deployer
     * @custom:senders anvil
     */
    function run() public broadcast {
        Senders.Sender storage deployer = sender(
            vm.envOr("deployer", string("anvil"))
        );

        deployer.create2("{{.ArtifactPath}}").deploy();
    }
}
`

	t := template.Must(template.New("library").Parse(libraryTemplate))
	var buf bytes.Buffer
	if err := t.Execute(&buf, tmpl); err != nil {
		return "", fmt.Errorf("failed to execute library template: %w", err)
	}
	return buf.String(), nil
}

// generateContractScript generates a contract deployment script
func (g *ScriptGeneratorAdapter) generateContractScript(tmpl *domain.ScriptTemplate) (string, error) {
	// Determine deployment method
	strategyMethod := "create3"
	if tmpl.Strategy == domain.StrategyCreate2 {
		strategyMethod = "create2"
	}

	// Build deployment call
	deployCall := fmt.Sprintf(`deployer.%s("%s")
            .setLabel(vm.envOr("LABEL", string("")))
            .deploy`, strategyMethod, tmpl.ArtifactPath)

	hasConstructor := tmpl.ConstructorInfo != nil && tmpl.ConstructorInfo.HasConstructor && len(tmpl.ConstructorInfo.Parameters) > 0
	if hasConstructor {
		deployCall += "(_getConstructorArgs());"
	} else {
		deployCall += "();"
	}

	// Generate constructor args if needed
	constructorSection := ""
	if hasConstructor {
		abi := &domain.ContractABI{
			HasConstructor: true,
			Constructor: &domain.Constructor{
				Inputs: tmpl.ConstructorInfo.Parameters,
			},
		}
		vars, encode := g.abiParser.GenerateConstructorArgs(abi)
		constructorSection = fmt.Sprintf(`

    /// @notice Get constructor arguments
    function _getConstructorArgs() internal pure returns (bytes memory) {
        // TODO: Update these constructor arguments
%s
        %s
    }`, vars, encode)
	}

	const contractTemplate = `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/src/TrebScript.sol";
import {Senders} from "treb-sol/src/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/src/internal/sender/Deployer.sol";

contract Deploy{{.ContractName}} is TrebScript {
    using Senders for Senders.Sender;
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    /**
     * @custom:env {sender:optional} deployer
     * @custom:senders anvil
     */
    function run() public broadcast {
        Senders.Sender storage deployer = sender(
            vm.envOr("deployer", string("anvil"))
        );

        // Deploy {{.ContractName}}
        {{.DeployCall}}
    }{{.ConstructorSection}}
}
`

	data := struct {
		*domain.ScriptTemplate
		DeployCall         string
		ConstructorSection string
	}{
		ScriptTemplate:     tmpl,
		DeployCall:         deployCall,
		ConstructorSection: constructorSection,
	}

	t := template.Must(template.New("contract").Parse(contractTemplate))
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute contract template: %w", err)
	}
	return buf.String(), nil
}

// generateProxyScript generates a proxy deployment script
func (g *ScriptGeneratorAdapter) generateProxyScript(tmpl *domain.ScriptTemplate) (string, error) {
	if tmpl.ProxyInfo == nil {
		return "", fmt.Errorf("proxy info is required for proxy scripts")
	}

	// Determine deployment method
	strategyMethod := "create3"
	if tmpl.Strategy == domain.StrategyCreate2 {
		strategyMethod = "create2"
	}

	// Generate initializer content
	initializerContent := ""
	if tmpl.ProxyInfo.InitializerInfo != nil && len(tmpl.ProxyInfo.InitializerInfo.Parameters) > 0 {
		method := &domain.Method{
			Name:   tmpl.ProxyInfo.InitializerInfo.MethodName,
			Inputs: tmpl.ProxyInfo.InitializerInfo.Parameters,
		}
		vars, encode := g.abiParser.GenerateInitializerArgs(method)
		initializerContent = fmt.Sprintf(`
        // TODO: Update these initializer arguments
%s
        %s`, vars, encode)
	} else {
		initializerContent = `
        // TODO: Update with initializer parameters
        // Example: return abi.encodeWithSignature("initialize(address)", owner);
        return "";`
	}

	const proxyTemplate = `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/src/TrebScript.sol";
import {Senders} from "treb-sol/src/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/src/internal/sender/Deployer.sol";
import { {{- .ProxyInfo.ProxyName -}} } from "{{.ProxyInfo.ProxyPath}}";

contract Deploy{{.ContractName}}Proxy is TrebScript {
    using Senders for Senders.Sender;
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    /**
     * @custom:env {sender:optional} deployer
     * @custom:env {string:optional} implementationLabel
     * @custom:env {string:optional} proxyLabel
     * @custom:senders anvil
     */
    function run() public broadcast {
        Senders.Sender storage deployer = sender(
            vm.envOr("deployer", string("anvil"))
        );

        // Deploy implementation
        address implementation = deployer
            .{{.StrategyMethod}}("{{.ArtifactPath}}")
            .setLabel(vm.envOr("implementationLabel", string("")))
            .deploy();

        // Deploy proxy
        deployer
            .{{.StrategyMethod}}("{{.ProxyInfo.ProxyArtifact}}")
            .setLabel(vm.envOr("proxyLabel", string("{{.ContractName}}")))
            .deploy(_getProxyConstructorArgs(implementation));
    }

    function _getProxyConstructorArgs(address implementation) internal pure returns (bytes memory) {
        // TODO: Update based on proxy type
        // For TransparentUpgradeableProxy:
        // return abi.encode(implementation, proxyAdmin, initData);
        
        // For UUPS/ERC1967 proxy:
        // return abi.encode(implementation, initData);
        
        bytes memory initData = _getInitializerData();
        return abi.encode(implementation, initData);
    }

    function _getInitializerData() internal pure returns (bytes memory) { {{- .InitializerContent}}
    }
}
`

	data := struct {
		*domain.ScriptTemplate
		StrategyMethod     string
		InitializerContent string
	}{
		ScriptTemplate:     tmpl,
		StrategyMethod:     strategyMethod,
		InitializerContent: initializerContent,
	}

	t := template.Must(template.New("proxy").Parse(proxyTemplate))
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute proxy template: %w", err)
	}
	return buf.String(), nil
}

// Ensure the adapter implements the interface
var _ usecase.ScriptGenerator = (*ScriptGeneratorAdapter)(nil)