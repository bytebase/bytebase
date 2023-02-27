import { grpc } from "@improbable-eng/grpc-web";
import { createChannel, createClientFactory } from "nice-grpc-web";
import { AuthServiceDefinition } from "@/types/proto/v1/auth_service";
import { IdentityProviderServiceDefinition } from "@/types/proto/v1/idp_service";
import { EnvironmentServiceDefinition } from "@/types/proto/v1/environment_service";
import { InstanceServiceDefinition } from "@/types/proto/v1/instance_service";

// Create each grpc service client.
// Reference: https://github.com/deeplay-io/nice-grpc/blob/master/packages/nice-grpc-web/README.md

const address = import.meta.env.BB_GRPC_LOCAL
  ? "http://localhost:8080"
  : window.location.origin;

const channel = createChannel(
  address,
  grpc.CrossBrowserHttpTransport({
    withCredentials: true,
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

export const identityProviderClient = clientFactory.create(
  IdentityProviderServiceDefinition,
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
