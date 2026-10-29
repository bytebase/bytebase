import type { TFunction } from "i18next";
import {
  CircleAlert,
  CircleCheck,
  CircleDashed,
  CircleX,
  EllipsisVertical,
  Info,
  Loader2,
  Play,
  RotateCw,
  TriangleAlert,
  UserCheck,
} from "lucide-react";
import { type ReactNode, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/app/router";
import { PROJECT_V1_ROUTE_SQL_REVIEW } from "@/app/router/handles";
import { HumanizeTs } from "@/components/HumanizeTs";
import { PlanCheckResultsDrawer } from "@/components/plan-check/PlanCheckSection";
import {
  STANDARD_RULE_TYPES,
  standardRuleText,
} from "@/components/sql-review/standardRules";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { badgeVariants } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app";
import { getTimeForPbTimestampProtoEs } from "@/types";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import {
  type IssueComment,
  IssueComment_ReviewMetadata_Priority,
  ReviewRun_Status,
  ReviewRun_Type,
} from "@/types/proto-es/v1/issue_service_pb";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";
import { extractDatabaseResourceName } from "@/utils";
import { engineNameV1 } from "@/utils/v1/instance";
import { useEffectiveStandardRules } from "../../hooks/useEffectiveStandardRules";
import { useFocusThread } from "../../hooks/useFocusThread";
import { usePlanCheckActions } from "../../hooks/usePlanCheckActions";
import type { PlanReviewState } from "../../hooks/usePlanReviewState";
import { useRuleReviewRerun } from "../../hooks/useRuleReviewRerun";
import { useSqlReviewV2 } from "../../hooks/useSqlReviewV2";
import {
  usePlanDetailStore,
  usePlanDetailStoreApi,
} from "../../shared/stores/usePlanDetailStore";
import { PROJECT_NAME_PREFIX } from "../../shell/constants";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import { countByName, findingsInOrder } from "../../utils/findingIndex";
import {
  type AutomatedReviewer,
  type AutomatedReviewRow,
  type FindingGroup,
  type FindingPriority,
  findingTitle,
  resolvedByPeople,
  rowNeedsAttention,
} from "../../utils/planGates";
import { openFindingsLabel, reviewerLabel } from "../../utils/reviewLabels";
import { reviewScope } from "../../utils/reviewScope";
import {
  priorityStyle,
  ReviewerIcon,
  reviewerName,
} from "../threads/ReviewResult";

// How many P0s resolved without a change a hover names before it counts the
// rest.
const NAMED_OVERRIDES = 3;
// How many databases the scope names before it counts them.
const NAMED_DATABASES = 3;
const NO_COMMENTS: IssueComment[] = [];

const runTypeOf = (reviewer: AutomatedReviewer): ReviewRun_Type | undefined =>
  reviewer === "rule"
    ? ReviewRun_Type.RULE
    : reviewer === "ai"
      ? ReviewRun_Type.AI
      : undefined;

// The machine reviewers, one line each, between Approval and Activity: the
// conclusion, the open findings by priority, what people resolved, and what
// the reviewer covered on hover. The findings themselves live in the editor;
// every count opens it there.
export function AutomatedReviewSection({ state }: { state: PlanReviewState }) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const { allowRunChecks, isRunningChecks, runChecks } = usePlanCheckActions();
  const ruleReview = useSqlReviewV2()?.ruleReview;
  const { rows } = state;
  const attention = rows.filter(rowNeedsAttention).length;
  const [checksDrawerOpen, setChecksDrawerOpen] = useState(false);
  const [confirmChecks, setConfirmChecks] = useState(false);
  const ruleRow = rows.find((row) => row.reviewer === "rule");
  const ruleResolved = ruleRow ? resolvedByPeople(ruleRow.findings) : 0;

  // Running the checks also starts the rule review on the server, which
  // posts its findings again and reopens what people resolved.
  const rerunAll = async () => {
    await runChecks();
    await ruleReview?.reload();
  };
  const requestRerunChecks = () => {
    if (ruleResolved > 0) setConfirmChecks(true);
    else void rerunAll();
  };

  if (rows.length === 0) return null;

  return (
    <div
      className="flex flex-col gap-y-2 px-4 pt-4"
      data-testid="automated-review"
    >
      <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
        <h4 className="text-sm font-medium text-main">
          {t("plan.automated-review.self")}
        </h4>
        <span className="text-sm text-control-light">
          {attention > 0
            ? t("plan.automated-review.needs-attention", {
                count: attention,
                total: rows.length,
              })
            : t("plan.automated-review.nothing-needs-attention")}
        </span>
        <div className="flex-1" />
        <Button
          appearance="link"
          className="text-control-light"
          onClick={() =>
            void router.push({
              name: PROJECT_V1_ROUTE_SQL_REVIEW,
              params: { projectId: page.projectId },
            })
          }
          size="xs"
        >
          {t("common.configure")}
        </Button>
        {allowRunChecks && (
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button
                  aria-label={t("common.more")}
                  appearance="secondary"
                  size="xs"
                />
              }
            >
              <EllipsisVertical className="size-4" />
            </DropdownMenuTrigger>
            <DropdownMenuContent>
              <DropdownMenuItem
                disabled={isRunningChecks}
                onClick={requestRerunChecks}
              >
                {t("plan.automated-review.rerun-all")}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>
      <div className="flex flex-col divide-y rounded-sm border">
        {rows.map((row) => (
          <ReviewerRow
            key={row.reviewer}
            onRerunChecks={requestRerunChecks}
            onViewChecks={() => setChecksDrawerOpen(true)}
            row={row}
            state={state}
          />
        ))}
      </div>
      <PlanCheckResultsDrawer
        includeRunFailure
        onOpenChange={setChecksDrawerOpen}
        open={checksDrawerOpen}
        planCheckRuns={page.planCheckRuns}
      />
      <RerunConfirmDialog
        confirmLabel={t("plan.automated-review.rerun-all")}
        description={t("plan.automated-review.rerun-all-description", {
          count: ruleResolved,
        })}
        onConfirm={() => void rerunAll()}
        onOpenChange={setConfirmChecks}
        open={confirmChecks}
        title={t("plan.automated-review.rerun-all-title")}
      />
    </div>
  );
}

function RerunConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel,
  onConfirm,
  disabled = false,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: string;
  confirmLabel: string;
  onConfirm: () => void;
  disabled?: boolean;
}) {
  const { t } = useTranslation();
  return (
    <AlertDialog onOpenChange={onOpenChange} open={open}>
      <AlertDialogContent>
        <AlertDialogTitle>{title}</AlertDialogTitle>
        <AlertDialogDescription>{description}</AlertDialogDescription>
        <AlertDialogFooter>
          <Button appearance="secondary" onClick={() => onOpenChange(false)}>
            {t("common.cancel")}
          </Button>
          <Button
            disabled={disabled}
            onClick={() => {
              onOpenChange(false);
              onConfirm();
            }}
          >
            {confirmLabel}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function ReviewerRow({
  row,
  state,
  onRerunChecks,
  onViewChecks,
}: {
  row: AutomatedReviewRow;
  state: PlanReviewState;
  onRerunChecks: () => void;
  onViewChecks: () => void;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const storeApi = usePlanDetailStoreApi();
  const { focusInStatement } = useFocusThread();
  const ruleReview = useSqlReviewV2()?.ruleReview;
  const { allowRunChecks, isRunningChecks } = usePlanCheckActions();
  const rerunRule = useRuleReviewRerun();
  const runType = runTypeOf(row.reviewer);
  const comments = useAppStore((s) =>
    page.issue ? s.getIssueComments(page.issue.name) : NO_COMMENTS
  );
  const [confirm, setConfirm] = useState<"rule" | "ai">();
  const rowRef = useRef<HTMLDivElement>(null);

  // Activity's "Show" brings this row into view.
  const reviewRowRequest = usePlanDetailStore((s) => s.reviewRowRequest);
  useEffect(() => {
    if (reviewRowRequest?.reviewer !== row.reviewer) return;
    rowRef.current?.scrollIntoView({ behavior: "smooth", block: "center" });
  }, [reviewRowRequest, row.reviewer]);

  // A count opens the editor on the first finding of its group and narrows
  // the walker to that group.
  const openGroup = (group: FindingGroup) => {
    if (runType === undefined) return;
    const first = findingsInOrder(comments, runType, page.plan.specs, group)[0];
    if (!first) return;
    storeApi.getState().setWalkerFilter({ runType, group });
    focusInStatement(first);
  };

  const ai = state.ai;
  let name: ReactNode;
  if (row.reviewer === "rule" || row.reviewer === "ai") {
    name = (
      <>
        <ReviewerIcon
          className="size-4 shrink-0 text-control-light"
          runType={runType ?? ReviewRun_Type.RULE}
        />
        {reviewerName(t, runType ?? ReviewRun_Type.RULE)}
      </>
    );
  } else {
    name = reviewerLabel(t, row.reviewer);
  }

  const actions: ReactNode[] = [];
  if (row.reviewer === "rule") {
    if (ruleReview?.loadFailed) {
      actions.push(
        <Button
          appearance="outline"
          key="retry"
          onClick={() => void ruleReview.reload()}
          size="xs"
        >
          <RotateCw className="size-3.5" />
          {t("plan.automated-review.retry")}
        </Button>
      );
    } else if (
      rerunRule.allowed &&
      row.conclusion !== "running" &&
      row.conclusion !== "loading"
    ) {
      // A rerun posts the findings again, reopening what people resolved.
      const start = () => {
        if (resolvedByPeople(row.findings) > 0) setConfirm("rule");
        else void rerunRule.run();
      };
      actions.push(
        <RunButton
          conclusion={row.conclusion}
          disabled={rerunRule.disabled}
          key="run"
          onClick={start}
        />
      );
    }
  }
  if (
    row.reviewer === "ai" &&
    ai.allowRun &&
    row.conclusion !== "running" &&
    row.conclusion !== "loading"
  ) {
    // Every finished run is paid for in model calls, so running it again
    // asks first.
    const settled =
      ai.run?.status === ReviewRun_Status.DONE && !ai.run.outdated;
    actions.push(
      <Tooltip
        content={ai.editing ? t("plan.ai-review.save-first") : undefined}
        key="run"
      >
        <RunButton
          conclusion={settled ? "passed" : row.conclusion}
          disabled={!ai.canStart}
          label={settled ? undefined : t("plan.ai-review.run")}
          onClick={() => (settled ? setConfirm("ai") : void ai.runReview())}
        />
      </Tooltip>
    );
  }
  // Plan checks run together, and the run restarts the rule review too.
  if (runType === undefined && allowRunChecks && row.conclusion !== "running") {
    actions.push(
      <RunButton
        conclusion={row.conclusion}
        disabled={isRunningChecks}
        key="run"
        onClick={onRerunChecks}
      />
    );
  }
  if (runType === undefined) {
    actions.push(
      <Button
        appearance="link"
        className="text-control-light"
        key="details"
        onClick={onViewChecks}
        size="xs"
      >
        {t("plan.automated-review.details")}
      </Button>
    );
  }

  // A reviewer without a result has covered nothing yet.
  const hasResult =
    row.conclusion !== "not-run" &&
    row.conclusion !== "loading" &&
    row.conclusion !== "running";

  return (
    <div
      className="flex flex-wrap items-center gap-x-3 gap-y-1 px-3 py-2 text-sm"
      data-testid={`automated-review-${row.reviewer}`}
      ref={rowRef}
    >
      <ConclusionIcon row={row} />
      <span className="flex flex-1 items-center gap-x-1.5 font-medium text-main sm:w-44 sm:flex-none">
        {name}
      </span>
      {/* Below the name on a narrow screen, lined up with it. */}
      <span className="order-last flex min-w-0 basis-full flex-wrap items-center gap-x-1.5 gap-y-1 pl-7 text-control sm:order-none sm:flex-1 sm:pl-0">
        <ConclusionText
          comments={comments}
          onOpen={openGroup}
          row={row}
          runType={runType}
        />
        {(row.reviewer === "rule" || row.reviewer === "ai") && hasResult && (
          <ScopeInfo reviewer={row.reviewer} />
        )}
      </span>
      {actions.length > 0 && (
        <span className="flex shrink-0 items-center gap-x-2">{actions}</span>
      )}
      <RerunConfirmDialog
        confirmLabel={t("plan.automated-review.rerun")}
        description={
          confirm === "ai"
            ? t("plan.ai-review.run-again-description")
            : t("plan.automated-review.rerun-reopens", {
                count: resolvedByPeople(row.findings),
              })
        }
        disabled={confirm === "ai" ? !ai.canStart : rerunRule.disabled}
        onConfirm={() =>
          confirm === "ai" ? void ai.runReview() : void rerunRule.run()
        }
        onOpenChange={(open) => !open && setConfirm(undefined)}
        open={confirm !== undefined}
        title={
          confirm === "ai"
            ? t("plan.ai-review.run-again-title")
            : t("plan.automated-review.rerun-rule-title")
        }
      />
    </div>
  );
}

// Run before a first result or after the SQL changed, Retry after a failed
// run, Rerun once there is a result.
function RunButton({
  conclusion,
  disabled,
  onClick,
  label,
}: {
  conclusion: AutomatedReviewRow["conclusion"];
  disabled: boolean;
  onClick: () => void;
  label?: string;
}) {
  const { t } = useTranslation();
  const first = conclusion === "not-run" || conclusion === "outdated";
  const text = first
    ? t("common.run")
    : conclusion === "failed"
      ? t("plan.automated-review.retry")
      : t("plan.automated-review.rerun");
  return (
    <Button
      appearance="outline"
      aria-label={label}
      disabled={disabled}
      onClick={onClick}
      size="xs"
    >
      {first ? (
        <Play className="size-3.5" />
      ) : (
        <RotateCw className="size-3.5" />
      )}
      {text}
    </Button>
  );
}

// Status color sits on the icon; the text stays readable. A P0 resolved
// without a change keeps the row amber after everything is resolved.
function ConclusionIcon({ row }: { row: AutomatedReviewRow }) {
  const className = "size-4 shrink-0";
  switch (row.conclusion) {
    case "passed":
      return <CircleCheck className={cn(className, "text-success")} />;
    case "resolved":
      return row.findings.overridden > 0 ? (
        <TriangleAlert className={cn(className, "text-warning")} />
      ) : (
        <UserCheck className={cn(className, "text-control-light")} />
      );
    case "open":
      return (
        <CircleAlert
          className={cn(
            className,
            row.findings.p0 > 0 ? "text-error" : "text-warning"
          )}
        />
      );
    case "failed":
      return <CircleX className={cn(className, "text-error")} />;
    case "running":
    case "loading":
      return (
        <Loader2 className={cn(className, "animate-spin text-control-light")} />
      );
    default:
      return <CircleDashed className={cn(className, "text-control-light")} />;
  }
}

const PRIORITY: Record<FindingPriority, IssueComment_ReviewMetadata_Priority> =
  {
    p0: IssueComment_ReviewMetadata_Priority.P0,
    p1: IssueComment_ReviewMetadata_Priority.P1,
    p2: IssueComment_ReviewMetadata_Priority.P2,
  };
const PRIORITIES: readonly FindingPriority[] = ["p0", "p1", "p2"];

// A count of findings that opens them in the statement editor, with what
// they are and where the click goes on hover.
function FindingCount({
  label,
  hint,
  className,
  onClick,
  children,
}: {
  label: string;
  hint: ReactNode;
  className?: string;
  onClick: () => void;
  children: ReactNode;
}) {
  const { t } = useTranslation();
  return (
    <Tooltip
      content={
        <span className="flex flex-col gap-y-1">
          {hint}
          <span className="text-main-text/70">
            {t("plan.automated-review.opens-in-sql")}
          </span>
        </span>
      }
      popupClassName="max-w-72"
      render={
        <Button
          appearance="link"
          aria-label={label}
          className={cn("h-auto p-0 font-normal", className)}
          onClick={onClick}
          size="xs"
        />
      }
    >
      {children}
    </Tooltip>
  );
}

// A finding's name: its rule, or for the AI review, its title.
const findingName = (t: TFunction, comment: IssueComment): string => {
  const ruleType =
    comment.reviewMetadata?.ruleType ??
    ReviewRuleType.REVIEW_RULE_TYPE_UNSPECIFIED;
  return ruleType === ReviewRuleType.REVIEW_RULE_TYPE_UNSPECIFIED
    ? findingTitle(comment)
    : standardRuleText(t, ruleType).title;
};

// The names of a group's findings for a hover: by line for the few P0s that
// need a look, by count per rule for the rest.
function useGroupNames(
  comments: readonly IssueComment[],
  runType: ReviewRun_Type | undefined,
  group: "overridden" | "resolved",
  count: number
): string {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  return useMemo(() => {
    if (runType === undefined || count === 0) return "";
    const findings = findingsInOrder(comments, runType, page.plan.specs, group);
    if (group === "overridden") {
      const named = findings.slice(0, NAMED_OVERRIDES).map((comment) => {
        const line = comment.statementAnchor?.startPosition?.line;
        return line
          ? t("plan.findings.name-at-line", {
              name: findingName(t, comment),
              line,
            })
          : findingName(t, comment);
      });
      const rest = findings.length - named.length;
      if (rest > 0) named.push(t("plan.findings.n-more", { count: rest }));
      return named.join(t("plan.findings.separator"));
    }
    return countByName(findings, (comment) => findingName(t, comment))
      .map(([name, times]) =>
        times > 1 ? t("plan.findings.name-times", { name, count: times }) : name
      )
      .join(t("plan.findings.separator"));
  }, [comments, count, group, page.plan.specs, runType, t]);
}

function ConclusionText({
  row,
  runType,
  comments,
  onOpen,
}: {
  row: AutomatedReviewRow;
  runType: ReviewRun_Type | undefined;
  comments: readonly IssueComment[];
  onOpen: (group: FindingGroup) => void;
}) {
  const { t } = useTranslation();
  const { findings } = row;
  const overriddenNames = useGroupNames(
    comments,
    runType,
    "overridden",
    findings.overridden
  );
  const resolvedNames = useGroupNames(
    comments,
    runType,
    "resolved",
    findings.resolved
  );
  const time = row.time && (
    <HumanizeTs
      className="text-control-light"
      tsMs={getTimeForPbTimestampProtoEs(row.time)}
    />
  );
  const isPlanCheck = runType === undefined;
  const parts: ReactNode[] = [];
  switch (row.conclusion) {
    case "loading":
      parts.push(t("common.loading"));
      break;
    case "not-run":
      parts.push(t("plan.automated-review.not-run"));
      break;
    case "running":
      parts.push(t("plan.automated-review.running"));
      break;
    case "outdated":
      parts.push(t("plan.automated-review.outdated"));
      break;
    case "failed":
      parts.push(
        <span className="break-words whitespace-normal text-main" key="error">
          {row.error || t("plan.automated-review.failed")}
        </span>
      );
      break;
    case "passed":
      parts.push(
        isPlanCheck && row.targets
          ? t("plan.automated-review.passed-on", { count: row.targets })
          : t("plan.automated-review.no-findings")
      );
      if (row.warnings) {
        parts.push(
          t("plan.automated-review.n-warnings", { count: row.warnings })
        );
      }
      break;
    case "open":
    case "resolved":
      if (isPlanCheck) {
        parts.push(
          <span className="font-medium text-main" key="errors">
            {openFindingsLabel(t, row)}
          </span>
        );
        if (row.warnings) {
          parts.push(
            t("plan.automated-review.n-warnings", { count: row.warnings })
          );
        }
        break;
      }
      if (findings.open > 0) {
        parts.push(
          <span className="flex items-center gap-x-1.5" key="open">
            {PRIORITIES.filter((priority) => findings[priority] > 0).map(
              (priority) => (
                <PriorityCount
                  count={findings[priority]}
                  key={priority}
                  onClick={() => onOpen(priority)}
                  priority={priority}
                />
              )
            )}
          </span>
        );
      }
      if (findings.resolved > 0) {
        parts.push(
          <FindingCount
            className="text-sm text-control underline decoration-control-border underline-offset-2 hover:text-main"
            hint={resolvedNames}
            key="resolved"
            label={t("plan.findings.n-resolved", { count: findings.resolved })}
            onClick={() => onOpen("resolved")}
          >
            {t("plan.findings.n-resolved", { count: findings.resolved })}
          </FindingCount>
        );
      }
      if (findings.overridden > 0) {
        parts.push(
          <FindingCount
            className="text-sm text-main underline decoration-warning underline-offset-2"
            hint={overriddenNames}
            key="overridden"
            label={t("plan.findings.n-overridden", {
              count: findings.overridden,
            })}
            onClick={() => onOpen("overridden")}
          >
            <TriangleAlert className="size-3.5 shrink-0 text-warning" />
            {t("plan.findings.n-overridden", { count: findings.overridden })}
          </FindingCount>
        );
      }
      break;
  }
  if (time && row.conclusion !== "loading" && row.conclusion !== "not-run") {
    parts.push(time);
  }
  return (
    <>
      {parts.map((part, index) => (
        <span
          className="flex items-center gap-x-1.5 whitespace-nowrap"
          key={index}
        >
          {index > 0 && <span className="text-control-light">·</span>}
          {part}
        </span>
      ))}
    </>
  );
}

// "P0 2", colored as the priority badge on each finding.
function PriorityCount({
  priority,
  count,
  onClick,
}: {
  priority: FindingPriority;
  count: number;
  onClick: () => void;
}) {
  const { t } = useTranslation();
  const style = priorityStyle(t, PRIORITY[priority]);
  return (
    <FindingCount
      className={badgeVariants({ variant: style?.variant })}
      hint={style?.description}
      label={t("plan.findings.n-priority", {
        count,
        priority: priority.toUpperCase(),
      })}
      onClick={onClick}
    >
      {priority.toUpperCase()}
      <span className="font-semibold">{count}</span>
    </FindingCount>
  );
}

// What the reviewer covered, on hover: the databases, and for the rule
// review, the rules it applies. A "No findings" means little without it.
function ScopeInfo({ reviewer }: { reviewer: "rule" | "ai" }) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const databasesByName = useAppStore((s) => s.databasesByName);
  const dbGroupsByName = useAppStore((s) => s.dbGroupsByName);
  const dbGroupViewByName = useAppStore((s) => s.dbGroupViewByName);
  const rules = useEffectiveStandardRules(
    `${PROJECT_NAME_PREFIX}${page.projectId}`
  );
  const scope = useMemo(
    () =>
      reviewScope(page.plan, {
        reviewer,
        sampleSize: page.project.ciSamplingSize ?? 0,
        matchedDatabases: (group) =>
          dbGroupViewByName[group] === DatabaseGroupView.FULL
            ? (dbGroupsByName[group]?.matchedDatabases.map(
                (database) => database.name
              ) ?? [])
            : undefined,
        engineOf: (database) =>
          databasesByName[database]?.instanceResource?.engine,
      }),
    [
      databasesByName,
      dbGroupViewByName,
      dbGroupsByName,
      page.plan,
      page.project.ciSamplingSize,
      reviewer,
    ]
  );
  if (scope.databases.length === 0) return null;

  const names =
    scope.databases.length <= NAMED_DATABASES
      ? scope.databases
          .map((database) => extractDatabaseResourceName(database).databaseName)
          .join(t("plan.findings.separator"))
      : t("plan.automated-review.n-databases", {
          count: scope.databases.length,
        });
  const lines = [
    scope.engines.length > 0
      ? t("plan.automated-review.scope-with-engines", {
          databases: names,
          engines: scope.engines.map(engineNameV1).join(", "),
        })
      : t("plan.automated-review.scope", { databases: names }),
  ];
  if (scope.total > scope.databases.length) {
    lines.push(
      t("plan.automated-review.sampled", {
        count: scope.databases.length,
        total: scope.total,
      })
    );
  }
  if (reviewer === "rule" && rules) {
    const counts = {
      on: rules.rules.length,
      total: STANDARD_RULE_TYPES.length,
    };
    lines.push(
      rules.source === "project"
        ? t("plan.automated-review.project-rules", counts)
        : t("plan.automated-review.workspace-rules", counts)
    );
  }
  return (
    <Tooltip
      content={
        <span className="flex flex-col gap-y-1" data-testid="review-scope">
          {lines.map((line) => (
            <span key={line}>{line}</span>
          ))}
        </span>
      }
      popupClassName="max-w-72"
      render={
        <Button
          appearance="link"
          aria-label={t("plan.automated-review.scope-label")}
          className="h-auto cursor-help p-0 text-control-light hover:text-control"
          size="xs"
        />
      }
    >
      <Info className="size-3.5" />
    </Tooltip>
  );
}
