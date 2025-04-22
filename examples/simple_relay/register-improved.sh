#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
WAIT_TIME=45  # Seconds to wait for transaction confirmation
AMOUNT="120000000000"
CHAIN_IDS="0001,0002"
GEO_ZONES="0001"
NUM_SERVICERS="1"
NATIVE_CHAIN="viper-test"

echo -e "${GREEN}Viper Network Enhanced Registration Script${NC}"
echo -e "${GREEN}========================================${NC}"

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

# Function to check if viper CLI is available
check_viper_cli() {
    if ! command -v viper &> /dev/null; then
        echo -e "${RED}Error: viper CLI not found. Make sure it's installed and in your PATH.${NC}"
        exit 1
    fi
}

# Function to check wallet balance
check_balance() {
    local address=$1
    echo -e "${BLUE}Checking balance for address: $address${NC}"
    
    # Try up to 3 times with a 5 second delay between attempts
    for i in {1..3}; do
        BALANCE_OUTPUT=$(viper wallet balance $address 2>&1)
        if [[ $? -eq 0 && "$BALANCE_OUTPUT" == *"$address"* ]]; then
            echo -e "${GREEN}Balance checked successfully.${NC}"
            echo "$BALANCE_OUTPUT"
            return 0
        fi
        echo -e "${YELLOW}Attempt $i: Failed to check balance. Retrying in 5 seconds...${NC}"
        sleep 5
    done
    
    echo -e "${RED}Failed to check balance after multiple attempts.${NC}"
    return 1
}

# Step 0: Check prerequisites
check_viper_cli

# Step 1: Create account
echo -e "\n${GREEN}Step 1: Creating account...${NC}"
CREATE_OUTPUT=$(viper wallet create-account $PRIVATE_KEY 2>&1)
STATUS=$?

echo "$CREATE_OUTPUT"

if [ $STATUS -ne 0 ]; then
    # Check if account already exists
    if [[ "$CREATE_OUTPUT" == *"already exists"* ]]; then
        echo -e "${YELLOW}Account already exists. Continuing with next steps.${NC}"
        # Extract the address from the error message
        ADDRESS=$(echo "$CREATE_OUTPUT" | grep -o '[a-f0-9]\{40\}')
        if [ -z "$ADDRESS" ]; then
            echo -e "${RED}Failed to extract address from output.${NC}"
            echo "$CREATE_OUTPUT"
            exit 1
        fi
    else
        echo -e "${RED}Failed to create account.${NC}"
        exit 1
    fi
else
    # Extract the address from the successful output
    ADDRESS=$(echo "$CREATE_OUTPUT" | grep "Address:" | awk '{print $2}')
    if [ -z "$ADDRESS" ]; then
        echo -e "${RED}Failed to extract address from output.${NC}"
        echo "$CREATE_OUTPUT"
        exit 1
    fi
fi

echo -e "${GREEN}Using account with address: ${YELLOW}$ADDRESS${NC}"

# Step 2: Check initial balance
echo -e "\n${GREEN}Step 2: Checking initial balance...${NC}"
check_balance "$ADDRESS"

# Step 3: Fund account
echo -e "\n${GREEN}Step 3: Funding account...${NC}"
echo -e "${YELLOW}Running: viper wallet transfer $FUNDED_ADDRESS $ADDRESS $AMOUNT $NATIVE_CHAIN \"\"${NC}"

# Save current password to prevent multiple prompts
echo -e "${BLUE}You'll be prompted for your wallet password.${NC}"
read -s -p "Enter your wallet password (will be used for all operations): " WALLET_PASSWORD
echo

# Try the transfer with password
echo "$WALLET_PASSWORD" | viper wallet transfer "$FUNDED_ADDRESS" "$ADDRESS" "$AMOUNT" "$NATIVE_CHAIN" "" --pwd -

if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to transfer funds using saved password. Trying without password...${NC}"
    # Try without password
    viper wallet transfer "$FUNDED_ADDRESS" "$ADDRESS" "$AMOUNT" "$NATIVE_CHAIN" ""
    if [ $? -ne 0 ]; then
        echo -e "${RED}Failed to transfer funds. Please check your funded address and try again.${NC}"
        exit 1
    fi
fi

# Step 4: Wait for confirmation
echo -e "\n${GREEN}Step 4: Waiting for transaction confirmation ($WAIT_TIME seconds)...${NC}"
for i in $(seq $WAIT_TIME -1 1); do
    echo -ne "${BLUE}$i seconds remaining...${NC}\r"
    sleep 1
done
echo -e "\n${GREEN}Wait complete.${NC}"

# Step 5: Verify funds were received
echo -e "\n${GREEN}Step 5: Verifying funds were received...${NC}"
check_balance "$ADDRESS"

# Step 6: Stake account
echo -e "\n${GREEN}Step 6: Staking account as requestor...${NC}"
echo -e "${YELLOW}Running: viper requestors stake $ADDRESS $AMOUNT $CHAIN_IDS $GEO_ZONES $NUM_SERVICERS $NATIVE_CHAIN${NC}"

# Try the staking with password
echo "$WALLET_PASSWORD" | viper requestors stake "$ADDRESS" "$AMOUNT" "$CHAIN_IDS" "$GEO_ZONES" "$NUM_SERVICERS" "$NATIVE_CHAIN" --pwd -

if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to stake account using saved password. Trying without password...${NC}"
    # Try without password
    viper requestors stake "$ADDRESS" "$AMOUNT" "$CHAIN_IDS" "$GEO_ZONES" "$NUM_SERVICERS" "$NATIVE_CHAIN"
    if [ $? -ne 0 ]; then
        echo -e "${RED}Failed to stake account. You may need more time for the transaction to be confirmed.${NC}"
        echo -e "${YELLOW}Try running this step manually after a few minutes:${NC}"
        echo -e "${YELLOW}viper requestors stake $ADDRESS $AMOUNT $CHAIN_IDS $GEO_ZONES $NUM_SERVICERS $NATIVE_CHAIN${NC}"
        exit 1
    fi
fi

# Step 7: Wait for staking confirmation
echo -e "\n${GREEN}Step 7: Waiting for staking confirmation ($WAIT_TIME seconds)...${NC}"
for i in $(seq $WAIT_TIME -1 1); do
    echo -ne "${BLUE}$i seconds remaining...${NC}\r"
    sleep 1
done
echo -e "\n${GREEN}Wait complete.${NC}"

# Step 8: Verify staking was successful
echo -e "\n${GREEN}Step 8: Verifying staking status...${NC}"
# Check if the account is listed as a requestor
REQUESTOR_CHECK=$(viper requestors get $ADDRESS 2>&1)
if [[ $? -eq 0 && "$REQUESTOR_CHECK" == *"$ADDRESS"* ]]; then
    echo -e "${GREEN}Staking verification successful!${NC}"
    echo "$REQUESTOR_CHECK"
else
    echo -e "${YELLOW}Could not verify staking status. This doesn't necessarily mean it failed.${NC}"
    echo -e "${YELLOW}The staking transaction may still be processing.${NC}"
    echo -e "${YELLOW}You can check manually with: viper requestors get $ADDRESS${NC}"
fi

# Success message
echo -e "\n${GREEN}Registration process completed!${NC}"
echo -e "To use this account in relay examples, set the environment variable:"
echo -e "${YELLOW}export VIPER_PRIVATE_KEY=$PRIVATE_KEY${NC}"

# Ask if user wants to set the environment variable now
read -p "Set the environment variable now? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    export VIPER_PRIVATE_KEY=$PRIVATE_KEY
    echo -e "${GREEN}Environment variable VIPER_PRIVATE_KEY has been set for this session.${NC}"
    echo -e "${YELLOW}To make it permanent, add it to your shell profile (e.g., ~/.bashrc or ~/.zshrc):${NC}"
    echo -e "${YELLOW}export VIPER_PRIVATE_KEY=$PRIVATE_KEY${NC}"
fi

echo -e "\n${GREEN}You can now run the relay example with:${NC} ${YELLOW}./run_with_key.sh${NC}"
echo -e "${GREEN}If you encounter 'hash is invalid' errors, the staking may not be confirmed yet.${NC}"
echo -e "${GREEN}In that case, wait a few minutes and try again.${NC}" 