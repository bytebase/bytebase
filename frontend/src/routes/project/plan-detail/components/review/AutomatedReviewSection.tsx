import type { TFunction } from "i18next";
import {
  CircleAlert,
  CircleCheck,
  CircleDashed,
  CircleSlash,
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
import { Badge } from "@/components/ui/badge";
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
} from "@/types/proto-es/v1/issue_service_pb";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";
import { extractDatabaseResourceName } from "@/utils";
import { engineNameV1 } from "@/utils/v1/instance";
import { useFocusThread } from "../../hooks/useFocusThread";
import { usePlanCheckActions } from "../../hooks/usePlanCheckActions";
import type { PlanReviewState } from "../../hooks/usePlanReviewState";
import { useReviewRerun } from "../../hooks/useReviewRerun";
import { useSqlReviewV2 } from "../../hooks/useSqlReviewV2";
import {
  usePlanDetailStore,
  usePlanDetailStoreApi,
} from "../../shared/stores/usePlanDetailStore";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import { countByName, findingsInOrder } from "../../utils/findingIndex";
import {
  type AutomatedReviewRow,
  type FindingGroup,
  type FindingPriority,
  findingTitle,
  rowNeedsAttention,
} from "../../utils/planGates";
import { openFindingsLabel, reviewerLabel } from "../../utils/reviewLabels";
import { aiReviewEnabled } from "../../utils/reviewRun";
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

// The review and the plan checks, one line each, between Approval and
// Activity: the conclusion, the open findings by priority, what people
// resolved, and what the review covered on hover. The findings themselves
// live in the editor; every count opens it there.
export function AutomatedReviewSection({ state }: { state: PlanReviewState }) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const { allowRunChecks, isRunningChecks, runChecks } = usePlanCheckActions();
  const rerun = useReviewRerun();
  const { rows } = state;
  const attention = rows.filter(rowNeedsAttention).length;
  // A review that is off checked nothing, so it is neither counted nor
  // summed up as needing no attention.
  const reviewed = rows.filter((row) => row.conclusion !== "off");
  const [checksDrawerOpen, setChecksDrawerOpen] = useState(false);
  const [confirmAllOpen, setConfirmAllOpen] = useState(false);
  const requestRerunChecks = () => void runChecks();
  // Running the checks does not start the review, so Re-run all starts both.
  // It leaves the review alone where its own Rerun is not offered. The review
  // starts first: the checks can finish at once, and the server would then
  // create the rollout on the review that was about to be replaced.
  const reviewReruns =
    rerun.allowed &&
    !rerun.disabled &&
    rows.some((row) => row.reviewer === "review" && row.conclusion !== "off");
  const rerunAll = async () => {
    if (reviewReruns) await rerun.run();
    await runChecks();
  };
  const requestRerunAll = () =>
    reviewReruns && rerun.confirmFirst
      ? setConfirmAllOpen(true)
      : void rerunAll();

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
        {(attention > 0 || reviewed.length === rows.length) && (
          <span
            className="text-sm text-control-light"
            data-testid="automated-review-summary"
          >
            {attention > 0
              ? t("plan.automated-review.needs-attention", {
                  count: attention,
                  total: reviewed.length,
                })
              : t("plan.automated-review.nothing-needs-attention")}
          </span>
        )}
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
                onClick={requestRerunAll}
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
        description={t("plan.automated-review.run-again-description")}
        disabled={isRunningChecks || !reviewReruns}
        onConfirm={() => void rerunAll()}
        onOpenChange={setConfirmAllOpen}
        open={confirmAllOpen}
        title={t("plan.automated-review.run-again-title")}
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
  onRerunChecks,
  onViewChecks,
}: {
  row: AutomatedReviewRow;
  onRerunChecks: () => void;
  onViewChecks: () => void;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const storeApi = usePlanDetailStoreApi();
  const { focusInStatement } = useFocusThread();
  const review = useSqlReviewV2()?.review;
  const { allowRunChecks, isRunningChecks } = usePlanCheckActions();
  const rerun = useReviewRerun();
  const isReview = row.reviewer === "review";
  const comments = useAppStore((s) =>
    page.issue ? s.getIssueComments(page.issue.name) : NO_COMMENTS
  );
  const [confirmOpen, setConfirmOpen] = useState(false);
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
    if (!isReview) return;
    const first = findingsInOrder(comments, page.plan.specs, group)[0];
    if (!first) return;
    storeApi.getState().setWalkerFilter({ group });
    focusInStatement(first);
  };

  const name: ReactNode = isReview ? (
    <>
      <ReviewerIcon className="size-4 shrink-0 text-control-light" />
      {reviewerName(t)}
    </>
  ) : (
    reviewerLabel(t, row.reviewer)
  );

  const actions: ReactNode[] = [];
  if (isReview) {
    if (review?.loadFailed) {
      actions.push(
        <Button
          appearance="outline"
          key="retry"
          onClick={() => void review.reload()}
          size="xs"
        >
          <RotateCw className="size-3.5" />
          {t("plan.automated-review.retry")}
        </Button>
      );
    } else if (
      rerun.allowed &&
      row.conclusion !== "running" &&
      row.conclusion !== "loading" &&
      row.conclusion !== "off"
    ) {
      actions.push(
        <Tooltip
          content={
            rerun.editing ? t("plan.automated-review.save-first") : undefined
          }
          key="run"
        >
          <RunButton
            conclusion={row.conclusion}
            disabled={rerun.disabled}
            onClick={() =>
              rerun.confirmFirst ? setConfirmOpen(true) : void rerun.run()
            }
          />
        </Tooltip>
      );
    }
  }
  // Plan checks run together. The review has its own Rerun.
  if (!isReview && allowRunChecks && row.conclusion !== "running") {
    actions.push(
      <RunButton
        conclusion={row.conclusion}
        disabled={isRunningChecks}
        key="run"
        onClick={onRerunChecks}
      />
    );
  }
  if (!isReview) {
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

  // A review without a result has covered nothing yet, and one that is off
  // covers nothing at all.
  const hasResult =
    row.conclusion !== "not-run" &&
    row.conclusion !== "loading" &&
    row.conclusion !== "running" &&
    row.conclusion !== "off";

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
        <ConclusionText comments={comments} onOpen={openGroup} row={row} />
        {isReview && hasResult && <ScopeInfo />}
      </span>
      {actions.length > 0 && (
        <span className="flex shrink-0 items-center gap-x-2">{actions}</span>
      )}
      {isReview && (
        <RerunConfirmDialog
          confirmLabel={t("plan.automated-review.rerun")}
          description={t("plan.automated-review.run-again-description")}
          disabled={rerun.disabled}
          onConfirm={() => void rerun.run()}
          onOpenChange={setConfirmOpen}
          open={confirmOpen}
          title={t("plan.automated-review.run-again-title")}
        />
      )}
    </div>
  );
}

// Run before a first result or after the SQL changed, Retry after a failed
// run, Rerun once there is a result.
function RunButton({
  conclusion,
  disabled,
  onClick,
}: {
  conclusion: AutomatedReviewRow["conclusion"];
  disabled: boolean;
  onClick: () => void;
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
    case "off":
      return <CircleSlash className={cn(className, "text-control-light")} />;
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

// A finding's name: its standard rule, or else its title.
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
  isReview: boolean,
  group: "overridden" | "resolved",
  count: number
): string {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  return useMemo(() => {
    if (!isReview || count === 0) return "";
    const findings = findingsInOrder(comments, page.plan.specs, group);
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
  }, [comments, count, group, isReview, page.plan.specs, t]);
}

function ConclusionText({
  row,
  comments,
  onOpen,
}: {
  row: AutomatedReviewRow;
  comments: readonly IssueComment[];
  onOpen: (group: FindingGroup) => void;
}) {
  const { t } = useTranslation();
  const { findings } = row;
  const isReview = row.reviewer === "review";
  const overriddenNames = useGroupNames(
    comments,
    isReview,
    "overridden",
    findings.overridden
  );
  const resolvedNames = useGroupNames(
    comments,
    isReview,
    "resolved",
    findings.resolved
  );
  // Reviewer rows finish seconds apart, so ages in seconds would tick out of
  // step on adjacent lines.
  const time = row.time && (
    <HumanizeTs
      className="text-control-light"
      precision="minute"
      tsMs={getTimeForPbTimestampProtoEs(row.time)}
    />
  );
  // The review's findings: the open ones by priority, then what people
  // resolved.
  const findingParts = (): ReactNode[] => {
    const counts: ReactNode[] = [];
    if (findings.open > 0) {
      counts.push(
        <span className="flex items-center gap-x-3" key="open">
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
      counts.push(
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
      counts.push(
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
    return counts;
  };
  const parts: ReactNode[] = [];
  switch (row.conclusion) {
    case "loading":
      parts.push(t("common.loading"));
      break;
    case "not-run":
      parts.push(t("plan.automated-review.not-run"));
      break;
    case "running":
      parts.push(
        isReview
          ? t("plan.automated-review.reviewing")
          : t("plan.automated-review.running")
      );
      break;
    case "outdated":
      parts.push(t("plan.automated-review.outdated"));
      break;
    case "off":
      parts.push(t("plan.automated-review.off-for-project"));
      break;
    case "failed":
      parts.push(
        <span className="break-words whitespace-normal text-main" key="error">
          {row.error || t("plan.automated-review.failed")}
        </span>
      );
      // A failed run changes no finding, so the earlier ones still stand.
      if (isReview) parts.push(...findingParts());
      break;
    case "passed":
      parts.push(
        !isReview && row.targets
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
      if (!isReview) {
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
      parts.push(...findingParts());
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

// The priority badge each finding carries, then how many are open. Only the
// count underlines on hover: a link's underline would run through the badge.
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
      className="group gap-x-1 hover:no-underline"
      hint={style?.description}
      label={t("plan.findings.n-priority", {
        count,
        priority: priority.toUpperCase(),
      })}
      onClick={onClick}
    >
      <Badge variant={style?.variant}>{priority.toUpperCase()}</Badge>
      <span className="text-sm text-control-light underline-offset-2 group-hover:text-main group-hover:underline">
        {count}
      </span>
    </FindingCount>
  );
}

// What the review covered, on hover: a pass over three databases says very
// little without it. The review reads a sample when the project sets a sample
// size, and the AI review reads engines the standard rules do not.
function ScopeInfo() {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const rules = useSqlReviewV2()?.standardRules;
  const aiReviewOn = useAppStore((s) => aiReviewEnabled(s.serverInfo));
  const databasesByName = useAppStore((s) => s.databasesByName);
  const dbGroupsByName = useAppStore((s) => s.dbGroupsByName);
  const dbGroupViewByName = useAppStore((s) => s.dbGroupViewByName);
  // Rules not known yet count as on.
  const rulesOn = !rules || rules.rules.length > 0;
  const scope = useMemo(
    () =>
      reviewScope(page.plan, {
        reviewers: { rules: rulesOn, ai: aiReviewOn },
        sampleSize: page.project.ciSamplingSize ?? 0,
        matchedDatabases: (group: string) =>
          dbGroupViewByName[group] === DatabaseGroupView.FULL
            ? (dbGroupsByName[group]?.matchedDatabases.map(
                (database) => database.name
              ) ?? [])
            : undefined,
        engineOf: (database: string) =>
          databasesByName[database]?.instanceResource?.engine,
      }),
    [
      aiReviewOn,
      databasesByName,
      dbGroupViewByName,
      dbGroupsByName,
      page.plan,
      page.project.ciSamplingSize,
      rulesOn,
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
  if (rules) {
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
