<#
Copyright 2024 Bytebase Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Build script for sensitive data and approval flow features (PowerShell version)
#>

param(
    [switch]$Check,
    [switch]$Generate,
    [switch]$Build,
    [switch]$Test,
    [switch]$Docs,
    [switch]$Clean,
    [switch]$All,
    [switch]$Help
)

# Configuration
$ProjectRoot = Split-Path -Path $MyInvocation.MyCommand.Definition -Parent
$ProtoDir = Join-Path -Path $ProjectRoot -ChildPath "backend/api/v1"
$GenDir = Join-Path -Path $ProjectRoot -ChildPath "backend/generated-go/v1"
$GoModFile = Join-Path -Path $ProjectRoot -ChildPath "go.mod.sensitive_data"

# Colors for output
$Red = "`e[31m"
$Green = "`e[32m"
$Yellow = "`e[33m"
$NC = "`e[0m" # No Color

# Check if required tools are installed
function Check-Tools {
    Write-Host "$Yellow Checking required tools...$NC"
    
    # Check protoc
    if (-not (Get-Command protoc -ErrorAction SilentlyContinue)) {
        Write-Host "$Red ERROR: protoc is not installed. Please install Protocol Buffers compiler.$NC"
        exit 1
    }
    
    # Check protoc-gen-go
    if (-not (Get-Command protoc-gen-go -ErrorAction SilentlyContinue)) {
        Write-Host "$Red ERROR: protoc-gen-go is not installed. Please install with 'go install google.golang.org/protobuf/cmd/protoc-gen-go@latest'$NC"
        exit 1
    }
    
    # Check protoc-gen-go-grpc
    if (-not (Get-Command protoc-gen-go-grpc -ErrorAction SilentlyContinue)) {
        Write-Host "$Red ERROR: protoc-gen-go-grpc is not installed. Please install with 'go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest'$NC"
        exit 1
    }
    
    # Check go
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Write-Host "$Red ERROR: Go is not installed. Please install Go.$NC"
        exit 1
    }
    
    Write-Host "$Green All required tools are installed.$NC"
}

# Generate Protobuf code
function Generate-Protobuf {
    Write-Host "$Yellow Generating Protobuf code...$NC"
    
    # Create generated directory if it doesn't exist
    if (-not (Test-Path -Path $GenDir)) {
        New-Item -ItemType Directory -Path $GenDir | Out-Null
    }
    
    # Generate sensitive level service
    protoc `
        --proto_path="$ProtoDir" `
        --go_out="$GenDir" `
        --go_opt=paths=source_relative `
        --go-grpc_out="$GenDir" `
        --go-grpc_opt=paths=source_relative `
        "$ProtoDir/sensitive_level_service.proto"
    
    # Generate approval flow service
    protoc `
        --proto_path="$ProtoDir" `
        --go_out="$GenDir" `
        --go_opt=paths=source_relative `
        --go-grpc_out="$GenDir" `
        --go-grpc_opt=paths=source_relative `
        "$ProtoDir/approval_flow_service.proto"
    
    Write-Host "$Green Protobuf code generated successfully.$NC"
}

# Build sensitive data service
function Build-Service {
    Write-Host "$Yellow Building sensitive data and approval flow services...$NC"
    
    # Create bin directory if it doesn't exist
    $BinDir = Join-Path -Path $ProjectRoot -ChildPath "bin"
    if (-not (Test-Path -Path $BinDir)) {
        New-Item -ItemType Directory -Path $BinDir | Out-Null
    }
    
    # Build sensitive level service
    go build -o "$BinDir/sensitive_level_service.exe" `
        "$ProjectRoot/backend/api/v1/sensitive_level_service.go"
    
    # Build approval flow service
    go build -o "$BinDir/approval_flow_service.exe" `
        "$ProjectRoot/backend/api/v1/approval_flow_service.go"
    
    Write-Host "$Green Services built successfully.$NC"
}

# Run tests
function Run-Tests {
    Write-Host "$Yellow Running tests...$NC"
    
    # Run sensitive level service tests
    go test -v "$ProjectRoot/backend/api/v1" -run "TestSensitiveLevelService"
    
    # Run approval flow service tests
    go test -v "$ProjectRoot/backend/api/v1" -run "TestApprovalFlowService"
    
    # Run integration tests
    go test -v "$ProjectRoot/backend/api/v1" -run "TestSensitiveLevelAndApprovalFlowIntegration"
    
    Write-Host "$Green All tests passed.$NC"
}

# Generate documentation
function Generate-Docs {
    Write-Host "$Yellow Generating documentation...$NC"
    
    # Create docs/api directory if it doesn't exist
    $DocsDir = Join-Path -Path $ProjectRoot -ChildPath "docs/api"
    if (-not (Test-Path -Path $DocsDir)) {
        New-Item -ItemType Directory -Path $DocsDir | Out-Null
    }
    
    # Generate API documentation from Protobuf
    protoc `
        --proto_path="$ProtoDir" `
        --doc_out="$DocsDir" `
        --doc_opt=markdown,sensitive_data_approval_flow_api.md `
        "$ProtoDir/sensitive_level_service.proto" `
        "$ProtoDir/approval_flow_service.proto"
    
    Write-Host "$Green Documentation generated successfully.$NC"
}

# Clean build artifacts
function Clean-Build {
    Write-Host "$Yellow Cleaning build artifacts...$NC"
    
    # Remove generated code
    if (Test-Path -Path $GenDir) {
        Remove-Item -Path $GenDir -Recurse -Force
    }
    
    # Remove binary files
    $BinDir = Join-Path -Path $ProjectRoot -ChildPath "bin"
    if (Test-Path -Path $BinDir) {
        Remove-Item -Path "$BinDir/sensitive_level_service.exe" -Force -ErrorAction SilentlyContinue
        Remove-Item -Path "$BinDir/approval_flow_service.exe" -Force -ErrorAction SilentlyContinue
    }
    
    # Remove test coverage files
    Remove-Item -Path "$ProjectRoot/coverage.out" -Force -ErrorAction SilentlyContinue
    if (Test-Path -Path "$ProjectRoot/coverage") {
        Remove-Item -Path "$ProjectRoot/coverage" -Recurse -Force
    }
    
    Write-Host "$Green Cleaned successfully.$NC"
}

# Show help
function Show-Help {
    Write-Host "Usage: $($MyInvocation.MyCommand.Name) [OPTIONS]"
    Write-Host ""
    Write-Host "Build script for sensitive data and approval flow features."
    Write-Host ""
    Write-Host "Options:"
    Write-Host "  -Check         Check required tools"
    Write-Host "  -Generate      Generate Protobuf code"
    Write-Host "  -Build         Build services"
    Write-Host "  -Test          Run tests"
    Write-Host "  -Docs          Generate documentation"
    Write-Host "  -Clean         Clean build artifacts"
    Write-Host "  -All           Run all steps (check, generate, build, test, docs)"
    Write-Host "  -Help          Show this help message"
    Write-Host ""
    Write-Host "Examples:"
    Write-Host "  .\build_sensitive_data.ps1 -All              # Run all steps"
    Write-Host "  .\build_sensitive_data.ps1 -Generate -Build  # Generate and build"
    Write-Host "  .\build_sensitive_data.ps1 -Test             # Run tests"
}

# Main function
function Main {
    # If help is requested, show help
    if ($Help) {
        Show-Help
        exit 0
    }
    
    # If no options specified, show help
    if (-not $Check -and -not $Generate -and -not $Build -and -not $Test -and -not $Docs -and -not $Clean -and -not $All) {
        Show-Help
        exit 0
    }
    
    # If all is specified, run all steps
    if ($All) {
        $Check = $true
        $Generate = $true
        $Build = $true
        $Test = $true
        $Docs = $true
    }
    
    # Run selected steps
    if ($Check) {
        Check-Tools
    }
    
    if ($Generate) {
        Generate-Protobuf
    }
    
    if ($Build) {
        Build-Service
    }
    
    if ($Test) {
        Run-Tests
    }
    
    if ($Docs) {
        Generate-Docs
    }
    
    if ($Clean) {
        Clean-Build
    }
    
    Write-Host "$Green Build process completed successfully.$NC"
}

# Run main function
Main