import { grpc } from "@improbable-eng/grpc-web";
import { Channel, createChannel, createClientFactory } from "nice-grpc-web";
import { AuthServiceDefinition } from "@/types/proto/v1/auth_service";
import { IdentityProviderServiceDefinition } from "@/types/proto/v1/idp_service";
import { EnvironmentServiceDefinition } from "@/types/proto/v1/environment_service";

// Create each grpc service client.
// Reference: https://github.com/deeplay-io/nice-grpc/blob/master/packages/nice-grpc-web/README.md

let channelCache: Channel | undefined = undefined;

const getChannel = () => {
  if (channelCache) {
    return channelCache;
  }

  const address = import.meta.env.BB_GRPC_LOCAL
    ? "http://localhost:8080"
    : window.location.origin;
  const channel = createChannel(
    address,
    grpc.CrossBrowserHttpTransport({
      withCredentials: true,
    })
  );
  channelCache = channel;
  return channelCache;
};

const clientFactory = createClientFactory();

export const authServiceClient = () => {
  return clientFactory.create(AuthServiceDefinition, getChannel());
};

export const environmentServiceClient = () => {
  return clientFactory.create(EnvironmentServiceDefinition, getChannel());
};

export const identityProviderClient = () => {
  return clientFactory.create(IdentityProviderServiceDefinition, getChannel());
};

// e.g. How to use `authServiceClient`?
//
// await authServiceClient().login({
//   email: "bb@bytebase.com",
//   password: "bb",
//   web: true,
// });
// const { users } = await authServiceClient().listUsers({});
