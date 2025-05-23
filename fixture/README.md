# Fixture Project

This is a test/development Foundry project that uses the forge-deploy library via symlink for development and testing.

## Setup

The project is set up with:
- **Symlinked Library**: `lib/forge-deploy` → `../../forge-deploy-lib` 
- **Sample Contract**: `SampleToken.sol` - Simple ERC20-like token
- **Deployment Script**: `DeploySampleToken.s.sol` - Uses `CreateXDeployment` base
- **Environment**: Configured with test private key and staging environment

## Usage

### Build
```bash
forge build
```

### Test Prediction
```bash
forge script lib/forge-deploy/script/PredictAddress.s.sol:PredictAddress \
    --sig "predict(string,string)" "SampleToken" "staging"
```

### Test CLI Integration
```bash
# From root directory
cd fixture
../bin/fdeploy predict SampleToken
```

### Test Deployment (Local)
```bash
# Start local anvil
anvil

# Deploy using forge script
forge script script/DeploySampleToken.s.sol:DeploySampleToken \
    --rpc-url http://localhost:8545 --broadcast
```

## Development Benefits

- **Live Updates**: Changes to `forge-deploy-lib` are immediately reflected
- **No Rebuild**: No need to reinstall library during development
- **Real Testing**: Test actual deployment flows with real contracts
- **CLI Testing**: Test fdeploy CLI integration end-to-end

## Files

- `src/SampleToken.sol` - Example contract for deployment testing
- `script/DeploySampleToken.s.sol` - Deployment script using forge-deploy
- `lib/forge-deploy/` - Symlink to development library
- `.env` - Test environment configuration
- `deployments.json` - fdeploy registry
