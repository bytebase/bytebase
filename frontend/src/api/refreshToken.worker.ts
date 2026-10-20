import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { AuthService } from "@/types/proto-es/v1/auth_service_pb";

const client = createClient(
  AuthService,
  createConnectTransport({
    baseUrl: import.meta.env.BB_GRPC_LOCAL || globalThis.location.origin,
    fetch: (input, init) => fetch(input, { ...init, credentials: "include" }),
  })
);

let pending: Promise<void> | undefined;

const scope = globalThis as unknown as {
  onconnect: (event: MessageEvent) => void;
};

scope.onconnect = (event) => {
  const port = event.ports[0];
  port.onmessage = async () => {
    // The worker owns the network request, so closing the initiating tab
    // does not hand an in-flight refresh to another tab.
    pending ??= client
      .refresh({}, { timeoutMs: 30000 })
      .then(() => {})
      .finally(() => {
        pending = undefined;
      });
    let result: { error?: string } = {};
    try {
      await pending;
    } catch (error) {
      result = {
        error:
          error instanceof Error ? error.message : "Session refresh failed",
      };
    }
    port.postMessage(result);
  };
};
