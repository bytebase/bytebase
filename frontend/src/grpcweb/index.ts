import { ActuatorServiceDefinition } from "@/types/proto/api/v1alpha/actuator_service";
import { AuditLogServiceDefinition } from "@/types/proto/api/v1alpha/audit_log_service";
import { AuthServiceDefinition } from "@/types/proto/api/v1alpha/auth_service";
import { CelServiceDefinition } from "@/types/proto/api/v1alpha/cel_service";
import { ChangelistServiceDefinition } from "@/types/proto/api/v1alpha/changelist_service";
import { DatabaseCatalogServiceDefinition } from "@/types/proto/api/v1alpha/database_catalog_service";
import { DatabaseGroupServiceDefinition } from "@/types/proto/api/v1alpha/database_group_service";
import { DatabaseServiceDefinition } from "@/types/proto/api/v1alpha/database_service";
import { GroupServiceDefinition } from "@/types/proto/api/v1alpha/group_service";
import { IdentityProviderServiceDefinition } from "@/types/proto/api/v1alpha/idp_service";
import { InstanceRoleServiceDefinition } from "@/types/proto/api/v1alpha/instance_role_service";
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
import { errorDetailsClientMiddleware } from "nice-grpc-error-details";
import {
  createChannel,
  createClientFactory,
  FetchTransport,
  WebsocketTransport,
} from "nice-grpc-web";
import {
  authInterceptorMiddleware,
  errorNotificationMiddleware,
  simulateLatencyMiddleware,
} from "./middlewares";

// Create each grpc service client.
// Reference: https://github.com/deeplay-io/nice-grpc/blob/master/packages/nice-grpc-web/README.md

const address = import.meta.env.BB_GRPC_LOCAL || window.location.origin;

const channel = createChannel(
  address,
  FetchTransport({
    credentials: "include",
  })
);
const websocketChannel = createChannel(
  window.location.origin,
  WebsocketTransport()
);

const clientFactory = createClientFactory()
  // A middleware that is attached first, will be invoked last.
  .use(authInterceptorMiddleware)
  .use(errorDetailsClientMiddleware)
  .use(errorNotificationMiddleware)
  .use(simulateLatencyMiddleware);
/**
 * Example to use error notification middleware.
 * Errors occurs during all requests will cause UI notifications automatically.
 * abcServiceClient.foo(requestParams, {
 *   // true if you want to suppress error notifications for this call
 *   silent: true,
 * })
 */

export const authServiceClient = clientFactory.create(
  AuthServiceDefinition,
  channel
);

export const userServiceClient = clientFactory.create(
  UserServiceDefinition,
  channel
);

export const roleServiceClient = clientFactory.create(
  RoleServiceDefinition,
  channel
);

export const instanceServiceClient = clientFactory.create(
  InstanceServiceDefinition,
  channel
);

export const policyServiceClient = clientFactory.create(
  OrgPolicyServiceDefinition,
  channel
);

export const projectServiceClient = clientFactory.create(
  ProjectServiceDefinition,
  channel
);

export const databaseServiceClient = clientFactory.create(
  DatabaseServiceDefinition,
  channel
);

export const databaseCatalogServiceClient = clientFactory.create(
  DatabaseCatalogServiceDefinition,
  channel
);

export const databaseGroupServiceClient = clientFactory.create(
  DatabaseGroupServiceDefinition,
  channel
);

export const identityProviderClient = clientFactory.create(
  IdentityProviderServiceDefinition,
  channel
);

export const riskServiceClient = clientFactory.create(
  RiskServiceDefinition,
  channel
);

export const settingServiceClient = clientFactory.create(
  SettingServiceDefinition,
  channel
);

export const sheetServiceClient = clientFactory.create(
  SheetServiceDefinition,
  channel
);

export const worksheetServiceClient = clientFactory.create(
  WorksheetServiceDefinition,
  channel
);

export const issueServiceClient = clientFactory.create(
  IssueServiceDefinition,
  channel
);

export const rolloutServiceClient = clientFactory.create(
  RolloutServiceDefinition,
  channel
);

export const planServiceClient = clientFactory.create(
  PlanServiceDefinition,
  channel
);

export const sqlServiceClient = clientFactory.create(
  SQLServiceDefinition,
  channel
);

export const sqlStreamingServiceClient = clientFactory.create(
  SQLServiceDefinition,
  websocketChannel
);

export const celServiceClient = clientFactory.create(
  CelServiceDefinition,
  channel
);

export const subscriptionServiceClient = clientFactory.create(
  SubscriptionServiceDefinition,
  channel
);

export const actuatorServiceClient = clientFactory.create(
  ActuatorServiceDefinition,
  channel
);

export const changelistServiceClient = clientFactory.create(
  ChangelistServiceDefinition,
  channel
);

export const auditLogServiceClient = clientFactory.create(
  AuditLogServiceDefinition,
  channel
);

export const groupServiceClient = clientFactory.create(
  GroupServiceDefinition,
  channel
);

export const reviewConfigServiceClient = clientFactory.create(
  ReviewConfigServiceDefinition,
  channel
);

export const workspaceServiceClient = clientFactory.create(
  WorkspaceServiceDefinition,
  channel
);

export const releaseServiceClient = clientFactory.create(
  ReleaseServiceDefinition,
  channel
);

export const instanceRoleServiceClient = clientFactory.create(
  InstanceRoleServiceDefinition,
  channel
);

// e.g. How to use `authServiceClient`?
//
// await authServiceClient.login({
//   email: "bb@bytebase.com",
//   password: "bb",
//   web: true,
// });
// const { users } = await authServiceClient.listUsers({});
