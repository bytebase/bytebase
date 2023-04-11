import {
  createChannel,
  createClientFactory,
  FetchTransport,
} from "nice-grpc-web";
import { AuthServiceDefinition } from "@/types/proto/v1/auth_service";
import { IdentityProviderServiceDefinition } from "@/types/proto/v1/idp_service";
import { EnvironmentServiceDefinition } from "@/types/proto/v1/environment_service";
import { InstanceServiceDefinition } from "@/types/proto/v1/instance_service";
import { ProjectServiceDefinition } from "@/types/proto/v1/project_service";
import { SQLServiceDefinition } from "@/types/proto/v1/sql_service";
import { RiskServiceDefinition } from "@/types/proto/v1/risk_service";
import { SettingServiceDefinition } from "@/types/proto/v1/setting_service";
import { ReviewServiceDefinition } from "@/types/proto/v1/review_service";
import { DatabaseServiceDefinition } from "@/types/proto/v1/database_service";

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

const clientFactory = createClientFactory();

export const authServiceClient = clientFactory.create(
  AuthServiceDefinition,
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

export const reviewServiceClient = clientFactory.create(
  ReviewServiceDefinition,
  channel
);

export const sqlClient = clientFactory.create(SQLServiceDefinition, channel);

// e.g. How to use `authServiceClient`?
//
// await authServiceClient.login({
//   email: "bb@bytebase.com",
//   password: "bb",
//   web: true,
// });
// const { users } = await authServiceClient.listUsers({});
