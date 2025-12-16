---
name: data-masking-setup
description: Configure dynamic data masking (classification, semantic types, rules)
---

# Data Masking Setup

## Overview

Configure Dynamic Data Masking (DDM) to protect sensitive data in SQL Editor query results. Masking is applied based on classification levels and semantic types.

## Prerequisites

- Have `bb.settings.set` permission (for workspace-level config)
- Have `bb.policies.create` permission (for masking rules)
- Have `bb.databaseCatalogs.update` permission (for column classification)

## Masking Precedence

1. **Masking Exemption** - If user has exemption, data is not masked
2. **Global Masking Rule** - Applied if no exemption
3. **Column Masking** - Applied if no global rule matches

## Workflow

### Step 1: Configure Data Classification

Define classification levels (e.g., Public, Internal, Confidential) and categories.

```
search_api(operationId="SettingService/UpdateSetting")
```
```
call_api(operationId="SettingService/UpdateSetting", body={
  "setting": {
    "name": "settings/DATA_CLASSIFICATION",
    "value": {
      "dataClassification": {
        "configs": [{
          "id": "default",
          "title": "Data Classification",
          "levels": [
            {"id": "1", "title": "Public"},
            {"id": "2", "title": "Internal"},
            {"id": "3", "title": "Confidential"}
          ],
          "classification": {
            "1": {"id": "1", "title": "General"},
            "1-1": {"id": "1-1", "title": "Public Info", "levelId": "1"},
            "2": {"id": "2", "title": "PII"},
            "2-1": {"id": "2-1", "title": "Email", "levelId": "2"},
            "2-2": {"id": "2-2", "title": "Phone", "levelId": "3"},
            "2-3": {"id": "2-3", "title": "SSN", "levelId": "3"}
          }
        }]
      }
    }
  },
  "updateMask": "value"
})
```

**Key:** Classification IDs like `2-1` are children of `2`. The `levelId` maps to sensitivity level.

### Step 2: Configure Semantic Types

Define how each type of sensitive data should be masked using algorithm configs.

```
call_api(operationId="SettingService/UpdateSetting", body={
  "setting": {
    "name": "settings/SEMANTIC_TYPES",
    "value": {
      "semanticType": {
        "types": [
          {
            "id": "email",
            "title": "Email Address",
            "algorithm": {
              "rangeMask": {
                "slices": [{"start": 0, "end": 5, "substitution": "***"}]
              }
            }
          },
          {
            "id": "phone",
            "title": "Phone Number",
            "algorithm": {
              "fullMask": {"substitution": "******"}
            }
          },
          {
            "id": "ssn",
            "title": "Social Security Number",
            "algorithm": {
              "md5Mask": {"salt": "random-salt"}
            }
          }
        ]
      }
    }
  },
  "updateMask": "value"
})
```

**Algorithm types (oneof):**
- `fullMask`: Replace entire value with substitution
- `rangeMask`: Replace specific character ranges
- `md5Mask`: Hash the value with optional salt
- `innerOuterMask`: Mask inner or outer characters

### Step 3: Create Global Masking Rule

Apply masking based on classification levels across all databases.

```
search_api(operationId="OrgPolicyService/CreatePolicy")
```
```
call_api(operationId="OrgPolicyService/CreatePolicy", body={
  "parent": "",
  "type": "MASKING_RULE",
  "policy": {
    "maskingRulePolicy": {
      "rules": [{
        "id": "rule-uuid-here",
        "condition": {
          "expression": "resource.classification_level == \"3\""
        },
        "semanticType": "ssn"
      }]
    }
  }
})
```

**Condition variables:**
- `resource.environment_id`, `resource.project_id`, `resource.instance_id`
- `resource.database_name`, `resource.table_name`, `resource.column_name`
- `resource.classification_level`

### Step 4: Classify Columns (Alternative to Global Rules)

Apply classification directly to database columns.

```
search_api(operationId="DatabaseCatalogService/UpdateDatabaseCatalog")
```
```
call_api(operationId="DatabaseCatalogService/UpdateDatabaseCatalog", body={
  "catalog": {
    "name": "instances/{instance-id}/databases/{database-name}/catalog",
    "schemas": [{
      "name": "public",
      "tables": [{
        "name": "users",
        "columns": [
          {"name": "email", "semanticType": "email", "classification": "2-1"},
          {"name": "phone", "semanticType": "phone", "classification": "2-2"},
          {"name": "ssn", "semanticType": "ssn", "classification": "2-3"}
        ]
      }]
    }]
  }
})
```

**Note:** This API overwrites all column configs for the database. Fetch existing catalog first if you need to preserve other columns.

### Step 5: Grant Masking Exemption (Optional)

Allow specific users to see unmasked data.

```
call_api(operationId="OrgPolicyService/CreatePolicy", body={
  "parent": "projects/{project-id}",
  "type": "MASKING_EXEMPTION",
  "policy": {
    "maskingExemptionPolicy": {
      "exemptions": [{
        "members": ["users/admin@example.com"],
        "condition": {
          "expression": "resource.instance_id == \"prod\" && request.time < timestamp(\"2025-12-31T23:59:59Z\")"
        }
      }]
    }
  }
})
```

**Member format:** `users/{email}` or `groups/{group-email}`

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| setting not found | Wrong setting name | Use `settings/DATA_CLASSIFICATION` or `settings/SEMANTIC_TYPES` |
| invalid classification | ID format wrong | Use hierarchical IDs: `1`, `1-1`, `1-2` |
| semantic type not found | Type ID doesn't exist | Create semantic type first (Step 2) |
| permission denied | Missing bb.settings.set | Check workspace admin permissions |
