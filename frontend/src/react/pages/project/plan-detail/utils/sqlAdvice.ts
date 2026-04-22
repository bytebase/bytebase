import type { AdviceOption } from "@/react/components/monaco/types";
import type {
  PlanCheckRun,
  PlanCheckRun_Result,
} from "@/types/proto-es/v1/plan_service_pb";
import { PlanCheckRun_Result_Type } from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";

const planCheckRunOrdinal = (name: string): number => {
  const uid = name.match(/(?:^|\/)planCheckRuns\/(\d+)(?:$|\/)/)?.[1];
  return uid ? Number.parseInt(uid, 10) : 0;
};

export const getSQLAdviceMarkers = (
  planCheckRuns: PlanCheckRun[]
): AdviceOption[] => {
  const latest = [...planCheckRuns].sort(
    (left, right) =>
      planCheckRunOrdinal(right.name) - planCheckRunOrdinal(left.name)
  )[0];
  if (!latest) {
    return [];
  }

  return latest.results
    .filter(
      (result) => result.type === PlanCheckRun_Result_Type.STATEMENT_ADVISE
    )
    .map(resultToAdviceOption)
    .filter((marker): marker is AdviceOption => marker !== undefined);
};

const resultToAdviceOption = (
  result: PlanCheckRun_Result
): AdviceOption | undefined => {
  if (
    result.status !== Advice_Level.ERROR &&
    result.status !== Advice_Level.WARNING
  ) {
    return undefined;
  }
  if (result.report.case !== "sqlReviewReport") {
    return undefined;
  }

  const position = result.report.value.startPosition;
  if (!position || position.line <= 0) {
    return undefined;
  }

  const column = position.column || 1;
  return {
    endColumn: column,
    endLineNumber: position.line,
    message: result.content,
    severity: result.status === Advice_Level.ERROR ? "ERROR" : "WARNING",
    source: `${result.title} (${result.code}) L${position.line}:C${column}`,
    startColumn: column,
    startLineNumber: position.line,
  };
};
