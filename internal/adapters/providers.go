package adapters

import (
	"github.com/google/wire"
	"github.com/trebuchet-org/treb-cli/internal/adapters/abi"
	"github.com/trebuchet-org/treb-cli/internal/adapters/anvil"
	"github.com/trebuchet-org/treb-cli/internal/adapters/blockchain"
	"github.com/trebuchet-org/treb-cli/internal/adapters/forge"
	"github.com/trebuchet-org/treb-cli/internal/adapters/fs"
	"github.com/trebuchet-org/treb-cli/internal/adapters/repository/contracts"
	"github.com/trebuchet-org/treb-cli/internal/adapters/repository/deployments"
	"github.com/trebuchet-org/treb-cli/internal/adapters/resolvers"
	"github.com/trebuchet-org/treb-cli/internal/adapters/template"
	"github.com/trebuchet-org/treb-cli/internal/adapters/verification"
	"github.com/trebuchet-org/treb-cli/internal/cli/interactive"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// Removed ProvideContractsIndexer - no longer needed as we use fs.ContractIndexerAdapter

// ProvideProjectPath provides the project path from RuntimeConfig
func ProvideProjectPath(cfg *config.RuntimeConfig) string {
	return cfg.ProjectRoot
}

// FSSet provides filesystem-based implementations
var FSSet = wire.NewSet(
	deployments.NewFileRepositoryFromConfig,
	wire.Bind(new(usecase.DeploymentRepository), new(*deployments.FileRepository)),
	wire.Bind(new(usecase.DeploymentRepositoryUpdater), new(*deployments.FileRepository)),

	deployments.NewPruner,
	wire.Bind(new(usecase.DeploymentRepositoryPruner), new(*deployments.Pruner)),

	fs.NewFileWriterAdapter,
	wire.Bind(new(usecase.FileWriter), new(*fs.FileWriterAdapter)),

	fs.NewLocalConfigStoreAdapter,
	wire.Bind(new(usecase.LocalConfigRepository), new(*fs.LocalConfigStoreAdapter)),
)

// TemplateSet provides template-based implementations
var TemplateSet = wire.NewSet(
	template.NewScriptGeneratorAdapter,
	wire.Bind(new(usecase.ScriptGenerator), new(*template.ScriptGeneratorAdapter)),
)

// InteractiveSet provides interactive implementations
var InteractiveSet = wire.NewSet(
	interactive.NewSelectorAdapter,
	wire.Bind(new(usecase.ContractSelector), new(*interactive.SelectorAdapter)),
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

// AnvilSet provides anvil-based implementations
var AnvilSet = wire.NewSet(
	anvil.NewManager,
	wire.Bind(new(usecase.AnvilManager), new(*anvil.Manager)),
)

// ScriptAdapters provides all adapters needed for script execution
var ScriptAdapters = wire.NewSet(
	// Contract resolution and indexing
	resolvers.NewContractResolver,
	wire.Bind(new(usecase.ContractResolver), new(*resolvers.ContractResolver)),

	contracts.NewRepository,
	wire.Bind(new(usecase.ContractRepository), new(*contracts.Repository)),

	// Script resolution
	resolvers.NewScriptResolver,
	wire.Bind(new(usecase.ScriptResolver), new(*resolvers.ScriptResolver)),

	// ABI handling
	abi.NewParser,
	wire.Bind(new(usecase.ABIParser), new(*abi.Parser)),

	abi.NewABIResolver,
	wire.Bind(new(usecase.ABIResolver), new(*abi.ABIResolver)),

	// Parameter handling
	resolvers.NewParameterResolver,
	wire.Bind(new(usecase.ParameterResolver), new(*resolvers.ParameterResolver)),

	// TODO: Add these bask
	// parameters.NewParameterPrompterAdapter,
	// wire.Bind(new(usecase.ParameterPrompter), new(*parameters.ParameterPrompterAdapter)),

	// Script execution
	forge.NewForgeAdapter,
	wire.Bind(new(usecase.ForgeScriptRunner), new(*forge.ForgeAdapter)),

	// Result hydration
	forge.NewRunResultHydrator,
	wire.Bind(new(usecase.RunResultHydrator), new(*forge.RunResultHydrator)),

	// Registry updates - RegistryStoreAdapter also implements RegistryUpdater

	// Library resolution
	resolvers.NewLibraryResolver,
	wire.Bind(new(usecase.LibraryResolver), new(*resolvers.LibraryResolver)),
)

// AllAdapters includes all adapter sets
var AllAdapters = wire.NewSet(
	// Provider functions
	ProvideProjectPath,

	// Adapter sets
	FSSet,
	TemplateSet,
	InteractiveSet,
	BlockchainSet,
	VerificationSet,
	AnvilSet,
	ScriptAdapters,
)
