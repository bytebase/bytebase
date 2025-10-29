#!/bin/bash

# Script to tag and push Bytebase image to Azure Container Registry
# Run this AFTER the Docker build completes

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Configuration
ACR_NAME="bytebase"
ACR_LOGIN_SERVER="bytebase.azurecr.io"
BYTEBASE_VERSION="3.11.1"
LOCAL_IMAGE="bytebase/bytebase:${BYTEBASE_VERSION}"
ACR_IMAGE="${ACR_LOGIN_SERVER}/bytebase/bytebase:${BYTEBASE_VERSION}"

echo -e "${GREEN}=== Bytebase ACR Push Script ===${NC}\n"

# Check if Docker image exists
echo -e "${YELLOW}Step 1: Checking if Docker image exists...${NC}"
if docker image inspect ${LOCAL_IMAGE} > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Image ${LOCAL_IMAGE} found${NC}"
else
    echo -e "${RED}✗ Image ${LOCAL_IMAGE} not found${NC}"
    echo "Please build the image first:"
    echo "  docker build -f scripts/Dockerfile -t ${LOCAL_IMAGE} ."
    exit 1
fi

# Check if Azure CLI is installed
echo -e "\n${YELLOW}Step 2: Checking Azure CLI...${NC}"
if command -v az > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Azure CLI is installed${NC}"
else
    echo -e "${RED}✗ Azure CLI is not installed${NC}"
    echo "Install from: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli"
    exit 1
fi

# Login to ACR
echo -e "\n${YELLOW}Step 3: Logging in to Azure Container Registry...${NC}"
echo "ACR: ${ACR_NAME}"
if az acr login --name ${ACR_NAME}; then
    echo -e "${GREEN}✓ Successfully logged in to ACR${NC}"
else
    echo -e "${RED}✗ Failed to login to ACR${NC}"
    echo "Make sure you have access to the ACR: ${ACR_NAME}"
    exit 1
fi

# Tag image for ACR
echo -e "\n${YELLOW}Step 4: Tagging image for ACR...${NC}"
echo "Tagging: ${LOCAL_IMAGE} -> ${ACR_IMAGE}"
if docker tag ${LOCAL_IMAGE} ${ACR_IMAGE}; then
    echo -e "${GREEN}✓ Image tagged successfully${NC}"
else
    echo -e "${RED}✗ Failed to tag image${NC}"
    exit 1
fi

# Push to ACR
echo -e "\n${YELLOW}Step 5: Pushing image to ACR...${NC}"
echo "This may take several minutes depending on your connection speed..."
if docker push ${ACR_IMAGE}; then
    echo -e "\n${GREEN}✓ Image successfully pushed to ACR!${NC}"
else
    echo -e "\n${RED}✗ Failed to push image to ACR${NC}"
    exit 1
fi

# Verify image in ACR
echo -e "\n${YELLOW}Step 6: Verifying image in ACR...${NC}"
if az acr repository show --name ${ACR_NAME} --repository bytebase/bytebase > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Image verified in ACR${NC}"

    # Show image details
    echo -e "\n${GREEN}Image Details:${NC}"
    az acr repository show-tags --name ${ACR_NAME} --repository bytebase/bytebase --output table
else
    echo -e "${YELLOW}⚠ Could not verify image in ACR${NC}"
fi

# Summary
echo -e "\n${GREEN}=== Push Complete! ===${NC}"
echo "Image location: ${ACR_IMAGE}"
echo ""
echo "Next steps:"
echo "1. Run the packaging validation:"
echo "   ./scripts/azure-marketplace/package.sh"
echo ""
echo "2. If validation passes, build the CNAB bundle:"
echo "   (The package.sh script will prompt you)"
echo ""
echo "3. Create your offer in Partner Center:"
echo "   https://partner.microsoft.com/dashboard"
