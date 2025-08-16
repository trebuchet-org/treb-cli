package adapters

import (
	"github.com/google/wire"
	"github.com/trebuchet-org/treb-cli/internal/adapters/anvil"
	"github.com/trebuchet-org/treb-cli/internal/adapters/blockchain"
	internalconfig "github.com/trebuchet-org/treb-cli/internal/adapters/config"
	"github.com/trebuchet-org/treb-cli/internal/adapters/forge"
	"github.com/trebuchet-org/treb-cli/internal/adapters/fs"
	"github.com/trebuchet-org/treb-cli/internal/adapters/interactive"
	"github.com/trebuchet-org/treb-cli/internal/adapters/safe"
	"github.com/trebuchet-org/treb-cli/internal/adapters/template"
	"github.com/trebuchet-org/treb-cli/internal/adapters/verification"
	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// Removed ProvideContractsIndexer - no longer needed as we use fs.ContractIndexerAdapter

// ProvideProjectPath provides the project path from RuntimeConfig
func ProvideProjectPath(cfg *config.RuntimeConfig) string {
	return cfg.ProjectRoot
}

// FSSet provides filesystem-based implementations
var FSSet = wire.NewSet(
	fs.NewRegistryStoreAdapter,
	wire.Bind(new(usecase.DeploymentStore), new(*fs.RegistryStoreAdapter)),
	wire.Bind(new(usecase.TransactionStore), new(*fs.RegistryStoreAdapter)),
	wire.Bind(new(usecase.SafeTransactionStore), new(*fs.RegistryStoreAdapter)),
	wire.Bind(new(usecase.RegistryPruner), new(*fs.RegistryStoreAdapter)),
	
	fs.NewFileWriterAdapter,
	wire.Bind(new(usecase.FileWriter), new(*fs.FileWriterAdapter)),
	
	fs.NewLocalConfigStoreAdapter,
	wire.Bind(new(usecase.LocalConfigStore), new(*fs.LocalConfigStoreAdapter)),
)

// ForgeSet provides forge-based implementations
var ForgeSet = wire.NewSet(
	forge.NewForgeExecutorAdapter,
	wire.Bind(new(usecase.ForgeExecutor), new(*forge.ForgeExecutorAdapter)),
	
	forge.NewABIParserAdapter,
	wire.Bind(new(usecase.ABIParser), new(*forge.ABIParserAdapter)),
)

// TemplateSet provides template-based implementations
var TemplateSet = wire.NewSet(
	template.NewScriptGeneratorAdapter,
	wire.Bind(new(usecase.ScriptGenerator), new(*template.ScriptGeneratorAdapter)),
)

// InteractiveSet provides interactive implementations
var InteractiveSet = wire.NewSet(
	interactive.NewSelectorAdapter,
	wire.Bind(new(usecase.InteractiveSelector), new(*interactive.SelectorAdapter)),
)

// ConfigSet provides configuration-based implementations
var ConfigSet = wire.NewSet(
	internalconfig.NewNetworkResolverAdapter,
	wire.Bind(new(usecase.NetworkResolver), new(*internalconfig.NetworkResolverAdapter)),
)

// BlockchainSet provides blockchain-based implementations
var BlockchainSet = wire.NewSet(
	blockchain.NewCheckerAdapter,
	wire.Bind(new(usecase.BlockchainChecker), new(*blockchain.CheckerAdapter)),
)

// VerificationSet provides verification-based implementations
var VerificationSet = wire.NewSet(
	verification.NewVerifierAdapter,
	wire.Bind(new(usecase.ContractVerifier), new(*verification.VerifierAdapter)),
)

// SafeSet provides Safe-based implementations
var SafeSet = wire.NewSet(
	safe.NewClientAdapter,
	wire.Bind(new(usecase.SafeClient), new(*safe.ClientAdapter)),
)

// AnvilSet provides anvil-based implementations
var AnvilSet = wire.NewSet(
	anvil.NewManager,
	wire.Bind(new(usecase.AnvilManager), new(*anvil.Manager)),
)

// AllAdapters includes all adapter sets
var AllAdapters = wire.NewSet(
	// Provider functions
	ProvideProjectPath,
	
	// Adapter sets
	FSSet,
	ForgeSet,
	TemplateSet,
	InteractiveSet,
	ConfigSet,
	BlockchainSet,
	VerificationSet,
	SafeSet,
	AnvilSet,
	ScriptAdapters,
)