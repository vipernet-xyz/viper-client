#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}Viper Network Simple Registration Script${NC}"
echo -e "${GREEN}------------------------------------${NC}"

# Check if we have a funded address
if [ "$#" -lt 1 ]; then
    echo -e "${YELLOW}Usage: $0 <funded_address> <private_key>${NC}"
    echo -e "${YELLOW}Example: $0 abc123yourfundedaddress d05f9453b62eccc67eae03fb5164904c8cc1405c2c666ca8a656c2a9db4a89ef${NC}"
    exit 1
fi

FUNDED_ADDRESS=$1
PRIVATE_KEY=$2

# If private key wasn't provided, ask for it
if [ -z "$PRIVATE_KEY" ]; then
    read -p "Enter the private key to register: " PRIVATE_KEY
    if [ -z "$PRIVATE_KEY" ]; then
        echo -e "${RED}No private key provided. Exiting.${NC}"
        exit 1
    fi
fi

# Step 1: Create account
echo -e "\n${GREEN}Step 1: Creating account...${NC}"
CREATE_OUTPUT=$(viper wallet create-account $PRIVATE_KEY)
if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to create account.${NC}"
    echo "$CREATE_OUTPUT"
    exit 1
fi

# Extract the address from the output
ADDRESS=$(echo "$CREATE_OUTPUT" | grep "Address:" | awk '{print $2}')
if [ -z "$ADDRESS" ]; then
    echo -e "${RED}Failed to extract address from output.${NC}"
    echo "$CREATE_OUTPUT"
    exit 1
fi

echo -e "${GREEN}Created account with address: ${YELLOW}$ADDRESS${NC}"

# Step 2: Fund account
echo -e "\n${GREEN}Step 2: Funding account...${NC}"
echo -e "${YELLOW}Running: viper wallet transfer $FUNDED_ADDRESS $ADDRESS 120000000000 viper-test \"\"${NC}"
viper wallet transfer "$FUNDED_ADDRESS" "$ADDRESS" "120000000000" "viper-test" ""
if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to transfer funds.${NC}"
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
echo -e "${YELLOW}Running: viper requestors stake $ADDRESS 120000000000 0001,0002 0001 1 viper-test${NC}"
viper requestors stake "$ADDRESS" "120000000000" "0001,0002" "0001" "1" "viper-test"
if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to stake account.${NC}"
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

echo -e "\n${GREEN}You can now run the relay example with:${NC} ${YELLOW}./run_with_key.sh${NC}" 