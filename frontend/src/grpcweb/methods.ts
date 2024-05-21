import { ActuatorServiceDefinition } from "@/types/proto/v1/actuator_service";
import { AnomalyServiceDefinition } from "@/types/proto/v1/anomaly_service";
import { AuditLogServiceDefinition } from "@/types/proto/v1/audit_log_service";
import { AuthServiceDefinition } from "@/types/proto/v1/auth_service";
import { BranchServiceDefinition } from "@/types/proto/v1/branch_service";
import { CelServiceDefinition } from "@/types/proto/v1/cel_service";
import { ChangelistServiceDefinition } from "@/types/proto/v1/changelist_service";
import { DatabaseServiceDefinition } from "@/types/proto/v1/database_service";
import { EnvironmentServiceDefinition } from "@/types/proto/v1/environment_service";
import { IdentityProviderServiceDefinition } from "@/types/proto/v1/idp_service";
import { InstanceRoleServiceDefinition } from "@/types/proto/v1/instance_role_service";
import { InstanceServiceDefinition } from "@/types/proto/v1/instance_service";
import { IssueServiceDefinition } from "@/types/proto/v1/issue_service";
import { OrgPolicyServiceDefinition } from "@/types/proto/v1/org_policy_service";
import { ProjectServiceDefinition } from "@/types/proto/v1/project_service";
import { RiskServiceDefinition } from "@/types/proto/v1/risk_service";
import { RoleServiceDefinition } from "@/types/proto/v1/role_service";
import { RolloutServiceDefinition } from "@/types/proto/v1/rollout_service";
import { SettingServiceDefinition } from "@/types/proto/v1/setting_service";
import { SheetServiceDefinition } from "@/types/proto/v1/sheet_service";
import { SQLServiceDefinition } from "@/types/proto/v1/sql_service";
import { SubscriptionServiceDefinition } from "@/types/proto/v1/subscription_service";
import { VCSConnectorServiceDefinition } from "@/types/proto/v1/vcs_connector_service";
import { VCSProviderServiceDefinition } from "@/types/proto/v1/vcs_provider_service";
import { WorksheetServiceDefinition } from "@/types/proto/v1/worksheet_service";

const ALL_METHODS = [
  ActuatorServiceDefinition,
  AnomalyServiceDefinition,
  AuditLogServiceDefinition,
  AuthServiceDefinition,
  BranchServiceDefinition,
  CelServiceDefinition,
  ChangelistServiceDefinition,
  DatabaseServiceDefinition,
  EnvironmentServiceDefinition,
  IdentityProviderServiceDefinition,
  InstanceRoleServiceDefinition,
  InstanceServiceDefinition,
  IssueServiceDefinition,
  OrgPolicyServiceDefinition,
  ProjectServiceDefinition,
  RiskServiceDefinition,
  RoleServiceDefinition,
  RolloutServiceDefinition,
  SettingServiceDefinition,
  SheetServiceDefinition,
  SQLServiceDefinition,
  SubscriptionServiceDefinition,
  VCSConnectorServiceDefinition,
  VCSProviderServiceDefinition,
  WorksheetServiceDefinition,
]
  .map((serviceDefinition) => {
    const methods: string[] = [];
    for (const method of Object.values(serviceDefinition.methods)) {
      const fullName = serviceDefinition.fullName;
      methods.push(`/${fullName}/${method.name}`);
    }
    return methods;
  })
  .flat();

export default ALL_METHODS;
