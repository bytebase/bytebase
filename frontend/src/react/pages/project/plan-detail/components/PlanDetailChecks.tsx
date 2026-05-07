import { create } from "@bufbuild/protobuf";
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
import { Badge } from "@/react/components/ui/badge";
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
import {
  projectNamePrefix,
  pushNotification,
  useCurrentUserV1,
  useDBGroupStore,
  useProjectV1Store,
} from "@/store";
import { extractUserEmail } from "@/store/modules/v1/common";
import {
  getDateForPbTimestampProtoEs,
  isValidDatabaseGroupName,
} from "@/types";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import {
  GetPlanCheckRunRequestSchema,
  GetPlanRequestSchema,
  type PlanCheckRun,
  type PlanCheckRun_Result,
  PlanCheckRun_Result_Type,
  RunPlanChecksRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { extractDatabaseResourceName } from "@/utils/v1/database";
import { usePlanDetailContext } from "../context/PlanDetailContext";
import {
  expandSpecTargets,
  getFilteredResultGroups,
  getPlanCheckSummary,
  type PlanCheckSummary,
  planCheckRunListForSpec,
} from "../utils/planCheck";

const PAGE_SIZE = 10;

export function PlanDetailChecks({
  selectedSpec,
}: {
  selectedSpec: Plan_Spec;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const { patchState } = page;
  const projectStore = useProjectV1Store();
  const dbGroupStore = useDBGroupStore();
  const currentUser = useCurrentUserV1().value;
  const [isRunningChecks, setIsRunningChecks] = useState(false);
  const [selectedResultStatus, setSelectedResultStatus] = useState<
    Advice_Level | undefined
  >();
  const projectName = `${projectNamePrefix}${page.projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));

  // Plan check results are keyed by per-database resource names even for
  // db-group specs (the scheduler expands targets at runtime). Pre-fetch any
  // db-group target this spec references so we can resolve it to the matched
  // databases when filtering check results below.
  const dbGroupTargets = useMemo(() => {
    if (selectedSpec.config.case === "changeDatabaseConfig") {
      return (selectedSpec.config.value.targets ?? []).filter(
        isValidDatabaseGroupName
      );
    }
    if (selectedSpec.config.case === "exportDataConfig") {
      return (selectedSpec.config.value.targets ?? []).filter(
        isValidDatabaseGroupName
      );
    }
    return [];
  }, [selectedSpec]);
  useEffect(() => {
    for (const target of dbGroupTargets) {
      void dbGroupStore
        .getOrFetchDBGroupByName(target, { view: DatabaseGroupView.FULL })
        .catch(() => undefined);
    }
  }, [dbGroupStore, dbGroupTargets]);
  // Stable join-key so useVueState can detect changes via Object.is
  // — returning a fresh array each call would trigger an update loop.
  const expandedTargetsKey = useVueState(() => {
    const targets = expandSpecTargets(selectedSpec, (name) => {
      if (!isValidDatabaseGroupName(name)) return undefined;
      const group = dbGroupStore.getDBGroupByName(name, DatabaseGroupView.FULL);
      return group.matchedDatabases.map((db) => db.name);
    });
    return targets.join("\u0000");
  });
  const expandedTargets = useMemo(
    () => (expandedTargetsKey ? expandedTargetsKey.split("\u0000") : []),
    [expandedTargetsKey]
  );
  const filteredPlanCheckRuns = useMemo(
    () =>
      planCheckRunListForSpec(
        page.planCheckRuns,
        selectedSpec,
        expandedTargets
      ),
    [page.planCheckRuns, selectedSpec, expandedTargets]
  );
  const summary = useMemo(
    () => getPlanCheckSummary(filteredPlanCheckRuns),
    [filteredPlanCheckRuns]
  );
  const allowRunChecks = useMemo(() => {
    if (page.plan.hasRollout) {
      return false;
    }
    if (extractUserEmail(page.plan.creator) === currentUser.email) {
      return true;
    }
    return hasProjectPermissionV2(project, "bb.planCheckRuns.run");
  }, [currentUser.email, page.plan.creator, page.plan.hasRollout, project]);

  const refreshChecks = useCallback(async () => {
    const nextPlan = await planServiceClientConnect.getPlan(
      create(GetPlanRequestSchema, {
        name: page.plan.name,
      })
    );

    let nextPlanCheckRuns: PlanCheckRun[] = [];
    try {
      const response = await planServiceClientConnect.getPlanCheckRun(
        create(GetPlanCheckRunRequestSchema, {
          name: `${page.plan.name}/planCheckRun`,
        })
      );
      nextPlanCheckRuns = [response];
    } catch {
      nextPlanCheckRuns = [];
    }

    patchState({
      plan: nextPlan,
      planCheckRuns: nextPlanCheckRuns,
    });
    return nextPlanCheckRuns;
  }, [page.plan.name, patchState]);

  const runChecks = useCallback(async () => {
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
  }, [page.plan.name, refreshChecks, t]);

  return (
    <div className="flex flex-col gap-y-1">
      <div className="flex items-center justify-between gap-2">
        <h3 className="textlabel uppercase">{t("plan.checks.self")}</h3>
        {allowRunChecks && (
          <Button
            disabled={isRunningChecks}
            onClick={() => void runChecks()}
            size="xs"
            variant="outline"
          >
            <Play className="h-3.5 w-3.5" />
            {t("plan.run")}
          </Button>
        )}
      </div>

      <div className="flex min-w-0 flex-wrap items-center gap-2 text-sm">
        {summary.total > 0 ? (
          <StatusCount
            selectedStatus={selectedResultStatus}
            summary={summary}
            onSelectStatus={setSelectedResultStatus}
          />
        ) : (
          <span className="text-sm text-control-placeholder">
            {t("plan.overview.no-checks")}
          </span>
        )}
      </div>

      <ChecksDialog
        onClose={() => setSelectedResultStatus(undefined)}
        open={selectedResultStatus !== undefined}
        planCheckRuns={filteredPlanCheckRuns}
        refreshChecks={refreshChecks}
        status={selectedResultStatus}
      />
    </div>
  );
}

function StatusCount({
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
    <div className="flex min-w-0 flex-wrap items-center gap-3 text-sm">
      {summary.running > 0 && (
        <div className="flex items-center gap-1 text-control">
          <Loader2 className="h-5 w-5 animate-spin" />
          <span>{t("common.running")}</span>
        </div>
      )}
      {[
        {
          count: summary.error,
          icon: <XCircle className="h-5 w-5" />,
          status: Advice_Level.ERROR,
          textClass: "text-error",
        },
        {
          count: summary.warning,
          icon: <AlertCircle className="h-5 w-5" />,
          status: Advice_Level.WARNING,
          textClass: "text-warning",
        },
        {
          count: summary.success,
          icon: <CheckCircle className="h-5 w-5" />,
          status: Advice_Level.SUCCESS,
          textClass: "text-success",
        },
      ]
        .filter((item) => item.count > 0)
        .map((item) => (
          <button
            key={item.status}
            className={cn(
              "flex items-center gap-1",
              item.textClass,
              selectedStatus === item.status
                ? "rounded-lg bg-gray-100 px-2 py-1"
                : selectedStatus !== undefined
                  ? "px-2 py-1"
                  : ""
            )}
            onClick={() =>
              onSelectStatus(
                selectedStatus === item.status ? undefined : item.status
              )
            }
            type="button"
          >
            {item.icon}
            <span>
              {item.status === Advice_Level.ERROR
                ? t("common.error")
                : item.status === Advice_Level.WARNING
                  ? t("common.warning")
                  : t("common.success")}
            </span>
            <span>{item.count}</span>
          </button>
        ))}
    </div>
  );
}

function ChecksDialog({
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
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [selectedStatus, setSelectedStatus] = useState<
    Advice_Level | undefined
  >(status);
  const [displayCount, setDisplayCount] = useState(PAGE_SIZE);
  const summary = useMemo(
    () => getPlanCheckSummary(planCheckRuns),
    [planCheckRuns]
  );

  useEffect(() => {
    if (!open) return;
    setSelectedStatus(status);
    setDisplayCount(PAGE_SIZE);
    const refresh = async () => {
      try {
        setIsRefreshing(true);
        await refreshChecks();
      } finally {
        setIsRefreshing(false);
      }
    };
    void refresh();
  }, [open, refreshChecks, status]);

  const filteredResultGroups = useMemo(() => {
    return getFilteredResultGroups({
      includeRunFailure: true,
      planCheckRuns,
      runFailureContent: t("common.failed"),
      runFailureTitle: t("common.failed"),
      selectedStatus,
    });
  }, [planCheckRuns, selectedStatus, t]);

  const displayedResultGroups = filteredResultGroups.slice(0, displayCount);
  const remainingCount = Math.max(
    filteredResultGroups.length - displayCount,
    0
  );

  return (
    <Sheet
      onOpenChange={(nextOpen) => {
        if (!nextOpen) onClose();
      }}
      open={open}
    >
      <SheetContent width="wide">
        <SheetHeader>
          <SheetTitle>{t("plan.navigator.checks")}</SheetTitle>
        </SheetHeader>
        <SheetBody className="gap-y-4">
          {summary.total > 0 && (
            <StatusCount
              selectedStatus={selectedStatus}
              summary={summary}
              onSelectStatus={setSelectedStatus}
            />
          )}
          {filteredResultGroups.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-control-light">
              {isRefreshing ? (
                <Loader2 className="mb-4 h-12 w-12 animate-spin opacity-50" />
              ) : (
                <CheckCircle className="mb-4 h-12 w-12 opacity-50" />
              )}
              <div className="text-lg">
                {selectedStatus !== undefined
                  ? t("plan.checks.no-results-match-filters")
                  : t("plan.checks.no-check-results")}
              </div>
            </div>
          ) : (
            <>
              <div className="divide-y">
                {displayedResultGroups.map((group) => (
                  <div key={group.key} className="px-2 py-4">
                    <div className="mb-2 flex flex-wrap items-start justify-between gap-2">
                      <div className="flex items-center gap-3">
                        {checkTypeIcon(group.type)}
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-medium">
                            {checkTypeLabel(group.type, t)}
                          </span>
                          {checkTypeDescription(group.type, t) && (
                            <Tooltip
                              content={checkTypeDescription(group.type, t)}
                            >
                              <CircleQuestionMark className="h-4 w-4 text-control-light" />
                            </Tooltip>
                          )}
                          {group.createTime && (
                            <span className="text-xs text-control-light">
                              {getDateForPbTimestampProtoEs(
                                group.createTime
                              )?.toLocaleString() ?? ""}
                            </span>
                          )}
                        </div>
                      </div>
                      <div className="min-w-0 max-w-[50%] text-sm text-control-light">
                        {group.target ? (
                          <span className="truncate">
                            {
                              extractDatabaseResourceName(group.target)
                                .databaseName
                            }
                          </span>
                        ) : null}
                      </div>
                    </div>
                    <div className="flex flex-col gap-y-2 pl-8">
                      {group.results.map((result, index) => (
                        <CheckResultCard
                          key={`${group.key}-${index}`}
                          result={result}
                        />
                      ))}
                    </div>
                  </div>
                ))}
              </div>
              {remainingCount > 0 && (
                <div className="flex justify-center py-4">
                  <button
                    className="text-sm text-accent hover:underline"
                    onClick={() =>
                      setDisplayCount((count) => count + PAGE_SIZE)
                    }
                    type="button"
                  >
                    {t("common.load-more")} ({remainingCount})
                  </button>
                </div>
              )}
            </>
          )}
        </SheetBody>
      </SheetContent>
    </Sheet>
  );
}

function CheckResultCard({ result }: { result: PlanCheckRun_Result }) {
  const { t } = useTranslation();
  const position =
    result.report?.case === "sqlReviewReport"
      ? result.report.value.startPosition
      : undefined;
  const affectedRows =
    result.report?.case === "sqlSummaryReport"
      ? result.report.value.affectedRows
      : undefined;

  return (
    <div className="flex items-start gap-3 rounded-lg border border-control-border bg-control-bg px-3 py-2">
      <div className="mt-0.5 shrink-0">{statusIcon(result.status)}</div>
      <div className="flex min-w-0 flex-1 flex-col gap-y-1">
        <div className="text-sm font-medium text-main">{result.title}</div>
        {result.content && (
          <div className="text-sm text-control">{result.content}</div>
        )}
        {position && (position.line > 0 || position.column > 0) && (
          <div className="mt-1 text-sm text-control-light">
            {position.line > 0 ? `${t("common.line")} ${position.line}` : ""}
            {position.line > 0 && position.column > 0 ? ", " : ""}
            {position.column > 0
              ? `${t("common.column")} ${position.column}`
              : ""}
          </div>
        )}
        {affectedRows !== undefined && (
          <div className="mt-1 flex items-center gap-1 text-sm">
            <Badge className="px-2 py-0 text-xs font-medium" variant="default">
              {t("task.check-type.affected-rows.self")}
            </Badge>
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

function checkTypeIcon(type: PlanCheckRun_Result_Type) {
  const className = "h-5 w-5 text-control-light";
  switch (type) {
    case PlanCheckRun_Result_Type.STATEMENT_ADVISE:
      return <SearchCode className={className} />;
    case PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT:
      return <FileCode className={className} />;
    case PlanCheckRun_Result_Type.GHOST_SYNC:
      return <Shield className={className} />;
    default:
      return <FileCode className={className} />;
  }
}

function checkTypeLabel(
  type: PlanCheckRun_Result_Type,
  t: (key: string) => string
) {
  switch (type) {
    case PlanCheckRun_Result_Type.STATEMENT_ADVISE:
      return t("task.check-type.sql-review.self");
    case PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT:
      return t("task.check-type.affected-rows.self");
    case PlanCheckRun_Result_Type.GHOST_SYNC:
      return t("task.online-migration.self");
    default:
      return t("plan.navigator.checks");
  }
}

function checkTypeDescription(
  type: PlanCheckRun_Result_Type,
  t: (key: string) => string
) {
  switch (type) {
    case PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT:
      return t("task.check-type.affected-rows.description");
    default:
      return "";
  }
}

function statusIcon(status: Advice_Level) {
  switch (status) {
    case Advice_Level.ERROR:
      return <XCircle className="h-4 w-4 text-error" />;
    case Advice_Level.WARNING:
      return <AlertCircle className="h-4 w-4 text-warning" />;
    default:
      return <CheckCircle className="h-4 w-4 text-success" />;
  }
}
