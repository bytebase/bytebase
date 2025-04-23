import { ActuatorServiceDefinition } from "@/types/proto/api/v1alpha/actuator_service";
import { AuditLogServiceDefinition } from "@/types/proto/api/v1alpha/audit_log_service";
import { AuthServiceDefinition } from "@/types/proto/api/v1alpha/auth_service";
import { CelServiceDefinition } from "@/types/proto/api/v1alpha/cel_service";
import { ChangelistServiceDefinition } from "@/types/proto/api/v1alpha/changelist_service";
import { DatabaseGroupServiceDefinition } from "@/types/proto/api/v1alpha/database_group_service";
import { DatabaseServiceDefinition } from "@/types/proto/api/v1alpha/database_service";
import { GroupServiceDefinition } from "@/types/proto/api/v1alpha/group_service";
import { IdentityProviderServiceDefinition } from "@/types/proto/api/v1alpha/idp_service";
import { InstanceServiceDefinition } from "@/types/proto/api/v1alpha/instance_service";
import { IssueServiceDefinition } from "@/types/proto/api/v1alpha/issue_service";
import { OrgPolicyServiceDefinition } from "@/types/proto/api/v1alpha/org_policy_service";
import { PlanServiceDefinition } from "@/types/proto/api/v1alpha/plan_service";
import { ProjectServiceDefinition } from "@/types/proto/api/v1alpha/project_service";
import { ReleaseServiceDefinition } from "@/types/proto/api/v1alpha/release_service";
import { ReviewConfigServiceDefinition } from "@/types/proto/api/v1alpha/review_config_service";
import { RiskServiceDefinition } from "@/types/proto/api/v1alpha/risk_service";
import { RoleServiceDefinition } from "@/types/proto/api/v1alpha/role_service";
import { RolloutServiceDefinition } from "@/types/proto/api/v1alpha/rollout_service";
import { SettingServiceDefinition } from "@/types/proto/api/v1alpha/setting_service";
import { SheetServiceDefinition } from "@/types/proto/api/v1alpha/sheet_service";
import { SQLServiceDefinition } from "@/types/proto/api/v1alpha/sql_service";
import { SubscriptionServiceDefinition } from "@/types/proto/api/v1alpha/subscription_service";
import { UserServiceDefinition } from "@/types/proto/api/v1alpha/user_service";
import { WorksheetServiceDefinition } from "@/types/proto/api/v1alpha/worksheet_service";
import { WorkspaceServiceDefinition } from "@/types/proto/api/v1alpha/workspace_service";

// The code of audit field in generated method options.
// A workaround code for the outdated ts-proto version in buf community plugins.
// This hack can be removed once the ts-proto version is upgraded.
// TODO(steven): remove me later.
const AUDIT_TAG_CODE = "800024";

// The methods that have audit field in their options.
// Format: /bytebase.v1.ServiceName/MethodName
export const ALL_METHODS_WITH_AUDIT = [
  ActuatorServiceDefinition,
  AuditLogServiceDefinition,
  AuthServiceDefinition,
  CelServiceDefinition,
  ChangelistServiceDefinition,
  DatabaseGroupServiceDefinition,
  DatabaseServiceDefinition,
  GroupServiceDefinition,
  IdentityProviderServiceDefinition,
  InstanceServiceDefinition,
  IssueServiceDefinition,
  OrgPolicyServiceDefinition,
  PlanServiceDefinition,
  ProjectServiceDefinition,
  ReleaseServiceDefinition,
  ReviewConfigServiceDefinition,
  RiskServiceDefinition,
  RoleServiceDefinition,
  RolloutServiceDefinition,
  SettingServiceDefinition,
  SheetServiceDefinition,
  SQLServiceDefinition,
  SubscriptionServiceDefinition,
  UserServiceDefinition,
  WorksheetServiceDefinition,
  WorkspaceServiceDefinition,
]
  .map((serviceDefinition) => {
    const methods: string[] = [];
    for (const method of Object.values(serviceDefinition.methods)) {
      const fullName = serviceDefinition.fullName;
      if (method.options._unknownFields?.[AUDIT_TAG_CODE]?.length > 0) {
        methods.push(`/${fullName}/${method.name}`);
      }
    }
    return methods;
  })
  .flat();
