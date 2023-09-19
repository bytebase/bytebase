import { type Ref, type InjectionKey, provide, inject } from "vue";

export type SQLResultViewContext = {
  dark: Ref<boolean>;
  disallowCopyingData: Ref<boolean>;
  keyword: Ref<string>;
  detail: Ref<{
    show: boolean;
    set: number; // The index of selected result set.
    row: number; // The row index of selected record.
    col: number; // The column index of selected cell.
  }>;
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
