import {
  createChannel,
  createClientFactory,
  FetchTransport,
} from "nice-grpc-web";
import { authInterceptorMiddleware } from "./middlewares";
import { AuthServiceDefinition } from "@/types/proto/v1/auth_service";
import { RoleServiceDefinition } from "@/types/proto/v1/role_service";
import { IdentityProviderServiceDefinition } from "@/types/proto/v1/idp_service";
import { EnvironmentServiceDefinition } from "@/types/proto/v1/environment_service";
import { InstanceServiceDefinition } from "@/types/proto/v1/instance_service";
import { OrgPolicyServiceDefinition } from "@/types/proto/v1/org_policy_service";
import { ProjectServiceDefinition } from "@/types/proto/v1/project_service";
import { SQLServiceDefinition } from "@/types/proto/v1/sql_service";
import { RiskServiceDefinition } from "@/types/proto/v1/risk_service";
import { SettingServiceDefinition } from "@/types/proto/v1/setting_service";
import { ReviewServiceDefinition } from "@/types/proto/v1/review_service";
import { DatabaseServiceDefinition } from "@/types/proto/v1/database_service";
import { SheetServiceDefinition } from "@/types/proto/v1/sheet_service";
import { InstanceRoleServiceDefinition } from "@/types/proto/v1/instance_role_service";
import { CelServiceDefinition } from "@/types/proto/v1/cel_service";
import { SubscriptionServiceDefinition } from "@/types/proto/v1/subscription_service";
import { ActuatorServiceDefinition } from "@/types/proto/v1/actuator_service";
import { ExternalVersionControlServiceDefinition } from "@/types/proto/v1/externalvs_service";

// Create each grpc service client.
// Reference: https://github.com/deeplay-io/nice-grpc/blob/master/packages/nice-grpc-web/README.md

const address = import.meta.env.BB_GRPC_LOCAL
  ? "http://localhost:8080"
  : window.location.origin;

const channel = createChannel(
  address,
  FetchTransport({
    credentials: "include",
  })
);

const clientFactory = createClientFactory().use(authInterceptorMiddleware);

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

export const instanceRoleServiceClient = clientFactory.create(
  InstanceRoleServiceDefinition,
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

export const reviewServiceClient = clientFactory.create(
  ReviewServiceDefinition,
  channel
);

export const sqlServiceClient = clientFactory.create(
  SQLServiceDefinition,
  channel
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

export const externalVersionControlServiceClient = clientFactory.create(
  ExternalVersionControlServiceDefinition,
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
