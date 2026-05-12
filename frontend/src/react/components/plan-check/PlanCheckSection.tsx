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
import { type ReactNode, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { Tooltip } from "@/react/components/ui/tooltip";
import { cn } from "@/react/lib/utils";
import {
  getFilteredResultGroups,
  getPlanCheckSummary,
  type PlanCheckSummary,
} from "@/react/pages/project/plan-detail/utils/planCheck";
import {
  getDateForPbTimestampProtoEs,
  getRuleLocalization,
  ruleTemplateMapV2,
  ruleTypeToString,
} from "@/types";
import {
  type PlanCheckRun,
  type PlanCheckRun_Result,
  PlanCheckRun_Result_Type,
} from "@/types/proto-es/v1/plan_service_pb";
import { SQLReviewRule_Type } from "@/types/proto-es/v1/review_config_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { extractDatabaseResourceName } from "@/utils/v1/database";

const PAGE_SIZE = 10;

interface PlanCheckSectionProps {
  planCheckRuns: PlanCheckRun[];
  // When provided, the section uses this summary instead of computing one
  // from planCheckRuns. Issue-detail uses this to fall back to
  // plan.planCheckRunStatusCount when planCheckRuns are not yet loaded.
  summaryOverride?: PlanCheckSummary;
  canRun: boolean;
  isRunning: boolean;
  runDisabled?: boolean;
  onRun: () => void | Promise<void>;
  // Called when the drawer opens; should refresh runs and return the latest.
  // Return value replaces what the drawer renders.
  onRefreshOnOpen?: () => Promise<PlanCheckRun[]>;
  // Optional trailing element rendered after status counts (e.g. affected rows).
  trailingSummary?: ReactNode;
  // If true, FAILED check runs without results render as a synthetic error
  // group. Used by plan-detail regular checks.
  includeRunFailure?: boolean;
  // Section heading style; defaults to uppercase.
  headingClassName?: string;
}

export function PlanCheckSection({
  planCheckRuns,
  summaryOverride,
  canRun,
  isRunning,
  runDisabled,
  onRun,
  onRefreshOnOpen,
  trailingSummary,
  includeRunFailure = false,
  headingClassName = "textlabel uppercase",
}: PlanCheckSectionProps) {
  const { t } = useTranslation();
  const [drawerOpen, setDrawerOpen] = useState(false);

  const summary = useMemo(
    () => summaryOverride ?? getPlanCheckSummary(planCheckRuns),
    [summaryOverride, planCheckRuns]
  );
  const hasAnyChecks = summary.total > 0;

  return (
    <div className="flex flex-col gap-y-1">
      <div className="flex items-center justify-between gap-2">
        <h3 className={headingClassName}>{t("plan.checks.self")}</h3>
        {canRun && (
          <Button
            disabled={isRunning || runDisabled}
            onClick={() => void onRun()}
            size="xs"
            variant="outline"
          >
            <Play className="h-3.5 w-3.5" />
            {t("plan.run")}
          </Button>
        )}
      </div>

      {hasAnyChecks ? (
        <div className="flex min-w-0 flex-wrap items-center gap-3">
          <button
            aria-label={t("plan.navigator.checks")}
            className="cursor-pointer text-left"
            onClick={() => setDrawerOpen(true)}
            type="button"
          >
            <PlanCheckSummaryRow summary={summary} />
          </button>
          {trailingSummary}
        </div>
      ) : (
        <span className="text-sm text-control-placeholder">
          {t("plan.overview.no-checks")}
        </span>
      )}

      <PlanCheckResultsDrawer
        includeRunFailure={includeRunFailure}
        onOpenChange={setDrawerOpen}
        onRefreshOnOpen={onRefreshOnOpen}
        open={drawerOpen}
        planCheckRuns={planCheckRuns}
      />
    </div>
  );
}

type StatusEntry = {
  count: number;
  icon: typeof XCircle;
  label: string;
  status: Advice_Level;
  textClass: string;
};

function getStatusEntries(
  summary: PlanCheckSummary,
  t: ReturnType<typeof useTranslation>["t"]
): StatusEntry[] {
  return [
    {
      count: summary.error,
      icon: XCircle,
      label: t("common.error"),
      status: Advice_Level.ERROR,
      textClass: "text-error",
    },
    {
      count: summary.warning,
      icon: AlertCircle,
      label: t("common.warning"),
      status: Advice_Level.WARNING,
      textClass: "text-warning",
    },
    {
      count: summary.success,
      icon: CheckCircle,
      label: t("common.success"),
      status: Advice_Level.SUCCESS,
      textClass: "text-success",
    },
  ].filter((entry) => entry.count > 0);
}

function PlanCheckSummaryRow({ summary }: { summary: PlanCheckSummary }) {
  const { t } = useTranslation();
  const entries = getStatusEntries(summary, t);
  return (
    <div className="flex min-w-0 flex-wrap items-center gap-3 text-sm">
      {summary.running > 0 && (
        <span className="flex items-center gap-1 text-control">
          <Loader2 className="size-4 animate-spin" />
          <span>{t("common.running")}</span>
          <span>{summary.running}</span>
        </span>
      )}
      {entries.map(({ count, icon: Icon, label, status, textClass }) => (
        <span className={cn("flex items-center gap-1", textClass)} key={status}>
          <Icon className="size-4" />
          <span>{label}</span>
          <span>{count}</span>
        </span>
      ))}
    </div>
  );
}

function PlanCheckFilterPills({
  onSelect,
  selectedStatus,
  summary,
}: {
  onSelect: (status: Advice_Level | undefined) => void;
  selectedStatus?: Advice_Level;
  summary: PlanCheckSummary;
}) {
  const { t } = useTranslation();
  const entries = getStatusEntries(summary, t);
  return (
    <div className="flex min-w-0 flex-wrap items-center gap-2 text-sm">
      {summary.running > 0 && (
        <div className="flex items-center gap-1 px-2 py-1 text-control">
          <Loader2 className="h-5 w-5 animate-spin" />
          <span>{t("common.running")}</span>
        </div>
      )}
      {entries.map(({ count, icon: Icon, label, status, textClass }) => {
        const isSelected = selectedStatus === status;
        return (
          <button
            className={cn(
              "flex cursor-pointer items-center gap-1 rounded-lg px-2 py-1 transition-colors",
              textClass,
              isSelected ? "bg-gray-100" : "hover:bg-gray-100"
            )}
            key={status}
            onClick={() => onSelect(isSelected ? undefined : status)}
            type="button"
          >
            <Icon className="h-5 w-5" />
            <span>{label}</span>
            <span>{count}</span>
          </button>
        );
      })}
    </div>
  );
}

interface PlanCheckResultsDrawerProps {
  open: boolean;
  onOpenChange: (next: boolean) => void;
  planCheckRuns: PlanCheckRun[];
  onRefreshOnOpen?: () => Promise<PlanCheckRun[]>;
  includeRunFailure?: boolean;
}

export function PlanCheckResultsDrawer({
  open,
  onOpenChange,
  planCheckRuns,
  onRefreshOnOpen,
  includeRunFailure = false,
}: PlanCheckResultsDrawerProps) {
  const { t } = useTranslation();
  const [selectedStatus, setSelectedStatus] = useState<
    Advice_Level | undefined
  >(undefined);
  const [displayCount, setDisplayCount] = useState(PAGE_SIZE);
  const [isRefreshing, setIsRefreshing] = useState(false);

  useEffect(() => {
    if (!open) return;
    setSelectedStatus(undefined);
    setDisplayCount(PAGE_SIZE);
    if (!onRefreshOnOpen) return;
    let canceled = false;
    const refresh = async () => {
      try {
        setIsRefreshing(true);
        await onRefreshOnOpen();
      } finally {
        if (!canceled) setIsRefreshing(false);
      }
    };
    void refresh();
    return () => {
      canceled = true;
    };
  }, [open, onRefreshOnOpen]);

  useEffect(() => {
    setDisplayCount(PAGE_SIZE);
  }, [selectedStatus]);

  const summary = useMemo(
    () => getPlanCheckSummary(planCheckRuns),
    [planCheckRuns]
  );
  const filteredResultGroups = useMemo(
    () =>
      getFilteredResultGroups({
        includeRunFailure,
        planCheckRuns,
        runFailureContent: t("common.failed"),
        runFailureTitle: t("common.failed"),
        selectedStatus,
      }),
    [includeRunFailure, planCheckRuns, selectedStatus, t]
  );
  const displayedResultGroups = filteredResultGroups.slice(0, displayCount);
  const remainingCount = Math.max(
    filteredResultGroups.length - displayCount,
    0
  );

  return (
    <Sheet onOpenChange={onOpenChange} open={open}>
      <SheetContent width="wide">
        <SheetHeader>
          <SheetTitle>{t("plan.navigator.checks")}</SheetTitle>
        </SheetHeader>
        <SheetBody className="gap-y-4">
          {summary.total > 0 && (
            <PlanCheckFilterPills
              onSelect={setSelectedStatus}
              selectedStatus={selectedStatus}
              summary={summary}
            />
          )}
          {isRefreshing && filteredResultGroups.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-control-light">
              <Loader2 className="mb-4 h-12 w-12 animate-spin opacity-50" />
            </div>
          ) : filteredResultGroups.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-control-light">
              <CheckCircle className="mb-4 h-12 w-12 opacity-50" />
              <div className="text-lg">
                {selectedStatus !== undefined
                  ? t("plan.checks.no-results-match-filters")
                  : t("plan.checks.no-check-results")}
              </div>
            </div>
          ) : (
            <>
              <div className="divide-y">
                {displayedResultGroups.map((group) => {
                  const description = getCheckTypeDescription(group.type, t);
                  return (
                    <div className="px-2 py-4" key={group.key}>
                      <div className="mb-2 flex flex-wrap items-start justify-between gap-2">
                        <div className="flex items-center gap-3">
                          <CheckTypeIcon type={group.type} />
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-medium">
                              {getCheckTypeLabel(group.type, t)}
                            </span>
                            {description && (
                              <Tooltip content={description}>
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
                        {group.target && (
                          <div className="min-w-0 max-w-[50%] truncate text-sm text-control-light">
                            {formatCheckTarget(group.target)}
                          </div>
                        )}
                      </div>
                      <div className="flex flex-col gap-y-2 pl-8">
                        {group.results.map((result, index) => (
                          <PlanCheckResultCard
                            key={`${group.key}-${index}`}
                            result={result}
                          />
                        ))}
                      </div>
                    </div>
                  );
                })}
              </div>
              {remainingCount > 0 && (
                <div className="flex justify-center py-4">
                  <button
                    className="cursor-pointer text-sm text-accent hover:underline"
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

export function PlanCheckResultCard({
  result,
}: {
  result: PlanCheckRun_Result;
}) {
  const { t } = useTranslation();
  const position =
    result.report?.case === "sqlReviewReport"
      ? result.report.value.startPosition
      : undefined;
  const affectedRows =
    result.report?.case === "sqlSummaryReport"
      ? result.report.value.affectedRows
      : undefined;
  const displayTitle = useMemo(
    () => resolveResultTitle(result.title, result.code, result.report?.case),
    [result.title, result.code, result.report?.case]
  );

  return (
    <div className="flex items-start gap-3 rounded-lg border border-control-border bg-control-bg px-3 py-2">
      <div className="mt-0.5 shrink-0">{statusIcon(result.status)}</div>
      <div className="flex min-w-0 flex-1 flex-col gap-y-1">
        <div className="text-sm font-medium text-main">{displayTitle}</div>
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
  return <Icon className={cn("h-5 w-5 text-control-light", className)} />;
}

function getCheckTypeLabel(
  type: PlanCheckRun_Result_Type,
  t: ReturnType<typeof useTranslation>["t"]
) {
  if (type === PlanCheckRun_Result_Type.STATEMENT_ADVISE) {
    return t("task.check-type.sql-review.self");
  }
  if (type === PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT) {
    return t("task.check-type.affected-rows.self");
  }
  if (type === PlanCheckRun_Result_Type.GHOST_SYNC) {
    return t("task.online-migration.self");
  }
  return t("plan.navigator.checks");
}

function getCheckTypeDescription(
  type: PlanCheckRun_Result_Type,
  t: ReturnType<typeof useTranslation>["t"]
) {
  if (type === PlanCheckRun_Result_Type.STATEMENT_ADVISE) {
    return t("task.check-type.sql-review.description");
  }
  if (type === PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT) {
    return t("task.check-type.affected-rows.description");
  }
  return "";
}

function statusIcon(status: Advice_Level) {
  if (status === Advice_Level.ERROR) {
    return <XCircle className="h-4 w-4 text-error" />;
  }
  if (status === Advice_Level.WARNING) {
    return <AlertCircle className="h-4 w-4 text-warning" />;
  }
  return <CheckCircle className="h-4 w-4 text-success" />;
}

function formatCheckTarget(target: string): string {
  if (!target) return "";
  try {
    return extractDatabaseResourceName(target).databaseName;
  } catch {
    return target;
  }
}

// Resolves SQL-review rule codes to their localized rule titles. Non-SQL-review
// results pass through unchanged. Mirrors what was previously duplicated inside
// IssueDetailChecks so all surfaces get localized titles.
function resolveResultTitle(
  title: string,
  code: number,
  reportCase: PlanCheckRun_Result["report"]["case"]
): string {
  const withCode = (message: string) =>
    code !== 0 ? `${message} #${code}` : message;
  if (title === "OK" || title === "Syntax error") return withCode(title);
  if (reportCase !== "sqlReviewReport") return title;
  if (!code) return title;
  const typeKey = title as keyof typeof SQLReviewRule_Type;
  const typeEnum = SQLReviewRule_Type[typeKey];
  if (typeEnum !== undefined) {
    for (const mapByType of ruleTemplateMapV2.values()) {
      const rule = mapByType.get(typeEnum);
      if (rule) {
        return withCode(
          getRuleLocalization(ruleTypeToString(rule.type), rule.engine).title
        );
      }
    }
  } else if (title.startsWith("builtin.")) {
    return withCode(getRuleLocalization(title).title);
  }
  return withCode(title);
}
