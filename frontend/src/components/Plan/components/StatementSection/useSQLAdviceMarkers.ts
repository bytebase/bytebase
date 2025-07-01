import { maxBy } from "lodash-es";
import { computed, type Ref } from "vue";
import type { AdviceOption } from "@/components/MonacoEditor";
import {
  PlanCheckRun,
  PlanCheckRun_Result_Status,
  PlanCheckRun_Type,
} from "@/types/proto/v1/plan_service";
import { Advice_Status, type Advice } from "@/types/proto-es/v1/sql_service_pb";
import { extractPlanCheckRunUID } from "@/utils";

export const useSQLAdviceMarkers = (
  isCreating: Ref<boolean>,
  planCheckRuns?: Ref<PlanCheckRun[]>,
  advices?: Ref<Advice[] | undefined>
) => {
  const markers = computed(() => {
    if (isCreating.value) {
      if (!advices) return [];
      if (!advices.value) return [];
      return advices.value.map<AdviceOption>((advice) => {
        const line = advice.startPosition?.line ?? 0;
        const column = advice.startPosition?.column ?? Number.MAX_SAFE_INTEGER;
        const code = advice.code;
        return {
          severity: advice.status === Advice_Status.ERROR ? "ERROR" : "WARNING",
          message: advice.content,
          source: `${advice.title} (${code}) L${line}:C${column}`,
          startLineNumber: line,
          endLineNumber: line,
          startColumn: column,
          endColumn: column,
        };
      });
    } else {
      if (!planCheckRuns) return [];
      if (!planCheckRuns.value) return [];
      const types: PlanCheckRun_Type[] = [
        PlanCheckRun_Type.DATABASE_STATEMENT_ADVISE,
      ];
      return types.flatMap((type) => {
        return getLatestAdviceOptions(
          planCheckRuns.value.filter((checkRun) => checkRun.type === type)
        );
      });
    }
  });
  return { markers };
};

const getLatestAdviceOptions = (planCheckRuns: PlanCheckRun[]) => {
  const latest = maxBy(planCheckRuns, (checkRun) =>
    parseInt(extractPlanCheckRunUID(checkRun.name), 10)
  );
  if (!latest) {
    return [];
  }
  const resultList = latest.results;
  return resultList
    .filter(
      (result) =>
        result.status === PlanCheckRun_Result_Status.ERROR ||
        result.status === PlanCheckRun_Result_Status.WARNING
    )
    .filter((result) => result.sqlReviewReport?.line !== undefined)
    .map<AdviceOption>((result) => {
      const line = result.sqlReviewReport!.line;
      const column = result.sqlReviewReport?.column ?? Number.MAX_SAFE_INTEGER;
      const code = result.code;
      return {
        severity:
          result.status === PlanCheckRun_Result_Status.ERROR
            ? "ERROR"
            : "WARNING",
        message: result.content,
        source: `${result.title} (${code}) L${line}:C${column}`,
        startLineNumber: line,
        endLineNumber: line,
        startColumn: column,
        endColumn: column,
      };
    });
};
