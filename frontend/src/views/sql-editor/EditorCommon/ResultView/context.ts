import { type Ref, type InjectionKey, provide, inject } from "vue";

export type SQLResultViewContext = {
  dark: Ref<boolean>;
  disallowCopyingData: Ref<boolean>;
};

export const KEY = Symbol(
  "bb.sql-editor.result-view"
) as InjectionKey<SQLResultViewContext>;

export const provideSQLResultViewContext = (context: SQLResultViewContext) => {
  provide(KEY, context);
};

export const useSQLResultViewContext = () => {
  return inject(KEY)!;
};
