import type { ComputedRef, InjectionKey, Ref } from "vue";
import { computed, inject, provide, ref } from "vue";
import { Issue_Type } from "@/types/proto/v1/issue_service";
import type { CheckReleaseResponse_CheckResult } from "@/types/proto/v1/release_service";
import { useIssueContext } from "../../logic";

export type SQLCheckContext = {
  enabled: ComputedRef<boolean>;
  // Key is the database name.
  resultMap: Ref<Record<string, CheckReleaseResponse_CheckResult>>;

  upsertResult: (key: string, result: CheckReleaseResponse_CheckResult) => void;
};

export const KEY = Symbol(
  "bb.issue.context.sql-checks"
) as InjectionKey<SQLCheckContext>;

export const useIssueSQLCheckContext = () => {
  return inject(KEY)!;
};

export const provideIssueSQLCheckContext = () => {
  const { isCreating, issue } = useIssueContext();
  const resultMap = ref<Record<string, CheckReleaseResponse_CheckResult>>({});

  const enabled = computed(() => {
    return (
      isCreating.value &&
      [Issue_Type.DATABASE_CHANGE, Issue_Type.DATABASE_DATA_EXPORT].includes(
        issue.value.type
      )
    );
  });

  const context: SQLCheckContext = {
    enabled,
    resultMap,
    upsertResult: (key, result) => {
      resultMap.value[key] = result;
    },
  };

  provide(KEY, context);

  return context;
};
