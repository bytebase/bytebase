import {
  CircleAlert,
  CircleCheck,
  CircleDashed,
  CircleX,
  EllipsisVertical,
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
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app";
import { getTimeForPbTimestampProtoEs } from "@/types";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import {
  type IssueComment,
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
import { findingsInOrder } from "../../utils/findingIndex";
import {
  type AutomatedReviewer,
  type AutomatedReviewRow,
  type FindingSet,
  findingTitle,
  rowNeedsAttention,
} from "../../utils/planGates";
import { openActionParts, reviewerLabel } from "../../utils/reviewLabels";
import { reviewScope } from "../../utils/reviewScope";
import { ReviewerIcon, reviewerName } from "../threads/ReviewResult";

// How many overridden findings the warning names before it counts the rest.
const NAMED_OVERRIDES = 3;
// How many databases the scope line names before it counts them.
const NAMED_DATABASES = 3;
const NO_COMMENTS: IssueComment[] = [];

const runTypeOf = (reviewer: AutomatedReviewer): ReviewRun_Type | undefined =>
  reviewer === "rule"
    ? ReviewRun_Type.RULE
    : reviewer === "ai"
      ? ReviewRun_Type.AI
      : undefined;

// The machine reviewers, one row each, between Approval and Activity. Each
// row gives one conclusion and what is left to do, split by what a finding
// asks: fix the change, accept a risk, or consider advice. Below it, what
// people already decided, which the editor folds away, and what the reviewer
// covered. The findings themselves live in the editor; every count opens it.
export function AutomatedReviewSection({ state }: { state: PlanReviewState }) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const { allowRunChecks, isRunningChecks, runChecks } = usePlanCheckActions();
  const ruleReview = useSqlReviewV2()?.ruleReview;
  const { rows } = state;
  const attention = rows.filter(rowNeedsAttention).length;
  const [checksDrawerOpen, setChecksDrawerOpen] = useState(false);
  const [confirmChecks, setConfirmChecks] = useState(false);
  const ruleResolved =
    rows.find((row) => row.reviewer === "rule")?.findings.resolved ?? 0;

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

  // A count opens the editor on the first finding of its set and narrows
  // the walker to that set.
  const openSet = (set: FindingSet) => {
    if (runType === undefined) return;
    const first = findingsInOrder(comments, runType, page.plan.specs, set)[0];
    if (!first) return;
    storeApi.getState().setWalkerFilter({ runType, set });
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
        if (row.findings.resolved > 0) setConfirm("rule");
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
      <RunButton
        conclusion={settled ? "passed" : row.conclusion}
        disabled={!ai.canStart}
        key="run"
        label={settled ? undefined : t("plan.ai-review.run")}
        onClick={() => (settled ? setConfirm("ai") : void ai.runReview())}
      />
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

  const details: ReactNode[] = [];
  if (row.reviewer === "ai" && ai.allowRun && ai.editing) {
    details.push(
      <p className="text-xs text-control-light" key="save-first">
        {t("plan.ai-review.save-first")}
      </p>
    );
  }
  const { findings } = row;
  const decided =
    findings.overridden > 0 ||
    findings.accepted.length > 0 ||
    findings.dismissed > 0 ||
    findings.earlier > 0;
  if (runType !== undefined && decided) {
    details.push(
      <FindingDecisions
        comments={comments}
        key="decisions"
        onOpen={openSet}
        row={row}
        runType={runType}
      />
    );
  }
  // A reviewer without a result has covered nothing yet.
  const hasResult =
    row.conclusion !== "not-run" &&
    row.conclusion !== "loading" &&
    row.conclusion !== "running";
  if ((row.reviewer === "rule" || row.reviewer === "ai") && hasResult) {
    details.push(<ScopeLine key="scope" reviewer={row.reviewer} />);
  }

  return (
    <div
      className="flex flex-col gap-y-1 px-3 py-2.5 text-sm"
      data-testid={`automated-review-${row.reviewer}`}
      ref={rowRef}
    >
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        <ConclusionIcon row={row} />
        <span className="flex w-44 shrink-0 items-center gap-x-1.5 font-medium text-main">
          {name}
        </span>
        <span className="min-w-0 flex-1 text-control">
          <ConclusionText onOpen={openSet} row={row} />
        </span>
        {actions.length > 0 && (
          <span className="flex shrink-0 items-center gap-x-2">{actions}</span>
        )}
      </div>
      {details.length > 0 && (
        <div className="flex gap-x-3">
          {/* Lines up with the conclusion, past the icon and the name. */}
          <span className="size-4 shrink-0" />
          <span className="hidden w-44 shrink-0 sm:block" />
          <div className="flex min-w-0 flex-1 flex-col gap-y-1">{details}</div>
        </div>
      )}
      <RerunConfirmDialog
        confirmLabel={t("plan.automated-review.rerun")}
        description={
          confirm === "ai"
            ? t("plan.ai-review.run-again-description")
            : t("plan.automated-review.rerun-reopens", {
                count: row.findings.resolved,
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
            row.findings.fix > 0 ? "text-error" : "text-warning"
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

function CountLink({
  label,
  onClick,
  emphasis = false,
}: {
  label: string;
  onClick: () => void;
  emphasis?: boolean;
}) {
  return (
    <Button
      appearance="link"
      className={cn(
        "h-auto p-0 text-sm text-main underline decoration-control-border underline-offset-2",
        emphasis && "font-medium"
      )}
      onClick={onClick}
      size="xs"
    >
      {label}
    </Button>
  );
}

function ConclusionText({
  row,
  onOpen,
}: {
  row: AutomatedReviewRow;
  onOpen: (set: FindingSet) => void;
}) {
  const { t } = useTranslation();
  const time = row.time && (
    <HumanizeTs
      className="text-control-light"
      tsMs={getTimeForPbTimestampProtoEs(row.time)}
    />
  );
  const isPlanCheck = runTypeOf(row.reviewer) === undefined;
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
        <span className="break-words text-main" key="error">
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
    case "resolved":
      parts.push(t("plan.automated-review.no-open-findings"));
      break;
    case "open":
      for (const part of openActionParts(t, row.findings)) {
        parts.push(
          isPlanCheck ? (
            <span className="font-medium text-main" key={part.action}>
              {part.label}
            </span>
          ) : (
            <CountLink
              emphasis={part.action !== "advice"}
              key={part.action}
              label={part.label}
              onClick={() => onOpen(part.action)}
            />
          )
        );
      }
      break;
  }
  if (time && row.conclusion !== "loading" && row.conclusion !== "not-run") {
    parts.push(time);
  }
  return (
    <span className="flex flex-wrap items-center gap-x-1.5">
      {parts.map((part, index) => (
        <span className="flex items-center gap-x-1.5" key={index}>
          {index > 0 && <span className="text-control-light">·</span>}
          {part}
        </span>
      ))}
    </span>
  );
}

// What people decided about the latest run's findings, which the editor
// folds away: a P0 resolved without a change first, since an approver must
// look at it, then the risks people accepted by rule, then counts only for
// dismissed advice and earlier runs.
function FindingDecisions({
  row,
  runType,
  comments,
  onOpen,
}: {
  row: AutomatedReviewRow;
  runType: ReviewRun_Type;
  comments: readonly IssueComment[];
  onOpen: (set: FindingSet) => void;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const { findings } = row;
  const findingName = (comment: IssueComment) => {
    const ruleType =
      comment.reviewMetadata?.ruleType ??
      ReviewRuleType.REVIEW_RULE_TYPE_UNSPECIFIED;
    return ruleType === ReviewRuleType.REVIEW_RULE_TYPE_UNSPECIFIED
      ? findingTitle(comment)
      : standardRuleText(t, ruleType).title;
  };
  const overridden = useMemo(
    () =>
      findings.overridden > 0
        ? findingsInOrder(comments, runType, page.plan.specs, "overridden")
        : [],
    [comments, findings.overridden, page.plan.specs, runType]
  );

  const lines: ReactNode[] = [];
  if (overridden.length > 0) {
    const named = overridden.slice(0, NAMED_OVERRIDES).map((comment) => {
      const line = comment.statementAnchor?.startPosition?.line;
      return line
        ? t("plan.findings.name-at-line", {
            name: findingName(comment),
            line,
          })
        : findingName(comment);
    });
    const rest = overridden.length - named.length;
    lines.push(
      <p
        className="flex items-start gap-x-1.5 text-main"
        data-testid="findings-overridden"
        key="overridden"
      >
        <TriangleAlert className="mt-0.5 size-4 shrink-0 text-warning" />
        <span className="min-w-0">
          <CountLink
            emphasis
            label={t("plan.findings.overridden-warning", {
              count: overridden.length,
            })}
            onClick={() => onOpen("overridden")}
          />{" "}
          <span className="text-control">
            {named.join(t("plan.findings.separator"))}
            {rest > 0 &&
              `${t("plan.findings.separator")}${t("plan.findings.n-more", {
                count: rest,
              })}`}
          </span>
        </span>
      </p>
    );
  }
  if (findings.accepted.length > 0) {
    const accepted = findings.accepted.reduce(
      (sum, group) => sum + group.count,
      0
    );
    lines.push(
      <p
        className="text-control"
        data-testid="findings-accepted"
        key="accepted"
      >
        <CountLink
          label={t("plan.findings.accepted", { count: accepted })}
          onClick={() => onOpen("accepted")}
        />{" "}
        {findings.accepted
          .map((group) => {
            const name =
              group.ruleType === ReviewRuleType.REVIEW_RULE_TYPE_UNSPECIFIED
                ? group.title
                : standardRuleText(t, group.ruleType).title;
            return group.count > 1
              ? t("plan.findings.name-times", { name, count: group.count })
              : name;
          })
          .join(t("plan.findings.separator"))}
      </p>
    );
  }
  const counts = [
    findings.dismissed > 0
      ? t("plan.findings.n-dismissed", { count: findings.dismissed })
      : "",
    findings.earlier > 0
      ? t("plan.findings.n-earlier", { count: findings.earlier })
      : "",
  ].filter(Boolean);
  if (counts.length > 0) {
    lines.push(
      <p className="text-xs text-control-light" key="counts">
        {counts.join(" · ")}
      </p>
    );
  }
  return lines.length > 0 ? <>{lines}</> : null;
}

// What the reviewer covered: the databases, and for the rule review, the
// rules it applies. A "No findings" means little without it.
function ScopeLine({ reviewer }: { reviewer: "rule" | "ai" }) {
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
  const parts = [
    scope.engines.length > 0
      ? t("plan.automated-review.scope-with-engines", {
          databases: names,
          engines: scope.engines.map(engineNameV1).join(", "),
        })
      : t("plan.automated-review.scope", { databases: names }),
  ];
  if (scope.total > scope.databases.length) {
    parts.push(
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
    parts.push(
      rules.source === "project"
        ? t("plan.automated-review.project-rules", counts)
        : t("plan.automated-review.workspace-rules", counts)
    );
  }
  return (
    <p className="text-xs text-control-light" data-testid="review-scope">
      {parts.join(" · ")}
    </p>
  );
}
