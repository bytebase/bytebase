import { maxBy } from "lodash-es";
import { computed, type Ref } from "vue";
import type { AdviceOption } from "@/components/MonacoEditor";
import { PlanCheckRun_Type } from "@/types/proto-es/v1/plan_service_pb";
import type { PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level, type Advice } from "@/types/proto-es/v1/sql_service_pb";
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
      return advices.value
        .filter((advice) => {
          // Filter out advices without valid position (line 0 means unknown)
          const line = advice.startPosition?.line ?? 0;
          return line > 0;
        })
        .map<AdviceOption>((advice) => {
          const line = advice.startPosition!.line;
          const column = advice.startPosition!.column || 1;
          const code = advice.code;
          return {
            severity:
              advice.status === Advice_Level.ERROR ? "ERROR" : "WARNING",
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
        result.status === Advice_Level.ERROR ||
        result.status === Advice_Level.WARNING
    )
    .filter(
      (result) =>
        result.report.case === "sqlReviewReport" &&
        result.report.value.startPosition !== undefined
    )
    .filter((result) => {
      // Filter out results without valid position (line 0 means unknown)
      const sqlReviewReport =
        result.report.case === "sqlReviewReport" ? result.report.value : null;
      const line = sqlReviewReport?.startPosition?.line ?? 0;
      return line > 0;
    })
    .map<AdviceOption>((result) => {
      const sqlReviewReport =
        result.report.case === "sqlReviewReport" ? result.report.value : null;
      const line = sqlReviewReport!.startPosition!.line;
      const column = sqlReviewReport!.startPosition!.column || 1;
      const code = result.code;
      return {
        severity: result.status === Advice_Level.ERROR ? "ERROR" : "WARNING",
        message: result.content,
        source: `${result.title} (${code}) L${line}:C${column}`,
        startLineNumber: line,
        endLineNumber: line,
        startColumn: column,
        endColumn: column,
      };
    });
};
