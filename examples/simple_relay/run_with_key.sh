#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}Viper Network Relay Runner Script${NC}"
echo -e "${GREEN}-------------------------------${NC}"

# Check if private key is provided as argument or in environment
PRIVATE_KEY=$1
if [ -z "$PRIVATE_KEY" ]; then
    PRIVATE_KEY=$VIPER_PRIVATE_KEY
    if [ -z "$PRIVATE_KEY" ]; then
        echo -e "${RED}No private key provided.${NC}"
        echo -e "${YELLOW}Usage: $0 <private_key>${NC}"
        echo -e "${YELLOW}Or set the VIPER_PRIVATE_KEY environment variable.${NC}"
        exit 1
    fi
    echo -e "${YELLOW}Using private key from environment variable.${NC}"
else
    echo -e "${YELLOW}Using private key from command line argument.${NC}"
    # Set the environment variable for this session
    export VIPER_PRIVATE_KEY=$PRIVATE_KEY
fi

echo -e "${GREEN}Running relay with private key:${NC} ${YELLOW}$PRIVATE_KEY${NC}"
echo

# Run the relay code with the environment variable set
echo -e "${GREEN}Executing relay:${NC}"
go run relay.go

echo -e "\n${GREEN}Relay execution completed.${NC}" 