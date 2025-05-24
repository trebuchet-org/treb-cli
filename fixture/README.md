# Fixture Project

This is a test/development Foundry project that uses the treb-sol library via symlink for development and testing.

## Setup

The project is set up with:
- **Symlinked Library**: `lib/treb-sol` → `../../treb-sol` 
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
forge script lib/treb-sol/script/PredictAddress.s.sol:PredictAddress \
    --sig "predict(string,string)" "SampleToken" "staging"
```

### Test CLI Integration
```bash
# From root directory
cd fixture
../bin/treb predict SampleToken
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

- **Live Updates**: Changes to `treb-sol` are immediately reflected
- **No Rebuild**: No need to reinstall library during development
- **Real Testing**: Test actual deployment flows with real contracts
- **CLI Testing**: Test treb CLI integration end-to-end

## Files

- `src/SampleToken.sol` - Example contract for deployment testing
- `script/DeploySampleToken.s.sol` - Deployment script using treb-sol
- `lib/treb-sol/` - Symlink to development library
- `.env` - Test environment configuration
- `deployments.json` - treb registry
