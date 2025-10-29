# Partner Center Setup Guide for Bytebase

This guide provides detailed, step-by-step instructions for creating your Bytebase offer in Microsoft Partner Center.

## Prerequisites

Before you start, ensure you have:
- ✅ Partner Center account with publisher access
- ✅ CNAB bundle uploaded to ACR: `bytebase.azurecr.io/bytebase:3.11.1`
- ✅ Legal documents ready (Terms of Service, Privacy Policy URLs)
- ✅ Marketing materials (screenshots, description, logo)
- ✅ Support contact information

---

## Part 1: Create New Offer (10 minutes)

### Step 1: Access Partner Center

1. Navigate to: https://partner.microsoft.com/dashboard
2. Sign in with your Microsoft account
3. Click on **"Commercial Marketplace"** in the left navigation
4. Click **"Overview"**

### Step 2: Create New Offer

1. Click **"+ New offer"** button (top right)
2. Select **"Azure Container"** from the dropdown
3. A popup will appear: **"New Azure Container offer"**

4. Fill in the form:
   - **Offer ID**: `bytebase`
     - This is permanent and cannot be changed
     - Must be unique in your publisher namespace
     - Use lowercase, no spaces (hyphens allowed)

   - **Offer alias**: `Bytebase - Database CI/CD`
     - Internal name for your reference
     - Can be changed later

5. Click **"Create"**

### Step 3: Initial Setup

The offer will be created and you'll see the offer management page with tabs:
- Offer setup
- Properties
- Offer listing
- Preview audience
- Technical configuration
- Plan overview
- Co-sell with Microsoft (optional)
- Resell through CSPs (optional)

---

## Part 2: Offer Setup (5 minutes)

Click on **"Offer setup"** tab.

### Customer Leads (Optional but Recommended)

1. **Connect a CRM system** (optional):
   - If you want to receive customer contact info
   - Options: Azure Table, Dynamics 365, Marketo, Salesforce, HTTPS endpoint
   - For now, you can skip this

2. **Test connection**: Click "Validate" if you configured CRM

3. Click **"Save draft"** at the bottom

---

## Part 3: Properties (5 minutes)

Click on **"Properties"** tab.

### Categories

Select up to 2 primary categories:
1. **Primary**: `Databases`
2. **Secondary**: `Developer Tools`

### Legal

You must provide legal documents:

1. **Standard Contract**:
   - ☑ Check "Use the Standard Contract for Microsoft's commercial marketplace"
   - This simplifies legal compliance

2. **Terms and Conditions** (If NOT using Standard Contract):
   - Enter URL to your Terms of Service
   - Example: `https://www.bytebase.com/terms`

3. **Privacy Policy**:
   - **Required**: Enter URL to your Privacy Policy
   - Example: `https://www.bytebase.com/privacy`

4. Click **"Save draft"**

---

## Part 4: Offer Listing (20 minutes)

Click on **"Offer listing"** tab.

This is what customers see on Azure Marketplace. Take time to make it compelling!

### Basic Information

1. **Name**: `Bytebase`
   - This is the display name customers see
   - Keep it simple and recognizable

2. **Search results summary** (100 characters max):
   ```
   Safe database schema change and version control for DevOps teams
   ```

3. **Short description** (256 characters max):
   ```
   Bytebase is a database CI/CD tool that helps DevOps teams manage database schema changes safely and efficiently. Deploy schema migrations, track changes, and enforce policies across all your databases from a single platform.
   ```

4. **Description** (3000 characters max):
   ```markdown
   ## Database Schema Change and Version Control

   Bytebase is an all-in-one database CI/CD solution that helps developers and DBAs manage database schema changes throughout the application development lifecycle.

   ### Key Features

   **Schema Migration**
   - Visual schema editor with SQL syntax highlighting
   - Automated migration scripts generation
   - Rollback support for failed migrations
   - Multi-environment deployment workflows

   **Version Control Integration**
   - GitOps-style database change management
   - Integrated with GitHub, GitLab, and Bitbucket
   - Pull request-based review workflow
   - Automated deployment on merge

   **Policy Enforcement**
   - Customizable approval workflows
   - SQL review and best practices enforcement
   - Automated schema validation
   - Compliance audit trails

   **Database Support**
   - MySQL, PostgreSQL, Oracle, SQL Server
   - MongoDB, Redis, Snowflake, BigQuery
   - And many more database engines

   **Team Collaboration**
   - Role-based access control (RBAC)
   - Change history and audit logs
   - Slack and email notifications
   - Multi-project and multi-environment support

   ### Why Bytebase?

   - **Safety**: Prevent schema migration failures with automated validation
   - **Efficiency**: Streamline database change workflows for your team
   - **Compliance**: Built-in audit trails and approval processes
   - **Scalability**: Manage hundreds of databases across environments

   ### Get Started

   Deploy Bytebase to your Azure Kubernetes Service (AKS) cluster in minutes. Configure your external PostgreSQL database for metadata storage, and you're ready to manage your database schema changes.

   ### Support

   - Documentation: https://docs.bytebase.com
   - Community: https://github.com/bytebase/bytebase
   - Enterprise Support: support@bytebase.com
   ```

### Links

1. **Privacy policy URL**: `https://www.bytebase.com/privacy`
2. **Support website**: `https://docs.bytebase.com`
3. **Engineering contact** (private, for Microsoft):
   - Name: `[Your Name]`
   - Email: `support@bytebase.com`
   - Phone: `[Your Phone]`

4. **Support contact** (public, shown to customers):
   - Name: `Bytebase Support`
   - Email: `support@bytebase.com`
   - Phone: `[Optional]`

### Marketing Materials

#### Supporting Documents (Optional)
- Upload PDFs, videos, or other marketing materials
- Examples: product briefs, case studies, whitepapers

#### Marketplace Media

**Logo (Required)**:
- Small (48x48 pixels): PNG format
- Medium (90x90 pixels): PNG format
- Large (216x216 pixels): PNG format
- Wide (255x115 pixels): PNG format

> **Tip**: Use your Bytebase logo from: `helm-charts/bytebase/Chart.yaml` (icon field)
> You'll need to download and resize it to the required dimensions.

**Screenshots (Required - At least 1, up to 5)**:
- Size: 1280x720 or 1920x1080 pixels
- Format: PNG
- Include captions (max 100 characters each)

Suggested screenshots:
1. **Main Dashboard**: Show database overview and recent changes
   - Caption: "Centralized database management dashboard"
2. **Schema Migration**: Show migration editor and workflow
   - Caption: "Visual schema migration editor with SQL support"
3. **Version Control**: Show GitOps integration
   - Caption: "GitOps-based database change management"
4. **Approval Workflow**: Show review and approval process
   - Caption: "Custom approval workflows for compliance"
5. **Audit Trail**: Show change history
   - Caption: "Complete audit trail of all database changes"

**Videos (Optional - Up to 4)**:
- YouTube or Vimeo links
- Include thumbnail (1280x720 pixels)
- Examples:
  - Product overview (2-3 minutes)
  - Getting started tutorial
  - Feature deep dives

### Additional Links (Optional)

- `https://www.bytebase.com/docs` (Documentation)
- `https://github.com/bytebase/bytebase` (GitHub)
- `https://www.bytebase.com/pricing` (Pricing)

5. Click **"Save draft"**

---

## Part 5: Preview Audience (2 minutes)

Click on **"Preview audience"** tab.

Add Azure subscription IDs that can access your offer before it goes live:

1. **Azure subscription ID**: Your test subscription ID
   - Format: `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`
   - Find this in Azure Portal → Subscriptions

2. **Description**: `Internal testing`

3. Click **"+ Add subscription ID"** to add more if needed

4. Click **"Save draft"**

---

## Part 6: Technical Configuration (10 minutes)

Click on **"Technical configuration"** tab.

This is where you link your CNAB bundle.

### Cluster Type

1. **Select cluster type**:
   - ☑ **Azure Kubernetes Service managed cluster**
   - ☐ Azure Arc-enabled Kubernetes cluster

### Kubernetes Application Package

1. **Package type**: Select `CNAB`

2. **Package location**:
   ```
   bytebase.azurecr.io/bytebase:3.11.1
   ```
   - This is your CNAB bundle path from Step 5 of README.md
   - Format: `<acr-name>.azurecr.io/<repository>:<tag>`
   - **Important**: Must be in CNAB/OCI format (not Docker v2 manifest)

3. **Grant access**:
   - Click **"Grant access"**
   - This allows Microsoft to access your ACR to copy images
   - Follow the prompts to authorize

### Application Insights (Optional)

- If you want telemetry, configure Application Insights
- For now, you can skip this

4. Click **"Save draft"**

---

## Part 7: Plan Overview - Create Plans (15 minutes)

Click on **"Plan overview"** tab.

You must create at least one plan. Let's create a BYOL plan:

### Create BYOL Plan

1. Click **"+ Create new plan"**

2. **Plan ID**: `bytebase-byol`
   - Permanent, cannot be changed
   - Lowercase, no spaces

3. **Plan name**: `Bytebase BYOL`
   - Display name customers see
   - Can be changed later

4. Click **"Create"**

### Configure the Plan

You'll see tabs for the plan:

#### Plan Setup Tab

1. **Azure regions**: Default is all regions (recommended)

2. **Plan type**: Select `BYOL (Bring Your Own License)`

3. **Visibility**:
   - ◉ **Public** (recommended for launch)
   - ○ Private (only for specific customers)

4. Click **"Save draft"**

#### Plan Listing Tab

1. **Plan name**: `Bytebase BYOL`

2. **Plan summary** (100 characters):
   ```
   Bring your own Bytebase license and deploy to your AKS cluster
   ```

3. **Plan description** (2000 characters):
   ```markdown
   ## Bytebase BYOL Plan

   Deploy Bytebase to your Azure Kubernetes Service (AKS) cluster with your own Bytebase license.

   ### What's Included

   - Full Bytebase application deployment via Helm
   - Deploy to your AKS cluster
   - Persistent storage configuration
   - External PostgreSQL database integration
   - ConfigureConfigurable resource limits

   ### Requirements

   - Active Bytebase license (purchase separately from bytebase.com)
   - Azure Kubernetes Service (AKS) cluster
   - External PostgreSQL database for metadata storage
   - Minimum 2 vCPU and 4GB RAM recommended

   ### License Information

   You must purchase a Bytebase license separately. Visit https://www.bytebase.com/pricing for license options and pricing.

   Contact support@bytebase.com for enterprise licensing or questions.

   ### Support

   - Documentation: https://docs.bytebase.com
   - Community Support: https://github.com/bytebase/bytebase
   - Enterprise Support: support@bytebase.com
   ```

4. Click **"Save draft"**

#### Pricing and Availability Tab

1. **Markets**:
   - Select all markets OR specific countries/regions
   - Recommended: **All markets**

2. **Pricing**:
   - BYOL has no marketplace pricing
   - Customers pay you directly for licenses

3. **Free trial**: Not applicable for BYOL

4. Click **"Save draft"**

#### Technical Configuration Tab (Plan-specific)

1. **Application version**: `3.11.1` (matches your Bytebase version)

2. **Package reference**: Should auto-populate from offer-level technical config

3. **Configuration settings** (optional):
   - You can override Helm values per plan
   - For BYOL, defaults are usually fine

4. Click **"Save draft"**

---

## Part 8: Co-sell with Microsoft (Optional - 10 minutes)

Click on **"Co-sell with Microsoft"** tab.

This is optional but recommended for increased visibility.

### Documents

1. **Upload co-sell documents**:
   - Solution overview (PDF)
   - Reference architecture diagram
   - Customer success stories

### Solution Areas

Select relevant areas:
- ☑ Application Development
- ☑ Data and AI
- ☑ DevOps

2. Click **"Save draft"**

---

## Part 9: Resell through CSPs (Optional - 2 minutes)

Click on **"Resell through CSPs"** tab.

Allow Cloud Solution Providers to resell your offer:

1. **CSP channel**:
   - ◉ No partners in the CSP program
   - ○ Any partner in the CSP program
   - ○ Specific partners in the CSP program

2. For wider reach, select **"Any partner in the CSP program"**

3. Click **"Save draft"**

---

## Part 10: Review and Publish (5 minutes)

### Final Review

1. Go through each tab and ensure all required fields are filled:
   - ✅ Offer setup
   - ✅ Properties
   - ✅ Offer listing
   - ✅ Preview audience
   - ✅ Technical configuration
   - ✅ Plan overview (at least 1 plan)

2. Look for any red warnings or missing fields

### Publish to Preview

1. Click **"Review and publish"** button (top right)

2. Review the submission checklist

3. Click **"Publish"**

4. Confirmation:
   ```
   Your offer has been submitted for publishing.
   Status: Publisher signoff (Preview)
   ```

### What Happens Next

**Phase 1: Publisher Signoff (Preview)** - 1-2 hours
- Microsoft validates your submission
- Your offer becomes available to preview audience
- Test your offer deployment

**Phase 2: Publisher Approval** - You must approve
- Test the preview version
- Deploy to your test AKS cluster
- Verify everything works
- Click **"Go live"** when ready

**Phase 3: Certification** - 2-4 weeks
- Microsoft security scanning
- Image vulnerability assessment
- Metadata review
- Deployment testing

**Phase 4: Live on Marketplace** - Automatic
- Once certified, offer goes live
- Available to all Azure customers
- Appears in search results

---

## Testing Your Preview Offer

Once in preview status:

1. **Access the preview**:
   - Use an Azure subscription you added to preview audience
   - Go to: https://portal.azure.com/#create/
   - Search for your offer (may take an hour to appear)

2. **Deploy to test AKS**:
   - Follow the deployment wizard
   - Configure PostgreSQL connection
   - Deploy to test namespace

3. **Verify functionality**:
   - Check pods are running: `kubectl get pods -n bytebase`
   - Access Bytebase UI via external URL
   - Test database connections
   - Verify license activation (BYOL)

4. **If issues found**:
   - Return to Partner Center
   - Update configuration
   - Re-submit for review

5. **When ready**:
   - Click **"Go live"** in Partner Center
   - Offer proceeds to certification

---

## Common Issues and Solutions

### Issue: "Package location not accessible"
**Solution**: Ensure you ran `az acr login` and granted Partner Center access to your ACR

### Issue: "Invalid CNAB bundle"
**Solution**:
- Re-run `./scripts/azure-marketplace/package.sh`
- Ensure `cpa verify` passes without errors
- Check manifest.yaml references correct images

### Issue: "Missing required screenshots"
**Solution**: Upload at least one screenshot (1280x720 or 1920x1080 PNG)

### Issue: "Legal document URL not accessible"
**Solution**: Ensure privacy policy and terms URLs are publicly accessible (200 OK response)

### Issue: "Plan pricing configuration invalid"
**Solution**: For BYOL, ensure no pricing meters are configured

---

## After Going Live

### Monitor Performance

1. **Analytics Dashboard**:
   - Partner Center → Commercial Marketplace → Analyze
   - Track page visits, deployments, revenue

2. **Customer Feedback**:
   - Monitor ratings and reviews
   - Respond to customer questions

3. **Support Tickets**:
   - Set up support process for marketplace customers
   - Track common issues

### Updating Your Offer

**To release new versions**:

1. Build new Docker image with updated version tag
2. Push to ACR
3. Run packaging tool with new version
4. In Partner Center:
   - Update Technical Configuration with new CNAB path
   - Update plan versions
   - Add changelog in description
5. Re-submit for certification

**Typical update timeline**: 1-2 weeks for re-certification

---

## Checklist

Before clicking "Review and publish", verify:

- [ ] Offer setup complete
- [ ] Properties configured (categories, legal)
- [ ] Offer listing complete (description, screenshots, logo)
- [ ] Preview audience added (at least one subscription)
- [ ] Technical configuration (CNAB bundle path, ACR access granted)
- [ ] At least one plan created and configured
- [ ] Plan description and pricing configured
- [ ] All required fields filled (no red warnings)
- [ ] Legal documents (privacy, terms) are publicly accessible
- [ ] Support contact information is correct

---

## Support

- **Partner Center Help**: https://partner.microsoft.com/support
- **Documentation**: https://learn.microsoft.com/en-us/partner-center/marketplace-offers/
- **Open a support ticket**: Partner Center → Support → New request

---

**Ready to publish?** Follow this guide step by step and you'll have your offer live on Azure Marketplace in about 4 weeks! 🚀
