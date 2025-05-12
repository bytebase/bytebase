import type { InjectionKey, Ref } from "vue";
import { inject, provide, ref } from "vue";
import type { CheckReleaseResponse_CheckResult } from "@/types/proto/v1/release_service";

export type SQLCheckContext = {
  // Key is the database name.
  resultMap: Ref<Record<string, CheckReleaseResponse_CheckResult>>;

  upsertResult: (key: string, result: CheckReleaseResponse_CheckResult) => void;
};

export const KEY = Symbol(
  "bb.plan.context.sql-checks"
) as InjectionKey<SQLCheckContext>;

export const usePlanSQLCheckContext = () => {
  return inject(KEY)!;
};

export const providePlanSQLCheckContext = () => {
  const resultMap = ref<Record<string, CheckReleaseResponse_CheckResult>>({});

  const context: SQLCheckContext = {
    resultMap,
    upsertResult: (key, result) => {
      resultMap.value[key] = result;
    },
  };

  provide(KEY, context);

  return context;
};
