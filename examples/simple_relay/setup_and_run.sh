#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}Viper Network Relay Setup Script${NC}"
echo "=================================="

# Step 1: Run the relay example to get new keys
echo -e "\n${YELLOW}Step 1: Generating new keys...${NC}"
OUTPUT=$(go run relay.go)

# Extract the values from the output
PRIVATE_KEY=$(echo "$OUTPUT" | grep "Private key:" | cut -d' ' -f3)
ADDRESS=$(echo "$OUTPUT" | grep "address:" | cut -d' ' -f5 | cut -d'(' -f1)
PUBLIC_KEY=$(echo "$OUTPUT" | grep "Public key:" | cut -d' ' -f3)

echo "Generated Keys:"
echo "Address: $ADDRESS"
echo "Public Key: $PUBLIC_KEY"
echo "Private Key: $PRIVATE_KEY"

# Check if we got the keys
if [ -z "$PRIVATE_KEY" ] || [ -z "$ADDRESS" ]; then
    echo -e "${RED}Failed to extract keys from output. Exiting.${NC}"
    exit 1
fi

# Step 2: Ask for a funded address
echo -e "\n${YELLOW}Step 2: Create account in wallet${NC}"
echo "Creating account with private key in wallet..."
echo -e "${GREEN}Running: viper wallet create-account $PRIVATE_KEY${NC}"
read -p "Execute this command? (y/n): " CONFIRM
if [[ $CONFIRM == [yY] ]]; then
    viper wallet create-account $PRIVATE_KEY
else
    echo "Skipping account creation. Continuing with script."
fi

# Step 3: Fund the account
echo -e "\n${YELLOW}Step 3: Fund the account${NC}"
read -p "Enter the address of your funded account: " FUNDED_ADDRESS
if [ -z "$FUNDED_ADDRESS" ]; then
    echo -e "${RED}No funded address provided. Exiting.${NC}"
    exit 1
fi

echo -e "${GREEN}Running: viper wallet transfer $FUNDED_ADDRESS $ADDRESS 120000000000 viper-test \"\"${NC}"
read -p "Execute this command? (y/n): " CONFIRM
if [[ $CONFIRM == [yY] ]]; then
    viper wallet transfer $FUNDED_ADDRESS $ADDRESS 120000000000 viper-test ""
else
    echo "Skipping funding. Continuing with script."
fi

# Step 4: Wait for transaction confirmation
echo -e "\n${YELLOW}Step 4: Waiting for transaction confirmation...${NC}"
echo "Waiting 15 seconds for the transaction to confirm..."
sleep 15

# Step 5: Stake the account
echo -e "\n${YELLOW}Step 5: Stake the account as a requestor${NC}"
echo -e "${GREEN}Running: viper requestors stake $ADDRESS 120000000000 0001,0002 0001 1 viper-test${NC}"
read -p "Execute this command? (y/n): " CONFIRM
if [[ $CONFIRM == [yY] ]]; then
    viper requestors stake $ADDRESS 120000000000 0001,0002 0001 1 viper-test
else
    echo "Skipping staking. Continuing with script."
fi

# Step 6: Run the relay example with the registered key
echo -e "\n${YELLOW}Step 6: Running the relay example with registered key${NC}"
echo -e "${GREEN}Running relay example with VIPER_PRIVATE_KEY=$PRIVATE_KEY${NC}"
read -p "Execute this command? (y/n): " CONFIRM
if [[ $CONFIRM == [yY] ]]; then
    VIPER_PRIVATE_KEY=$PRIVATE_KEY go run relay.go
else
    echo "Skipping final relay execution."
fi

echo -e "\n${GREEN}Script completed!${NC}"
echo "For future use, run with:"
echo -e "${YELLOW}VIPER_PRIVATE_KEY=$PRIVATE_KEY go run relay.go${NC}" 