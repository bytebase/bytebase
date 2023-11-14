import type monaco from "monaco-editor";
import { Ref, computed, watch } from "vue";
import { UNKNOWN_ID } from "@/types";
import {
  extractDatabaseResourceName,
  extractInstanceResourceName,
} from "@/utils";
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
      instanceId: "",
      database: "",
    };
    const ctx = context.value;
    if (ctx) {
      const instance = extractInstanceResourceName(ctx.instance);
      if (instance && instance !== String(UNKNOWN_ID)) {
        p.instanceId = ctx.instance;
      }
      const database = extractDatabaseResourceName(ctx.database ?? "").database;
      if (database && database !== String(UNKNOWN_ID)) {
        p.database = database;
      }
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
