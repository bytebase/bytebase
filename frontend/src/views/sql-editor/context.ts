import Emittery from "emittery";
import { InjectionKey, inject, provide, Ref, ref } from "vue";
import { ComposedDatabase } from "@/types";
import {
  DatabaseMetadata,
  ExternalTableMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";

type SQLEditorEvents = Emittery<{
  "save-sheet": { title: string };
  "alter-schema": {
    databaseUID: string;
    schema: string;
    table: string;
  };
}>;

export type SelectedDatabaseSchema = {
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table?: TableMetadata;
  externalTable?: ExternalTableMetadata;
};

export type SQLEditorContext = {
  showAIChatBox: Ref<boolean>;
  selectedDatabaseSchemaByDatabaseName: Ref<
    Map<string, SelectedDatabaseSchema>
  >;

  events: SQLEditorEvents;
};

export const KEY = Symbol(
  "bb.sql-editor.context"
) as InjectionKey<SQLEditorContext>;

export const useSQLEditorContext = () => {
  return inject(KEY)!;
};

export const provideSQLEditorContext = () => {
  const context: SQLEditorContext = {
    showAIChatBox: ref(false),
    selectedDatabaseSchemaByDatabaseName: ref(new Map()),
    events: new Emittery(),
  };

  provide(KEY, context);

  return context;
};
