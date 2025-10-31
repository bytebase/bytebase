# Publishing Bytebase to Azure Marketplace

This guide walks you through publishing Bytebase as an Azure Container Offer on Azure Marketplace.

## Prerequisites

1. **Azure Account Setup**
   - Active Azure subscription
   - Partner Center account with publisher access
   - Azure Container Registry (ACR) in your publishing tenant

2. **Technical Requirements**
   - Docker installed locally
   - Azure CLI installed and configured
   - Access to push images to your ACR
   - Kubernetes knowledge (for testing)

3. **Tools**
   - Microsoft's container-package-app tool (Docker image)

## Directory Structure

```
bytebase/
├── scripts/azure-marketplace/
│   ├── manifest.yaml              # Package metadata
│   ├── createUiDefinition.json    # Azure portal deployment UI
│   ├── mainTemplate.json          # ARM template for deployment
│   ├── values-azure.yaml          # Azure-specific Helm values
│   └── README.md                  # This file
├── helm-charts/bytebase/          # Your existing Helm chart
│   ├── Chart.yaml
│   ├── values.yaml
│   └── templates/
│       ├── statefulset.yaml       # Needs modification (see below)
│       └── ...
└── scripts/
    └── Dockerfile                 # Your existing Dockerfile
```

## Step-by-Step Publishing Guide

### Step 1: Modify Helm Chart for Azure Marketplace

Azure Marketplace requires images to use the `global.azure.images` pattern. You need to update your `helm-charts/bytebase/templates/statefulset.yaml` file.

#### Current Image Reference (Line 114-119):
```yaml
{{- if $registryMirrorHost }}
image: {{ trimSuffix "/" $registryMirrorHost }}/bytebase/bytebase:{{ $version }}
{{- else }}
image: bytebase/bytebase:{{ $version }}
{{- end }}
```

#### Updated Image Reference (Add Azure support):
```yaml
{{- if .Values.global.azure.images.bytebase }}
# Azure Marketplace image path
image: {{ .Values.global.azure.images.bytebase.registry }}/{{ .Values.global.azure.images.bytebase.repository }}:{{ .Values.global.azure.images.bytebase.tag }}
{{- else if $registryMirrorHost }}
# Registry mirror path
image: {{ trimSuffix "/" $registryMirrorHost }}/bytebase/bytebase:{{ $version }}
{{- else }}
# Default Docker Hub path
image: bytebase/bytebase:{{ $version }}
{{- end }}
```

This change allows Azure Marketplace to retag and host images in their registry while maintaining backward compatibility with existing deployments.

### Step 2: Set Up Azure Container Registry

```bash
# Login to Azure
az login

# Create resource group (if needed)
az group create --name bytebase-marketplace --location eastus

# Create Azure Container Registry
az acr create \
  --resource-group bytebase-marketplace \
  --name bytebaseacr \
  --sku Premium \
  --location eastus

# Login to ACR
az acr login --name bytebaseacr

# Get ACR login server
ACR_LOGIN_SERVER=$(az acr show --name bytebaseacr --query loginServer --output tsv)
echo "ACR Login Server: $ACR_LOGIN_SERVER"
```

### Step 3: Push Bytebase Image to ACR

You have two options: use the official Bytebase image from Docker Hub, or build your own.

#### Option A: Use Official Image (Recommended)

```bash
# Pull the official Bytebase image
docker pull bytebase/bytebase:3.11.1

# Tag for ACR
docker tag bytebase/bytebase:3.11.1 $ACR_LOGIN_SERVER/bytebase/bytebase:3.11.1

# Push to ACR
docker push $ACR_LOGIN_SERVER/bytebase/bytebase:3.11.1
```

#### Option B: Build From Source

```bash
# From the root of your repository
cd /Users/ecmadao/Develop/Bytebase/ecmadao/bytebase

# Build the Docker image
docker build -f scripts/Dockerfile -t bytebase/bytebase:3.11.1 .

# Tag for ACR
docker tag bytebase/bytebase:3.11.1 $ACR_LOGIN_SERVER/bytebase/bytebase:3.11.1

# Push to ACR
docker push $ACR_LOGIN_SERVER/bytebase/bytebase:3.11.1
```

**Automated Script:**
```bash
./scripts/azure-marketplace/push-to-acr.sh
```

### Step 4: Update manifest.yaml with Your ACR

Edit `scripts/azure-marketplace/manifest.yaml` and update the registry and image references:

```yaml
registries:
  - name: <your-acr-name>.azurecr.io
    url: <your-acr-name>.azurecr.io

images:
  - image: <your-acr-name>.azurecr.io/bytebase/bytebase:3.11.1
    platform: linux/amd64
```

### Step 5: Build and Publish CNAB Bundle

The CNAB bundle must be built and published by Microsoft's CPA tool to ensure proper OCI artifact format.

#### Automated Script (Recommended):

```bash
cd azure-marketplace
./package.sh
```

This script will:
1. Validate all artifacts with `cpa verify`
2. Optionally build and publish the CNAB bundle with `cpa buildbundle`

#### Manual Process:

**Step 5a: Verify artifacts**

```bash
cd /Users/ecmadao/Develop/Bytebase/ecmadao/bytebase

docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $(pwd):/data \
  -w /data/azure-marketplace \
  mcr.microsoft.com/container-package-app:latest \
  cpa verify
```

**Step 5b: Build and publish CNAB bundle**

**IMPORTANT:** The CPA tool needs Azure CLI authentication to push to ACR. You must mount your Azure credentials:

```bash
# Ensure you're logged in to Azure
az login
az acr login --name <your-acr-name>

# Run CPA with Azure credentials mounted
docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $(pwd)/..:/data \
  -v $HOME/.azure:/root/.azure \
  -w /data/azure-marketplace \
  mcr.microsoft.com/container-package-app:latest \
  /bin/bash -c "az acr login --name <your-acr-name> && cpa buildbundle --force"
```

**What this does:**
- Packages Helm chart + ARM template + UI definition into a CNAB bundle
- Creates Porter invocation images with proper OCI annotations
- Publishes to ACR with CNAB-compliant manifest format (required by Partner Center)
- Tags bundle as `<your-acr-name>.azurecr.io/bytebase:3.11.1`

**Verify the bundle:**

```bash
# Check the bundle manifest has proper CNAB annotations
az acr manifest show --registry <your-acr-name> --name bytebase:3.11.1 | grep "io.cnab"
```

You should see annotations like:
- `io.cnab.runtime_version: 1.2.0`
- `org.opencontainers.artifactType: application/vnd.cnab.manifest.v1`

**Troubleshooting:**

Common authentication issues include:
- CPA tool cannot access Docker credentials
- Azure CLI session not properly mounted
- ACR permissions not configured correctly

The key is that the CPA tool itself must push the bundle - manual `docker push` will not create the proper CNAB/OCI artifact format that Partner Center requires.

### Step 6: Test Your createUiDefinition

Before submitting to Partner Center, test your deployment UI:

1. Go to Azure Portal CreateUI Sandbox:
   https://portal.azure.com/#blade/Microsoft_Azure_CreateUIDef/SandboxBlade

2. Copy the contents of `scripts/azure-marketplace/createUiDefinition.json`

3. Paste and test the UI flow

4. Verify all fields, validations, and outputs work correctly

### Step 7: Create Partner Center Offer

For detailed, step-by-step instructions on creating your Partner Center offer, see:

📖 **[PARTNER_CENTER_GUIDE.md](./PARTNER_CENTER_GUIDE.md)** - Complete walkthrough with field-by-field instructions

**Quick Overview:**

1. **Access Partner Center**: https://partner.microsoft.com/dashboard
2. **Create Offer**: Commercial Marketplace → New Offer → Azure Container
3. **Offer ID**: `bytebase` (permanent, must be unique)
4. **Configure Sections**:
   - Offer Properties (categories, legal documents)
   - Offer Listing (description, screenshots, logo)
   - Preview Audience (test subscription IDs)
   - **Technical Configuration**: `<your-acr-name>.azurecr.io/bytebase:3.11.1`
   - Plans (create at least one - BYOL recommended for initial launch)
5. **Review and Publish**: Submit for certification

**Required Materials:**
- Legal documents (Privacy Policy, Terms of Service URLs)
- Logo files (48x48, 90x90, 216x216, 255x115 PNG)
- Screenshots (at least 1, max 5, 1280x720 or 1920x1080 PNG)
- Support contact information

**Timeline:** 2-4 weeks for Microsoft certification after submission

### Step 8: Submit for Certification

1. **Review and Submit**
   - Review all sections in Partner Center
   - Click "Review and publish"
   - Submit for certification

2. **Certification Process**
   - Microsoft performs security scanning on your images
   - Reviews all marketplace metadata
   - Tests deployment flow
   - **Timeline**: Typically 2-4 weeks

3. **Address Feedback**
   - Microsoft may request changes
   - Address any security vulnerabilities
   - Update and resubmit as needed

### Step 9: Publish

Once certification passes:
1. Approve the preview version
2. Click "Go live"
3. Your offer becomes available on Azure Marketplace!

## Testing Your Deployment

Before submitting, test locally:

```bash
# Install your Helm chart with Azure values
helm install bytebase-test ./helm-charts/bytebase \
  -f scripts/azure-marketplace/values-azure.yaml \
  --set bytebase.option.externalPg.url="postgresql://user:pass@host:5432/db" \
  --set bytebase.option.external-url="https://test.example.com" \
  --namespace bytebase \
  --create-namespace

# Verify deployment
kubectl get pods -n bytebase
kubectl logs -n bytebase -l app=bytebase

# Clean up
helm uninstall bytebase-test -n bytebase
```

## Pricing Strategy Considerations

### BYOL (Recommended for Initial Launch)
**Pros:**
- Faster to market (simpler implementation)
- Full control over pricing and billing
- No Azure Metering API integration needed
- Flexible pricing changes

**Cons:**
- Separate billing (not on Azure invoice)
- Requires your own license management system

### Marketplace Metered Billing
**Pros:**
- Unified Azure billing (appears on customer's Azure invoice)
- Azure handles payment processing
- Builds trust with enterprise customers

**Cons:**
- Requires Azure Metering API integration in your application
- More complex to implement and test
- Longer certification process

## Marketplace Revenue Share

Microsoft takes a commission on marketplace transactions:
- **Standard**: ~20% for transactable offers
- **IP co-sell eligible**: ~3% (requires co-sell setup)

BYOL offers still have a marketplace fee (~3%) for listing.

## Updating Your Offer

After publishing, to release new versions:

1. Build new Docker image with updated version tag
2. Push to ACR
3. Update `manifest.yaml` with new version
4. Run `cpa buildbundle` to create new CNAB
5. In Partner Center:
   - Update Technical Configuration with new CNAB path
   - Increment plan version
   - Submit for re-certification

## Important: CNAB Bundle Format

**Critical Requirement:** Partner Center requires CNAB bundles in proper OCI artifact format, not regular Docker v2 manifests.

**What This Means:**
- ✅ Bundle must be published by CPA tool (`cpa buildbundle`)
- ❌ Manual `docker push` will NOT work (creates wrong manifest type)
- ✅ Bundle must have OCI annotations like `io.cnab.runtime_version` and `org.opencontainers.artifactType`

**How to Verify:**
```bash
# Check your bundle format
az acr manifest show --registry <your-acr-name> --name bytebase:3.11.1

# Should show:
# - annotations.io.cnab.runtime_version: "1.2.0"
# - annotations.org.opencontainers.artifactType: "application/vnd.cnab.manifest.v1"
# - mediaType: "application/vnd.oci.image.manifest.v1+json" (for manifests)
```

If your bundle shows `mediaType: "application/vnd.docker.distribution.manifest.v2+json"` instead, it was pushed incorrectly and Partner Center will reject it with error: *"The artifact you selected is not a valid Cloud Native Application Bundle (CNAB)"*

**Solution:** Delete the incorrect bundle and republish using the CPA tool with Azure CLI authentication as described in Step 5b.

## Common Issues and Solutions

### Issue 1: Partner Center Rejects Bundle
**Problem**: Error "The artifact you selected is not a valid Cloud Native Application Bundle (CNAB)"
**Solution**:
- Verify bundle format with `az acr manifest show` command
- If incorrect format, republish using CPA tool (not docker push)
- Follow Step 5b instructions for proper authentication

### Issue 2: Image Pull Errors
**Problem**: ACR images can't be pulled during packaging
**Solution**: Ensure you're logged into ACR: `az acr login --name <acr-name>`

### Issue 3: CPA Authentication Failures
**Problem**: `cpa buildbundle` fails with authentication errors
**Solution**:
- Ensure Azure CLI is logged in: `az login`
- Login to ACR: `az acr login --name <acr-name>`
- Mount Azure credentials when running CPA: `-v $HOME/.azure:/root/.azure`
- See Step 5b for complete command

### Issue 4: Validation Failures
**Problem**: `cpa verify` fails with Helm chart errors
**Solution**:
- Ensure Helm chart is unpacked (not .tgz)
- Verify `Chart.yaml` apiVersion is v2
- Check all template references are valid

### Issue 5: createUiDefinition Errors
**Problem**: UI definition doesn't load in sandbox
**Solution**:
- Validate JSON syntax
- Check all parameter references match ARM template
- Test incrementally (comment out sections to isolate issues)

### Issue 6: Security Scan Failures
**Problem**: Certification blocked by vulnerability scan
**Solution**:
- Update base images in Dockerfile
- Run `docker scan` locally before submitting
- Address all HIGH and CRITICAL vulnerabilities

## Support and Resources

- **Azure Marketplace Docs**: https://learn.microsoft.com/en-us/partner-center/marketplace-offers/marketplace-containers
- **Prepare Azure container technical assets for a Kubernetes app**: https://learn.microsoft.com/en-us/partner-center/marketplace-offers/azure-container-technical-assets-kubernetes
- **Mastering the Marketplace**: https://microsoft.github.io/Mastering-the-Marketplace/container/
- **CreateUI Sandbox**: https://portal.azure.com/#blade/Microsoft_Azure_CreateUIDef/SandboxBlade
- **Partner Center**: https://partner.microsoft.com/dashboard
