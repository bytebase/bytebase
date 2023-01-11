import { grpc } from "@improbable-eng/grpc-web";
import { createChannel, createClientFactory } from "nice-grpc-web";
import { AuthServiceDefinition } from "@/types/proto/v1/auth_service";

// Create each grpc service client.
// Reference: https://github.com/deeplay-io/nice-grpc/blob/master/packages/nice-grpc-web/README.md

const channel = createChannel(
  "http://localhost:8080",
  grpc.CrossBrowserHttpTransport({
    withCredentials: true,
  })
);

const clientFactory = createClientFactory();

export const authServiceClient = clientFactory.create(
  AuthServiceDefinition,
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
