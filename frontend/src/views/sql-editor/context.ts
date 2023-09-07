import Emittery from "emittery";
import { InjectionKey, inject, provide, Ref, ref } from "vue";

type SQLEditorEvents = Emittery<{
  // nothing by now
}>;

export type SQLEditorContext = {
  showAIChatBox: Ref<boolean>;

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
    events: new Emittery(),
  };

  provide(KEY, context);

  return context;
};
