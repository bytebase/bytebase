import { type InjectionKey, inject, provide, type Ref } from "vue";

export type SQLResultViewContext = {
  dark: Ref<boolean>;
  disallowCopyingData: Ref<boolean>;
  detail: Ref<
    | {
        set: number; // The index of selected result set.
        row: number; // The row index of selected record.
        col: number; // The column index of selected cell.
      }
    | undefined
  >;
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
