package adapters

import (
	"github.com/google/wire"
	"github.com/trebuchet-org/treb-cli/internal/adapters/contracts"
	"github.com/trebuchet-org/treb-cli/internal/adapters/environment"
	"github.com/trebuchet-org/treb-cli/internal/adapters/forge"
	"github.com/trebuchet-org/treb-cli/internal/adapters/parameters"
	"github.com/trebuchet-org/treb-cli/internal/adapters/parser"
	"github.com/trebuchet-org/treb-cli/internal/adapters/progress"
	"github.com/trebuchet-org/treb-cli/internal/adapters/registry"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ScriptAdapters provides all adapters needed for script execution
var ScriptAdapters = wire.NewSet(
	// Contract resolution and indexing
	contracts.NewContractResolverAdapter,
	wire.Bind(new(usecase.ContractResolver), new(*contracts.ContractResolverAdapter)),
	wire.Bind(new(usecase.ContractIndexer), new(*contracts.ContractResolverAdapter)),

	// Script resolution
	contracts.NewScriptResolverAdapter,
	wire.Bind(new(usecase.ScriptResolver), new(*contracts.ScriptResolverAdapter)),

	// Parameter handling
	parameters.NewParameterResolverAdapter,
	wire.Bind(new(usecase.ParameterResolver), new(*parameters.ParameterResolverAdapter)),
	
	parameters.NewParameterPrompterAdapter,
	wire.Bind(new(usecase.ParameterPrompter), new(*parameters.ParameterPrompterAdapter)),

	// Script execution
	forge.NewScriptExecutorAdapter,
	wire.Bind(new(usecase.ScriptExecutor), new(*forge.ScriptExecutorAdapter)),

	// Execution parsing
	parser.NewExecutionParserAdapter,
	wire.Bind(new(usecase.ExecutionParser), new(*parser.ExecutionParserAdapter)),

	// Registry updates
	registry.NewUpdaterAdapter,
	wire.Bind(new(usecase.RegistryUpdater), new(*registry.UpdaterAdapter)),

	// Environment building
	environment.NewBuilderAdapter,
	wire.Bind(new(usecase.EnvironmentBuilder), new(*environment.BuilderAdapter)),

	// Library resolution
	registry.NewLibraryResolverAdapter,
	wire.Bind(new(usecase.LibraryResolver), new(*registry.LibraryResolverAdapter)),

	// Progress reporting
	progress.NewSpinnerProgressReporter,
	wire.Bind(new(usecase.ProgressReporter), new(*progress.SpinnerProgressReporter)),
)