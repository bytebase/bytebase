import { Check, Code2, Loader2, MessageSquareMore, Rocket } from "lucide-react";
import type { ReactNode } from "react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { MAIN_SCROLL_RESTORATION_ID } from "@/app/router/NavigationScrollRestoration";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useOnKeyChange } from "@/hooks/useOnKeyChange";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app";
import { ApprovalStatus, IssueStatus } from "@/types/proto-es/v1/common_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { getRolloutStatus } from "@/utils";
import { sqlReviewV2Enabled } from "@/utils/featureGates";
import { getReviewBadge } from "../utils/reviewBadge";
import { DeployBranch } from "./components/deploy/DeployBranch";
import { PlanDetailChangesBranch } from "./components/PlanDetailChangesBranch";
import { PlanDetailDeployFuture } from "./components/PlanDetailDeployFuture";
import { PlanDetailDeployWaiting } from "./components/PlanDetailDeployWaiting";
import { PlanDetailHeader } from "./components/PlanDetailHeader";
import { PlanDetailHeaderDetails } from "./components/PlanDetailHeaderDetails";
import { PlanReviewSection } from "./components/review/PlanReviewSection";
import { useUnresolvedThreadTotal } from "./components/threads/useUnresolvedThreadCounts";
import { useIssueCommentThreadsSync } from "./hooks/useIssueCommentThreadsSync";
import { usePlacementSync } from "./hooks/usePlacementSync";
import { usePlanReviewState } from "./hooks/usePlanReviewState";
import { SqlReviewV2Provider } from "./hooks/useSqlReviewV2";
import { PlanDetailStoreProvider } from "./shared/stores/PlanDetailStoreProvider";
import { planPhaseAnchorId } from "./shell/focusPhase";
import { usePlanDetailPage } from "./shell/hooks/usePlanDetailPage";
import {
  PlanDetailProvider,
  usePlanDetailContext,
} from "./shell/PlanDetailContext";
import {
  buildChangesSummary,
  buildDeploySummary,
  buildReviewSummary,
  buildReviewSummaryV2,
} from "./utils/phaseSummary";
import { rowNeedsAttention } from "./utils/planGates";
import { isReleaseBackedPlan } from "./utils/spec";

type PhaseStatus = "completed" | "closed" | "active" | "future";

export function ProjectPlanDetailPage(props: {
  projectId: string;
  planId: string;
  routeHash?: string;
  routeName?: string;
  routeQuery?: Record<string, unknown>;
  specId?: string;
  stageId?: string;
  taskId?: string;
}) {
  return (
    <PlanDetailStoreProvider key={`${props.projectId}/${props.planId}`}>
      <ProjectPlanDetailPageInner {...props} />
    </PlanDetailStoreProvider>
  );
}

function ProjectPlanDetailPageInner({
  projectId,
  planId,
  routeHash,
  routeName,
  routeQuery,
  specId,
  stageId,
  taskId,
}: {
  projectId: string;
  planId: string;
  routeHash?: string;
  routeName?: string;
  routeQuery?: Record<string, unknown>;
  specId?: string;
  stageId?: string;
  taskId?: string;
}) {
  const { t } = useTranslation();
  const [pageHost, setPageHost] = useState<HTMLDivElement | null>(null);
  // The sticky title row only shows its bottom border once content scrolls
  // under it (GitHub issue-header pattern) — flat at the top, separated when
  // stuck. The dashboard main pane owns vertical scrolling.
  const [headerStuck, setHeaderStuck] = useState(false);
  useEffect(() => {
    if (!pageHost) return;
    const scrollHost = pageHost.closest<HTMLElement>(
      `[data-scroll-restoration-id="${MAIN_SCROLL_RESTORATION_ID}"]`
    );
    if (!scrollHost) return;
    const onScroll = () => setHeaderStuck(scrollHost.scrollTop > 0);
    onScroll();
    scrollHost.addEventListener("scroll", onScroll, { passive: true });
    return () => scrollHost.removeEventListener("scroll", onScroll);
  }, [pageHost]);
  const [selectedSpecId, setSelectedSpecId] = useState(specId ?? "");
  const page = usePlanDetailPage({
    projectId,
    planId,
    routeHash,
    routeName,
    routeQuery,
    specId,
    stageId,
    taskId,
  });
  useIssueCommentThreadsSync(page.issue, page.aiReviewRun);
  usePlacementSync({
    issueName: page.issue?.name,
    projectId,
    ready: page.ready,
    specs: page.plan.specs,
  });
  const setViewedPlan = useAppStore((s) => s.setViewedPlan);
  const isReleaseBacked = isReleaseBackedPlan(page.plan.specs);
  useEffect(() => {
    if (!page.ready) return;
    setViewedPlan({ name: page.plan.name, isReleaseBacked });
  }, [page.ready, page.plan.name, isReleaseBacked, setViewedPlan]);
  // Route-driven spec changes reset during render, so the first paint after a
  // Back/Forward or tab navigation already shows the destination spec. The
  // keyed provider remounts this subtree when the plan itself changes.
  useOnKeyChange(`${page.pageKey}:${specId ?? ""}`, () => {
    if (!page.isCreating && specId) {
      setSelectedSpecId(specId);
    }
  });

  // A create route has no persisted spec resource in its URL. Derive the
  // initial selection during render so the first ready paint already shows the
  // first client-local spec, without a placeholder route or a follow-up effect.
  const effectiveSelectedSpecId =
    selectedSpecId || page.plan.specs[0]?.id || "";

  return (
    <PlanDetailProvider value={page}>
      <SqlReviewV2Provider>
        <div
          ref={setPageHost}
          data-testid="plan-detail-page"
          className="relative min-h-full overflow-x-clip bg-control-bg/50"
        >
          <div
            className={cn(
              "flex min-h-full flex-col",
              page.ready ? "" : "invisible pointer-events-none"
            )}
          >
            {/* Only the title/action row is sticky (GitHub issue-header pattern);
              the description + metadata below scroll away beneath it. The
              dashboard main pane is the scroll container, so top-0 pins to it;
              z-20 keeps it above the phase rail nodes (z-10). The border appears
              only when stuck, so the row is flat at the top. */}
            <header
              className={cn(
                "sticky top-0 z-20 shrink-0 bg-background",
                headerStuck && "border-b"
              )}
            >
              <PlanDetailHeader />
            </header>
            <PlanDetailHeaderDetails />

            <main className="min-h-[calc(100vh-4rem)] min-w-0 flex-1">
              <PlanPhaseRail
                onSelectedSpecIdChange={setSelectedSpecId}
                selectedSpecId={effectiveSelectedSpecId}
              />
            </main>
          </div>

          {!page.ready && (
            <div className="absolute inset-0 flex flex-col items-center justify-center gap-y-3 bg-background">
              <Loader2 className="h-8 w-8 animate-spin text-accent" />
              <div className="text-sm text-control-light">
                {t("common.loading")}
              </div>
            </div>
          )}

          <AlertDialog
            open={page.pendingLeaveConfirm}
            onOpenChange={(open) => {
              if (!open) {
                page.resolveLeaveConfirm(false);
              }
            }}
          >
            <AlertDialogContent>
              <AlertDialogTitle>
                {t("common.leave-without-saving")}
              </AlertDialogTitle>
              <AlertDialogFooter>
                <Button
                  appearance="outline"
                  onClick={() => page.resolveLeaveConfirm(false)}
                >
                  {t("common.cancel")}
                </Button>
                <Button
                  variant="destructive"
                  onClick={() => page.resolveLeaveConfirm(true)}
                >
                  {t("common.discard-changes")}
                </Button>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </div>
      </SqlReviewV2Provider>
    </PlanDetailProvider>
  );
}

// The three phases. Before the rollout exists, Changes and Review form one
// loop and Deploy lists what the rollout waits on; afterwards Deploy leads.
function PlanPhaseRail({
  onSelectedSpecIdChange,
  selectedSpecId,
}: {
  onSelectedSpecIdChange: (specId: string) => void;
  selectedSpecId: string;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const unresolvedThreads = useUnresolvedThreadTotal(page.issue?.name);
  const reviewState = usePlanReviewState();
  const isGitOpsPlan = useMemo(
    () => isReleaseBackedPlan(page.plan.specs),
    [page.plan.specs]
  );
  // CI/CD UI (sheet-backed) plans always surface the review phase — even
  // before an issue exists it shows as an upcoming step. Only GitOps
  // release-backed plans, which bypass review entirely, hide it.
  const reviewVisible = !isGitOpsPlan;
  const selectPhase = page.expandPhase;
  // The machine reviews run on drafts too, so under SQL Review V2 Review is
  // open from the first save of an issue.
  const reviewFromDraft = !!reviewState;

  const phaseConfigs = useMemo(() => {
    const hasIssue = !!page.issue;
    const isDraftIssue = page.issue?.draft === true;
    const hasSubmittedIssue = hasIssue && !isDraftIssue;
    const effectiveRollout = isDraftIssue ? undefined : page.rollout;
    const hasRollout = !!effectiveRollout;
    const isIssueClosed =
      page.issue?.status === IssueStatus.CANCELED ||
      page.issue?.status === IssueStatus.DONE;
    const allTasks =
      effectiveRollout?.stages.flatMap((stage) => stage.tasks) ?? [];
    const allDone =
      allTasks.length > 0 &&
      allTasks.every(
        (task) =>
          task.status === Task_Status.DONE ||
          task.status === Task_Status.SKIPPED
      );

    let review: PhaseStatus = "future";
    if (hasSubmittedIssue || (reviewFromDraft && hasIssue)) {
      if (page.issue?.status === IssueStatus.CANCELED) {
        review = "closed";
      } else {
        review = hasRollout || isIssueClosed ? "completed" : "active";
      }
    }

    const changesStatus: PhaseStatus =
      page.isCreating || (reviewVisible && !hasSubmittedIssue && !hasRollout)
        ? "active"
        : "completed";
    const deployStatus: PhaseStatus = hasRollout
      ? allDone
        ? "completed"
        : "active"
      : "future";

    const changesBadge =
      changesStatus === "active" && !page.isCreating
        ? {
            label: page.issue?.draft
              ? t("common.draft")
              : t("plan.lifecycle.incomplete"),
            variant: page.issue?.draft
              ? ("default" as const)
              : ("destructive" as const),
          }
        : undefined;
    const rawReviewBadge = getReviewBadge({
      hasIssue: hasSubmittedIssue,
      issueStatus: page.issue?.status,
      hasRollout,
      approvalStatus: page.issue?.approvalStatus,
    });
    // "Skipped" names the approval outcome, not the review; under SQL Review
    // V2 the summary says "No approval required" instead.
    const reviewBadge =
      rawReviewBadge &&
      !(
        reviewFromDraft && page.issue?.approvalStatus === ApprovalStatus.SKIPPED
      )
        ? {
            label: t(rawReviewBadge.labelKey),
            variant: rawReviewBadge.variant,
          }
        : undefined;
    const deployBadge = (() => {
      if (deployStatus !== "active" || !effectiveRollout) return undefined;
      const rolloutStatus = getRolloutStatus(effectiveRollout);
      if (rolloutStatus === Task_Status.FAILED) {
        return { label: t("common.failed"), variant: "destructive" as const };
      }
      if (
        rolloutStatus === Task_Status.RUNNING ||
        rolloutStatus === Task_Status.PENDING
      ) {
        return {
          label: t("common.in-progress"),
          variant: "secondary" as const,
        };
      }
      if (rolloutStatus === Task_Status.CANCELED) {
        return { label: t("common.canceled"), variant: "default" as const };
      }
      // A NOT_STARTED aggregate can still mean earlier stages already finished
      // (partially deployed) — at the phase level that's progress, not idle.
      if (
        allTasks.some(
          (task) =>
            task.status === Task_Status.DONE ||
            task.status === Task_Status.SKIPPED
        )
      ) {
        return {
          label: t("common.in-progress"),
          variant: "secondary" as const,
        };
      }
      return { label: t("common.not-started"), variant: "default" as const };
    })();

    return {
      changes: { status: changesStatus, badge: changesBadge },
      review: { status: review, badge: reviewBadge },
      deploy: { status: deployStatus, badge: deployBadge },
    };
  }, [
    page.isCreating,
    page.issue,
    page.rollout,
    reviewFromDraft,
    reviewVisible,
    t,
  ]);

  // A small dot on the Review node marks findings or discussions that need
  // someone.
  const reviewNeedsAttention =
    !!reviewState &&
    (reviewState.rows.some(rowNeedsAttention) ||
      reviewState.openDiscussions > 0);
  const reviewSummary = reviewState
    ? buildReviewSummaryV2(
        page.issue,
        t,
        reviewState.rows,
        reviewState.openDiscussions
      )
    : buildReviewSummary(page.issue, t, unresolvedThreads);

  let deployFuture: ReactNode;
  // A closed issue waits on nothing, so Deploy describes the phase instead.
  if (reviewState && page.issue?.status === IssueStatus.OPEN && !isGitOpsPlan) {
    deployFuture = (
      <PlanDetailDeployWaiting
        issue={page.issue}
        plan={page.plan}
        state={reviewState}
      />
    );
  } else if (page.issue?.draft) {
    deployFuture = (
      <p className="mt-0.5 text-sm text-control-light">
        {sqlReviewV2Enabled()
          ? t("plan.phase.deploy-description-v2")
          : t("plan.phase.deploy-description")}
      </p>
    );
  } else {
    deployFuture = <PlanDetailDeployFuture />;
  }

  return (
    // No row gap here: each PhaseSection's pb-6 carries the spacing, which
    // the connector line fills so it stays continuous.
    <div className="flex min-w-0 flex-col pb-6 pl-2 pr-4 pt-4 xl:pr-8 2xl:pr-12">
      <PhaseSection
        anchorId={planPhaseAnchorId("changes")}
        badge={phaseConfigs.changes.badge}
        expanded={page.activePhases.has("changes")}
        icon={<Code2 className="size-3 md:size-4" />}
        label={t("plan.navigator.changes")}
        onSelect={() => selectPhase("changes")}
        status={phaseConfigs.changes.status}
        onToggle={() => page.togglePhase("changes")}
        summary={buildChangesSummary(page.plan, t, {
          includeChecks: !reviewState,
        })}
      >
        <PlanDetailChangesBranch
          onSelectedSpecIdChange={onSelectedSpecIdChange}
          selectedSpecId={selectedSpecId}
        />
      </PhaseSection>

      {reviewVisible && (
        <PhaseSection
          anchorId={planPhaseAnchorId("review")}
          attention={reviewNeedsAttention}
          badge={phaseConfigs.review.badge}
          expanded={page.activePhases.has("review")}
          icon={<MessageSquareMore className="size-3 md:size-4" />}
          label={t("plan.navigator.review")}
          onSelect={() => selectPhase("review")}
          status={phaseConfigs.review.status}
          onToggle={() => page.togglePhase("review")}
          summary={reviewSummary}
          future={
            <p className="mt-0.5 text-sm text-control-light">
              {t("plan.phase.review-description")}
            </p>
          }
        >
          <PlanReviewSection />
        </PhaseSection>
      )}

      <PhaseSection
        anchorId={planPhaseAnchorId("deploy")}
        badge={phaseConfigs.deploy.badge}
        expanded={page.activePhases.has("deploy")}
        icon={<Rocket className="size-3 md:size-4" />}
        isLast
        label={t("plan.navigator.deploy")}
        status={phaseConfigs.deploy.status}
        onSelect={() => selectPhase("deploy")}
        onToggle={() => page.togglePhase("deploy")}
        summary={buildDeploySummary(
          page.issue?.draft ? undefined : page.rollout,
          t
        )}
        future={deployFuture}
      >
        <DeployBranch />
      </PhaseSection>
    </div>
  );
}

function PhaseSection({
  attention = false,
  badge,
  label,
  icon,
  future,
  isLast = false,
  expanded,
  status,
  summary,
  children,
  onToggle,
  onSelect,
  anchorId,
}: {
  attention?: boolean;
  badge?: {
    label: string;
    variant: "default" | "secondary" | "warning" | "success" | "destructive";
  };
  label: string;
  icon: ReactNode;
  future?: ReactNode;
  isLast?: boolean;
  expanded: boolean;
  status: PhaseStatus;
  summary?: string;
  children: ReactNode;
  onToggle: () => void;
  onSelect: () => void;
  anchorId?: string;
}) {
  const { t } = useTranslation();
  // Finished is quiet, in progress is outlined, and only something that needs
  // a person gets saturated color: the attention dot.
  const dotClass =
    status === "completed"
      ? "bg-control-bg text-control-light"
      : status === "closed"
        ? "bg-control-placeholder text-accent-text"
        : status === "active"
          ? "border-2 border-accent bg-background text-accent"
          : "border-2 border-dashed border-control-border bg-control-bg text-control-light";

  return (
    <div className={cn("flex scroll-mt-16", isLast && "mb-48")} id={anchorId}>
      <div className="flex w-10 shrink-0 flex-col items-center md:w-16">
        <div
          className="relative z-10 mt-0.5 flex h-5 w-5 shrink-0 cursor-pointer items-center justify-center md:h-7 md:w-7"
          onClick={onSelect}
        >
          <div
            className={cn(
              "flex h-5 w-5 items-center justify-center rounded-full md:h-7 md:w-7",
              dotClass
            )}
          >
            {status === "completed" ? (
              <Check className="size-3 md:size-4" />
            ) : (
              icon
            )}
          </div>
          {attention && (
            <span
              aria-hidden="true"
              className="absolute -top-0.5 -right-0.5 size-2.5 rounded-full border-2 border-background bg-warning"
              data-testid="phase-attention"
            />
          )}
        </div>
        {/* -mt-1 tucks the rail under the node so it reads as connected. */}
        {!isLast && (
          <div className="-mt-1 min-h-[20px] w-0.5 flex-1 bg-block-border" />
        )}
      </div>

      <div className="min-w-0 flex-1 pb-6">
        {status === "future" ? (
          <div className="py-0.5">
            <span className="textlabel uppercase text-control-light">
              {label}
            </span>
            {future ?? (
              <div className="mt-0.5 text-sm text-control-light">{summary}</div>
            )}
          </div>
        ) : !expanded ? (
          <div className="cursor-pointer py-0.5" onClick={onToggle}>
            <div className="flex items-center gap-2">
              <span className="textlabel uppercase">{label}</span>
              {badge && <Badge variant={badge.variant}>{badge.label}</Badge>}
              <div className="flex-1" />
              <span className="shrink-0 text-xs text-control-light">
                {t("plan.phase.show-details")}
              </span>
            </div>
            {summary && (
              <div className="mt-0.5 text-sm text-control">{summary}</div>
            )}
          </div>
        ) : (
          <div className="flex flex-col">
            <div className="flex items-center gap-2 py-0.5">
              <span className="textlabel uppercase text-main">{label}</span>
              {badge && <Badge variant={badge.variant}>{badge.label}</Badge>}
              <div className="flex-1" />
              <span
                className="shrink-0 cursor-pointer text-xs text-control-light hover:text-control"
                onClick={onToggle}
              >
                {t("plan.phase.hide-details")}
              </span>
            </div>
            <div className="mt-1 overflow-hidden rounded-sm border bg-background">
              {children}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
