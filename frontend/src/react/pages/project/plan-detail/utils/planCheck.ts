import { create } from "@bufbuild/protobuf";
import type { Timestamp as TimestampPb } from "@bufbuild/protobuf/wkt";
import {
  type Plan_Spec,
  type PlanCheckRun,
  type PlanCheckRun_Result,
  PlanCheckRun_Result_Type,
  PlanCheckRun_ResultSchema,
  PlanCheckRun_Status,
  PlanCheckRunSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import type { CheckReleaseResponse_CheckResult } from "@/types/proto-es/v1/release_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";

export interface PlanCheckSummary {
  error: number;
  running: number;
  success: number;
  total: number;
  warning: number;
}

export interface ResultGroup {
  key: string;
  type: PlanCheckRun_Result_Type;
  target: string;
  createTime?: TimestampPb;
  results: PlanCheckRun_Result[];
}

export const getPlanCheckSummary = (
  planCheckRuns: PlanCheckRun[]
): PlanCheckSummary => {
  let running = 0;
  let success = 0;
  let warning = 0;
  let error = 0;

  for (const checkRun of planCheckRuns) {
    if (checkRun.status === PlanCheckRun_Status.RUNNING) running++;
    if (checkRun.status === PlanCheckRun_Status.FAILED) error++;
    for (const result of checkRun.results) {
      if (result.status === Advice_Level.ERROR) error++;
      else if (result.status === Advice_Level.WARNING) warning++;
      else if (result.status === Advice_Level.SUCCESS) success++;
    }
  }

  return {
    error,
    running,
    success,
    total: running + success + warning + error,
    warning,
  };
};

const targetsForSpec = (spec: Plan_Spec): string[] => {
  if (spec.config.case === "changeDatabaseConfig") {
    return spec.config.value.targets ?? [];
  }
  if (spec.config.case === "exportDataConfig") {
    return spec.config.value.targets ?? [];
  }
  return [];
};

// The plan-check scheduler expands a db-group spec into per-database targets
// before running checks (see backend/runner/plancheck/derive.go), so result.target
// is always a database resource name. The spec, however, still stores the
// db-group name. The caller is responsible for resolving any db-group target to
// its matched database names before calling this function; otherwise checks for
// db-group specs would silently render empty.
export const planCheckRunListForSpec = (
  planCheckRuns: PlanCheckRun[],
  spec: Plan_Spec,
  expandedTargets?: string[]
): PlanCheckRun[] => {
  const targets = expandedTargets ?? targetsForSpec(spec);
  const targetSet = new Set(targets);
  if (targetSet.size === 0) {
    return [];
  }
  return planCheckRuns.flatMap((run) => {
    const matchingResults = run.results.filter((result) =>
      targetSet.has(result.target)
    );
    if (matchingResults.length > 0) {
      return [{ ...run, results: matchingResults }];
    }
    if (
      (run.status === PlanCheckRun_Status.RUNNING ||
        run.status === PlanCheckRun_Status.FAILED) &&
      run.results.length === 0
    ) {
      return [{ ...run, results: [] }];
    }
    return [];
  });
};

export const expandSpecTargets = (
  spec: Plan_Spec,
  resolveDatabaseGroup: (name: string) => string[] | undefined
): string[] => {
  const raw = targetsForSpec(spec);
  return raw.flatMap((target) => {
    const databases = resolveDatabaseGroup(target);
    return databases ?? [target];
  });
};

export const getFilteredResultGroups = ({
  includeRunFailure = false,
  planCheckRuns,
  runFailureContent,
  runFailureTitle,
  selectedStatus,
}: {
  includeRunFailure?: boolean;
  planCheckRuns: PlanCheckRun[];
  runFailureContent?: string;
  runFailureTitle?: string;
  selectedStatus?: Advice_Level;
}): ResultGroup[] => {
  const groups: ResultGroup[] = [];
  const groupMap = new Map<string, ResultGroup>();

  for (const checkRun of planCheckRuns) {
    if (
      includeRunFailure &&
      checkRun.status === PlanCheckRun_Status.FAILED &&
      (selectedStatus === undefined || selectedStatus === Advice_Level.ERROR)
    ) {
      groups.push({
        createTime: checkRun.createTime,
        key: `failed-${checkRun.name}`,
        results: [
          create(PlanCheckRun_ResultSchema, {
            code: 0,
            content: checkRun.error || runFailureContent,
            status: Advice_Level.ERROR,
            title: runFailureTitle,
          }),
        ],
        target: "",
        type: PlanCheckRun_Result_Type.TYPE_UNSPECIFIED,
      });
    }

    for (const result of checkRun.results) {
      if (selectedStatus !== undefined && result.status !== selectedStatus) {
        continue;
      }
      const key = `${result.type}-${result.target}`;
      let group = groupMap.get(key);
      if (!group) {
        group = {
          createTime: checkRun.createTime,
          key,
          results: [],
          target: result.target,
          type: result.type,
        };
        groupMap.set(key, group);
        groups.push(group);
      }
      group.results.push(result);
    }
  }

  return groups;
};

export const transformReleaseCheckResultsToPlanCheckRuns = (
  results: CheckReleaseResponse_CheckResult[]
): PlanCheckRun[] => {
  const allResults: PlanCheckRun_Result[] = [];

  for (const result of results) {
    for (const advice of result.advices) {
      allResults.push(
        create(PlanCheckRun_ResultSchema, {
          code: advice.code,
          content: advice.content,
          report: {
            case: "sqlReviewReport",
            value: {
              startPosition: advice.startPosition,
            },
          },
          status:
            advice.status === Advice_Level.ERROR
              ? Advice_Level.ERROR
              : advice.status === Advice_Level.WARNING
                ? Advice_Level.WARNING
                : Advice_Level.SUCCESS,
          target: result.target,
          title: advice.title,
          type: PlanCheckRun_Result_Type.STATEMENT_ADVISE,
        })
      );
    }
  }

  if (allResults.length === 0) {
    return [];
  }

  return [
    create(PlanCheckRunSchema, {
      createTime: { nanos: 0, seconds: BigInt(Math.floor(Date.now() / 1000)) },
      name: "check-run-0",
      results: allResults,
      status: PlanCheckRun_Status.DONE,
    }),
  ];
};
