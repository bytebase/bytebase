import { errorDetailsClientMiddleware } from "nice-grpc-error-details";
import {
  createChannel,
  createClientFactory,
  FetchTransport,
  WebsocketTransport,
} from "nice-grpc-web";
import { ActuatorServiceDefinition } from "@/types/proto/v1/actuator_service";
import { AnomalyServiceDefinition } from "@/types/proto/v1/anomaly_service";
import { AuditLogServiceDefinition } from "@/types/proto/v1/audit_log_service";
import { AuthServiceDefinition } from "@/types/proto/v1/auth_service";
import { BranchServiceDefinition } from "@/types/proto/v1/branch_service";
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
import { VCSConnectorServiceDefinition } from "@/types/proto/v1/vcs_connector_service";
import { VCSProviderServiceDefinition } from "@/types/proto/v1/vcs_provider_service";
import { WorksheetServiceDefinition } from "@/types/proto/v1/worksheet_service";
import { WorkspaceServiceDefinition } from "@/types/proto/v1/workspace_service";
import {
  authInterceptorMiddleware,
  errorNotificationMiddleware,
  simulateLatencyMiddleware,
} from "./middlewares";

// Create each grpc service client.
// Reference: https://github.com/deeplay-io/nice-grpc/blob/master/packages/nice-grpc-web/README.md

// const address = import.meta.env.BB_GRPC_LOCAL || window.location.origin;
const address = `${location.protocol}//${location.hostname}:8080`;

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

export const roleServiceClient = clientFactory.create(
  RoleServiceDefinition,
  channel
);

export const environmentServiceClient = clientFactory.create(
  EnvironmentServiceDefinition,
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

export const vcsProviderServiceClient = clientFactory.create(
  VCSProviderServiceDefinition,
  channel
);

export const vcsConnectorServiceClient = clientFactory.create(
  VCSConnectorServiceDefinition,
  channel
);

export const anomalyServiceClient = clientFactory.create(
  AnomalyServiceDefinition,
  channel
);

export const branchServiceClient = clientFactory.create(
  BranchServiceDefinition,
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

// e.g. How to use `authServiceClient`?
//
// await authServiceClient.login({
//   email: "bb@bytebase.com",
//   password: "bb",
//   web: true,
// });
// const { users } = await authServiceClient.listUsers({});
