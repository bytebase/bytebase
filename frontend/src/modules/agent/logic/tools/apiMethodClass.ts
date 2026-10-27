import { getOption, hasOption } from "@bufbuild/protobuf";
import { AccessGrantService } from "@/types/proto-es/v1/access_grant_service_pb";
import { ActuatorService } from "@/types/proto-es/v1/actuator_service_pb";
import { AIService } from "@/types/proto-es/v1/ai_service_pb";
import {
  AuthMethod,
  auth_method,
  MCPMethodClass,
  mcp_method_class,
  permission,
} from "@/types/proto-es/v1/annotation_pb";
import { AuditLogService } from "@/types/proto-es/v1/audit_log_service_pb";
import { AuthService } from "@/types/proto-es/v1/auth_service_pb";
import { CelService } from "@/types/proto-es/v1/cel_service_pb";
import { ChangelogService } from "@/types/proto-es/v1/changelog_service_pb";
import { DatabaseCatalogService } from "@/types/proto-es/v1/database_catalog_service_pb";
import { DatabaseGroupService } from "@/types/proto-es/v1/database_group_service_pb";
import { DatabaseService } from "@/types/proto-es/v1/database_service_pb";
import { GroupService } from "@/types/proto-es/v1/group_service_pb";
import { IdentityProviderService } from "@/types/proto-es/v1/idp_service_pb";
import { InstanceRoleService } from "@/types/proto-es/v1/instance_role_service_pb";
import { InstanceService } from "@/types/proto-es/v1/instance_service_pb";
import { IssueService } from "@/types/proto-es/v1/issue_service_pb";
import { OrgPolicyService } from "@/types/proto-es/v1/org_policy_service_pb";
import { PlanService } from "@/types/proto-es/v1/plan_service_pb";
import { ProjectService } from "@/types/proto-es/v1/project_service_pb";
import { QueryHistoryService } from "@/types/proto-es/v1/query_history_service_pb";
import { ReleaseService } from "@/types/proto-es/v1/release_service_pb";
import { ReviewConfigService } from "@/types/proto-es/v1/review_config_service_pb";
import { RevisionService } from "@/types/proto-es/v1/revision_service_pb";
import { RoleService } from "@/types/proto-es/v1/role_service_pb";
import { RolloutService } from "@/types/proto-es/v1/rollout_service_pb";
import { SavedQueryService } from "@/types/proto-es/v1/saved_query_service_pb";
import { ServiceAccountService } from "@/types/proto-es/v1/service_account_service_pb";
import { SettingService } from "@/types/proto-es/v1/setting_service_pb";
import { SheetService } from "@/types/proto-es/v1/sheet_service_pb";
import { SQLService } from "@/types/proto-es/v1/sql_service_pb";
import { SubscriptionService } from "@/types/proto-es/v1/subscription_service_pb";
import { UserService } from "@/types/proto-es/v1/user_service_pb";
import { WorkloadIdentityService } from "@/types/proto-es/v1/workload_identity_service_pb";
import { WorkspaceService } from "@/types/proto-es/v1/workspace_service_pb";
import { endpoints } from "./gen/openapi-index";

const endpointPaths = new Map(
  endpoints.map((endpoint) => [endpoint.operationId, endpoint.path])
);
const getEndpointPath = (operationId: string): string | undefined => {
  const fullId = operationId.includes("/")
    ? `bytebase.v1.${operationId.replace("/", ".")}`
    : operationId;
  return endpointPaths.get(fullId);
};

const services = [
  AIService,
  AccessGrantService,
  ActuatorService,
  AuditLogService,
  AuthService,
  CelService,
  ChangelogService,
  DatabaseCatalogService,
  DatabaseGroupService,
  DatabaseService,
  GroupService,
  IdentityProviderService,
  InstanceRoleService,
  InstanceService,
  IssueService,
  OrgPolicyService,
  PlanService,
  ProjectService,
  QueryHistoryService,
  ReleaseService,
  ReviewConfigService,
  RevisionService,
  RoleService,
  RolloutService,
  SavedQueryService,
  ServiceAccountService,
  SettingService,
  SheetService,
  SQLService,
  SubscriptionService,
  UserService,
  WorkloadIdentityService,
  WorkspaceService,
];

const methodClasses = new Map<string, MCPMethodClass>();
type ApiAuthorization =
  | { authMethod: "IAM"; permission: string }
  | { authMethod: "CUSTOM" };
const authorizations = new Map<string, ApiAuthorization>();
for (const service of services) {
  for (const method of service.methods) {
    const path = `/${service.typeName}/${method.name}`;
    const authMethod = getOption(method, auth_method);
    if (authMethod === AuthMethod.IAM) {
      authorizations.set(path, {
        authMethod: "IAM",
        permission: getOption(method, permission),
      });
    } else if (authMethod === AuthMethod.CUSTOM) {
      authorizations.set(path, { authMethod: "CUSTOM" });
    }
    if (!hasOption(method, mcp_method_class)) {
      continue;
    }
    const methodClass = getOption(method, mcp_method_class);
    if (methodClass !== MCPMethodClass.MCP_METHOD_CLASS_UNSPECIFIED) {
      methodClasses.set(path, methodClass);
    }
  }
}

export const getApiAuthorization = (
  operationId: string
): ApiAuthorization | undefined => {
  const path = getEndpointPath(operationId);
  return path ? authorizations.get(path) : undefined;
};

export const getApiMethodClass = (
  operationId: string
): MCPMethodClass | undefined => {
  const path = getEndpointPath(operationId);
  return path ? methodClasses.get(path) : undefined;
};

export const apiCallNeedsConfirmation = (operationId: string): boolean => {
  const path = getEndpointPath(operationId);
  return (
    !path ||
    path === "/bytebase.v1.SQLService/Query" ||
    methodClasses.get(path) !== MCPMethodClass.READ
  );
};
