import { ref, watchEffect } from "vue";
import type { ConnectionState } from "../lsp-client";

export const useLSPConnectionState = () => {
  const connectionState = ref<ConnectionState["state"]>();
  const connectionHeartbeat = ref<ConnectionState["heartbeat"]>();

  import("../lsp-client").then(
    ({ connectionState: state, connectionHeartbeat: heartbeat }) => {
      watchEffect(() => {
        connectionState.value = state.value;
      });
      watchEffect(() => {
        connectionHeartbeat.value = heartbeat.value;
      });
    }
  );

  return { connectionState, connectionHeartbeat };
};
