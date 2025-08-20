import { debounce } from "lodash-es";
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

  // Debounce LSP metadata updates to reduce WebSocket requests
  const debouncedSetMetadata = debounce(async (params: SetMetadataParams) => {
    if (readonly.value) {
      return;
    }

    // Initialize LSP client if not already initialized.
    try {
      const { executeCommand, initializeLSPClient } = await import(
        "../lsp-client"
      );
      const client = await initializeLSPClient();
      const result = await executeCommand(client, "setMetadata", [params]);
      console.debug(
        `[MonacoEditor] setMetadata(${JSON.stringify(params)}): ${JSON.stringify(
          result
        )}`
      );
    } catch (err) {
      console.error("[MonacoEditor] Failed to initialize LSP client", err);
    }
  }, 500); // 500ms debounce to significantly reduce LSP requests

  watch(
    [() => JSON.stringify(params.value), () => readonly.value],
    () => {
      debouncedSetMetadata(params.value);
    },
    { immediate: true }
  );
};
