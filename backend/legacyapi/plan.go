package api

import (
	"fmt"
)

// PlanType is the type for a plan.
type PlanType int

const (
	// FREE is the plan type for FREE.
	FREE PlanType = iota
	// TEAM is the plan type for TEAM.
	TEAM
	// ENTERPRISE is the plan type for ENTERPRISE.
	ENTERPRISE
)

// String returns the string format of plan type.
func (p PlanType) String() string {
	switch p {
	case FREE:
		return "FREE"
	case TEAM:
		return "TEAM"
	case ENTERPRISE:
		return "ENTERPRISE"
	}
	return ""
}

// Priority returns the priority of the plan type.
// Higher priority means the plan supports more features.
func (p PlanType) Priority() int {
	switch p {
	case FREE:
		return 1
	case TEAM:
		return 2
	case ENTERPRISE:
		return 3
	}
	return 0
}

// FeatureType is the type of a feature.
type FeatureType string

const (
	// Admin & Security.

	// FeatureSSO allows user to manage SSO provider and authenticate (login) with SSO.
	FeatureSSO FeatureType = "bb.feature.sso"
	// Feature2FA allows user to manage 2FA provider and authenticate (login) with 2FA.
	Feature2FA FeatureType = "bb.feature.2fa"
	// FeatureDisallowSignup allows user to change the disallow signup flag.
	FeatureDisallowSignup FeatureType = "bb.feature.disallow-signup"
	// FeatureRBAC enables RBAC.
	//
	// - Workspace level RBAC
	// - Project level RBAC.
	FeatureRBAC FeatureType = "bb.feature.rbac"

	// FeatureWatermark enables full-screen watermark.
	FeatureWatermark FeatureType = "bb.feature.watermark"

	// FeatureAuditLog enables viewing audit logs.
	FeatureAuditLog FeatureType = "bb.feature.audit-log"

	// FeatureCustomRole enables customizing roles.
	FeatureCustomRole FeatureType = "bb.feature.custom-role"

	// Branding.

	// FeatureBranding enables customized branding.
	//
	// Currently, we only support customizing the logo.
	FeatureBranding FeatureType = "bb.feature.branding"

	// Change Workflow.

	// FeatureDBAWorkflow enforces the DBA workflow.
	//
	// - Developers can't create and view instances since they are exclusively by DBA, they can
	//   only access database.
	// - Developers can't create database.
	// - Developers can't query and export data directly. They must request corresponding permissions first.
	FeatureDBAWorkflow FeatureType = "bb.feature.dba-workflow"
	// FeatureIMApproval integrates IM approval into Bytebase, allowing users approve Bytebase issues on the IM.
	FeatureIMApproval FeatureType = "bb.feature.im.approval"
	// FeatureMultiTenancy allows user to enable batch mode for the project.
	//
	// Batch mode allows user to track a group of homogeneous database changes together.
	// e.g. A game studio may deploy many servers, each server is fully isolated with its
	// own database. When a new game version is released, it may require to upgrade the
	// underlying database schema, then batch mode will help the studio to track the
	// schema change across all databases.
	FeatureMultiTenancy FeatureType = "bb.feature.multi-tenancy"
	// FeatureOnlineMigration allows user to perform online-migration.
	FeatureOnlineMigration FeatureType = "bb.feature.online-migration"
	// FeatureSchemaDrift detects if there occurs schema drift.
	// See https://bytebase.com/docs/features/drift-detection
	FeatureSchemaDrift FeatureType = "bb.feature.schema-drift"
	// FeatureSQLReview allows user to specify schema policy for the environment
	//
	// e.g. One can configure rules for database schema or SQL query.
	FeatureSQLReview FeatureType = "bb.feature.sql-review"
	// FeatureMybatisSQLReview allows review mybatis sql.
	FeatureMybatisSQLReview FeatureType = "bb.feature.mybatis-sql-review"
	// FeatureTaskScheduleTime allows user to run task at a scheduled time.
	FeatureTaskScheduleTime FeatureType = "bb.feature.task-schedule-time"
	// FeatureEncryptedSecrets is a feature that allows user to setting the encrypted secrets for the database.
	FeatureEncryptedSecrets FeatureType = "bb.feature.encrypted-secrets"
	// FeatureDatabaseGrouping allows user to create database/schema groups.
	FeatureDatabaseGrouping FeatureType = "bb.feature.database-grouping"
	// FeatureSchemaTemplate allows user to create and use the schema template.
	FeatureSchemaTemplate FeatureType = "bb.feature.schema-template"

	// VCS Integration.

	// FeatureVCSSchemaWriteBack allows user to enable the latest schema write-back in VCS workflow.
	FeatureVCSSchemaWriteBack FeatureType = "bb.feature.vcs-schema-write-back"
	// FeatureVCSSheetSync allows to sync sheets from the VCS in VCS workflow.
	FeatureVCSSheetSync FeatureType = "bb.feature.vcs-sheet-sync"
	// FeatureVCSSQLReviewWorkflow allows user to enable the SQL review CI in VCS workflow.
	FeatureVCSSQLReviewWorkflow FeatureType = "bb.feature.vcs-sql-review"

	// Database management.

	// FeaturePITR allows user to perform point-in-time recovery for databases.
	FeaturePITR FeatureType = "bb.feature.pitr"
	// FeatureReadReplicaConnection allows user to set a read replica connection
	// including host and port to data source.
	FeatureReadReplicaConnection FeatureType = "bb.feature.read-replica-connection"
	// FeatureInstanceSSHConnection provides SSH connection for instances.
	FeatureInstanceSSHConnection FeatureType = "bb.feature.instance-ssh-connection"
	// FeatureSyncSchemaAllVersions allows user to sync the base database schema all versions into target database.
	FeatureSyncSchemaAllVersions FeatureType = "bb.feature.sync-schema-all-versions"
	// FeatureIndexAdvisor provides the index advisor for databases.
	FeatureIndexAdvisor FeatureType = "bb.feature.index-advisor"

	// Policy Control.

	// FeatureApprovalPolicy allows user to specify approval policy for the environment
	//
	// e.g. One can configure to NOT require approval for dev environment while require
	//      manual approval for production.
	FeatureApprovalPolicy FeatureType = "bb.feature.approval-policy"
	// FeatureBackupPolicy allows user to specify backup policy for the environment
	//
	// e.g. One can configure to NOT require backup for dev environment while require
	//      weekly backup for staging and daily backup for production.
	FeatureBackupPolicy FeatureType = "bb.feature.backup-policy"
	// FeatureEnvironmentTierPolicy allows user to set the tier of an environment.
	//
	// e.g. set the tier to "PROTECTED" for the production environment.
	FeatureEnvironmentTierPolicy FeatureType = "bb.feature.environment-tier-policy"
	// FeatureSensitiveData allows user to annotate and protect sensitive data.
	FeatureSensitiveData FeatureType = "bb.feature.sensitive-data"
	// FeatureAccessControl allows user to config the access control.
	FeatureAccessControl FeatureType = "bb.feature.access-control"
	// FeatureCustomApproval enables custom risk level definition and custom
	// approval chain definition.
	FeatureCustomApproval FeatureType = "bb.feature.custom-approval"

	// Collaboration.

	// FeatureSharedSQLScript enables sharing sql script.
	FeatureSharedSQLScript FeatureType = "bb.feature.shared-sql-script"

	// Plugins.

	// FeaturePluginOpenAI enables AI features powered by OpenAI.
	FeaturePluginOpenAI FeatureType = "bb.feature.plugin.openai"
)

// Name returns a readable name of the feature.
func (e FeatureType) Name() string {
	switch e {
	// Admin & Security
	case FeatureSSO:
		return "SSO"
	case Feature2FA:
		return "2FA"
	case FeatureDisallowSignup:
		return "Disallow singup"
	case FeatureRBAC:
		return "RBAC"
	case FeatureWatermark:
		return "Watermark"
	case FeatureAuditLog:
		return "Audit log"
	case FeatureCustomRole:
		return "Custom role"
	// Branding
	case FeatureBranding:
		return "Branding"
	// Change Workflow
	case FeatureDBAWorkflow:
		return "DBA workflow"
	case FeatureIMApproval:
		return "IM approval integration"
	case FeatureMultiTenancy:
		return "Multi-tenancy"
	case FeatureOnlineMigration:
		return "Online schema migration"
	case FeatureSchemaDrift:
		return "Schema drift"
	case FeatureMybatisSQLReview:
		return "Mybatis SQL review"
	case FeatureSQLReview:
		return "SQL review"
	case FeatureTaskScheduleTime:
		return "Task schedule time"
	case FeatureEncryptedSecrets:
		return "Encrypted secrets"
	case FeatureDatabaseGrouping:
		return "Database grouping"
	case FeatureSchemaTemplate:
		return "Schema template"
	// VCS Integration
	case FeatureVCSSchemaWriteBack:
		return "Schema write-back"
	case FeatureVCSSheetSync:
		return "Sync sheets from VCS"
	case FeatureVCSSQLReviewWorkflow:
		return "VCS SQL review workflow"
	// Database management
	case FeaturePITR:
		return "Point-in-time Recovery"
	case FeatureReadReplicaConnection:
		return "Read replica connection"
	case FeatureInstanceSSHConnection:
		return "Instance SSH connection"
	case FeatureSyncSchemaAllVersions:
		return "Synchronize schema all versions"
	case FeatureIndexAdvisor:
		return "Index advisor"
	// Policy Control
	case FeatureApprovalPolicy:
		return "Approval policy"
	case FeatureBackupPolicy:
		return "Backup policy"
	case FeatureEnvironmentTierPolicy:
		return "Environment tier"
	case FeatureSensitiveData:
		return "Sensitive data"
	case FeatureAccessControl:
		return "Access Control"
	case FeatureCustomApproval:
		return "Custom Approval"
	// Collaboration
	case FeatureSharedSQLScript:
		return "Shared SQL script"
	// Plugins
	case FeaturePluginOpenAI:
		return "OpenAI"
	}
	return ""
}

// AccessErrorMessage returns a error message with feature name and minimum supported plan.
func (e FeatureType) AccessErrorMessage() string {
	plan := e.minimumSupportedPlan()
	return fmt.Sprintf("%s is a %s feature, please upgrade to access it.", e.Name(), plan.String())
}

// minimumSupportedPlan will find the minimum plan which supports the target feature.
func (e FeatureType) minimumSupportedPlan() PlanType {
	for i, enabled := range FeatureMatrix[e] {
		if enabled {
			return PlanType(i)
		}
	}

	return ENTERPRISE
}

// FeatureMatrix is a map from the a particular feature to the respective enablement of a particular
// plan in [FREE, TEAM, Enterprise].
var FeatureMatrix = map[FeatureType][3]bool{
	// Admin & Security
	FeatureSSO:            {false, false, true},
	Feature2FA:            {false, false, true},
	FeatureDisallowSignup: {false, false, true},
	FeatureRBAC:           {true, true, true},
	FeatureWatermark:      {false, false, true},
	FeatureAuditLog:       {false, false, true},
	FeatureCustomRole:     {false, false, true},
	// Branding
	FeatureBranding: {false, false, true},
	// Change Workflow
	FeatureDBAWorkflow:      {false, false, true},
	FeatureIMApproval:       {false, false, true},
	FeatureMultiTenancy:     {false, false, true},
	FeatureOnlineMigration:  {false, true, true},
	FeatureSchemaDrift:      {false, false, true},
	FeatureMybatisSQLReview: {false, false, true},
	FeatureSQLReview:        {true, true, true},
	FeatureTaskScheduleTime: {false, true, true},
	FeatureEncryptedSecrets: {false, true, true},
	FeatureDatabaseGrouping: {false, false, true},
	FeatureSchemaTemplate:   {false, false, true},
	// VCS Integration
	FeatureVCSSchemaWriteBack:   {false, true, true},
	FeatureVCSSheetSync:         {false, true, true},
	FeatureVCSSQLReviewWorkflow: {true, true, true},
	// Database management
	FeaturePITR:                  {false, true, true},
	FeatureReadReplicaConnection: {false, false, true},
	FeatureInstanceSSHConnection: {false, false, true},
	FeatureSyncSchemaAllVersions: {false, true, true},
	FeatureIndexAdvisor:          {false, false, true},
	// Policy Control
	FeatureApprovalPolicy:        {false, true, true},
	FeatureBackupPolicy:          {false, true, true},
	FeatureEnvironmentTierPolicy: {false, false, true},
	FeatureSensitiveData:         {false, false, true},
	FeatureAccessControl:         {false, false, true},
	FeatureCustomApproval:        {false, false, true},
	// Collaboration
	FeatureSharedSQLScript: {false, true, true},
	// Plugins
	FeaturePluginOpenAI: {false, false, true},
}

// InstanceLimitFeature is the map for instance feature. Only allowed to access these feature for activate instance.
var InstanceLimitFeature = map[FeatureType]bool{
	// Change Workflow
	FeatureIMApproval:       true,
	FeatureSchemaDrift:      true,
	FeatureSQLReview:        true,
	FeatureEncryptedSecrets: true,
	FeatureTaskScheduleTime: true,
	FeatureOnlineMigration:  true,
	// VCS Integration
	FeatureVCSSchemaWriteBack:   true,
	FeatureVCSSQLReviewWorkflow: true,
	FeatureMybatisSQLReview:     true,
	// Database management
	FeaturePITR:                  true,
	FeatureReadReplicaConnection: true,
	FeatureInstanceSSHConnection: true,
	FeatureDatabaseGrouping:      true,
	FeatureSyncSchemaAllVersions: true,
	FeatureIndexAdvisor:          true,
	// Policy Control
	FeatureSensitiveData:  true,
	FeatureCustomApproval: true,
}

// Feature returns whether a particular feature is available in a particular plan.
func Feature(feature FeatureType, plan PlanType) bool {
	matrix, ok := FeatureMatrix[feature]
	if !ok {
		return false
	}
	return matrix[plan]
}
