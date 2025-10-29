#!/bin/bash

# Azure Marketplace Packaging Script for Bytebase
# This script helps automate the packaging process for Azure Marketplace submission

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
BYTEBASE_VERSION="3.11.1"
PACKAGE_TOOL_IMAGE="mcr.microsoft.com/container-package-app:latest"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
AZURE_MARKETPLACE_DIR="${REPO_ROOT}/scripts/azure-marketplace"

echo -e "${GREEN}=== Bytebase Azure Marketplace Packaging Tool ===${NC}\n"

# Function to print step headers
print_step() {
    echo -e "\n${YELLOW}>>> $1${NC}\n"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check prerequisites
print_step "Step 1: Checking prerequisites"

if ! command_exists docker; then
    echo -e "${RED}Error: Docker is not installed${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Docker is installed${NC}"

if ! command_exists az; then
    echo -e "${YELLOW}Warning: Azure CLI is not installed. You'll need it for ACR operations.${NC}"
    echo "Install from: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli"
else
    echo -e "${GREEN}✓ Azure CLI is installed${NC}"
fi

if ! command_exists helm; then
    echo -e "${YELLOW}Warning: Helm is not installed. Recommended for local testing.${NC}"
else
    echo -e "${GREEN}✓ Helm is installed${NC}"
fi

# Check if required files exist
print_step "Step 2: Verifying Azure Marketplace artifacts"

required_files=(
    "scripts/azure-marketplace/manifest.yaml"
    "scripts/azure-marketplace/createUiDefinition.json"
    "scripts/azure-marketplace/mainTemplate.json"
    "scripts/azure-marketplace/values-azure.yaml"
    "helm-charts/bytebase/Chart.yaml"
    "helm-charts/bytebase/values.yaml"
)

all_files_exist=true
for file in "${required_files[@]}"; do
    if [ -f "${REPO_ROOT}/${file}" ]; then
        echo -e "${GREEN}✓ ${file}${NC}"
    else
        echo -e "${RED}✗ ${file} not found${NC}"
        all_files_exist=false
    fi
done

if [ "$all_files_exist" = false ]; then
    echo -e "\n${RED}Error: Some required files are missing${NC}"
    exit 1
fi

# Pull the packaging tool
print_step "Step 3: Pulling Microsoft's container-package-app tool"
docker pull ${PACKAGE_TOOL_IMAGE}

# Run validation
print_step "Step 4: Running validation (cpa verify)"
echo "This will validate all your Azure Marketplace artifacts..."

cd "${REPO_ROOT}"

if docker run --rm \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v "${REPO_ROOT}":/data \
    ${PACKAGE_TOOL_IMAGE} \
    cpa verify; then
    echo -e "\n${GREEN}✓ Validation passed!${NC}"
else
    echo -e "\n${RED}✗ Validation failed!${NC}"
    echo "Please review the errors above and fix them before proceeding."
    exit 1
fi

# Ask if user wants to proceed with build
print_step "Step 5: Build CNAB bundle"
echo "Validation passed successfully!"
echo -e "${YELLOW}Do you want to build and upload the CNAB bundle now?${NC}"
echo "Note: This requires:"
echo "  1. Your Docker image to be pushed to Azure Container Registry"
echo "  2. You to be logged in to ACR (az acr login --name <acr-name>)"
echo ""
read -p "Proceed with build? (y/N): " -n 1 -r
echo

if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "\n${YELLOW}Building CNAB bundle...${NC}"

    if docker run --rm \
        -v /var/run/docker.sock:/var/run/docker.sock \
        -v "${REPO_ROOT}":/data \
        ${PACKAGE_TOOL_IMAGE} \
        cpa buildbundle; then
        echo -e "\n${GREEN}✓ CNAB bundle built and uploaded successfully!${NC}"
        echo -e "\nYou can now proceed to Partner Center to create your offer."
        echo "Follow the instructions in azure-marketplace/README.md (Step 7)"
    else
        echo -e "\n${RED}✗ CNAB bundle build failed!${NC}"
        echo "Common issues:"
        echo "  1. Not logged in to ACR: az acr login --name <acr-name>"
        echo "  2. Docker images not pushed to ACR"
        echo "  3. Image references in manifest.yaml don't match ACR"
        exit 1
    fi
else
    echo -e "\n${YELLOW}Skipping CNAB build.${NC}"
    echo "When you're ready, run:"
    echo "  docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -v \$(pwd):/data ${PACKAGE_TOOL_IMAGE} cpa buildbundle"
fi

# Summary
print_step "Next Steps"
echo "1. If you haven't already:"
echo "   - Set up Azure Container Registry"
echo "   - Build and push your Docker image to ACR"
echo "   - Update scripts/azure-marketplace/manifest.yaml with your ACR details"
echo ""
echo "2. Test your createUiDefinition.json:"
echo "   https://portal.azure.com/#blade/Microsoft_Azure_CreateUIDef/SandboxBlade"
echo ""
echo "3. Create your offer in Partner Center:"
echo "   https://partner.microsoft.com/dashboard"
echo ""
echo "4. For detailed instructions, see:"
echo "   scripts/azure-marketplace/README.md"
echo ""
echo -e "${GREEN}Good luck with your Azure Marketplace submission! 🚀${NC}"
