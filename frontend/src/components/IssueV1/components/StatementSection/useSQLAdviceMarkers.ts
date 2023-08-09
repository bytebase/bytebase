import { computed } from "vue";
import { maxBy } from "lodash-es";

import {
  PlanCheckRun,
  PlanCheckRun_Result_Status,
  PlanCheckRun_Type,
} from "@/types/proto/v1/rollout_service";
import { AdviceOption } from "@/components/MonacoEditor";
import { useIssueContext } from "../../logic";

export const useSQLAdviceMarkers = () => {
  const { isCreating, issue, selectedTask } = useIssueContext();
  const markers = computed(() => {
    if (isCreating.value) return [];

    const task = selectedTask.value;
    const planCheckRunList = issue.value.planCheckRunList.filter((checkRun) => {
      task; // TODO: match task and plan
      return true;
    });

    const types: PlanCheckRun_Type[] = [
      PlanCheckRun_Type.DATABASE_STATEMENT_ADVISE,
      PlanCheckRun_Type.DATABASE_STATEMENT_SYNTAX,
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
    .filter((result) => result.line !== undefined)
    .map<AdviceOption>((result) => {
      return {
        severity:
          result.status === PlanCheckRun_Result_Status.ERROR
            ? "ERROR"
            : "WARNING",
        message: result.content,
        source: `${result.title} (${result.code})`,
        startLineNumber: result.line!,
        // We don't know the actual column yet, so we show the marker at then end of the line
        startColumn: Number.MAX_SAFE_INTEGER,
        endLineNumber: result.line!,
        // We don't know the actual column yet, so we show the marker at then end of the line
        endColumn: Number.MAX_SAFE_INTEGER,
      };
    });
};
