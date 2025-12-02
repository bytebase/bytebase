#!/bin/bash

# Copyright 2024 Bytebase Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Build script for sensitive data and approval flow features

set -euo pipefail

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROTO_DIR="${PROJECT_ROOT}/backend/api/v1"
GEN_DIR="${PROJECT_ROOT}/backend/generated-go/v1"
GO_MOD_FILE="${PROJECT_ROOT}/go.mod.sensitive_data"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if required tools are installed
check_tools() {
    echo -e "${YELLOW}Checking required tools...${NC}"
    
    # Check protoc
    if ! command -v protoc &> /dev/null; then
        echo -e "${RED}ERROR: protoc is not installed. Please install Protocol Buffers compiler.${NC}"
        exit 1
    fi
    
    # Check protoc-gen-go
    if ! command -v protoc-gen-go &> /dev/null; then
        echo -e "${RED}ERROR: protoc-gen-go is not installed. Please install with 'go install google.golang.org/protobuf/cmd/protoc-gen-go@latest'${NC}"
        exit 1
    fi
    
    # Check protoc-gen-go-grpc
    if ! command -v protoc-gen-go-grpc &> /dev/null; then
        echo -e "${RED}ERROR: protoc-gen-go-grpc is not installed. Please install with 'go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest'${NC}"
        exit 1
    fi
    
    # Check go
    if ! command -v go &> /dev/null; then
        echo -e "${RED}ERROR: Go is not installed. Please install Go.${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}All required tools are installed.${NC}"
}

# Generate Protobuf code
generate_protobuf() {
    echo -e "${YELLOW}Generating Protobuf code...${NC}"
    
    # Create generated directory if it doesn't exist
    mkdir -p "${GEN_DIR}"
    
    # Generate sensitive level service
    protoc \
        --proto_path="${PROTO_DIR}" \
        --go_out="${GEN_DIR}" \
        --go_opt=paths=source_relative \
        --go-grpc_out="${GEN_DIR}" \
        --go-grpc_opt=paths=source_relative \
        "${PROTO_DIR}/sensitive_level_service.proto"
    
    # Generate approval flow service
    protoc \
        --proto_path="${PROTO_DIR}" \
        --go_out="${GEN_DIR}" \
        --go_opt=paths=source_relative \
        --go-grpc_out="${GEN_DIR}" \
        --go-grpc_opt=paths=source_relative \
        "${PROTO_DIR}/approval_flow_service.proto"
    
    echo -e "${GREEN}Protobuf code generated successfully.${NC}"
}

# Build sensitive data service
build_service() {
    echo -e "${YELLOW}Building sensitive data and approval flow services...${NC}"
    
    # Build sensitive level service
    go build -o "${PROJECT_ROOT}/bin/sensitive_level_service" \
        "${PROJECT_ROOT}/backend/api/v1/sensitive_level_service.go"
    
    # Build approval flow service
    go build -o "${PROJECT_ROOT}/bin/approval_flow_service" \
        "${PROJECT_ROOT}/backend/api/v1/approval_flow_service.go"
    
    echo -e "${GREEN}Services built successfully.${NC}"
}

# Run tests
run_tests() {
    echo -e "${YELLOW}Running tests...${NC}"
    
    # Run sensitive level service tests
    go test -v "${PROJECT_ROOT}/backend/api/v1" -run "TestSensitiveLevelService"
    
    # Run approval flow service tests
    go test -v "${PROJECT_ROOT}/backend/api/v1" -run "TestApprovalFlowService"
    
    # Run integration tests
    go test -v "${PROJECT_ROOT}/backend/api/v1" -run "TestSensitiveLevelAndApprovalFlowIntegration"
    
    echo -e "${GREEN}All tests passed.${NC}"
}

# Generate documentation
generate_docs() {
    echo -e "${YELLOW}Generating documentation...${NC}"
    
    # Generate API documentation from Protobuf
    protoc \
        --proto_path="${PROTO_DIR}" \
        --doc_out="${PROJECT_ROOT}/docs/api" \
        --doc_opt=markdown,sensitive_data_approval_flow_api.md \
        "${PROTO_DIR}/sensitive_level_service.proto" \
        "${PROTO_DIR}/approval_flow_service.proto"
    
    echo -e "${GREEN}Documentation generated successfully.${NC}"
}

# Clean build artifacts
clean() {
    echo -e "${YELLOW}Cleaning build artifacts...${NC}"
    
    # Remove generated code
    rm -rf "${GEN_DIR}"
    
    # Remove binary files
    rm -f "${PROJECT_ROOT}/bin/sensitive_level_service"
    rm -f "${PROJECT_ROOT}/bin/approval_flow_service"
    
    # Remove test coverage files
    rm -f "${PROJECT_ROOT}/coverage.out"
    rm -rf "${PROJECT_ROOT}/coverage"
    
    echo -e "${GREEN}Cleaned successfully.${NC}"
}

# Show help
show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Build script for sensitive data and approval flow features."
    echo ""
    echo "Options:"
    echo "  -h, --help          Show this help message"
    echo "  -c, --check         Check required tools"
    echo "  -g, --generate      Generate Protobuf code"
    echo "  -b, --build         Build services"
    echo "  -t, --test          Run tests"
    echo "  -d, --docs          Generate documentation"
    echo "  -C, --clean         Clean build artifacts"
    echo "  -a, --all           Run all steps (check, generate, build, test, docs)"
    echo ""
    echo "Examples:"
    echo "  $0 --all              # Run all steps"
    echo "  $0 --generate --build # Generate and build"
    echo "  $0 --test             # Run tests"
}

# Main function
main() {
    # Parse command line arguments
    local check=false
    local generate=false
    local build=false
    local test=false
    local docs=false
    local clean=false
    local all=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -c|--check)
                check=true
                ;;
            -g|--generate)
                generate=true
                ;;
            -b|--build)
                build=true
                ;;
            -t|--test)
                test=true
                ;;
            -d|--docs)
                docs=true
                ;;
            -C|--clean)
                clean=true
                ;;
            -a|--all)
                all=true
                ;;
            *)
                echo -e "${RED}ERROR: Unknown option $1${NC}"
                show_help
                exit 1
                ;;
        esac
        shift
    done
    
    # If no options specified, show help
    if ! $check && ! $generate && ! $build && ! $test && ! $docs && ! $clean && ! $all; then
        show_help
        exit 0
    fi
    
    # If all is specified, run all steps
    if $all; then
        check=true
        generate=true
        build=true
        test=true
        docs=true
    fi
    
    # Run selected steps
    if $check; then
        check_tools
    fi
    
    if $generate; then
        generate_protobuf
    fi
    
    if $build; then
        build_service
    fi
    
    if $test; then
        run_tests
    fi
    
    if $docs; then
        generate_docs
    fi
    
    if $clean; then
        clean
    fi
    
    echo -e "${GREEN}Build process completed successfully.${NC}"
}

# Run main function
main "$@"