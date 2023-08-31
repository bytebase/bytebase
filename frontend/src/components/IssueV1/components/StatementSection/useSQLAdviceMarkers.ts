import { maxBy } from "lodash-es";
import { computed } from "vue";
import { AdviceOption } from "@/components/MonacoEditor";
import {
  PlanCheckRun,
  PlanCheckRun_Result_Status,
  PlanCheckRun_Type,
} from "@/types/proto/v1/rollout_service";
import { planCheckRunListForTask, useIssueContext } from "../../logic";

export const useSQLAdviceMarkers = () => {
  const { isCreating, issue, selectedTask } = useIssueContext();
  const markers = computed(() => {
    if (isCreating.value) return [];

    const task = selectedTask.value;
    const planCheckRunList = planCheckRunListForTask(issue.value, task);

    const types: PlanCheckRun_Type[] = [
      PlanCheckRun_Type.DATABASE_STATEMENT_ADVISE,
    ];
    return types.flatMap((type) => {
      return getLatestAdviceOptions(
        planCheckRunList.filter((checkRun) => checkRun.type === type)
      );
    });
  });
  return { markers };
};

const getLatestAdviceOptions = (planCheckRunList: PlanCheckRun[]) => {
  const latest = maxBy(planCheckRunList, (checkRun) =>
    parseInt(checkRun.uid, 10)
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
      const code = result.sqlReviewReport?.code ?? result.code;
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
