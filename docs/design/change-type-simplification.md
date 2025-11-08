# Change Type Simplification: From DDL/DML to Versioned/Adhoc

## Background

Bytebase currently supports two workflows for database changes:

1. **UI Workflow**: Users manually create changes through the UI and must select either "Edit Schema (DDL)" or "Change Data (DML)"
2. **GitOps Workflow**: All changes come from version control with associated versions

Currently, changes are categorized primarily by SQL syntax (DDL vs DML), which serves two purposes:
- Triggering different risk assessment rules
- Determining approval flows

Changes in the UI workflow do not have versions, while GitOps changes always have versions. This creates inconsistency and limits flexibility.

## Problem

The current DDL/DML categorization creates several issues:

### 1. User Confusion
Users often don't understand the difference between DDL and DML, or don't know which category their change falls into. Questions like "Is TRUNCATE DDL or DML?" are common.

### 2. Mixed Statements Not Supported
Real-world changes frequently contain both DDL and DML in a single script:
```sql
ALTER TABLE users ADD COLUMN status VARCHAR(20);
UPDATE users SET status = 'active' WHERE last_login > NOW() - INTERVAL '30 days';
```
Forcing users to categorize this as either DDL or DML is artificial.

### 3. Syntax vs Intent Mismatch
The approval process should be based on the **intent** of the change, not the SQL syntax:
- A planned schema migration affecting multiple environments needs architecture review
- An urgent data fix for a specific user needs quick database owner approval

Both could technically be DDL or DML, but they need different approval processes.

### 4. Terminology Confusion
- The terms "DDL" and "DML" are technical SQL concepts that don't map cleanly to user intent
- Users care about whether they're doing a coordinated migration or a one-off fix, not SQL syntax categories

## Proposed Solution

Replace the DDL/DML distinction with a **VERSIONED_CHANGE/ADHOC_CHANGE** distinction that reflects change intent rather than SQL syntax.

This is primarily a **terminology change** to make the system more intuitive. The names "VERSIONED_CHANGE" and "ADHOC_CHANGE" better describe what users are actually doing, and they align with a future vision where UI workflow could support optional versioning.

### Change Types

**VERSIONED_CHANGE** (currently called "DDL" or "Edit Schema"):
- **Intent**: Coordinated changes that are part of planned releases, typically rolling out across multiple environments
- **Typical scenarios**:
  - Feature releases with schema changes
  - Data migrations as part of a version release
  - Coordinated refactoring across environments
  - Any change that should be applied consistently dev → staging → prod
- **Current behavior**:
  - GitOps workflow: Has version extracted from file name (e.g., `V001__add_column.sql`)
  - UI workflow: No version currently (just indicates intent to coordinate across environments)
- **Future vision**: UI workflow could optionally support version input for ordering
- **Can contain**: DDL, DML, or both (SQL type doesn't matter, intent does)
- **Key characteristic**: Represents changes tied to application releases or coordinated rollouts

**ADHOC_CHANGE** (currently called "DML" or "Change Data"):
- **Intent**: One-off changes for specific situations, typically for a single database
- **Typical scenarios**:
  - Customer-specific data fixes
  - Production hotfixes that don't need to propagate to other environments
  - Cleanup tasks or data corrections
  - Emergency fixes that won't be repeated elsewhere
- **Current behavior**: UI workflow only, never has a version
- **Can contain**: DDL, DML, or both (SQL type doesn't matter, intent does)
- **Key characteristic**: Represents situational fixes, not coordinated releases

**Important Note**: The distinction is about **coordination and intent**, not SQL syntax. Both types can contain DDL and DML statements. Choose based on whether the change is part of a coordinated release (VERSIONED_CHANGE) or a one-off fix (ADHOC_CHANGE).

### Key Principles

1. **Users select change type explicitly**: Users choose between "Versioned Migration" or "Adhoc Change" to indicate their intent
2. **Clearer terminology**: Replace confusing "DDL/DML" with intent-based "Versioned/Adhoc"
3. **SQL analysis is automatic**: The system parses SQL to detect actual statement types (CREATE, ALTER, UPDATE, etc.) for risk assessment
4. **Intent-based approvals**: Risk rules match on change type (versioned vs adhoc) combined with SQL characteristics

## Design Details

### Risk Assessment Model

Risk sources are renamed to better reflect change intent:
- `DDL` → `VERSIONED_CHANGE`
- `DML` → `ADHOC_CHANGE`

This is primarily a semantic rename that aligns the naming with actual use cases:
- What was called "DDL" (schema changes) typically represents planned, versioned migrations
- What was called "DML" (data changes) typically represents one-off, adhoc data fixes

Risk rules can now express intent-based conditions using the clearer terminology:
- "Versioned changes to production with schema alterations require Senior DBA approval"
- "Adhoc changes affecting more than 1000 rows require Database Owner approval"
- "Versioned migrations in dev environment are auto-approved"

### SQL Analysis

The system automatically detects:
- Statement types present (CREATE, ALTER, DROP, INSERT, UPDATE, DELETE, etc.)
- Whether DDL statements exist
- Whether DML statements exist
- Estimated affected rows
- Target tables

This information is used in risk rule conditions without requiring user categorization.

### UI Terminology Options

The internal naming uses `VERSIONED_CHANGE` and `ADHOC_CHANGE`, but the UI should use more user-friendly terminology. Several options to consider:

#### Option 1: Migration-Focused
- **VERSIONED_CHANGE** → "Database Migration"
- **ADHOC_CHANGE** → "Data Fix"
- **Pros**: Clear intent, simple language, familiar terms
- **Cons**: "Migration" might suggest only schema changes to some users

#### Option 2: Intent-Based
- **VERSIONED_CHANGE** → "Planned Change"
- **ADHOC_CHANGE** → "Quick Fix"
- **Pros**: Emphasizes coordination vs urgency
- **Cons**: "Planned" is somewhat generic

#### Option 3: Scope-Based
- **VERSIONED_CHANGE** → "Multi-Environment Change"
- **ADHOC_CHANGE** → "Single-Database Change"
- **Pros**: Very explicit about scope
- **Cons**: Wordy, and VERSIONED_CHANGE could target single database too

#### Option 4: Direct Translation
- **VERSIONED_CHANGE** → "Versioned Change"
- **ADHOC_CHANGE** → "Adhoc Change"
- **Pros**: Matches internal naming, consistent with future versioning support
- **Cons**: Users might expect "Versioned" to have actual version numbers currently

**Recommendation**: Option 1 ("Database Migration" / "Data Fix") provides the best balance of clarity and intent while avoiding technical jargon.

### User Experience

**UI Workflow - Before:**
```
Step 1: Choose "Edit Schema (DDL)" or "Change Data (DML)"
Step 2: Select target databases
Step 3: Write SQL
```

**UI Workflow - After:**
```
Step 1: Choose change type (see terminology options above)
Step 2: Select target databases
Step 3: Write SQL
→ System evaluates risk rules based on change type + SQL content
```

**GitOps Workflow:**
- No change required
- All GitOps changes are treated as VERSIONED_CHANGE (versions extracted from file names)

### Real-World Workflow Scenarios

Understanding how developers and teams actually work helps clarify when to use each change type:

#### Developer Perspective

**Scenario 1: Feature Release with Schema + Data Changes**
```sql
-- Part of v2.5.0 release
ALTER TABLE users ADD COLUMN subscription_tier VARCHAR(20);
UPDATE users SET subscription_tier = 'free' WHERE subscription_tier IS NULL;
```
- **Change Type**: VERSIONED_CHANGE
- **Why**: Coordinated release going through dev → staging → prod
- **Workflow**: Developer tests in dev, submits for review, team lead approves, rolls out to all environments
- **Risk Rules**: Can require architecture review in prod

**Scenario 2: Production Hotfix for Single Customer**
```sql
-- Fix corrupted data for customer #12345
UPDATE orders SET status = 'completed' WHERE customer_id = 12345 AND status = 'stuck';
```
- **Change Type**: ADHOC_CHANGE
- **Why**: One-off fix for specific issue, not going to other environments
- **Workflow**: Developer submits directly to prod, database owner approves
- **Risk Rules**: Can have lighter approval for small affected rows

**Scenario 3: Emergency Performance Fix**
```sql
-- Production is slow, drop unused index immediately
DROP INDEX users_old_email_idx;
```
- **Change Type**: Could be either:
  - ADHOC_CHANGE if truly urgent and prod-only
  - VERSIONED_CHANGE if should be applied to all environments
- **Consideration**: Even emergency fixes might need to go to all envs to keep them consistent

**Scenario 4: Data Backfill After Migration**
```sql
-- After adding subscription_tier column yesterday
UPDATE users SET subscription_tier = 'free' WHERE subscription_tier IS NULL;
```
- **Change Type**: Often ADHOC_CHANGE
- **Why**: The schema migration already happened; this is cleanup
- **Alternative**: Could be part of the original VERSIONED_CHANGE if planned together

#### Team Lead Perspective

**Governance Considerations:**
- Risk rules should consider both change type AND other factors (affected rows, SQL type, environment)
- Example: Even ADHOC_CHANGE with DELETE or high affected rows should require strong approval
- Teams may want to enforce: "All production VERSIONED_CHANGE must have version number" (future capability)
- Audit trail: Need to track who chose which change type and why

**Preventing Approval Bypassing:**
Risk rules should be configured so that dangerous operations require approval regardless of change type:
```
High Risk Rule:
  (VERSIONED_CHANGE OR ADHOC_CHANGE)
  AND environment == "prod"
  AND (sql_type IN ["DROP", "TRUNCATE", "DELETE"] OR affected_rows > 1000)
  → Requires: Senior DBA approval
```

### Approval Flow Examples

**Example 1: Coordinated Schema Migration**
- Type: VERSIONED_CHANGE
- SQL: `ALTER TABLE users DROP COLUMN legacy_email;`
- Environment: Production
- Affected Rows: 0 (DDL)
- Risk Level: HIGH (production + schema change)
- Required Approval: Senior DBA + Tech Lead
- Rationale: Structural changes need architecture review

**Example 2: Small Data Fix**
- Type: ADHOC_CHANGE
- SQL: `UPDATE users SET email = 'corrected@example.com' WHERE id = 12345;`
- Environment: Production
- Affected Rows: 1
- Risk Level: LOW
- Required Approval: Database Owner
- Rationale: Minimal impact, quick fix

**Example 3: Large Data Migration (Part of Release)**
- Type: VERSIONED_CHANGE
- SQL: `INSERT INTO new_table SELECT * FROM old_table;`
- Environment: Production
- Affected Rows: 500,000
- Risk Level: HIGH (large affected rows)
- Required Approval: Senior DBA + Tech Lead
- Rationale: Even though it's DML, it's part of coordinated migration

**Example 4: Urgent Adhoc Fix with High Impact**
- Type: ADHOC_CHANGE
- SQL: `DELETE FROM sessions WHERE expired_at < NOW() - INTERVAL '1 year';`
- Environment: Production
- Affected Rows: 10,000
- Risk Level: HIGH (DELETE + high affected rows)
- Required Approval: Senior DBA + Database Owner
- Rationale: Even adhoc changes can be risky; approval based on SQL characteristics

## Migration Strategy

### Phase 1: Rename Risk Sources (Breaking Change)
- Rename DDL → VERSIONED_CHANGE
- Rename DML → ADHOC_CHANGE
- Automatically migrate all existing risk rules to use new names
- Update all code references from DDL/DML to VERSIONED_CHANGE/ADHOC_CHANGE

### Phase 2: UI Updates
- Update UI labels from "Edit Schema (DDL)" / "Change Data (DML)" to user-friendly terminology
- Choose one of the options described in "UI Terminology Options" section
- No functional changes to UI workflow in this phase
- (Future: Optional version input for UI workflow can be added later)

### Phase 3: Documentation and Communication
- Update user documentation to reflect new terminology
- Provide migration guide for existing risk rules (mostly automatic)
- Communicate changes to users and administrators

## Benefits

1. **Clearer terminology**: Users understand "migration vs fix" better than "DDL vs DML"
2. **Intent-based categorization**: Matches what users are trying to accomplish
3. **Better approval flows**: Different processes for migrations vs hotfixes make more sense
4. **Supports mixed statements**: No forced categorization of hybrid scripts (DDL+DML)
5. **Future-ready**: The "VERSIONED_CHANGE" name aligns with a future where UI workflow could support optional versioning
6. **No functional disruption**: This is primarily a rename - existing workflows continue to work the same way

## Future Possibilities (Not in Scope)

While this design focuses on terminology improvement, it sets the foundation for potential future enhancements:

**Optional Versioning in UI Workflow**: UI users could optionally provide version numbers for VERSIONED_CHANGE types, enabling:
- Ordering enforcement across environments for UI-created migrations
- Better alignment between GitOps and UI workflows
- Version-based change history tracking

This is intentionally left out of the current design to keep the scope focused on terminology improvement.

## Non-Goals

- This design does not change the underlying execution model
- This design does not affect how SQL statements are parsed or executed
- This design does not add new functionality to UI workflow (only terminology changes)
