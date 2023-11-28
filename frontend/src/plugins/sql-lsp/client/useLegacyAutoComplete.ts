import { computedAsync } from "@vueuse/core";
import type monaco from "monaco-editor";
import { Ref, computed, watch } from "vue";
import { useDBSchemaV1Store, useDatabaseV1Store } from "@/store";
import {
  SQLDialect,
  UNKNOWN_ID,
  dialectOfEngineV1,
  unknownDatabase,
} from "@/types";
import {
  DatabaseMetadata,
  DatabaseMetadataView,
} from "@/types/proto/v1/database_service";
import {
  Table,
  type ConnectionScope,
  type MonacoModule,
  type Schema,
} from "../types";
import { useLanguageClient } from "./useLanguageClient";

export type AutoCompleteContext = {
  instance: string; // instances/{instance}
  database?: string; // instances/{instance}/databases/{database_name}
};

export const useLegacyAutoComplete = (
  monaco: MonacoModule,
  editor: monaco.editor.IStandaloneCodeEditor,
  context: Ref<AutoCompleteContext | undefined>
) => {
  const { changeConnectionScope, changeDialect, changeSchema } =
    useLanguageClient();
  const database = computed(() => {
    if (!context.value || !context.value.database) {
      return unknownDatabase();
    }
    return useDatabaseV1Store().getDatabaseByName(context.value.database);
  });
  const dialect = computed((): SQLDialect => {
    if (database.value.uid === String(UNKNOWN_ID)) {
      return "MYSQL";
    }
    return dialectOfEngineV1(database.value.instanceEntity.engine);
  });
  const connectionScope = computed((): ConnectionScope => {
    if (database.value.uid !== String(UNKNOWN_ID)) {
      return "database";
    }
    return "instance";
  });
  const metadata = computedAsync(() => {
    if (database.value.uid === String(UNKNOWN_ID)) {
      return DatabaseMetadata.fromPartial({
        name: `${database.value.name}/metadata`,
      });
    }

    return useDBSchemaV1Store().getOrFetchDatabaseMetadata({
      database: database.value.name,
      skipCache: false,
      view: DatabaseMetadataView.DATABASE_METADATA_VIEW_BASIC,
    });
  }, DatabaseMetadata.fromPartial({ name: unknownDatabase().name }));

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
