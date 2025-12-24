import { maxBy } from "lodash-es";
import { computed, type Ref } from "vue";
import type { AdviceOption } from "@/components/MonacoEditor";
import type { PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import { PlanCheckRun_Result_Type } from "@/types/proto-es/v1/plan_service_pb";
import { type Advice, Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { extractPlanCheckRunUID } from "@/utils";
import { type IssueContext } from "../../logic";

export const useSQLAdviceMarkers = (
  context: IssueContext,
  advices: Ref<Advice[] | undefined> | undefined
) => {
  const { isCreating } = context;
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
      const { selectedTask, getPlanCheckRunsForTask } = context;
      const planCheckRunList = getPlanCheckRunsForTask(selectedTask.value);

      // With consolidated model, filter results by type instead of checkRuns
      return getLatestAdviceOptions(
        planCheckRunList,
        PlanCheckRun_Result_Type.STATEMENT_ADVISE
      );
    }
  });
  return { markers };
};

const getLatestAdviceOptions = (
  planCheckRunList: PlanCheckRun[],
  resultType: PlanCheckRun_Result_Type
) => {
  const latest = maxBy(planCheckRunList, (checkRun) =>
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
        result.report?.case === "sqlReviewReport" &&
        result.report.value.startPosition !== undefined
    )
    .filter((result) => {
      // Filter out results without valid position (line 0 means unknown)
      const sqlReviewReport =
        result.report?.case === "sqlReviewReport" ? result.report.value : null;
      const line = sqlReviewReport?.startPosition?.line ?? 0;
      return line > 0;
    })
    .map<AdviceOption>((result) => {
      const sqlReviewReport =
        result.report?.case === "sqlReviewReport" ? result.report.value : null;
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
