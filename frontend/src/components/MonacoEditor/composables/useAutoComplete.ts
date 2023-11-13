import type monaco from "monaco-editor";
import { Ref, computed, watch } from "vue";
import { UNKNOWN_ID } from "@/types";
import { extractDatabaseResourceName } from "@/utils";
import { executeCommand, useLSPClient } from "../lsp-client";
import type { MonacoModule } from "../types";

export type AutoCompleteContext = {
  instance: string; // instances/{instance}
  database?: string; // instances/{instance}/databases/{database_name}
};

export const useAutoComplete = (
  monaco: MonacoModule,
  editor: monaco.editor.IStandaloneCodeEditor,
  context: Ref<AutoCompleteContext | undefined>
) => {
  const client = useLSPClient();
  const params = computed(() => {
    const p = {
      instanceId: context.value?.instance ?? "instances/-1",
      database: extractDatabaseResourceName(context.value?.database ?? "")
        .database,
    };
    if (p.database === String(UNKNOWN_ID)) {
      p.database = "";
    }
    return p;
  });
  watch(
    () => JSON.stringify(params.value),
    async () => {
      const result = executeCommand(client, "setMetadata", [params.value]);
      console.debug(
        `setMetadata(${JSON.stringify(params.value)}): ${JSON.stringify(
          result
        )}`
      );
    },
    { immediate: true }
  );
};
