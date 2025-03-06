import { ActuatorServiceDefinition } from "@/types/proto/v1/actuator_service";
import { AnomalyServiceDefinition } from "@/types/proto/v1/anomaly_service";
import { AuditLogServiceDefinition } from "@/types/proto/v1/audit_log_service";
import { AuthServiceDefinition } from "@/types/proto/v1/auth_service";
import { CelServiceDefinition } from "@/types/proto/v1/cel_service";
import { ChangelistServiceDefinition } from "@/types/proto/v1/changelist_service";
import { DatabaseGroupServiceDefinition } from "@/types/proto/v1/database_group_service";
import { DatabaseServiceDefinition } from "@/types/proto/v1/database_service";
import { EnvironmentServiceDefinition } from "@/types/proto/v1/environment_service";
import { GroupServiceDefinition } from "@/types/proto/v1/group_service";
import { IdentityProviderServiceDefinition } from "@/types/proto/v1/idp_service";
import { InstanceServiceDefinition } from "@/types/proto/v1/instance_service";
import { IssueServiceDefinition } from "@/types/proto/v1/issue_service";
import { OrgPolicyServiceDefinition } from "@/types/proto/v1/org_policy_service";
import { PlanServiceDefinition } from "@/types/proto/v1/plan_service";
import { ProjectServiceDefinition } from "@/types/proto/v1/project_service";
import { ReleaseServiceDefinition } from "@/types/proto/v1/release_service";
import { ReviewConfigServiceDefinition } from "@/types/proto/v1/review_config_service";
import { RiskServiceDefinition } from "@/types/proto/v1/risk_service";
import { RoleServiceDefinition } from "@/types/proto/v1/role_service";
import { RolloutServiceDefinition } from "@/types/proto/v1/rollout_service";
import { SettingServiceDefinition } from "@/types/proto/v1/setting_service";
import { SheetServiceDefinition } from "@/types/proto/v1/sheet_service";
import { SQLServiceDefinition } from "@/types/proto/v1/sql_service";
import { SubscriptionServiceDefinition } from "@/types/proto/v1/subscription_service";
import { UserServiceDefinition } from "@/types/proto/v1/user_service";
import { WorksheetServiceDefinition } from "@/types/proto/v1/worksheet_service";
import { WorkspaceServiceDefinition } from "@/types/proto/v1/workspace_service";

// The code of audit field in generated method options.
// A workaround code for the outdated ts-proto version in buf community plugins.
// This hack can be removed once the ts-proto version is upgraded.
// TODO(steven): remove me later.
const AUDIT_TAG_CODE = "800024";

// The methods that have audit field in their options.
// Format: /bytebase.v1.ServiceName/MethodName
export const ALL_METHODS_WITH_AUDIT = [
  ActuatorServiceDefinition,
  AnomalyServiceDefinition,
  AuditLogServiceDefinition,
  AuthServiceDefinition,
  CelServiceDefinition,
  ChangelistServiceDefinition,
  DatabaseGroupServiceDefinition,
  DatabaseServiceDefinition,
  EnvironmentServiceDefinition,
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
