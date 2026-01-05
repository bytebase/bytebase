import { getOption } from "@bufbuild/protobuf";
import { ActuatorService } from "@/types/proto-es/v1/actuator_service_pb";
import { audit } from "@/types/proto-es/v1/annotation_pb";
import { AuditLogService } from "@/types/proto-es/v1/audit_log_service_pb";
import { AuthService } from "@/types/proto-es/v1/auth_service_pb";
import { CelService } from "@/types/proto-es/v1/cel_service_pb";
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
import { ReleaseService } from "@/types/proto-es/v1/release_service_pb";
import { ReviewConfigService } from "@/types/proto-es/v1/review_config_service_pb";
import { RevisionService } from "@/types/proto-es/v1/revision_service_pb";
import { RoleService } from "@/types/proto-es/v1/role_service_pb";
import { RolloutService } from "@/types/proto-es/v1/rollout_service_pb";
import { SettingService } from "@/types/proto-es/v1/setting_service_pb";
import { SheetService } from "@/types/proto-es/v1/sheet_service_pb";
import { SQLService } from "@/types/proto-es/v1/sql_service_pb";
import { SubscriptionService } from "@/types/proto-es/v1/subscription_service_pb";
import { UserService } from "@/types/proto-es/v1/user_service_pb";
import { WorksheetService } from "@/types/proto-es/v1/worksheet_service_pb";
import { WorkspaceService } from "@/types/proto-es/v1/workspace_service_pb";

// The methods that have audit field in their options.
// Format: /bytebase.v1.ServiceName/MethodName
export const ALL_METHODS_WITH_AUDIT = [
  ActuatorService,
  AuditLogService,
  AuthService,
  CelService,
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
  ReleaseService,
  RevisionService,
  ReviewConfigService,
  RoleService,
  RolloutService,
  SettingService,
  SheetService,
  SQLService,
  SubscriptionService,
  UserService,
  WorksheetService,
  WorkspaceService,
]
  .reduce((list, service) => {
    for (const method of service.methods) {
      const auditOption = getOption(method, audit);
      if (auditOption) {
        list.push(`/${service.typeName}/${method.name}`);
      }
    }
    return list;
  }, [] as string[])
  .sort();
