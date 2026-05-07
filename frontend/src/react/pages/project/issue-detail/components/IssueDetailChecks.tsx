import { create } from "@bufbuild/protobuf";
import type { Timestamp as TimestampPb } from "@bufbuild/protobuf/wkt";
import {
  AlertCircle,
  CheckCircle,
  CircleQuestionMark,
  FileCode,
  Loader2,
  Play,
  SearchCode,
  Shield,
  XCircle,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { planServiceClientConnect } from "@/connect";
import { Button } from "@/react/components/ui/button";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { pushNotification, useCurrentUserV1, useProjectV1Store } from "@/store";
import { extractUserEmail, projectNamePrefix } from "@/store/modules/v1/common";
import {
  getDateForPbTimestampProtoEs,
  getRuleLocalization,
  ruleTemplateMapV2,
  ruleTypeToString,
} from "@/types";
import {
  GetPlanCheckRunRequestSchema,
  GetPlanRequestSchema,
  type PlanCheckRun,
  type PlanCheckRun_Result,
  PlanCheckRun_Result_Type,
  PlanCheckRun_ResultSchema,
  PlanCheckRun_Status,
  RunPlanChecksRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { SQLReviewRule_Type } from "@/types/proto-es/v1/review_config_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { extractDatabaseResourceName } from "@/utils/v1/database";
import { useIssueDetailContext } from "../context/IssueDetailContext";

interface ResultGroup {
  key: string;
  type: PlanCheckRun_Result_Type;
  target: string;
  createTime?: TimestampPb;
  results: PlanCheckRun_Result[];
}

const PAGE_SIZE = 10;

export function IssueDetailChecks() {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const projectStore = useProjectV1Store();
  const currentUser = useVueState(() => useCurrentUserV1().value);
  const [isRunningChecks, setIsRunningChecks] = useState(false);
  const [selectedResultStatus, setSelectedResultStatus] = useState<
    Advice_Level | undefined
  >();
  const projectName = `${projectNamePrefix}${page.projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));
  const summary = useMemo(() => getPlanCheckSummary(page), [page]);
  const hasAnyChecks = summary.total > 0;
  const allowRunChecks = useMemo(() => {
    if (!page.plan) {
      return false;
    }
    // Once a rollout exists, the plan is frozen — re-running checks produces
    // the same result and is misleading for the user.
    if (page.plan.hasRollout) {
      return false;
    }
    if (extractUserEmail(page.plan.creator) === currentUser.email) {
      return true;
    }
    return hasProjectPermissionV2(project, "bb.planCheckRuns.run");
  }, [currentUser.email, page.plan, project]);

  const refreshChecks = useCallback(async () => {
    const planName = page.plan?.name;
    if (!planName) {
      return [];
    }
    const nextPlan = await planServiceClientConnect.getPlan(
      create(GetPlanRequestSchema, {
        name: planName,
      })
    );

    let nextPlanCheckRuns: PlanCheckRun[] = [];
    try {
      const response = await planServiceClientConnect.getPlanCheckRun(
        create(GetPlanCheckRunRequestSchema, {
          name: `${planName}/planCheckRun`,
        })
      );
      nextPlanCheckRuns = [response];
    } catch {
      nextPlanCheckRuns = [];
    }

    page.patchState({
      plan: nextPlan,
      planCheckRuns: nextPlanCheckRuns,
    });
    return nextPlanCheckRuns;
  }, [page.plan?.name, page.patchState]);

  const runChecks = useCallback(async () => {
    if (!page.plan?.name) {
      return;
    }

    try {
      setIsRunningChecks(true);
      await planServiceClientConnect.runPlanChecks(
        create(RunPlanChecksRequestSchema, {
          name: page.plan.name,
        })
      );
      await refreshChecks();
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("plan.checks.started"),
      });
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("plan.checks.failed-to-run"),
        description: String(error),
      });
    } finally {
      setIsRunningChecks(false);
    }
  }, [page.plan?.name, refreshChecks, t]);

  if (!page.plan) {
    return null;
  }

  return (
    <div className="flex flex-col gap-y-1">
      <div className="flex items-center justify-between">
        <h3 className="textlabel">{t("plan.navigator.checks")}</h3>
        {allowRunChecks && (
          <Button
            disabled={isRunningChecks}
            onClick={() => {
              void runChecks();
            }}
            size="xs"
            variant="outline"
          >
            <Play className="h-3.5 w-3.5" />
            {t("plan.run")}
          </Button>
        )}
      </div>
      <div className="flex items-center gap-2">
        {hasAnyChecks && (
          <IssueDetailChecksStatusCount
            selectedStatus={selectedResultStatus}
            summary={summary}
            onSelectStatus={setSelectedResultStatus}
          />
        )}
        {!hasAnyChecks && (
          <span className="text-sm text-control-placeholder">
            {t("plan.overview.no-checks")}
          </span>
        )}
      </div>

      <IssueDetailChecksDialog
        onClose={() => setSelectedResultStatus(undefined)}
        open={selectedResultStatus !== undefined}
        planCheckRuns={page.planCheckRuns}
        refreshChecks={refreshChecks}
        status={selectedResultStatus}
      />
    </div>
  );
}

function IssueDetailChecksStatusCount({
  onSelectStatus,
  selectedStatus,
  summary,
}: {
  onSelectStatus: (status: Advice_Level | undefined) => void;
  selectedStatus?: Advice_Level;
  summary: PlanCheckSummary;
}) {
  const { t } = useTranslation();

  return (
    <div className="flex items-center gap-3">
      {summary.running > 0 && (
        <div className="flex items-center gap-1 text-control">
          <Loader2 className="h-5 w-5 animate-spin" />
          <span>{t("task.status.running")}</span>
        </div>
      )}
      {summary.error > 0 && (
        <button
          className={cn(
            "flex items-center gap-1 text-error",
            selectedStatus === Advice_Level.ERROR
              ? "rounded-lg bg-gray-100 px-2 py-1"
              : selectedStatus !== undefined
                ? "px-2 py-1"
                : ""
          )}
          onClick={() =>
            onSelectStatus(
              selectedStatus === Advice_Level.ERROR
                ? undefined
                : Advice_Level.ERROR
            )
          }
          type="button"
        >
          <XCircle className="h-5 w-5" />
          <span>{summary.error}</span>
        </button>
      )}
      {summary.warning > 0 && (
        <button
          className={cn(
            "flex items-center gap-1 text-warning",
            selectedStatus === Advice_Level.WARNING
              ? "rounded-lg bg-gray-100 px-2 py-1"
              : selectedStatus !== undefined
                ? "px-2 py-1"
                : ""
          )}
          onClick={() =>
            onSelectStatus(
              selectedStatus === Advice_Level.WARNING
                ? undefined
                : Advice_Level.WARNING
            )
          }
          type="button"
        >
          <AlertCircle className="h-5 w-5" />
          <span>{summary.warning}</span>
        </button>
      )}
      {summary.success > 0 && (
        <button
          className={cn(
            "flex items-center gap-1 text-success",
            selectedStatus === Advice_Level.SUCCESS
              ? "rounded-lg bg-gray-100 px-2 py-1"
              : selectedStatus !== undefined
                ? "px-2 py-1"
                : ""
          )}
          onClick={() =>
            onSelectStatus(
              selectedStatus === Advice_Level.SUCCESS
                ? undefined
                : Advice_Level.SUCCESS
            )
          }
          type="button"
        >
          <CheckCircle className="h-5 w-5" />
          <span>{summary.success}</span>
        </button>
      )}
    </div>
  );
}

function IssueDetailChecksDialog({
  onClose,
  open,
  planCheckRuns,
  refreshChecks,
  status,
}: {
  onClose: () => void;
  open: boolean;
  planCheckRuns: PlanCheckRun[];
  refreshChecks: () => Promise<PlanCheckRun[]>;
  status?: Advice_Level;
}) {
  const { t } = useTranslation();
  const [isLoading, setIsLoading] = useState(false);
  const [resolvedPlanCheckRuns, setResolvedPlanCheckRuns] =
    useState<PlanCheckRun[]>(planCheckRuns);

  useEffect(() => {
    setResolvedPlanCheckRuns(planCheckRuns);
  }, [planCheckRuns]);

  useEffect(() => {
    if (!open) {
      return;
    }

    let canceled = false;

    const load = async () => {
      try {
        setIsLoading(true);
        const runs = await refreshChecks();
        if (!canceled) {
          setResolvedPlanCheckRuns(runs);
        }
      } finally {
        if (!canceled) {
          setIsLoading(false);
        }
      }
    };

    void load();

    return () => {
      canceled = true;
    };
  }, [open, refreshChecks]);

  return (
    <Sheet onOpenChange={(nextOpen) => !nextOpen && onClose()} open={open}>
      <SheetContent
        className="w-[calc(100vw-2rem)] max-w-[calc(100vw-2rem)] sm:max-w-[40rem]"
        width="standard"
      >
        <SheetHeader>
          <SheetTitle>{t("plan.navigator.checks")}</SheetTitle>
        </SheetHeader>
        <SheetBody className="py-4">
          <IssueDetailChecksView
            defaultStatus={status}
            isLoading={isLoading}
            planCheckRuns={resolvedPlanCheckRuns}
          />
        </SheetBody>
      </SheetContent>
    </Sheet>
  );
}

function IssueDetailChecksView({
  defaultStatus,
  isLoading = false,
  planCheckRuns,
}: {
  defaultStatus?: Advice_Level;
  isLoading?: boolean;
  planCheckRuns: PlanCheckRun[];
}) {
  const { t } = useTranslation();
  const [selectedStatus, setSelectedStatus] = useState<
    Advice_Level | undefined
  >(defaultStatus);
  const [displayCount, setDisplayCount] = useState(PAGE_SIZE);

  useEffect(() => {
    setSelectedStatus(defaultStatus);
  }, [defaultStatus]);

  const summary = useMemo(
    () => computePlanCheckStatusSummary(planCheckRuns),
    [planCheckRuns]
  );
  const hasAnyStatus = summary.total > 0;
  const hasFilters = selectedStatus !== undefined;
  const filteredResultGroups = useMemo(
    () => buildFilteredResultGroups(planCheckRuns, selectedStatus),
    [planCheckRuns, selectedStatus]
  );
  const displayedResultGroups = filteredResultGroups.slice(0, displayCount);
  const hasMore = filteredResultGroups.length > displayCount;
  const remainingCount = filteredResultGroups.length - displayCount;

  useEffect(() => {
    setDisplayCount(PAGE_SIZE);
  }, [selectedStatus]);

  return (
    <div className="flex flex-1 flex-col">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          {hasAnyStatus && (
            <div className="flex items-center gap-3">
              {summary.error > 0 && (
                <button
                  className={cn(
                    "flex items-center gap-1 text-error",
                    selectedStatus === Advice_Level.ERROR
                      ? "rounded-lg bg-gray-100 px-2 py-1"
                      : "px-2 py-1"
                  )}
                  onClick={() =>
                    setSelectedStatus((current) =>
                      current === Advice_Level.ERROR
                        ? undefined
                        : Advice_Level.ERROR
                    )
                  }
                  type="button"
                >
                  <XCircle className="h-5 w-5" />
                  <span>{t("common.error")}</span>
                  <span>{summary.error}</span>
                </button>
              )}
              {summary.warning > 0 && (
                <button
                  className={cn(
                    "flex items-center gap-1 text-warning",
                    selectedStatus === Advice_Level.WARNING
                      ? "rounded-lg bg-gray-100 px-2 py-1"
                      : "px-2 py-1"
                  )}
                  onClick={() =>
                    setSelectedStatus((current) =>
                      current === Advice_Level.WARNING
                        ? undefined
                        : Advice_Level.WARNING
                    )
                  }
                  type="button"
                >
                  <AlertCircle className="h-5 w-5" />
                  <span>{t("common.warning")}</span>
                  <span>{summary.warning}</span>
                </button>
              )}
              {summary.success > 0 && (
                <button
                  className={cn(
                    "flex items-center gap-1 text-success",
                    selectedStatus === Advice_Level.SUCCESS
                      ? "rounded-lg bg-gray-100 px-2 py-1"
                      : "px-2 py-1"
                  )}
                  onClick={() =>
                    setSelectedStatus((current) =>
                      current === Advice_Level.SUCCESS
                        ? undefined
                        : Advice_Level.SUCCESS
                    )
                  }
                  type="button"
                >
                  <CheckCircle className="h-5 w-5" />
                  <span>{t("common.success")}</span>
                  <span>{summary.success}</span>
                </button>
              )}
            </div>
          )}
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        {isLoading ? (
          <div className="flex flex-col items-center justify-center py-12">
            <Loader2 className="h-5 w-5 animate-spin text-control-light" />
          </div>
        ) : filteredResultGroups.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12">
            <CheckCircle className="mb-4 h-12 w-12 text-control-light opacity-50" />
            <div className="text-lg text-control-light">
              {hasFilters
                ? t("plan.checks.no-results-match-filters")
                : t("plan.checks.no-check-results")}
            </div>
          </div>
        ) : (
          <div>
            <div className="divide-y">
              {displayedResultGroups.map((group) => (
                <div key={group.key} className="px-2 py-4">
                  <div className="mb-2 flex flex-wrap items-start justify-between gap-2">
                    <div className="flex shrink-0 items-center gap-3">
                      <CheckTypeIcon
                        className="h-5 w-5 text-control-light"
                        type={group.type}
                      />
                      <div className="flex flex-row items-center gap-2">
                        <span className="text-sm font-medium">
                          {getCheckTypeLabel(group.type, t)}
                        </span>
                        {getCheckTypeDescription(group.type, t) && (
                          <Tooltip
                            content={getCheckTypeDescription(group.type, t)}
                          >
                            <CircleQuestionMark className="h-4 w-4 text-control-light" />
                          </Tooltip>
                        )}
                        {group.createTime && (
                          <span className="text-sm text-control-light">
                            {formatTimestamp(group.createTime)}
                          </span>
                        )}
                      </div>
                    </div>
                    {group.target && (
                      <div className="min-w-0 max-w-[50%] truncate text-sm text-control-light">
                        {formatCheckTarget(group.target)}
                      </div>
                    )}
                  </div>

                  <div className="flex flex-col gap-y-2 pl-8">
                    {group.results.map((result, index) => (
                      <IssueDetailCheckResultCard
                        key={`${group.key}-${index}`}
                        affectedRows={
                          result.report?.case === "sqlSummaryReport"
                            ? result.report.value.affectedRows
                            : undefined
                        }
                        code={result.code}
                        content={result.content}
                        position={
                          result.report?.case === "sqlReviewReport" &&
                          result.report.value.startPosition
                            ? result.report.value.startPosition
                            : undefined
                        }
                        reportType={result.report?.case}
                        status={getCheckResultStatus(result.status)}
                        title={result.title}
                      />
                    ))}
                  </div>
                </div>
              ))}
            </div>

            {hasMore && (
              <div className="flex justify-center py-4">
                <button
                  className="text-sm text-accent hover:underline"
                  onClick={() =>
                    setDisplayCount((current) => current + PAGE_SIZE)
                  }
                  type="button"
                >
                  {t("common.load-more")} ({remainingCount})
                </button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function IssueDetailCheckResultCard({
  affectedRows,
  code,
  content,
  position,
  reportType,
  status,
  title,
}: {
  affectedRows?: bigint;
  code?: number;
  content?: string;
  position?: { line: number; column: number };
  reportType?: "sqlReviewReport" | "sqlSummaryReport";
  status: "SUCCESS" | "WARNING" | "ERROR";
  title: string;
}) {
  const { t } = useTranslation();
  const displayTitle = useMemo(
    () => getCheckResultTitle(title, code, reportType),
    [code, reportType, title]
  );

  return (
    <div className="flex items-start gap-3 rounded-lg border bg-gray-50 px-3 py-2">
      {status === "ERROR" ? (
        <XCircle className="h-5 w-5 shrink-0 text-error" />
      ) : status === "WARNING" ? (
        <AlertCircle className="h-5 w-5 shrink-0 text-warning" />
      ) : (
        <CheckCircle className="h-5 w-5 shrink-0 text-success" />
      )}

      <div className="flex min-w-0 flex-1 flex-col gap-y-1">
        <div className="text-sm font-medium text-main">{displayTitle}</div>
        {content && <div className="text-sm text-control">{content}</div>}
        {position && (position.line > 0 || position.column > 0) && (
          <div className="mt-1 text-sm text-control-light">
            {position.line > 0 && (
              <span>
                {t("common.line")} {position.line}
              </span>
            )}
            {position.line > 0 && position.column > 0 && <span>, </span>}
            {position.column > 0 && (
              <span>
                {t("common.column")} {position.column}
              </span>
            )}
          </div>
        )}
        {affectedRows !== undefined && (
          <div className="mt-1 flex items-center gap-1 text-sm">
            <span className="inline-flex items-center rounded-full bg-white px-2 py-0.5 text-xs text-control">
              {t("task.check-type.affected-rows.self")}
            </span>
            <span>{String(affectedRows)}</span>
            <span className="text-control opacity-80">
              ({t("task.check-type.affected-rows.description")})
            </span>
          </div>
        )}
      </div>
    </div>
  );
}

interface PlanCheckSummary {
  running: number;
  success: number;
  warning: number;
  error: number;
  total: number;
}

function getPlanCheckSummary(
  page: ReturnType<typeof useIssueDetailContext>
): PlanCheckSummary {
  if (page.planCheckRuns.length > 0) {
    return computePlanCheckStatusSummary(page.planCheckRuns);
  }

  const statusCount = page.plan?.planCheckRunStatusCount || {};
  const running =
    statusCount[PlanCheckRun_Status[PlanCheckRun_Status.RUNNING]] || 0;
  const success = statusCount[Advice_Level[Advice_Level.SUCCESS]] || 0;
  const warning = statusCount[Advice_Level[Advice_Level.WARNING]] || 0;
  const error =
    (statusCount[Advice_Level[Advice_Level.ERROR]] || 0) +
    (statusCount[PlanCheckRun_Status[PlanCheckRun_Status.FAILED]] || 0);
  const total = running + success + warning + error;
  return { running, success, warning, error, total };
}

function computePlanCheckStatusSummary(runs: PlanCheckRun[]): PlanCheckSummary {
  let running = 0;
  let success = 0;
  let warning = 0;
  let error = 0;

  for (const checkRun of runs) {
    if (checkRun.status === PlanCheckRun_Status.RUNNING) {
      running++;
    }
    if (checkRun.status === PlanCheckRun_Status.FAILED) {
      error++;
    }
    for (const result of checkRun.results) {
      if (result.status === Advice_Level.ERROR) {
        error++;
      } else if (result.status === Advice_Level.WARNING) {
        warning++;
      } else if (result.status === Advice_Level.SUCCESS) {
        success++;
      }
    }
  }

  return {
    error,
    running,
    success,
    total: running + success + warning + error,
    warning,
  };
}

function buildFilteredResultGroups(
  planCheckRuns: PlanCheckRun[],
  selectedStatus?: Advice_Level
) {
  const groups: ResultGroup[] = [];
  const groupMap = new Map<string, ResultGroup>();

  for (const checkRun of planCheckRuns) {
    if (
      checkRun.status === PlanCheckRun_Status.FAILED &&
      (selectedStatus === undefined || selectedStatus === Advice_Level.ERROR)
    ) {
      groups.push({
        createTime: checkRun.createTime,
        key: `failed-${checkRun.name}`,
        results: [
          create(PlanCheckRun_ResultSchema, {
            code: 0,
            content: checkRun.error || "Plan check run failed",
            status: Advice_Level.ERROR,
            title: "Check Failed",
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
      const group = groupMap.get(key) ?? {
        createTime: checkRun.createTime,
        key,
        results: [],
        target: result.target,
        type: result.type,
      };
      group.results.push(result);
      if (!groupMap.has(key)) {
        groupMap.set(key, group);
        groups.push(group);
      }
    }
  }

  return groups;
}

function CheckTypeIcon({
  className,
  type,
}: {
  className?: string;
  type: PlanCheckRun_Result_Type;
}) {
  const Icon =
    type === PlanCheckRun_Result_Type.STATEMENT_ADVISE
      ? SearchCode
      : type === PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT
        ? FileCode
        : type === PlanCheckRun_Result_Type.GHOST_SYNC
          ? Shield
          : FileCode;
  return <Icon className={className} />;
}

function getCheckTypeLabel(
  type: PlanCheckRun_Result_Type,
  t: ReturnType<typeof useTranslation>["t"]
) {
  if (type === PlanCheckRun_Result_Type.STATEMENT_ADVISE) {
    return t("task.check-type.sql-review.self");
  }
  if (type === PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT) {
    return t("task.check-type.summary-report");
  }
  if (type === PlanCheckRun_Result_Type.GHOST_SYNC) {
    return t("task.check-type.ghost-sync");
  }
  return String(type);
}

function getCheckTypeDescription(
  type: PlanCheckRun_Result_Type,
  t: ReturnType<typeof useTranslation>["t"]
) {
  if (type === PlanCheckRun_Result_Type.STATEMENT_ADVISE) {
    return t("task.check-type.sql-review.description");
  }
  return undefined;
}

function getCheckResultStatus(
  status: Advice_Level
): "SUCCESS" | "WARNING" | "ERROR" {
  if (status === Advice_Level.ERROR) {
    return "ERROR";
  }
  if (status === Advice_Level.WARNING) {
    return "WARNING";
  }
  return "SUCCESS";
}

function getCheckResultTitle(
  title: string,
  code?: number,
  reportType?: "sqlReviewReport" | "sqlSummaryReport"
) {
  const messageWithCode = (message: string) => {
    if (code !== undefined && code !== 0) {
      return `${message} #${code}`;
    }
    return message;
  };

  if (title === "OK" || title === "Syntax error") {
    return messageWithCode(title);
  }

  if (reportType === "sqlReviewReport") {
    if (!code) {
      return title;
    }
    const typeKey = title as keyof typeof SQLReviewRule_Type;
    const typeEnum = SQLReviewRule_Type[typeKey];
    if (typeEnum !== undefined) {
      for (const mapByType of ruleTemplateMapV2.values()) {
        const rule = mapByType.get(typeEnum);
        if (rule) {
          return messageWithCode(
            getRuleLocalization(ruleTypeToString(rule.type), rule.engine).title
          );
        }
      }
    } else if (title.startsWith("builtin.")) {
      return messageWithCode(getRuleLocalization(title).title);
    }
  }

  return messageWithCode(title);
}

function formatTimestamp(timestamp: TimestampPb) {
  return getDateForPbTimestampProtoEs(timestamp)?.toLocaleString() ?? "";
}

function formatCheckTarget(target: string) {
  if (!target) {
    return "";
  }
  try {
    return extractDatabaseResourceName(target).databaseName;
  } catch {
    return target;
  }
}
