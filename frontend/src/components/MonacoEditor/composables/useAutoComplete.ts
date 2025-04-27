import * as monaco from "monaco-editor";
import type { Ref } from "vue";
import { computed, watch } from "vue";
import { UNKNOWN_ID } from "@/types";
import {
  extractDatabaseResourceName,
  extractInstanceResourceName,
} from "@/utils";
import type { MonacoModule } from "../types";

export type AutoCompleteContextScene = "query" | "all";

export type AutoCompleteContext = {
  instance: string; // instances/{instance}
  database?: string; // instances/{instance}/databases/{database_name}
  schema?: string;
  scene?: AutoCompleteContextScene;
};

type SetMetadataParams = {
  instanceId: string; // instances/{instance}
  databaseName: string;
  schema?: string;
  scene?: AutoCompleteContextScene;
};

export const useAutoComplete = async (
  monaco: MonacoModule,
  editor: monaco.editor.IStandaloneCodeEditor,
  context: Ref<AutoCompleteContext | undefined>,
  readonly: Ref<boolean | undefined>
) => {
  const params = computed(() => {
    const p: SetMetadataParams = {
      instanceId: "",
      databaseName: "",
      scene: context.value?.scene,
    };
    const ctx = context.value;
    if (ctx) {
      const instance = extractInstanceResourceName(ctx.instance);
      if (instance && instance !== String(UNKNOWN_ID)) {
        p.instanceId = ctx.instance;
      }
      const { databaseName } = extractDatabaseResourceName(ctx.database ?? "");
      if (databaseName && databaseName !== String(UNKNOWN_ID)) {
        p.databaseName = databaseName;
      }
      if (ctx.schema !== undefined) {
        p.schema = ctx.schema;
      }
    }
    return p;
  });

  watch(
    [() => JSON.stringify(params.value), () => readonly.value],
    async () => {
      if (readonly.value) {
        return;
      }

      // Initialize LSP client if not already initialized.
      try {
        const { executeCommand, initializeLSPClient } = await import(
          "../lsp-client"
        );
        const client = await initializeLSPClient();
        const result = await executeCommand(client, "setMetadata", [
          params.value,
        ]);
        console.debug(
          `[MonacoEditor] setMetadata(${JSON.stringify(params.value)}): ${JSON.stringify(
            result
          )}`
        );
      } catch (err) {
        console.error("[MonacoEditor] Failed to initialize LSP client", err);
      }
    },
    { immediate: true }
  );
};
