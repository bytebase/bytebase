import Emittery from "emittery";
import { InjectionKey, inject, provide, Ref, ref } from "vue";
import {
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";

type SQLEditorEvents = Emittery<{
  // nothing by now
}>;

export type SQLEditorContext = {
  showAIChatBox: Ref<boolean>;
  selectedDatabaseSchema: Ref<
    { schema: SchemaMetadata; table: TableMetadata } | undefined
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
    selectedDatabaseSchema: ref(),
    events: new Emittery(),
  };

  provide(KEY, context);

  return context;
};
