import { maxBy } from "lodash-es";
import { computed, type Ref } from "vue";
import type { AdviceOption } from "@/components/MonacoEditor";
import {
  Plan_Spec,
  PlanCheckRun,
  PlanCheckRun_Result_Status,
  PlanCheckRun_Type,
} from "@/types/proto/v1/plan_service";
import { Advice_Status, type Advice } from "@/types/proto/v1/sql_service";
import { extractPlanCheckRunUID } from "@/utils";
import { planCheckRunListForSpec, type PlanContext } from "../../logic";

export const useSQLAdviceMarkers = (
  context: PlanContext,
  advices: Ref<Advice[] | undefined> | undefined
) => {
  const { isCreating } = context;
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
      const { selectedSpec, planCheckRunList: contextPlanCheckRunList } = context;
      const planCheckRunList = planCheckRunListForSpec(
        contextPlanCheckRunList.value,
        selectedSpec.value as Plan_Spec
      );
      const types: PlanCheckRun_Type[] = [
        PlanCheckRun_Type.DATABASE_STATEMENT_ADVISE,
      ];
      return types.flatMap((type) => {
        return getLatestAdviceOptions(
          planCheckRunList.filter((checkRun) => checkRun.type === type)
        );
      });
    }
  });
  return { markers };
};

const getLatestAdviceOptions = (planCheckRunList: PlanCheckRun[]) => {
  const latest = maxBy(planCheckRunList, (checkRun) =>
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
