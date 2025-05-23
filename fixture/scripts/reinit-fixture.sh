#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🔄 Reinitializing fixture project...${NC}"

# Get the current directory
FIXTURE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$FIXTURE_DIR"

# Check if .env file exists
if [ ! -f ".env" ]; then
    echo -e "${RED}❌ .env file not found!${NC}"
    exit 1
fi

# Load current environment
source .env

# Extract current deployer address from private key
if [ -z "$DEPLOYER_PRIVATE_KEY" ]; then
    echo -e "${RED}❌ DEPLOYER_PRIVATE_KEY not found in .env!${NC}"
    exit 1
fi

echo -e "${YELLOW}📍 Getting current deployer address...${NC}"
OLD_DEPLOYER=$(cast wallet address "$DEPLOYER_PRIVATE_KEY")
echo "Current deployer: $OLD_DEPLOYER"

# Create new wallet
echo -e "${YELLOW}🔑 Creating new deployer wallet...${NC}"
NEW_WALLET_OUTPUT=$(cast wallet new)
NEW_ADDRESS=$(echo "$NEW_WALLET_OUTPUT" | grep "Address:" | awk '{print $2}')
NEW_PRIVATE_KEY=$(echo "$NEW_WALLET_OUTPUT" | grep "Private key:" | awk '{print $3}')

echo "New deployer address: $NEW_ADDRESS"

# Get RPC URLs from environment or use defaults
RPC_URLS=(
    "${RPC_URL_ALFAJORES:-https://alfajores-forno.celo-testnet.org}"
    "${RPC_URL_SEPOLIA:-https://sepolia.infura.io/v3/YOUR_INFURA_KEY}"
    "${RPC_URL_ARBITRUM_SEPOLIA:-https://sepolia-rollup.arbitrum.io/rpc}"
)

NETWORK_NAMES=(
    "Celo Alfajores"
    "Sepolia"
    "Arbitrum Sepolia"
)

# Transfer funds from old to new deployer
echo -e "${YELLOW}💸 Transferring testnet tokens to new deployer...${NC}"

for i in "${!RPC_URLS[@]}"; do
    RPC_URL="${RPC_URLS[$i]}"
    NETWORK="${NETWORK_NAMES[$i]}"
    
    echo -e "\n${YELLOW}Checking balance on ${NETWORK}...${NC}"
    
    # Get balance of old deployer
    BALANCE=$(cast balance "$OLD_DEPLOYER" --rpc-url "$RPC_URL" 2>/dev/null || echo "0")
    
    if [ "$BALANCE" = "0" ]; then
        echo "No balance on $NETWORK, skipping..."
        continue
    fi
    
    echo "Balance: $BALANCE wei"
    
    # Calculate amount to send (leave 0.01 ETH for gas)
    GAS_RESERVE="10000000000000000" # 0.01 ETH in wei
    AMOUNT_TO_SEND=$(echo "$BALANCE - $GAS_RESERVE" | bc)
    
    if [ $(echo "$AMOUNT_TO_SEND > 0" | bc) -eq 1 ]; then
        echo "Transferring $(cast to-unit "$AMOUNT_TO_SEND" ether) ETH to new deployer..."
        
        # Send transaction
        TX_HASH=$(cast send "$NEW_ADDRESS" --value "$AMOUNT_TO_SEND" \
            --private-key "$DEPLOYER_PRIVATE_KEY" \
            --rpc-url "$RPC_URL" \
            --json 2>/dev/null | jq -r '.transactionHash' || echo "")
        
        if [ -n "$TX_HASH" ]; then
            echo -e "${GREEN}✅ Transfer successful! TX: $TX_HASH${NC}"
            
            # Wait for confirmation
            echo "Waiting for confirmation..."
            cast receipt "$TX_HASH" --rpc-url "$RPC_URL" --confirmations 1 >/dev/null 2>&1
        else
            echo -e "${RED}❌ Transfer failed on $NETWORK${NC}"
        fi
    else
        echo "Insufficient balance to transfer (need to keep gas reserve)"
    fi
done

# Clean deployment artifacts
echo -e "\n${YELLOW}🧹 Cleaning deployment artifacts...${NC}"

# Remove deployment directories
if [ -d "deployments" ]; then
    echo "Removing deployments directory..."
    rm -rf deployments
fi

if [ -d "broadcast" ]; then
    echo "Removing broadcast directory..."
    rm -rf broadcast
fi

# Remove registry file
if [ -f "fdeploy-registry.json" ]; then
    echo "Removing registry file..."
    rm -f fdeploy-registry.json
fi

# Remove deployments.json if it exists
if [ -f "deployments.json" ]; then
    echo "Removing deployments.json..."
    rm -f deployments.json
fi

# Remove cache directories
if [ -d "cache" ]; then
    echo "Removing forge cache..."
    rm -rf cache
fi

if [ -d "out" ]; then
    echo "Removing build artifacts..."
    rm -rf out
fi

# Update .env file with new private key
echo -e "\n${YELLOW}📝 Updating .env file...${NC}"

# Create backup of current .env
cp .env .env.backup.$(date +%Y%m%d_%H%M%S)

# Update DEPLOYER_PRIVATE_KEY in .env
if grep -q "^DEPLOYER_PRIVATE_KEY=" .env; then
    # On macOS, use -i '' for in-place editing
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s/^DEPLOYER_PRIVATE_KEY=.*/DEPLOYER_PRIVATE_KEY=$NEW_PRIVATE_KEY/" .env
    else
        sed -i "s/^DEPLOYER_PRIVATE_KEY=.*/DEPLOYER_PRIVATE_KEY=$NEW_PRIVATE_KEY/" .env
    fi
else
    echo "DEPLOYER_PRIVATE_KEY=$NEW_PRIVATE_KEY" >> .env
fi

# Rebuild contracts
echo -e "\n${YELLOW}🔨 Rebuilding contracts...${NC}"
forge build

# Verify new setup
echo -e "\n${GREEN}✅ Fixture reinitialized successfully!${NC}"
echo -e "${GREEN}New deployer address: $NEW_ADDRESS${NC}"
echo -e "${GREEN}Private key has been updated in .env${NC}"
echo -e "${YELLOW}⚠️  Old .env backed up with timestamp${NC}"

# Show new balances
echo -e "\n${YELLOW}💰 New deployer balances:${NC}"
for i in "${!RPC_URLS[@]}"; do
    RPC_URL="${RPC_URLS[$i]}"
    NETWORK="${NETWORK_NAMES[$i]}"
    
    BALANCE=$(cast balance "$NEW_ADDRESS" --rpc-url "$RPC_URL" 2>/dev/null || echo "0")
    if [ "$BALANCE" != "0" ]; then
        BALANCE_ETH=$(cast to-unit "$BALANCE" ether)
        echo "$NETWORK: $BALANCE_ETH ETH"
    fi
done

echo -e "\n${GREEN}🎉 Ready for fresh deployments!${NC}"