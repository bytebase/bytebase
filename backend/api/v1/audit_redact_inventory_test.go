package v1

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// The inventory is the oracle the coverage sweep cannot be. That sweep proves
// annotated fields get blanked; it cannot tell an unannotated credential from
// an ordinary field the row intentionally keeps, because both survive
// redaction identically.
//
// So every string and bytes field reachable from an audit payload is listed
// here, minus the annotated ones. A field missing from the list fails the
// build; clearing that failure means annotating it or adding a line saying the
// row may keep it. Annotated fields are absent by design, so the list reads as
// "what the audit row writes down" and a new credential shows up as one line of
// diff.
//
// The population is wider than the audited RPCs: WrapUnary writes a row on
// `needAudit(ctx) || mcpPolicyDenied`, so gate-refused methods carrying no
// audit annotation are in scope. Every registered Any type is in scope too,
// since those reach the row without passing marshalAuditPayload — which is why
// the registry is enforced at its call sites for this list to mean anything.
//
// The roots are the payloads createAuditLog marshals onto the row — request,
// response, status, service_data — and not the row itself. storepb.AuditLog's
// own columns, its RequestMetadata and its MCPDelegation are assigned in Go
// from headers and verified grant state and are reviewed there; the annotation
// is no remedy for them, since bytebase.store does not import bytebase.v1 and
// marshalAuditPayload never sees them, so a line here would record a decision
// that had only one branch. TestLintAuditAnyFieldsAreRegistered does walk the
// row, because its remedy — registering what an Any may pack — is the live
// enforcement for the row's own service_data.

// TestLintAuditPayloadInventory fails when a string or bytes field joins or
// leaves the audited surface without anyone deciding what the audit row may
// record for it.
func TestLintAuditPayloadInventory(t *testing.T) {
	found := auditRecordedScalarFields(t)

	recorded := map[string]bool{}
	for _, field := range auditRecordedFields {
		require.False(t, recorded[field], "%s is listed twice", field)
		recorded[field] = true
	}

	var undecided, unresolvable, unreachable []string
	for _, field := range found {
		if !recorded[field] {
			undecided = append(undecided, field)
		}
	}
	// A listed name stops meaning anything in two different ways, and the fix
	// differs, so they are reported apart rather than as one "stale" bucket.
	for field := range recorded {
		switch {
		case slices.Contains(found, field):
		case inventoryFieldExists(field):
			unreachable = append(unreachable, field)
		default:
			unresolvable = append(unresolvable, field)
		}
	}
	slices.Sort(undecided)
	slices.Sort(unresolvable)
	slices.Sort(unreachable)

	require.Empty(t, undecided,
		"these reach an audit payload and nobody has decided about them: annotate the field "+
			"(bytebase.v1.audit_behavior) = SENSITIVE or OMIT, or add it to auditRecordedFields to record that "+
			"the audit row may keep it")
	require.Empty(t, unresolvable,
		"these name a field that does not exist, which reads as coverage while asserting nothing; drop them")
	require.Empty(t, unreachable,
		"these no longer reach an audit payload, or are now annotated; drop them so the list keeps meaning something")

	require.True(t, slices.IsSorted(auditRecordedFields),
		"auditRecordedFields is kept sorted so a new entry reads as one line of diff")
}

// auditRecordedScalarFields is every (message, string-or-bytes field) pair the
// audit row can record, derived from the descriptors rather than from reading.
// Reading is how the first sweep of this surface missed ListInstanceDatabase,
// whose request carries an entire Instance.
func auditRecordedScalarFields(t *testing.T) []string {
	t.Helper()
	types := map[protoreflect.FullName]bool{}
	var walk func(protoreflect.MessageDescriptor)
	walk = func(descriptor protoreflect.MessageDescriptor) {
		if types[descriptor.FullName()] {
			return
		}
		types[descriptor.FullName()] = true
		fields := descriptor.Fields()
		for i := range fields.Len() {
			field := fields.Get(i)
			if auditBehaviorOf(field) != v1pb.AuditBehavior_AUDIT_BEHAVIOR_UNSPECIFIED {
				continue
			}
			if sub := submessageOf(field); sub != nil {
				walk(sub)
			}
		}
	}

	// Every v1 method, both directions.
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if fd.Package() != "bytebase.v1" {
			return true
		}
		services := fd.Services()
		for i := range services.Len() {
			methods := services.Get(i).Methods()
			for j := range methods.Len() {
				walk(methods.Get(j).Input())
				walk(methods.Get(j).Output())
			}
		}
		return true
	})

	// The status a failed RPC leaves on the row. The v1 read API happens to
	// reach it too; it is a root of its own so the list does not rest on that.
	walk((&spb.Status{}).ProtoReflect().Descriptor())

	// Every registered Any type. These reach the row packed, so no descriptor
	// walk finds them: registering a type is what pulls its fields in here.
	for _, packed := range auditAnyRegistry {
		for _, name := range packed {
			descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(name)
			require.NoError(t, err, "registered Any type %s is not in the descriptor registry", name)
			md, ok := descriptor.(protoreflect.MessageDescriptor)
			require.True(t, ok, "registered Any type %s is not a message", name)
			walk(md)
		}
	}

	var found []string
	for name := range types {
		descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(name)
		if err != nil {
			continue
		}
		md, ok := descriptor.(protoreflect.MessageDescriptor)
		if !ok {
			continue
		}
		fields := md.Fields()
		for i := range fields.Len() {
			field := fields.Get(i)
			if auditBehaviorOf(field) != v1pb.AuditBehavior_AUDIT_BEHAVIOR_UNSPECIFIED {
				continue
			}
			kind := field.Kind()
			if field.IsMap() {
				kind = field.MapValue().Kind()
			}
			if kind == protoreflect.StringKind || kind == protoreflect.BytesKind {
				found = append(found, fmt.Sprintf("%s.%s", name, field.Name()))
			}
		}
	}
	slices.Sort(found)
	return slices.Compact(found)
}

// inventoryFieldExists reports whether a "message.field" entry still resolves.
func inventoryFieldExists(entry string) bool {
	cut := strings.LastIndex(entry, ".")
	if cut <= 0 {
		return false
	}
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(entry[:cut]))
	if err != nil {
		return false
	}
	md, ok := descriptor.(protoreflect.MessageDescriptor)
	return ok && md.Fields().ByName(protoreflect.Name(entry[cut+1:])) != nil
}

// auditRecordedFields is every string and bytes field an audit payload may
// record, sorted, one per line.
//
// The rule for adding one is mechanical: if the value would let someone
// authenticate, authorize, or act as somebody — or is an unbounded body, a
// blob, or personal data the row has no business keeping — it belongs on the
// annotation instead of here.
//
// Names that look like credentials and are not, so the next reader does not
// re-litigate them: DataSourceExternalSecret's url, engine_name, secret_name,
// password_key_name and mount_path say WHERE the secret lives, never what it
// is; DataSource.AzureCredential's tenant_id and client_id name the principal
// rather than authenticate as it; google.protobuf.Any.type_url and value are
// the envelope of a packed message whose own fields come in through
// auditAnyRegistry.
var auditRecordedFields = []string{
	"bytebase.v1.AIChatResponse.content",
	"bytebase.v1.AIChatToolCall.arguments",
	"bytebase.v1.AIChatToolCall.id",
	"bytebase.v1.AIChatToolCall.metadata",
	"bytebase.v1.AIChatToolCall.name",
	"bytebase.v1.AISetting.endpoint",
	"bytebase.v1.AISetting.model",
	"bytebase.v1.AISetting.version",
	"bytebase.v1.AccessGrant.container",
	"bytebase.v1.AccessGrant.creator",
	"bytebase.v1.AccessGrant.issue",
	"bytebase.v1.AccessGrant.name",
	"bytebase.v1.AccessGrant.query",
	"bytebase.v1.AccessGrant.reason",
	"bytebase.v1.AccessGrant.schema",
	"bytebase.v1.AccessGrant.targets",
	"bytebase.v1.ActivateAccessGrantRequest.name",
	"bytebase.v1.ActuatorInfo.default_project",
	"bytebase.v1.ActuatorInfo.external_url",
	"bytebase.v1.ActuatorInfo.git_commit",
	"bytebase.v1.ActuatorInfo.unlicensed_features",
	"bytebase.v1.ActuatorInfo.version",
	"bytebase.v1.ActuatorInfo.workspace",
	"bytebase.v1.AddDataSourceRequest.name",
	"bytebase.v1.AddWebhookRequest.project",
	"bytebase.v1.AdminExecuteRequest.container",
	"bytebase.v1.AdminExecuteRequest.name",
	"bytebase.v1.AdminExecuteRequest.schema",
	"bytebase.v1.AdminExecuteRequest.statement",
	"bytebase.v1.Advice.content",
	"bytebase.v1.Advice.title",
	"bytebase.v1.Algorithm.FullMask.substitution",
	"bytebase.v1.Algorithm.InnerOuterMask.substitution",
	"bytebase.v1.Algorithm.MD5Mask.salt",
	"bytebase.v1.Algorithm.RangeMask.Slice.substitution",
	"bytebase.v1.Announcement.link",
	"bytebase.v1.Announcement.text",
	"bytebase.v1.AppIMSetting.DingTalk.client_id",
	"bytebase.v1.AppIMSetting.DingTalk.robot_code",
	"bytebase.v1.AppIMSetting.Feishu.app_id",
	"bytebase.v1.AppIMSetting.Lark.app_id",
	"bytebase.v1.AppIMSetting.Teams.client_id",
	"bytebase.v1.AppIMSetting.Teams.tenant_id",
	"bytebase.v1.AppIMSetting.Wecom.agent_id",
	"bytebase.v1.AppIMSetting.Wecom.corp_id",
	"bytebase.v1.ApprovalFlow.roles",
	"bytebase.v1.ApprovalTemplate.description",
	"bytebase.v1.ApprovalTemplate.title",
	"bytebase.v1.ApproveIssueRequest.comment",
	"bytebase.v1.ApproveIssueRequest.name",
	"bytebase.v1.AuditLog.method",
	"bytebase.v1.AuditLog.name",
	"bytebase.v1.AuditLog.request",
	"bytebase.v1.AuditLog.resource",
	"bytebase.v1.AuditLog.response",
	"bytebase.v1.AuditLog.user",
	"bytebase.v1.AuthenticationInfo.workspace",
	"bytebase.v1.BatchCancelTaskRunsRequest.parent",
	"bytebase.v1.BatchCancelTaskRunsRequest.task_runs",
	"bytebase.v1.BatchCreateRevisionsRequest.parent",
	"bytebase.v1.BatchCreateSheetsRequest.parent",
	"bytebase.v1.BatchDeleteProjectsRequest.names",
	"bytebase.v1.BatchDeparseResponse.expressions",
	"bytebase.v1.BatchGetDatabasesRequest.names",
	"bytebase.v1.BatchGetDatabasesRequest.parent",
	"bytebase.v1.BatchGetGroupsRequest.names",
	"bytebase.v1.BatchGetProjectsRequest.names",
	"bytebase.v1.BatchGetUsersRequest.names",
	"bytebase.v1.BatchParseRequest.expressions",
	"bytebase.v1.BatchRunTasksRequest.parent",
	"bytebase.v1.BatchRunTasksRequest.tasks",
	"bytebase.v1.BatchSkipTasksRequest.parent",
	"bytebase.v1.BatchSkipTasksRequest.reason",
	"bytebase.v1.BatchSkipTasksRequest.tasks",
	"bytebase.v1.BatchSyncDatabasesRequest.names",
	"bytebase.v1.BatchSyncDatabasesRequest.parent",
	"bytebase.v1.BatchSyncInstancesRequest.parent",
	"bytebase.v1.BatchUpdateDatabasesRequest.parent",
	"bytebase.v1.BatchUpdateInstancesRequest.parent",
	"bytebase.v1.BatchUpdateIssuesStatusRequest.issues",
	"bytebase.v1.BatchUpdateIssuesStatusRequest.parent",
	"bytebase.v1.BatchUpdateIssuesStatusRequest.reason",
	"bytebase.v1.Binding.members",
	"bytebase.v1.Binding.role",
	"bytebase.v1.BindingDelta.member",
	"bytebase.v1.BindingDelta.role",
	"bytebase.v1.CancelPlanCheckRunRequest.name",
	"bytebase.v1.CancelPurchaseRequest.comment",
	"bytebase.v1.CancelPurchaseRequest.feedback",
	"bytebase.v1.ChangePasswordRequest.name",
	"bytebase.v1.Changelog.name",
	"bytebase.v1.Changelog.plan_title",
	"bytebase.v1.Changelog.schema",
	"bytebase.v1.Changelog.task_run",
	"bytebase.v1.CheckConstraintMetadata.expression",
	"bytebase.v1.CheckConstraintMetadata.name",
	"bytebase.v1.CheckReleaseRequest.custom_rules",
	"bytebase.v1.CheckReleaseRequest.parent",
	"bytebase.v1.CheckReleaseRequest.targets",
	"bytebase.v1.CheckReleaseResponse.CheckResult.file",
	"bytebase.v1.CheckReleaseResponse.CheckResult.target",
	"bytebase.v1.ColumnCatalog.classification",
	"bytebase.v1.ColumnCatalog.labels",
	"bytebase.v1.ColumnCatalog.name",
	"bytebase.v1.ColumnCatalog.semantic_type",
	"bytebase.v1.ColumnMetadata.character_set",
	"bytebase.v1.ColumnMetadata.collation",
	"bytebase.v1.ColumnMetadata.comment",
	"bytebase.v1.ColumnMetadata.default",
	"bytebase.v1.ColumnMetadata.default_constraint_name",
	"bytebase.v1.ColumnMetadata.name",
	"bytebase.v1.ColumnMetadata.on_update",
	"bytebase.v1.ColumnMetadata.type",
	"bytebase.v1.CompositeTypeAttribute.collation",
	"bytebase.v1.CompositeTypeAttribute.comment",
	"bytebase.v1.CompositeTypeAttribute.name",
	"bytebase.v1.CompositeTypeAttribute.type",
	"bytebase.v1.CompositeTypeMetadata.comment",
	"bytebase.v1.CompositeTypeMetadata.name",
	"bytebase.v1.ConfirmRecoveryCodesRequest.name",
	"bytebase.v1.CreateAccessGrantRequest.parent",
	"bytebase.v1.CreateDatabaseGroupRequest.database_group_id",
	"bytebase.v1.CreateDatabaseGroupRequest.parent",
	"bytebase.v1.CreateGroupRequest.group_email",
	"bytebase.v1.CreateIdentityProviderRequest.identity_provider_id",
	"bytebase.v1.CreateInstanceRequest.instance_id",
	"bytebase.v1.CreateInstanceRequest.parent",
	"bytebase.v1.CreateIssueCommentRequest.parent",
	"bytebase.v1.CreateIssueRequest.parent",
	"bytebase.v1.CreatePlanRequest.parent",
	"bytebase.v1.CreatePolicyRequest.parent",
	"bytebase.v1.CreateProjectRequest.project_id",
	"bytebase.v1.CreateReleaseRequest.parent",
	"bytebase.v1.CreateReleaseRequest.release_id_template",
	"bytebase.v1.CreateReleaseRequest.release_id_timezone",
	"bytebase.v1.CreateRevisionRequest.parent",
	"bytebase.v1.CreateRoleRequest.role_id",
	"bytebase.v1.CreateRolloutRequest.parent",
	"bytebase.v1.CreateRolloutRequest.target",
	"bytebase.v1.CreateSavedQueryRequest.parent",
	"bytebase.v1.CreateServiceAccountRequest.parent",
	"bytebase.v1.CreateServiceAccountRequest.service_account_id",
	"bytebase.v1.CreateSheetRequest.parent",
	"bytebase.v1.CreateWorkloadIdentityRequest.parent",
	"bytebase.v1.CreateWorkloadIdentityRequest.workload_identity_id",
	"bytebase.v1.DataClassificationSetting.DataClassificationConfig.DataClassification.id",
	"bytebase.v1.DataClassificationSetting.DataClassificationConfig.DataClassification.title",
	"bytebase.v1.DataClassificationSetting.DataClassificationConfig.Level.title",
	"bytebase.v1.DataClassificationSetting.DataClassificationConfig.id",
	"bytebase.v1.DataClassificationSetting.DataClassificationConfig.title",
	"bytebase.v1.DataSource.Address.host",
	"bytebase.v1.DataSource.Address.port",
	"bytebase.v1.DataSource.AzureCredential.client_id",
	"bytebase.v1.DataSource.AzureCredential.tenant_id",
	"bytebase.v1.DataSource.authentication_database",
	"bytebase.v1.DataSource.database",
	"bytebase.v1.DataSource.extra_connection_parameters",
	"bytebase.v1.DataSource.host",
	"bytebase.v1.DataSource.id",
	"bytebase.v1.DataSource.instance_id",
	"bytebase.v1.DataSource.master_name",
	"bytebase.v1.DataSource.master_username",
	"bytebase.v1.DataSource.port",
	"bytebase.v1.DataSource.project_id",
	"bytebase.v1.DataSource.region",
	"bytebase.v1.DataSource.replica_set",
	"bytebase.v1.DataSource.service_name",
	"bytebase.v1.DataSource.sid",
	"bytebase.v1.DataSource.ssh_host",
	"bytebase.v1.DataSource.ssh_port",
	"bytebase.v1.DataSource.ssh_user",
	"bytebase.v1.DataSource.username",
	"bytebase.v1.DataSource.warehouse_id",
	"bytebase.v1.DataSourceExternalSecret.AppRoleAuthOption.mount_path",
	"bytebase.v1.DataSourceExternalSecret.engine_name",
	"bytebase.v1.DataSourceExternalSecret.password_key_name",
	"bytebase.v1.DataSourceExternalSecret.secret_name",
	"bytebase.v1.DataSourceExternalSecret.url",
	"bytebase.v1.Database.effective_environment",
	"bytebase.v1.Database.environment",
	"bytebase.v1.Database.labels",
	"bytebase.v1.Database.name",
	"bytebase.v1.Database.project",
	"bytebase.v1.Database.release",
	"bytebase.v1.Database.sync_error",
	"bytebase.v1.DatabaseCatalog.name",
	"bytebase.v1.DatabaseGroup.Database.name",
	"bytebase.v1.DatabaseGroup.name",
	"bytebase.v1.DatabaseGroup.title",
	"bytebase.v1.DatabaseMetadata.character_set",
	"bytebase.v1.DatabaseMetadata.collation",
	"bytebase.v1.DatabaseMetadata.name",
	"bytebase.v1.DatabaseMetadata.owner",
	"bytebase.v1.DatabaseMetadata.search_path",
	"bytebase.v1.DatabaseSDLSchema.content_type",
	"bytebase.v1.DatabaseSDLSchema.schema",
	"bytebase.v1.DatabaseSchema.schema",
	"bytebase.v1.DeleteDatabaseGroupRequest.name",
	"bytebase.v1.DeleteGroupRequest.name",
	"bytebase.v1.DeleteIdentityProviderRequest.name",
	"bytebase.v1.DeleteInstanceRequest.name",
	"bytebase.v1.DeletePolicyRequest.name",
	"bytebase.v1.DeleteProjectRequest.name",
	"bytebase.v1.DeleteReleaseRequest.name",
	"bytebase.v1.DeleteReviewConfigRequest.name",
	"bytebase.v1.DeleteRevisionRequest.name",
	"bytebase.v1.DeleteRoleRequest.name",
	"bytebase.v1.DeleteSavedQueryRequest.name",
	"bytebase.v1.DeleteServiceAccountRequest.name",
	"bytebase.v1.DeleteUserRequest.name",
	"bytebase.v1.DeleteWorkloadIdentityRequest.name",
	"bytebase.v1.DeleteWorkspaceRequest.name",
	"bytebase.v1.DependencyColumn.column",
	"bytebase.v1.DependencyColumn.schema",
	"bytebase.v1.DependencyColumn.table",
	"bytebase.v1.DependencyTable.schema",
	"bytebase.v1.DependencyTable.table",
	"bytebase.v1.DiffMetadataRequest.name",
	"bytebase.v1.DiffMetadataResponse.diff",
	"bytebase.v1.DiffSchemaRequest.changelog",
	"bytebase.v1.DiffSchemaRequest.name",
	"bytebase.v1.DiffSchemaRequest.schema",
	"bytebase.v1.DiffSchemaResponse.diff",
	"bytebase.v1.DimensionConstraint.dimension",
	"bytebase.v1.DimensionalConfig.data_type",
	"bytebase.v1.DisableMFARequest.name",
	"bytebase.v1.EmailSetting.SMTPConfig.host",
	"bytebase.v1.EmailSetting.SMTPConfig.username",
	"bytebase.v1.EmailSetting.from",
	"bytebase.v1.EmailSetting.from_name",
	"bytebase.v1.EnableMFARequest.name",
	"bytebase.v1.EnumTypeMetadata.comment",
	"bytebase.v1.EnumTypeMetadata.name",
	"bytebase.v1.EnumTypeMetadata.values",
	"bytebase.v1.EnvironmentSetting.Environment.id",
	"bytebase.v1.EnvironmentSetting.Environment.name",
	"bytebase.v1.EnvironmentSetting.Environment.tags",
	"bytebase.v1.EnvironmentSetting.Environment.title",
	"bytebase.v1.EventMetadata.character_set_client",
	"bytebase.v1.EventMetadata.collation_connection",
	"bytebase.v1.EventMetadata.comment",
	"bytebase.v1.EventMetadata.definition",
	"bytebase.v1.EventMetadata.name",
	"bytebase.v1.EventMetadata.sql_mode",
	"bytebase.v1.EventMetadata.time_zone",
	"bytebase.v1.ExchangeTokenRequest.email",
	"bytebase.v1.ExportAuditLogsRequest.filter",
	"bytebase.v1.ExportAuditLogsRequest.order_by",
	"bytebase.v1.ExportAuditLogsRequest.page_token",
	"bytebase.v1.ExportAuditLogsRequest.parent",
	"bytebase.v1.ExportAuditLogsResponse.next_page_token",
	"bytebase.v1.ExportRequest.container",
	"bytebase.v1.ExportRequest.data_source_id",
	"bytebase.v1.ExportRequest.name",
	"bytebase.v1.ExportRequest.schema",
	"bytebase.v1.ExportRequest.statement",
	"bytebase.v1.ExportResponse.applied_access_grant",
	"bytebase.v1.ExtensionMetadata.description",
	"bytebase.v1.ExtensionMetadata.name",
	"bytebase.v1.ExtensionMetadata.schema",
	"bytebase.v1.ExtensionMetadata.version",
	"bytebase.v1.ExternalTableMetadata.external_database_name",
	"bytebase.v1.ExternalTableMetadata.external_server_name",
	"bytebase.v1.ExternalTableMetadata.name",
	"bytebase.v1.FieldMapping.display_name",
	"bytebase.v1.FieldMapping.groups",
	"bytebase.v1.FieldMapping.identifier",
	"bytebase.v1.FieldMapping.phone",
	"bytebase.v1.ForeignKeyMetadata.columns",
	"bytebase.v1.ForeignKeyMetadata.match_type",
	"bytebase.v1.ForeignKeyMetadata.name",
	"bytebase.v1.ForeignKeyMetadata.on_delete",
	"bytebase.v1.ForeignKeyMetadata.on_update",
	"bytebase.v1.ForeignKeyMetadata.referenced_columns",
	"bytebase.v1.ForeignKeyMetadata.referenced_schema",
	"bytebase.v1.ForeignKeyMetadata.referenced_table",
	"bytebase.v1.FunctionMetadata.character_set_client",
	"bytebase.v1.FunctionMetadata.collation_connection",
	"bytebase.v1.FunctionMetadata.comment",
	"bytebase.v1.FunctionMetadata.database_collation",
	"bytebase.v1.FunctionMetadata.definition",
	"bytebase.v1.FunctionMetadata.name",
	"bytebase.v1.FunctionMetadata.signature",
	"bytebase.v1.FunctionMetadata.sql_mode",
	"bytebase.v1.GenerationMetadata.expression",
	"bytebase.v1.GetAccessGrantRequest.name",
	"bytebase.v1.GetAuthenticationRestrictionRequest.workspace",
	"bytebase.v1.GetChangelogRequest.name",
	"bytebase.v1.GetDatabaseCatalogRequest.name",
	"bytebase.v1.GetDatabaseGroupRequest.name",
	"bytebase.v1.GetDatabaseMetadataRequest.filter",
	"bytebase.v1.GetDatabaseMetadataRequest.name",
	"bytebase.v1.GetDatabaseRequest.name",
	"bytebase.v1.GetDatabaseSDLSchemaRequest.name",
	"bytebase.v1.GetDatabaseSchemaRequest.name",
	"bytebase.v1.GetGroupRequest.name",
	"bytebase.v1.GetIamPolicyRequest.resource",
	"bytebase.v1.GetIdentityProviderRequest.name",
	"bytebase.v1.GetInstanceRequest.name",
	"bytebase.v1.GetIssueRequest.name",
	"bytebase.v1.GetPlanCheckRunRequest.name",
	"bytebase.v1.GetPlanRequest.name",
	"bytebase.v1.GetPolicyRequest.name",
	"bytebase.v1.GetProjectRequest.name",
	"bytebase.v1.GetQueryHistoryRequest.name",
	"bytebase.v1.GetReleaseRequest.name",
	"bytebase.v1.GetReviewConfigRequest.name",
	"bytebase.v1.GetRevisionRequest.name",
	"bytebase.v1.GetRoleRequest.name",
	"bytebase.v1.GetRolloutRequest.name",
	"bytebase.v1.GetSavedQueryPolicyRequest.resource",
	"bytebase.v1.GetSavedQueryRequest.name",
	"bytebase.v1.GetSchemaStringRequest.name",
	"bytebase.v1.GetSchemaStringRequest.object",
	"bytebase.v1.GetSchemaStringRequest.schema",
	"bytebase.v1.GetSchemaStringResponse.schema_string",
	"bytebase.v1.GetServiceAccountRequest.name",
	"bytebase.v1.GetSettingRequest.name",
	"bytebase.v1.GetSheetRequest.name",
	"bytebase.v1.GetTaskRunLogRequest.parent",
	"bytebase.v1.GetTaskRunRequest.name",
	"bytebase.v1.GetTaskRunSessionRequest.parent",
	"bytebase.v1.GetUserRequest.name",
	"bytebase.v1.GetWorkloadIdentityRequest.name",
	"bytebase.v1.GetWorkspaceRequest.name",
	"bytebase.v1.GridLevel.density",
	"bytebase.v1.Group.description",
	"bytebase.v1.Group.email",
	"bytebase.v1.Group.name",
	"bytebase.v1.Group.source",
	"bytebase.v1.Group.title",
	"bytebase.v1.GroupMember.member",
	"bytebase.v1.IamPolicy.etag",
	"bytebase.v1.IdentityProvider.domain",
	"bytebase.v1.IdentityProvider.name",
	"bytebase.v1.IdentityProvider.title",
	"bytebase.v1.IndexMetadata.comment",
	"bytebase.v1.IndexMetadata.definition",
	"bytebase.v1.IndexMetadata.expressions",
	"bytebase.v1.IndexMetadata.name",
	"bytebase.v1.IndexMetadata.opclass_names",
	"bytebase.v1.IndexMetadata.parent_index_name",
	"bytebase.v1.IndexMetadata.parent_index_schema",
	"bytebase.v1.IndexMetadata.type",
	"bytebase.v1.Instance.engine_version",
	"bytebase.v1.Instance.environment",
	"bytebase.v1.Instance.external_link",
	"bytebase.v1.Instance.labels",
	"bytebase.v1.Instance.name",
	"bytebase.v1.Instance.title",
	"bytebase.v1.InstanceResource.engine_version",
	"bytebase.v1.InstanceResource.environment",
	"bytebase.v1.InstanceResource.name",
	"bytebase.v1.InstanceResource.title",
	"bytebase.v1.InstanceRole.attribute",
	"bytebase.v1.InstanceRole.name",
	"bytebase.v1.InstanceRole.role_name",
	"bytebase.v1.InstanceRole.valid_until",
	"bytebase.v1.Issue.Approver.principal",
	"bytebase.v1.Issue.access_grant",
	"bytebase.v1.Issue.creator",
	"bytebase.v1.Issue.description",
	"bytebase.v1.Issue.labels",
	"bytebase.v1.Issue.name",
	"bytebase.v1.Issue.plan",
	"bytebase.v1.Issue.title",
	"bytebase.v1.IssueComment.IssueUpdate.from_description",
	"bytebase.v1.IssueComment.IssueUpdate.from_labels",
	"bytebase.v1.IssueComment.IssueUpdate.from_title",
	"bytebase.v1.IssueComment.IssueUpdate.to_description",
	"bytebase.v1.IssueComment.IssueUpdate.to_labels",
	"bytebase.v1.IssueComment.IssueUpdate.to_title",
	"bytebase.v1.IssueComment.comment",
	"bytebase.v1.IssueComment.creator",
	"bytebase.v1.IssueComment.name",
	"bytebase.v1.IssueComment.payload",
	"bytebase.v1.KerberosConfig.instance",
	"bytebase.v1.KerberosConfig.kdc_host",
	"bytebase.v1.KerberosConfig.kdc_port",
	"bytebase.v1.KerberosConfig.kdc_transport_protocol",
	"bytebase.v1.KerberosConfig.primary",
	"bytebase.v1.KerberosConfig.realm",
	"bytebase.v1.LDAPIdentityProviderConfig.base_dn",
	"bytebase.v1.LDAPIdentityProviderConfig.bind_dn",
	"bytebase.v1.LDAPIdentityProviderConfig.host",
	"bytebase.v1.LDAPIdentityProviderConfig.user_filter",
	"bytebase.v1.LDAPIdentityProviderTestRequestContext.username",
	"bytebase.v1.Label.group",
	"bytebase.v1.Label.value",
	"bytebase.v1.LeaveWorkspaceRequest.name",
	"bytebase.v1.ListAccessGrantsRequest.filter",
	"bytebase.v1.ListAccessGrantsRequest.order_by",
	"bytebase.v1.ListAccessGrantsRequest.page_token",
	"bytebase.v1.ListAccessGrantsRequest.parent",
	"bytebase.v1.ListAccessGrantsResponse.next_page_token",
	"bytebase.v1.ListChangelogsRequest.filter",
	"bytebase.v1.ListChangelogsRequest.page_token",
	"bytebase.v1.ListChangelogsRequest.parent",
	"bytebase.v1.ListChangelogsResponse.next_page_token",
	"bytebase.v1.ListDatabaseGroupsRequest.parent",
	"bytebase.v1.ListDatabasesRequest.filter",
	"bytebase.v1.ListDatabasesRequest.order_by",
	"bytebase.v1.ListDatabasesRequest.page_token",
	"bytebase.v1.ListDatabasesRequest.parent",
	"bytebase.v1.ListDatabasesResponse.next_page_token",
	"bytebase.v1.ListGroupsRequest.filter",
	"bytebase.v1.ListGroupsRequest.page_token",
	"bytebase.v1.ListGroupsResponse.next_page_token",
	"bytebase.v1.ListIdentityProvidersRequest.parent",
	"bytebase.v1.ListInstanceDatabaseRequest.name",
	"bytebase.v1.ListInstanceDatabaseResponse.databases",
	"bytebase.v1.ListInstanceRolesRequest.page_token",
	"bytebase.v1.ListInstanceRolesRequest.parent",
	"bytebase.v1.ListInstanceRolesResponse.next_page_token",
	"bytebase.v1.ListInstancesRequest.filter",
	"bytebase.v1.ListInstancesRequest.order_by",
	"bytebase.v1.ListInstancesRequest.page_token",
	"bytebase.v1.ListInstancesRequest.parent",
	"bytebase.v1.ListInstancesResponse.next_page_token",
	"bytebase.v1.ListIssueCommentsRequest.page_token",
	"bytebase.v1.ListIssueCommentsRequest.parent",
	"bytebase.v1.ListIssueCommentsResponse.next_page_token",
	"bytebase.v1.ListIssuesRequest.filter",
	"bytebase.v1.ListIssuesRequest.order_by",
	"bytebase.v1.ListIssuesRequest.page_token",
	"bytebase.v1.ListIssuesRequest.parent",
	"bytebase.v1.ListIssuesRequest.query",
	"bytebase.v1.ListIssuesResponse.next_page_token",
	"bytebase.v1.ListPlansRequest.filter",
	"bytebase.v1.ListPlansRequest.page_token",
	"bytebase.v1.ListPlansRequest.parent",
	"bytebase.v1.ListPlansResponse.next_page_token",
	"bytebase.v1.ListPoliciesRequest.parent",
	"bytebase.v1.ListProjectsRequest.filter",
	"bytebase.v1.ListProjectsRequest.order_by",
	"bytebase.v1.ListProjectsRequest.page_token",
	"bytebase.v1.ListProjectsResponse.next_page_token",
	"bytebase.v1.ListQueryHistoriesRequest.filter",
	"bytebase.v1.ListQueryHistoriesRequest.page_token",
	"bytebase.v1.ListQueryHistoriesRequest.parent",
	"bytebase.v1.ListQueryHistoriesResponse.next_page_token",
	"bytebase.v1.ListReleaseCategoriesRequest.parent",
	"bytebase.v1.ListReleaseCategoriesResponse.categories",
	"bytebase.v1.ListReleasesRequest.filter",
	"bytebase.v1.ListReleasesRequest.page_token",
	"bytebase.v1.ListReleasesRequest.parent",
	"bytebase.v1.ListReleasesResponse.next_page_token",
	"bytebase.v1.ListRevisionsRequest.page_token",
	"bytebase.v1.ListRevisionsRequest.parent",
	"bytebase.v1.ListRevisionsResponse.next_page_token",
	"bytebase.v1.ListRolloutsRequest.filter",
	"bytebase.v1.ListRolloutsRequest.page_token",
	"bytebase.v1.ListRolloutsRequest.parent",
	"bytebase.v1.ListRolloutsResponse.next_page_token",
	"bytebase.v1.ListSavedQueriesRequest.filter",
	"bytebase.v1.ListSavedQueriesRequest.order_by",
	"bytebase.v1.ListSavedQueriesRequest.page_token",
	"bytebase.v1.ListSavedQueriesRequest.parent",
	"bytebase.v1.ListSavedQueriesResponse.next_page_token",
	"bytebase.v1.ListServiceAccountsRequest.filter",
	"bytebase.v1.ListServiceAccountsRequest.page_token",
	"bytebase.v1.ListServiceAccountsRequest.parent",
	"bytebase.v1.ListServiceAccountsResponse.next_page_token",
	"bytebase.v1.ListTaskRunsRequest.parent",
	"bytebase.v1.ListUsersRequest.filter",
	"bytebase.v1.ListUsersRequest.page_token",
	"bytebase.v1.ListUsersResponse.next_page_token",
	"bytebase.v1.ListWorkloadIdentitiesRequest.filter",
	"bytebase.v1.ListWorkloadIdentitiesRequest.page_token",
	"bytebase.v1.ListWorkloadIdentitiesRequest.parent",
	"bytebase.v1.ListWorkloadIdentitiesResponse.next_page_token",
	"bytebase.v1.LoginRequest.email",
	"bytebase.v1.LoginRequest.idp_name",
	"bytebase.v1.LoginRequest.workspace",
	"bytebase.v1.MCPDelegation.client_id",
	"bytebase.v1.MCPDelegation.correlation_id",
	"bytebase.v1.MCPDelegation.resource",
	"bytebase.v1.MCPDelegation.scope",
	"bytebase.v1.MCPEngineEnforcement.note",
	"bytebase.v1.MCPInfo.workspace",
	"bytebase.v1.MCPMethod.method",
	"bytebase.v1.MCPMethod.operation_id",
	"bytebase.v1.MCPMethod.permission",
	"bytebase.v1.MaskingExemptionPolicy.Exemption.members",
	"bytebase.v1.MaskingReason.algorithm",
	"bytebase.v1.MaskingReason.context",
	"bytebase.v1.MaskingReason.masking_rule_id",
	"bytebase.v1.MaskingReason.semantic_type_id",
	"bytebase.v1.MaskingReason.semantic_type_title",
	"bytebase.v1.MaskingRulePolicy.MaskingRule.id",
	"bytebase.v1.MaskingRulePolicy.MaskingRule.semantic_type",
	"bytebase.v1.MaterializedViewMetadata.comment",
	"bytebase.v1.MaterializedViewMetadata.definition",
	"bytebase.v1.MaterializedViewMetadata.name",
	"bytebase.v1.MoveMySavedQueriesRequest.parent",
	"bytebase.v1.MoveMySavedQueriesRequest.source_folder",
	"bytebase.v1.MoveMySavedQueriesRequest.target_folder",
	"bytebase.v1.OAuth2IdentityProviderConfig.auth_url",
	"bytebase.v1.OAuth2IdentityProviderConfig.client_id",
	"bytebase.v1.OAuth2IdentityProviderConfig.scopes",
	"bytebase.v1.OAuth2IdentityProviderConfig.token_url",
	"bytebase.v1.OAuth2IdentityProviderConfig.user_info_url",
	"bytebase.v1.OIDCIdentityProviderConfig.auth_endpoint",
	"bytebase.v1.OIDCIdentityProviderConfig.client_id",
	"bytebase.v1.OIDCIdentityProviderConfig.issuer",
	"bytebase.v1.OIDCIdentityProviderConfig.scopes",
	"bytebase.v1.ObjectSchema.semantic_type",
	"bytebase.v1.PackageMetadata.definition",
	"bytebase.v1.PackageMetadata.name",
	"bytebase.v1.PaymentInfo.currency",
	"bytebase.v1.PaymentInfo.invoice_url",
	"bytebase.v1.PaymentInfo.next_period_price",
	"bytebase.v1.PaymentInfo.period_end",
	"bytebase.v1.PaymentInfo.period_start",
	"bytebase.v1.PaymentInfo.total_price",
	"bytebase.v1.PermissionDeniedDetail.method",
	"bytebase.v1.PermissionDeniedDetail.required_permissions",
	"bytebase.v1.PermissionDeniedDetail.resources",
	"bytebase.v1.Plan.ChangeDatabaseConfig.release",
	"bytebase.v1.Plan.ChangeDatabaseConfig.sheet",
	"bytebase.v1.Plan.ChangeDatabaseConfig.targets",
	"bytebase.v1.Plan.CreateDatabaseConfig.character_set",
	"bytebase.v1.Plan.CreateDatabaseConfig.cluster",
	"bytebase.v1.Plan.CreateDatabaseConfig.collation",
	"bytebase.v1.Plan.CreateDatabaseConfig.database",
	"bytebase.v1.Plan.CreateDatabaseConfig.environment",
	"bytebase.v1.Plan.CreateDatabaseConfig.owner",
	"bytebase.v1.Plan.CreateDatabaseConfig.table",
	"bytebase.v1.Plan.CreateDatabaseConfig.target",
	"bytebase.v1.Plan.RolloutStageSummary.stage",
	"bytebase.v1.Plan.Spec.id",
	"bytebase.v1.Plan.creator",
	"bytebase.v1.Plan.description",
	"bytebase.v1.Plan.issue",
	"bytebase.v1.Plan.name",
	"bytebase.v1.Plan.title",
	"bytebase.v1.PlanCheckRun.Result.content",
	"bytebase.v1.PlanCheckRun.Result.target",
	"bytebase.v1.PlanCheckRun.Result.title",
	"bytebase.v1.PlanCheckRun.error",
	"bytebase.v1.PlanCheckRun.name",
	"bytebase.v1.Policy.name",
	"bytebase.v1.PrepareSampleProjectInstanceRequest.parent",
	"bytebase.v1.PreviewTaskRunRollbackRequest.name",
	"bytebase.v1.PreviewTaskRunRollbackResponse.statement",
	"bytebase.v1.ProcedureMetadata.character_set_client",
	"bytebase.v1.ProcedureMetadata.collation_connection",
	"bytebase.v1.ProcedureMetadata.comment",
	"bytebase.v1.ProcedureMetadata.database_collation",
	"bytebase.v1.ProcedureMetadata.definition",
	"bytebase.v1.ProcedureMetadata.name",
	"bytebase.v1.ProcedureMetadata.signature",
	"bytebase.v1.ProcedureMetadata.sql_mode",
	"bytebase.v1.Project.data_classification_config_id",
	"bytebase.v1.Project.labels",
	"bytebase.v1.Project.name",
	"bytebase.v1.Project.title",
	"bytebase.v1.QueryHistory.creator",
	"bytebase.v1.QueryHistory.database",
	"bytebase.v1.QueryHistory.error",
	"bytebase.v1.QueryHistory.name",
	"bytebase.v1.QueryHistory.statement",
	"bytebase.v1.QueryRequest.container",
	"bytebase.v1.QueryRequest.data_source_id",
	"bytebase.v1.QueryRequest.name",
	"bytebase.v1.QueryRequest.schema",
	"bytebase.v1.QueryRequest.statement",
	"bytebase.v1.QueryResponse.applied_access_grant",
	"bytebase.v1.QueryResult.PostgresError.code",
	"bytebase.v1.QueryResult.PostgresError.column_name",
	"bytebase.v1.QueryResult.PostgresError.constraint_name",
	"bytebase.v1.QueryResult.PostgresError.data_type_name",
	"bytebase.v1.QueryResult.PostgresError.detail",
	"bytebase.v1.QueryResult.PostgresError.file",
	"bytebase.v1.QueryResult.PostgresError.hint",
	"bytebase.v1.QueryResult.PostgresError.internal_query",
	"bytebase.v1.QueryResult.PostgresError.message",
	"bytebase.v1.QueryResult.PostgresError.routine",
	"bytebase.v1.QueryResult.PostgresError.schema_name",
	"bytebase.v1.QueryResult.PostgresError.severity",
	"bytebase.v1.QueryResult.PostgresError.table_name",
	"bytebase.v1.QueryResult.PostgresError.where",
	"bytebase.v1.QueryResult.column_names",
	"bytebase.v1.QueryResult.column_type_names",
	"bytebase.v1.QueryResult.error",
	"bytebase.v1.QueryResult.statement",
	"bytebase.v1.RegenerateRecoveryCodesRequest.name",
	"bytebase.v1.RejectIssueRequest.comment",
	"bytebase.v1.RejectIssueRequest.name",
	"bytebase.v1.Release.File.path",
	"bytebase.v1.Release.File.sheet",
	"bytebase.v1.Release.File.sheet_sha256",
	"bytebase.v1.Release.File.version",
	"bytebase.v1.Release.VCSSource.url",
	"bytebase.v1.Release.category",
	"bytebase.v1.Release.creator",
	"bytebase.v1.Release.name",
	"bytebase.v1.RemoveDataSourceRequest.name",
	"bytebase.v1.RequestIssueRequest.comment",
	"bytebase.v1.RequestIssueRequest.name",
	"bytebase.v1.RequestMetadata.caller_ip",
	"bytebase.v1.RequestMetadata.caller_supplied_user_agent",
	"bytebase.v1.RequestPasswordResetRequest.email",
	"bytebase.v1.RequestPasswordResetRequest.workspace",
	"bytebase.v1.RequestReauthCodeRequest.name",
	"bytebase.v1.ResetPasswordRequest.email",
	"bytebase.v1.RetryIssueApprovalRequest.name",
	"bytebase.v1.ReviewConfig.name",
	"bytebase.v1.ReviewConfig.resources",
	"bytebase.v1.ReviewConfig.title",
	"bytebase.v1.Revision.deleter",
	"bytebase.v1.Revision.file",
	"bytebase.v1.Revision.name",
	"bytebase.v1.Revision.release",
	"bytebase.v1.Revision.sheet",
	"bytebase.v1.Revision.sheet_sha256",
	"bytebase.v1.Revision.task_run",
	"bytebase.v1.Revision.version",
	"bytebase.v1.RevokeAccessGrantRequest.name",
	"bytebase.v1.Role.description",
	"bytebase.v1.Role.name",
	"bytebase.v1.Role.permissions",
	"bytebase.v1.Role.title",
	"bytebase.v1.RoleGrant.role",
	"bytebase.v1.RoleGrant.user",
	"bytebase.v1.Rollout.name",
	"bytebase.v1.Rollout.title",
	"bytebase.v1.RolloutPolicy.roles",
	"bytebase.v1.RotateDirectorySyncTokenRequest.name",
	"bytebase.v1.RunPlanChecksRequest.name",
	"bytebase.v1.RunPlanChecksRequest.spec_id",
	"bytebase.v1.SQLEditorThemeSetting.id",
	"bytebase.v1.SQLEditorThemeSetting.monaco_base",
	"bytebase.v1.SQLEditorThemeSetting.name",
	"bytebase.v1.SQLReviewRule.NamingRulePayload.format",
	"bytebase.v1.SQLReviewRule.StringArrayRulePayload.list",
	"bytebase.v1.SQLReviewRule.StringRulePayload.value",
	"bytebase.v1.SampleInfo.Instance.instance",
	"bytebase.v1.SavedQuery.creator",
	"bytebase.v1.SavedQuery.database",
	"bytebase.v1.SavedQuery.folder",
	"bytebase.v1.SavedQuery.name",
	"bytebase.v1.SavedQuery.project",
	"bytebase.v1.SavedQuery.title",
	"bytebase.v1.SavedQueryBinding.members",
	"bytebase.v1.SavedQueryPolicy.etag",
	"bytebase.v1.SchemaCatalog.name",
	"bytebase.v1.SchemaMetadata.comment",
	"bytebase.v1.SchemaMetadata.name",
	"bytebase.v1.SchemaMetadata.owner",
	"bytebase.v1.SearchAuditLogsRequest.filter",
	"bytebase.v1.SearchAuditLogsRequest.order_by",
	"bytebase.v1.SearchAuditLogsRequest.page_token",
	"bytebase.v1.SearchAuditLogsRequest.parent",
	"bytebase.v1.SearchAuditLogsResponse.next_page_token",
	"bytebase.v1.SearchIssuesRequest.filter",
	"bytebase.v1.SearchIssuesRequest.order_by",
	"bytebase.v1.SearchIssuesRequest.page_token",
	"bytebase.v1.SearchIssuesRequest.parent",
	"bytebase.v1.SearchIssuesRequest.query",
	"bytebase.v1.SearchIssuesResponse.next_page_token",
	"bytebase.v1.SearchMyAccessGrantsRequest.filter",
	"bytebase.v1.SearchMyAccessGrantsRequest.order_by",
	"bytebase.v1.SearchMyAccessGrantsRequest.page_token",
	"bytebase.v1.SearchMyAccessGrantsRequest.parent",
	"bytebase.v1.SearchMyAccessGrantsResponse.next_page_token",
	"bytebase.v1.SearchProjectsRequest.filter",
	"bytebase.v1.SearchProjectsRequest.order_by",
	"bytebase.v1.SearchProjectsRequest.page_token",
	"bytebase.v1.SearchProjectsResponse.next_page_token",
	"bytebase.v1.SearchQueryHistoriesRequest.filter",
	"bytebase.v1.SearchQueryHistoriesRequest.page_token",
	"bytebase.v1.SearchQueryHistoriesRequest.parent",
	"bytebase.v1.SearchQueryHistoriesResponse.next_page_token",
	"bytebase.v1.SearchSavedQueriesRequest.filter",
	"bytebase.v1.SearchSavedQueriesRequest.page_token",
	"bytebase.v1.SearchSavedQueriesRequest.parent",
	"bytebase.v1.SearchSavedQueriesResponse.next_page_token",
	"bytebase.v1.SearchSavedQueryFoldersRequest.filter",
	"bytebase.v1.SearchSavedQueryFoldersRequest.parent",
	"bytebase.v1.SearchSavedQueryFoldersResponse.folders",
	"bytebase.v1.SemanticTypeSetting.SemanticType.description",
	"bytebase.v1.SemanticTypeSetting.SemanticType.icon",
	"bytebase.v1.SemanticTypeSetting.SemanticType.id",
	"bytebase.v1.SemanticTypeSetting.SemanticType.title",
	"bytebase.v1.SendEmailLoginCodeRequest.email",
	"bytebase.v1.SendEmailLoginCodeRequest.workspace",
	"bytebase.v1.SequenceMetadata.cache_size",
	"bytebase.v1.SequenceMetadata.comment",
	"bytebase.v1.SequenceMetadata.data_type",
	"bytebase.v1.SequenceMetadata.increment",
	"bytebase.v1.SequenceMetadata.last_value",
	"bytebase.v1.SequenceMetadata.max_value",
	"bytebase.v1.SequenceMetadata.min_value",
	"bytebase.v1.SequenceMetadata.name",
	"bytebase.v1.SequenceMetadata.owner_column",
	"bytebase.v1.SequenceMetadata.owner_table",
	"bytebase.v1.SequenceMetadata.start",
	"bytebase.v1.ServiceAccount.email",
	"bytebase.v1.ServiceAccount.name",
	"bytebase.v1.ServiceAccount.title",
	"bytebase.v1.SetIamPolicyRequest.etag",
	"bytebase.v1.SetIamPolicyRequest.resource",
	"bytebase.v1.SetSavedQueryPolicyRequest.resource",
	"bytebase.v1.Setting.name",
	"bytebase.v1.Sheet.name",
	"bytebase.v1.SignupRequest.email",
	"bytebase.v1.SignupRequest.title",
	"bytebase.v1.SpatialIndexConfig.method",
	"bytebase.v1.Stage.environment",
	"bytebase.v1.Stage.id",
	"bytebase.v1.Stage.name",
	"bytebase.v1.StartMFAEnrollmentRequest.name",
	"bytebase.v1.StorageConfig.buffering",
	"bytebase.v1.StorageConfig.data_compression",
	"bytebase.v1.StorageConfig.sort_in_tempdb",
	"bytebase.v1.StorageConfig.tablespace",
	"bytebase.v1.StorageConfig.work_tablespace",
	"bytebase.v1.StreamMetadata.comment",
	"bytebase.v1.StreamMetadata.definition",
	"bytebase.v1.StreamMetadata.name",
	"bytebase.v1.StreamMetadata.owner",
	"bytebase.v1.StreamMetadata.table_name",
	"bytebase.v1.Subscription.etag",
	"bytebase.v1.Subscription.org_name",
	"bytebase.v1.SwitchWorkspaceRequest.workspace",
	"bytebase.v1.SyncDatabaseRequest.name",
	"bytebase.v1.SyncDatabases.databases",
	"bytebase.v1.SyncInstanceRequest.name",
	"bytebase.v1.SyncInstanceResponse.databases",
	"bytebase.v1.TableCatalog.classification",
	"bytebase.v1.TableCatalog.name",
	"bytebase.v1.TableMetadata.charset",
	"bytebase.v1.TableMetadata.collation",
	"bytebase.v1.TableMetadata.comment",
	"bytebase.v1.TableMetadata.create_options",
	"bytebase.v1.TableMetadata.engine",
	"bytebase.v1.TableMetadata.name",
	"bytebase.v1.TableMetadata.owner",
	"bytebase.v1.TableMetadata.primary_key_type",
	"bytebase.v1.TableMetadata.sharding_info",
	"bytebase.v1.TableMetadata.sorting_keys",
	"bytebase.v1.TablePartitionMetadata.expression",
	"bytebase.v1.TablePartitionMetadata.name",
	"bytebase.v1.TablePartitionMetadata.use_default",
	"bytebase.v1.TablePartitionMetadata.value",
	"bytebase.v1.TagPolicy.tags",
	"bytebase.v1.Task.DatabaseCreate.sheet",
	"bytebase.v1.Task.DatabaseUpdate.release",
	"bytebase.v1.Task.DatabaseUpdate.sheet",
	"bytebase.v1.Task.name",
	"bytebase.v1.Task.skipped_reason",
	"bytebase.v1.Task.spec_id",
	"bytebase.v1.Task.target",
	"bytebase.v1.TaskMetadata.comment",
	"bytebase.v1.TaskMetadata.condition",
	"bytebase.v1.TaskMetadata.definition",
	"bytebase.v1.TaskMetadata.id",
	"bytebase.v1.TaskMetadata.name",
	"bytebase.v1.TaskMetadata.owner",
	"bytebase.v1.TaskMetadata.predecessors",
	"bytebase.v1.TaskMetadata.schedule",
	"bytebase.v1.TaskMetadata.warehouse",
	"bytebase.v1.TaskRun.creator",
	"bytebase.v1.TaskRun.detail",
	"bytebase.v1.TaskRun.name",
	"bytebase.v1.TaskRunLog.name",
	"bytebase.v1.TaskRunLogEntry.CommandExecute.CommandResponse.error",
	"bytebase.v1.TaskRunLogEntry.CommandExecute.statement",
	"bytebase.v1.TaskRunLogEntry.ComputeDiff.error",
	"bytebase.v1.TaskRunLogEntry.DatabaseSync.error",
	"bytebase.v1.TaskRunLogEntry.GhostMigration.error",
	"bytebase.v1.TaskRunLogEntry.PriorBackup.PriorBackupDetail.Item.Table.database",
	"bytebase.v1.TaskRunLogEntry.PriorBackup.PriorBackupDetail.Item.Table.schema",
	"bytebase.v1.TaskRunLogEntry.PriorBackup.PriorBackupDetail.Item.Table.table",
	"bytebase.v1.TaskRunLogEntry.PriorBackup.error",
	"bytebase.v1.TaskRunLogEntry.ReleaseFileExecute.file_path",
	"bytebase.v1.TaskRunLogEntry.ReleaseFileExecute.version",
	"bytebase.v1.TaskRunLogEntry.RetryInfo.error",
	"bytebase.v1.TaskRunLogEntry.SchemaDump.error",
	"bytebase.v1.TaskRunLogEntry.TransactionControl.error",
	"bytebase.v1.TaskRunLogEntry.replica_id",
	"bytebase.v1.TaskRunSession.Postgres.Session.application_name",
	"bytebase.v1.TaskRunSession.Postgres.Session.blocked_by_pids",
	"bytebase.v1.TaskRunSession.Postgres.Session.client_addr",
	"bytebase.v1.TaskRunSession.Postgres.Session.client_port",
	"bytebase.v1.TaskRunSession.Postgres.Session.datname",
	"bytebase.v1.TaskRunSession.Postgres.Session.pid",
	"bytebase.v1.TaskRunSession.Postgres.Session.query",
	"bytebase.v1.TaskRunSession.Postgres.Session.state",
	"bytebase.v1.TaskRunSession.Postgres.Session.usename",
	"bytebase.v1.TaskRunSession.Postgres.Session.wait_event",
	"bytebase.v1.TaskRunSession.Postgres.Session.wait_event_type",
	"bytebase.v1.TaskRunSession.name",
	"bytebase.v1.TessellationConfig.scheme",
	"bytebase.v1.TestEmailSettingRequest.parent",
	"bytebase.v1.TestEmailSettingRequest.to",
	"bytebase.v1.TestEmailSettingResponse.error",
	"bytebase.v1.TestIdentityProviderResponse.claims",
	"bytebase.v1.TestIdentityProviderResponse.user_info",
	"bytebase.v1.TestWebhookRequest.project",
	"bytebase.v1.TestWebhookResponse.error",
	"bytebase.v1.TriggerMetadata.body",
	"bytebase.v1.TriggerMetadata.character_set_client",
	"bytebase.v1.TriggerMetadata.collation_connection",
	"bytebase.v1.TriggerMetadata.comment",
	"bytebase.v1.TriggerMetadata.event",
	"bytebase.v1.TriggerMetadata.name",
	"bytebase.v1.TriggerMetadata.sql_mode",
	"bytebase.v1.TriggerMetadata.timing",
	"bytebase.v1.UndeleteInstanceRequest.name",
	"bytebase.v1.UndeleteProjectRequest.name",
	"bytebase.v1.UndeleteReleaseRequest.name",
	"bytebase.v1.UndeleteServiceAccountRequest.name",
	"bytebase.v1.UndeleteUserRequest.name",
	"bytebase.v1.UndeleteWorkloadIdentityRequest.name",
	"bytebase.v1.UpdateDataSourceRequest.name",
	"bytebase.v1.UpdateEmailRequest.email",
	"bytebase.v1.UpdateEmailRequest.name",
	"bytebase.v1.UpdateIssueCommentRequest.parent",
	"bytebase.v1.UpdatePurchaseRequest.etag",
	"bytebase.v1.UpdateSavedQueryStarRequest.name",
	"bytebase.v1.User.email",
	"bytebase.v1.User.name",
	"bytebase.v1.User.title",
	"bytebase.v1.User.workspace",
	"bytebase.v1.VCSUser.display_name",
	"bytebase.v1.VCSUser.user_id",
	"bytebase.v1.VCSUser.user_name",
	"bytebase.v1.VerifyCheckoutSessionResponse.status",
	"bytebase.v1.ViewMetadata.comment",
	"bytebase.v1.ViewMetadata.definition",
	"bytebase.v1.ViewMetadata.name",
	"bytebase.v1.Webhook.name",
	"bytebase.v1.Webhook.title",
	"bytebase.v1.WorkloadIdentity.email",
	"bytebase.v1.WorkloadIdentity.name",
	"bytebase.v1.WorkloadIdentity.title",
	"bytebase.v1.WorkloadIdentityConfig.allowed_audiences",
	"bytebase.v1.WorkloadIdentityConfig.issuer_url",
	"bytebase.v1.WorkloadIdentityConfig.subject_pattern",
	"bytebase.v1.Workspace.logo",
	"bytebase.v1.Workspace.name",
	"bytebase.v1.Workspace.title",
	"bytebase.v1.WorkspaceProfileSetting.domains",
	"bytebase.v1.WorkspaceProfileSetting.external_url",
	"bytebase.v1.WorkspaceProfileSetting.sql_editor_theme_id",
	"google.api.expr.v1alpha1.Constant.bytes_value",
	"google.api.expr.v1alpha1.Constant.string_value",
	"google.api.expr.v1alpha1.Expr.Call.function",
	"google.api.expr.v1alpha1.Expr.Comprehension.accu_var",
	"google.api.expr.v1alpha1.Expr.Comprehension.iter_var",
	"google.api.expr.v1alpha1.Expr.Comprehension.iter_var2",
	"google.api.expr.v1alpha1.Expr.CreateStruct.Entry.field_key",
	"google.api.expr.v1alpha1.Expr.CreateStruct.message_name",
	"google.api.expr.v1alpha1.Expr.Ident.name",
	"google.api.expr.v1alpha1.Expr.Select.field",
	"google.protobuf.Any.type_url",
	"google.protobuf.Any.value",
	"google.protobuf.FieldMask.paths",
	"google.rpc.Status.message",
	"google.type.Expr.description",
	"google.type.Expr.expression",
	"google.type.Expr.location",
	"google.type.Expr.title",
}

// TestAuditRowNeedsNoRedactionBeyondTheAnyPayloads pins the narrowing in
// createAuditLog. The row is not walked as a whole; service_data and
// status.details are redacted where they are assigned, and everything else is
// left alone because it holds nothing annotated.
//
// That is a fact about today's protos, not a guarantee, so it is asserted
// rather than assumed: annotate a field on RequestMetadata or MCPDelegation —
// caller_supplied_user_agent is caller-controlled and the likeliest candidate —
// and this fails, because nothing would redact it.
func TestAuditRowNeedsNoRedactionBeyondTheAnyPayloads(t *testing.T) {
	for _, message := range []proto.Message{&storepb.RequestMetadata{}, &storepb.MCPDelegation{}} {
		descriptor := message.ProtoReflect().Descriptor()
		require.Nil(t, planFor(descriptor),
			"%s is assigned onto the audit row unredacted; annotating a field under it needs redaction wired up "+
				"in createAuditLog first", descriptor.FullName())
	}

	// The row's own plan may cover the two Any fields and nothing else.
	plan := planFor((&storepb.AuditLog{}).ProtoReflect().Descriptor())
	require.NotNil(t, plan)
	fields := (&storepb.AuditLog{}).ProtoReflect().Descriptor().Fields()
	var covered []string
	for number := range plan.fields {
		covered = append(covered, string(fields.ByNumber(number).Name()))
	}
	slices.Sort(covered)
	require.Equal(t, []string{"service_data", "status"}, covered,
		"a field on the audit row started needing redaction; redact it where it is assigned in createAuditLog")
}

// TestLintAuditAnyFieldsAreRegistered is the descriptor half. It catches a new
// google.protobuf.Any FIELD reaching an audit row — a field the walk can see,
// even though the type packed inside it is chosen in Go and cannot be seen.
// Without it a new Any field would be governed by nothing: the runtime drops
// what it cannot place, so the payload would vanish silently.
func TestLintAuditAnyFieldsAreRegistered(t *testing.T) {
	found := auditReachableAnyFields(t)

	var unregistered []string
	for _, field := range found {
		if _, ok := auditAnyRegistry[protoreflect.FullName(field)]; !ok {
			unregistered = append(unregistered, field)
		}
	}
	slices.Sort(unregistered)
	require.Empty(t, unregistered,
		"these carry a google.protobuf.Any onto an audit row and auditAnyRegistry does not name what may be "+
			"packed in them, so the runtime would drop whatever they carry: register the permitted types")

	var stale []protoreflect.FullName
	for field := range auditAnyRegistry {
		if !slices.Contains(found, string(field)) {
			stale = append(stale, field)
		}
	}
	slices.SortFunc(stale, func(a, b protoreflect.FullName) int { return strings.Compare(string(a), string(b)) })
	require.Empty(t, stale, "these no longer reach an audit row; drop them so the registry keeps meaning something")
}

// auditReachableAnyFields is every google.protobuf.Any-typed field reachable
// from a message that can reach an audit payload, including the audit row's own
// directly assigned fields.
func auditReachableAnyFields(t *testing.T) []string {
	t.Helper()
	seen := map[protoreflect.FullName]bool{}
	var found []string
	var walk func(protoreflect.MessageDescriptor)
	walk = func(descriptor protoreflect.MessageDescriptor) {
		if seen[descriptor.FullName()] {
			return
		}
		seen[descriptor.FullName()] = true
		fields := descriptor.Fields()
		for i := range fields.Len() {
			field := fields.Get(i)
			// An annotated field never reaches the row, so neither does an Any
			// at or under it. Skipping matches what the redactor does and what
			// the inventory walk counts as reachable.
			if auditBehaviorOf(field) != v1pb.AuditBehavior_AUDIT_BEHAVIOR_UNSPECIFIED {
				continue
			}
			sub := submessageOf(field)
			if sub == nil {
				continue
			}
			if sub.FullName() == anyFullName {
				found = append(found, string(field.FullName()))
				continue
			}
			walk(sub)
		}
	}

	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if fd.Package() != "bytebase.v1" {
			return true
		}
		services := fd.Services()
		for i := range services.Len() {
			methods := services.Get(i).Methods()
			for j := range methods.Len() {
				walk(methods.Get(j).Input())
				walk(methods.Get(j).Output())
			}
		}
		return true
	})
	walk((&storepb.AuditLog{}).ProtoReflect().Descriptor())

	slices.Sort(found)
	return slices.Compact(found)
}
