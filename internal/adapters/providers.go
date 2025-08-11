package adapters

import (
	"github.com/google/wire"
	"github.com/trebuchet-org/treb-cli/internal/adapters/config"
	"github.com/trebuchet-org/treb-cli/internal/adapters/forge"
	"github.com/trebuchet-org/treb-cli/internal/adapters/fs"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// FSSet provides filesystem-based implementations
var FSSet = wire.NewSet(
	fs.NewRegistryStoreAdapter,
	wire.Bind(new(usecase.DeploymentStore), new(*fs.RegistryStoreAdapter)),
	
	fs.NewContractIndexerAdapter,
	wire.Bind(new(usecase.ContractIndexer), new(*fs.ContractIndexerAdapter)),
)

// ForgeSet provides forge-based implementations
var ForgeSet = wire.NewSet(
	forge.NewForgeExecutorAdapter,
	wire.Bind(new(usecase.ForgeExecutor), new(*forge.ForgeExecutorAdapter)),
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
	ConfigSet,
)