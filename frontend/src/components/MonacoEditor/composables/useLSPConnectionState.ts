import { ref, watchEffect } from "vue";
import type { ConnectionState } from "../lsp-client";

export const useLSPConnectionState = () => {
  const connectionState = ref<ConnectionState["state"]>();

  import("../lsp-client").then(({ connectionState: state }) => {
    watchEffect(() => {
      connectionState.value = state.value;
    });
  });

  return { connectionState };
};
