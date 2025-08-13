package adapters

import (
	"github.com/google/wire"
	"github.com/trebuchet-org/treb-cli/internal/adapters/config"
	"github.com/trebuchet-org/treb-cli/internal/adapters/forge"
	"github.com/trebuchet-org/treb-cli/internal/adapters/fs"
	"github.com/trebuchet-org/treb-cli/internal/adapters/interactive"
	"github.com/trebuchet-org/treb-cli/internal/adapters/template"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// FSSet provides filesystem-based implementations
var FSSet = wire.NewSet(
	fs.NewRegistryStoreAdapter,
	wire.Bind(new(usecase.DeploymentStore), new(*fs.RegistryStoreAdapter)),
	
	fs.NewContractIndexerAdapter,
	wire.Bind(new(usecase.ContractIndexer), new(*fs.ContractIndexerAdapter)),
	
	fs.NewFileWriterAdapter,
	wire.Bind(new(usecase.FileWriter), new(*fs.FileWriterAdapter)),
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
	config.NewNetworkResolverAdapter,
	wire.Bind(new(usecase.NetworkResolver), new(*config.NetworkResolverAdapter)),
)

// AllAdapters includes all adapter sets
var AllAdapters = wire.NewSet(
	FSSet,
	ForgeSet,
	TemplateSet,
	InteractiveSet,
	ConfigSet,
)