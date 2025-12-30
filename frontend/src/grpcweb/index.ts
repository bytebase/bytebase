import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { ActuatorService } from "@/types/proto-es/v1/actuator_service_pb";
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
import {
  authInterceptor,
  errorNotificationInterceptor,
  activeInterceptor,
} from "./middlewares";

const address = import.meta.env.BB_GRPC_LOCAL || window.location.origin;

const transport = createConnectTransport({
  baseUrl: address,
  useBinaryFormat: true,
  interceptors: [
    authInterceptor,
    activeInterceptor,
    errorNotificationInterceptor,
  ],
  fetch: (input, init) => fetch(input, { ...init, credentials: "include" }),
});

export const actuatorServiceClientConnect = createClient(
  ActuatorService,
  transport
);

export const authServiceClientConnect = createClient(AuthService, transport);

export const auditLogServiceClientConnect = createClient(
  AuditLogService,
  transport
);

export const subscriptionServiceClientConnect = createClient(
  SubscriptionService,
  transport
);

export const workspaceServiceClientConnect = createClient(
  WorkspaceService,
  transport
);

export const settingServiceClientConnect = createClient(
  SettingService,
  transport
);

export const celServiceClientConnect = createClient(CelService, transport);

export const databaseCatalogServiceClientConnect = createClient(
  DatabaseCatalogService,
  transport
);

export const instanceRoleServiceClientConnect = createClient(
  InstanceRoleService,
  transport
);

export const instanceServiceClientConnect = createClient(
  InstanceService,
  transport
);

export const roleServiceClientConnect = createClient(RoleService, transport);

export const groupServiceClientConnect = createClient(GroupService, transport);

export const databaseGroupServiceClientConnect = createClient(
  DatabaseGroupService,
  transport
);

export const orgPolicyServiceClientConnect = createClient(
  OrgPolicyService,
  transport
);

export const reviewConfigServiceClientConnect = createClient(
  ReviewConfigService,
  transport
);

export const revisionServiceClientConnect = createClient(
  RevisionService,
  transport
);

export const identityProviderServiceClientConnect = createClient(
  IdentityProviderService,
  transport
);

export const issueServiceClientConnect = createClient(IssueService, transport);

export const sheetServiceClientConnect = createClient(SheetService, transport);

export const userServiceClientConnect = createClient(UserService, transport);

export const releaseServiceClientConnect = createClient(
  ReleaseService,
  transport
);

export const worksheetServiceClientConnect = createClient(
  WorksheetService,
  transport
);

export const sqlServiceClientConnect = createClient(SQLService, transport);

export const planServiceClientConnect = createClient(PlanService, transport);

export const projectServiceClientConnect = createClient(
  ProjectService,
  transport
);

export const rolloutServiceClientConnect = createClient(
  RolloutService,
  transport
);

export const databaseServiceClientConnect = createClient(
  DatabaseService,
  transport
);
