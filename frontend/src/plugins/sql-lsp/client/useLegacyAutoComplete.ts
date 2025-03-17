import { computedAsync } from "@vueuse/core";
import type monaco from "monaco-editor";
import type { Ref } from "vue";
import { computed, watch } from "vue";
import { useDBSchemaV1Store, useDatabaseV1Store } from "@/store";
import type { SQLDialect } from "@/types";
import {
  dialectOfEngineV1,
  isValidDatabaseName,
  unknownDatabase,
} from "@/types";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import type { Table } from "../types";
import { type ConnectionScope, type MonacoModule, type Schema } from "../types";

export type AutoCompleteContext = {
  instance: string; // instances/{instance}
  database?: string; // instances/{instance}/databases/{database_name}
};

export const useLegacyAutoComplete = async (
  monaco: MonacoModule,
  editor: monaco.editor.IStandaloneCodeEditor,
  context: Ref<AutoCompleteContext | undefined>
) => {
  const { useLanguageClient } = await import("./useLanguageClient");

  const { changeConnectionScope, changeDialect, changeSchema } =
    useLanguageClient();
  const database = computed(() => {
    if (!context.value || !context.value.database) {
      return unknownDatabase();
    }
    return useDatabaseV1Store().getDatabaseByName(context.value.database);
  });
  const dialect = computed((): SQLDialect => {
    if (!isValidDatabaseName(database.value.name)) {
      return "MYSQL";
    }
    return dialectOfEngineV1(database.value.instanceResource.engine);
  });
  const connectionScope = computed((): ConnectionScope => {
    if (isValidDatabaseName(database.value.name)) {
      return "database";
    }
    return "instance";
  });
  const metadata = computedAsync(
    () => {
      if (!isValidDatabaseName(database.value.name)) {
        return DatabaseMetadata.fromPartial({
          name: `${database.value.name}/metadata`,
        });
      }

      return useDBSchemaV1Store().getOrFetchDatabaseMetadata({
        database: database.value.name,
        skipCache: false,
      });
    },
    DatabaseMetadata.fromPartial({ name: unknownDatabase().name })
  );

  watch(
    dialect,
    (dialect) => {
      changeDialect(dialect);
    },
    { immediate: true }
  );
  watch(
    connectionScope,
    (connectionScope) => {
      changeConnectionScope(connectionScope);
    },
    { immediate: true }
  );
  watch(
    [metadata, database],
    ([metadata, database]) => {
      if (`${database.name}/metadata` !== metadata.name) {
        return;
      }
      const schema: Schema = {
        databases: [
          {
            name: database.databaseName,
            tables: metadata.schemas.flatMap((s) =>
              s.tables.map<Table>((t) => ({
                database: database.databaseName,
                name: nameForTable(t.name, s.name),
                columns: t.columns.map((c) => ({
                  name: c.name,
                })),
              }))
            ),
          },
        ],
      };
      changeSchema(schema);
    },
    { immediate: true }
  );
};

const nameForTable = (table: string, schema: string = "") => {
  if (schema) return `${schema}.${table}`;
  return table;
};
