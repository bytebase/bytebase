import type { InjectionKey, Ref } from "vue";
import { inject, provide, ref } from "vue";

export type SQLCheckContext = {
  runSQLCheck: Ref<(() => Promise<boolean>) | undefined>;
};

export const KEY = Symbol(
  "bb.sql-check.context"
) as InjectionKey<SQLCheckContext>;

export const useSQLCheckContext = () => {
  return inject(KEY)!;
};

export const provideSQLCheckContext = () => {
  const context: SQLCheckContext = {
    runSQLCheck: ref(),
  };

  provide(KEY, context);

  return context;
};
