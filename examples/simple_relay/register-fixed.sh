#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}Viper Network Relay Registration Script${NC}"
echo -e "${GREEN}-------------------------------------${NC}"

# Check if we have the necessary parameters
if [ "$#" -lt 1 ]; then
    echo -e "${YELLOW}Usage: $0 <funded_address> [private_key] [address]${NC}"
    echo -e "${YELLOW}If private_key and address are not provided, they will be generated.${NC}"
    exit 1
fi

FUNDED_ADDRESS=$1
PRIVATE_KEY=$2
ADDRESS=$3

# If no private key is provided, generate one by running the relay example directly
if [ -z "$PRIVATE_KEY" ] || [ -z "$ADDRESS" ]; then
    echo -e "${YELLOW}Generating new keys...${NC}"
    
    # Run the relay example and capture its output
    OUTPUT=$(go run relay.go)
    
    # Write output to a temporary file for easier parsing
    echo "$OUTPUT" > temp_output.txt
    
    # Extract the address and private key using grep
    PRIVATE_KEY=$(grep -o "Client private key: [^ ]*" temp_output.txt | cut -d' ' -f4)
    ADDRESS=$(grep -o "Client address: [^ ]*" temp_output.txt | cut -d' ' -f3)
    
    # Remove the temporary file
    rm temp_output.txt
    
    if [ -z "$PRIVATE_KEY" ] || [ -z "$ADDRESS" ]; then
        echo -e "${RED}Failed to extract keys from relay output.${NC}"
        echo -e "${RED}Output was:${NC}"
        echo "$OUTPUT"
        exit 1
    fi
    
    echo -e "${GREEN}Successfully generated keys:${NC}"
    echo -e "  Address: ${YELLOW}$ADDRESS${NC}"
    echo -e "  Private Key: ${YELLOW}$PRIVATE_KEY${NC}"
else
    echo -e "${GREEN}Using provided credentials:${NC}"
    echo -e "  Address: ${YELLOW}$ADDRESS${NC}"
    echo -e "  Private Key: ${YELLOW}$PRIVATE_KEY${NC}"
fi

# Confirm before proceeding
echo
echo -e "${YELLOW}This script will:${NC}"
echo -e "  1. Create an account with the private key"
echo -e "  2. Transfer 120000000000 tokens from $FUNDED_ADDRESS to $ADDRESS"
echo -e "  3. Wait 15 seconds for confirmation"
echo -e "  4. Stake the account as a requestor"
echo
read -p "Continue? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${RED}Operation cancelled.${NC}"
    exit 1
fi

# Step 1: Create account
echo -e "\n${GREEN}Step 1: Creating account...${NC}"
viper wallet create-account $PRIVATE_KEY
if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to create account.${NC}"
    exit 1
fi

# Step 2: Fund account
echo -e "\n${GREEN}Step 2: Funding account...${NC}"
echo -e "${YELLOW}Transfer command: viper wallet transfer $FUNDED_ADDRESS $ADDRESS 120000000000 viper-test \"\"${NC}"
viper wallet transfer "$FUNDED_ADDRESS" "$ADDRESS" "120000000000" "viper-test" ""
if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to transfer funds.${NC}"
    echo -e "${YELLOW}Note: The transfer command format is:${NC}"
    echo -e "${YELLOW}viper wallet transfer <fromAddr> <toAddr> <amount> <nativeChainID> <memo>${NC}"
    exit 1
fi

# Step 3: Wait for confirmation
echo -e "\n${GREEN}Step 3: Waiting for transaction confirmation (15 seconds)...${NC}"
for i in {15..1}; do
    echo -ne "$i seconds remaining...\r"
    sleep 1
done
echo -e "\nWait complete."

# Step 4: Stake account
echo -e "\n${GREEN}Step 4: Staking account as requestor...${NC}"
# Update the staking command based on the error message:
# Usage: viper requestors stake <fromAddr> <amount> <ChainIDs> <geoZones> <numServicers> <nativeChainID>
echo -e "${YELLOW}Staking command: viper requestors stake $ADDRESS 120000000000 0001,0002 0001 1 viper-test${NC}"
viper requestors stake "$ADDRESS" "120000000000" "0001,0002" "0001" "1" "viper-test"
if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to stake account.${NC}"
    echo -e "${YELLOW}Note: The staking command format is:${NC}"
    echo -e "${YELLOW}viper requestors stake <fromAddr> <amount> <ChainIDs> <geoZones> <numServicers> <nativeChainID>${NC}"
    exit 1
fi

# Success message
echo -e "\n${GREEN}Registration complete!${NC}"
echo -e "To use this account in relay examples, set the environment variable:"
echo -e "${YELLOW}export VIPER_PRIVATE_KEY=$PRIVATE_KEY${NC}"

# Ask if user wants to set the environment variable now
read -p "Set the environment variable now? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    export VIPER_PRIVATE_KEY=$PRIVATE_KEY
    echo -e "${GREEN}Environment variable VIPER_PRIVATE_KEY has been set for this session.${NC}"
    echo -e "${YELLOW}To make it permanent, add it to your shell profile.${NC}"
fi

echo -e "\n${GREEN}You can now run the relay example with your registered account.${NC}" 