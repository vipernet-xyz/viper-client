#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
WAIT_TIME=45           # Seconds to wait for transaction confirmation
MAX_RETRIES=5          # Maximum number of retry attempts
RETRY_DELAY=15         # Seconds to wait between retries
AMOUNT="120000000000"
CHAIN_IDS="0001,0002"
GEO_ZONES="0001"
NUM_SERVICERS="1"
NATIVE_CHAIN="viper-test"

echo -e "${GREEN}Viper Network Auto-Retry Registration Script${NC}"
echo -e "${GREEN}============================================${NC}"

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

# Function to retry a command until it succeeds
retry_command() {
    local cmd=$1
    local description=$2
    local max_attempts=$3
    local delay=$4
    local attempt=1
    local output=""
    local status=1
    
    while [ $attempt -le $max_attempts ]; do
        echo -e "${BLUE}Attempt $attempt of $max_attempts: $description${NC}"
        output=$(eval "$cmd" 2>&1)
        status=$?
        
        echo "$output"
        
        if [ $status -eq 0 ]; then
            echo -e "${GREEN}Success!${NC}"
            echo "$output" > .last_command_output
            return 0
        else
            echo -e "${YELLOW}Failed. Waiting $delay seconds before retrying...${NC}"
            sleep $delay
            attempt=$((attempt + 1))
        fi
    done
    
    echo -e "${RED}All $max_attempts attempts failed for: $description${NC}"
    return 1
}

# Function to check wallet balance with retry
check_balance_with_retry() {
    local address=$1
    local max_attempts=$2
    local delay=$3
    
    echo -e "${BLUE}Checking balance for address: $address${NC}"
    
    if retry_command "viper wallet query account-balance $address" "Checking balance" $max_attempts $delay; then
        local balance_output=$(cat .last_command_output)
        if [[ "$balance_output" == *"$address"* ]]; then
            echo -e "${GREEN}Balance check successful.${NC}"
            return 0
        fi
    fi
    
    echo -e "${YELLOW}Balance check inconclusive. Continuing anyway...${NC}"
    return 0  # Return success to continue the process
}

# Function to extract address from account creation output
extract_address() {
    local output=$1
    local address=""
    
    # First try the success pattern
    address=$(echo "$output" | grep "Address:" | awk '{print $2}')
    
    # If that doesn't work, try to find a hex address pattern
    if [ -z "$address" ]; then
        address=$(echo "$output" | grep -o '[a-f0-9]\{40\}')
    fi
    
    echo "$address"
}

# Prompt for wallet password once
echo -e "${BLUE}You'll be prompted for your wallet password.${NC}"
read -s -p "Enter your wallet password (will be used for all operations): " WALLET_PASSWORD
echo

# Step 0: Check prerequisites
check_viper_cli

# Step 1: Create account with retry
echo -e "\n${GREEN}Step 1: Creating account...${NC}"
retry_command "viper wallet create-account $PRIVATE_KEY" "Creating account" 3 5
CREATE_OUTPUT=$(cat .last_command_output)
STATUS=$?

# Check if account creation worked or if account already exists
if [ $STATUS -ne 0 ]; then
    if [[ "$CREATE_OUTPUT" == *"already exists"* ]]; then
        echo -e "${YELLOW}Account already exists. Continuing with next steps.${NC}"
    else
        echo -e "${RED}Failed to create account after multiple attempts. Stopping.${NC}"
        exit 1
    fi
fi

# Extract the address from the output
ADDRESS=$(extract_address "$CREATE_OUTPUT")
if [ -z "$ADDRESS" ]; then
    echo -e "${RED}Failed to extract address from output. Stopping.${NC}"
    echo "$CREATE_OUTPUT"
    exit 1
fi

echo -e "${GREEN}Using account with address: ${YELLOW}$ADDRESS${NC}"

# Step 2: Check initial balance
echo -e "\n${GREEN}Step 2: Checking initial balance...${NC}"
check_balance_with_retry "$ADDRESS" 3 5

# Step 3: Fund account with retry
echo -e "\n${GREEN}Step 3: Funding account...${NC}"
echo -e "${YELLOW}Running: viper wallet transfer $FUNDED_ADDRESS $ADDRESS $AMOUNT $NATIVE_CHAIN \"\"${NC}"

# First try with password
transfer_with_pwd_cmd="echo \"$WALLET_PASSWORD\" | viper wallet transfer \"$FUNDED_ADDRESS\" \"$ADDRESS\" \"$AMOUNT\" \"$NATIVE_CHAIN\" \"\" --pwd -"
if ! retry_command "$transfer_with_pwd_cmd" "Transferring funds with password" 3 10; then
    echo -e "${YELLOW}Trying transfer without password prompt...${NC}"
    
    # Then try without password
    transfer_cmd="viper wallet transfer \"$FUNDED_ADDRESS\" \"$ADDRESS\" \"$AMOUNT\" \"$NATIVE_CHAIN\" \"\""
    if ! retry_command "$transfer_cmd" "Transferring funds" $MAX_RETRIES $RETRY_DELAY; then
        echo -e "${RED}Failed to transfer funds after multiple attempts. Stopping.${NC}"
        exit 1
    fi
fi

echo -e "${GREEN}Funds transfer initiated. Waiting for confirmation...${NC}"

# Step 4: Wait for confirmation with progressive checking
echo -e "\n${GREEN}Step 4: Waiting for funds to appear in wallet...${NC}"
success=0
attempts=0
max_balance_checks=10
balance_check_interval=15

while [ $success -eq 0 ] && [ $attempts -lt $max_balance_checks ]; do
    attempts=$((attempts + 1))
    echo -e "${BLUE}Balance check attempt $attempts of $max_balance_checks...${NC}"
    
    check_output=$(viper wallet query account-balance "$ADDRESS" 2>&1)
    if [[ $? -eq 0 && "$check_output" == *"$ADDRESS"* ]]; then
        echo "$check_output"
        
        # Look for a non-zero balance
        # This looks for values that aren't 0 or 0.0 - regex pattern matches "0" or "0.0" with possible spaces
        if ! [[ "$check_output" =~ $ADDRESS[[:space:]]+([0][[:space:]]+|[0].[0][[:space:]]+) ]]; then
            echo -e "${GREEN}Funds have arrived!${NC}"
            success=1
            break
        fi
    fi
    
    if [ $attempts -lt $max_balance_checks ]; then
        echo -e "${YELLOW}Funds not visible yet. Waiting $balance_check_interval seconds before next check...${NC}"
        sleep $balance_check_interval
    fi
done

if [ $success -eq 0 ]; then
    echo -e "${YELLOW}Warning: Could not verify funds arrival after $attempts checks.${NC}"
    echo -e "${YELLOW}Do you want to continue with staking anyway? (y/n)${NC}"
    read -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${RED}Stopping as requested.${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}Fund transfer confirmed!${NC}"
fi

# Step 5: Stake account with retry
echo -e "\n${GREEN}Step 5: Staking account as requestor...${NC}"
echo -e "${YELLOW}Running: viper requestors stake $ADDRESS $AMOUNT $CHAIN_IDS $GEO_ZONES $NUM_SERVICERS $NATIVE_CHAIN${NC}"

# First try with password
stake_with_pwd_cmd="echo \"$WALLET_PASSWORD\" | viper requestors stake \"$ADDRESS\" \"$AMOUNT\" \"$CHAIN_IDS\" \"$GEO_ZONES\" \"$NUM_SERVICERS\" \"$NATIVE_CHAIN\" --pwd -"
if ! retry_command "$stake_with_pwd_cmd" "Staking account with password" 3 10; then
    echo -e "${YELLOW}Trying staking without password prompt...${NC}"
    
    # Then try without password
    stake_cmd="viper requestors stake \"$ADDRESS\" \"$AMOUNT\" \"$CHAIN_IDS\" \"$GEO_ZONES\" \"$NUM_SERVICERS\" \"$NATIVE_CHAIN\""
    if ! retry_command "$stake_cmd" "Staking account" $MAX_RETRIES $RETRY_DELAY; then
        echo -e "${RED}Failed to stake account after multiple attempts.${NC}"
        echo -e "${YELLOW}Do you want to continue anyway? The staking may have succeeded despite errors. (y/n)${NC}"
        read -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo -e "${RED}Stopping as requested.${NC}"
            exit 1
        fi
    fi
fi

# Step 6: Wait for staking confirmation with progressive checking
echo -e "\n${GREEN}Step 6: Waiting for staking confirmation...${NC}"
success=0
attempts=0
max_stake_checks=10
stake_check_interval=15

while [ $success -eq 0 ] && [ $attempts -lt $max_stake_checks ]; do
    attempts=$((attempts + 1))
    echo -e "${BLUE}Staking verification attempt $attempts of $max_stake_checks...${NC}"
    
    stake_check=$(viper requestors get "$ADDRESS" 2>&1)
    if [[ $? -eq 0 && "$stake_check" == *"$ADDRESS"* ]]; then
        echo "$stake_check"
        echo -e "${GREEN}Staking verification successful!${NC}"
        success=1
        break
    else
        if [ $attempts -lt $max_stake_checks ]; then
            echo -e "${YELLOW}Staking not visible yet. Waiting $stake_check_interval seconds before next check...${NC}"
            sleep $stake_check_interval
        fi
    fi
done

if [ $success -eq 0 ]; then
    echo -e "${YELLOW}Warning: Could not verify staking after $attempts checks.${NC}"
    echo -e "${YELLOW}Do you want to continue anyway? (y/n)${NC}"
    read -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${RED}Stopping as requested.${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}Staking confirmed!${NC}"
fi

# Clean up
rm -f .last_command_output 2>/dev/null

# Success message
echo -e "\n${GREEN}Registration process completed!${NC}"
echo -e "To use this account in relay examples, set the environment variable:"
echo -e "${YELLOW}export VIPER_PRIVATE_KEY=$PRIVATE_KEY${NC}"

# Set the environment variable
export VIPER_PRIVATE_KEY=$PRIVATE_KEY
echo -e "${GREEN}Environment variable VIPER_PRIVATE_KEY has been set for this session.${NC}"
echo -e "${YELLOW}To make it permanent, add it to your shell profile (e.g., ~/.bashrc or ~/.zshrc).${NC}"

echo -e "\n${GREEN}You can now run the relay example with:${NC} ${YELLOW}./run_with_key.sh${NC}"
echo -e "${GREEN}If you encounter 'hash is invalid' errors, the staking may not be confirmed yet.${NC}"
echo -e "${GREEN}In that case, wait a few more minutes and try again.${NC}" 