import { maxBy } from "lodash-es";
import { computed, type Ref } from "vue";
import type { AdviceOption } from "@/components/MonacoEditor";
import type { PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import { PlanCheckRun_Result_Type } from "@/types/proto-es/v1/plan_service_pb";
import { type Advice, Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
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
      // With consolidated model, filter results by type instead of checkRuns
      return getLatestAdviceOptions(
        planCheckRuns.value,
        PlanCheckRun_Result_Type.STATEMENT_ADVISE
      );
    }
  });
  return { markers };
};

const getLatestAdviceOptions = (
  planCheckRuns: PlanCheckRun[],
  resultType: PlanCheckRun_Result_Type
) => {
  const latest = maxBy(planCheckRuns, (checkRun) =>
    parseInt(extractPlanCheckRunUID(checkRun.name), 10)
  );
  if (!latest) {
    return [];
  }
  // Filter results by type
  const resultList = latest.results.filter(
    (result) => result.type === resultType
  );
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
