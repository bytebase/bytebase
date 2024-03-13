import Emittery from "emittery";
import { InjectionKey, inject, provide, Ref, ref } from "vue";
import { useSQLEditorStore } from "@/store";
import { ComposedDatabase, SQLEditorTab } from "@/types";
import {
  DatabaseMetadata,
  ExternalTableMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";

type SQLEditorEvents = Emittery<{
  "save-sheet": { tab: SQLEditorTab; editTitle?: boolean };
  "alter-schema": {
    databaseUID: string;
    schema: string;
    table: string;
  };
  "format-content": undefined;
  "tree-ready": undefined;
  "project-context-ready": {
    project: string;
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

  maybeSwitchProject: (project: string) => Promise<string>;
};

export const KEY = Symbol(
  "bb.sql-editor.context"
) as InjectionKey<SQLEditorContext>;

export const useSQLEditorContext = () => {
  return inject(KEY)!;
};

export const provideSQLEditorContext = () => {
  const editorStore = useSQLEditorStore();
  const context: SQLEditorContext = {
    showAIChatBox: ref(false),
    selectedDatabaseSchemaByDatabaseName: ref(new Map()),
    events: new Emittery(),

    maybeSwitchProject: (project) => {
      if (editorStore.project !== project) {
        editorStore.project = project;
        return context.events.once("project-context-ready").then(() => project);
      }
      return Promise.resolve(editorStore.project);
    },
  };

  provide(KEY, context);

  return context;
};
