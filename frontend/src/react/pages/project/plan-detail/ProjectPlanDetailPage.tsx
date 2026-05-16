import { Code2, Loader2, MessageSquareMore, Rocket, X } from "lucide-react";
import type { CSSProperties, ReactNode } from "react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/react/components/ui/alert-dialog";
import { Badge } from "@/react/components/ui/badge";
import { Button } from "@/react/components/ui/button";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { cn } from "@/react/lib/utils";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { getReviewBadge } from "../utils/reviewBadge";
import { DeployBranch } from "./components/deploy/DeployBranch";
import { DeployTaskDetailPanel } from "./components/deploy/DeployTaskDetailPanel";
import { PlanDetailReviewApprovalFlow } from "./components/PlanDetailApprovalFlow";
import { PlanDetailChangesBranch } from "./components/PlanDetailChangesBranch";
import { PlanDetailDeployFuture } from "./components/PlanDetailDeployFuture";
import { PlanDetailHeader } from "./components/PlanDetailHeader";
import { PlanDetailStoreProvider } from "./shared/stores/PlanDetailStoreProvider";
import { INLINE_TASK_PANEL_BREAKPOINT_PX } from "./shell/constants";
import { usePlanDetailPage } from "./shell/hooks/usePlanDetailPage";
import {
  PlanDetailProvider,
  usePlanDetailContext,
} from "./shell/PlanDetailContext";
import {
  buildChangesSummary,
  buildDeploySummary,
  buildReviewSummary,
} from "./utils/phaseSummary";
import { isReleaseBackedPlan } from "./utils/spec";

export function ProjectPlanDetailPage(props: {
  projectId: string;
  planId: string;
  routeName?: string;
  routeQuery?: Record<string, unknown>;
  specId?: string;
}) {
  return (
    <PlanDetailStoreProvider>
      <ProjectPlanDetailPageInner {...props} />
    </PlanDetailStoreProvider>
  );
}

function ProjectPlanDetailPageInner({
  projectId,
  planId,
  routeName,
  routeQuery,
  specId,
}: {
  projectId: string;
  planId: string;
  routeName?: string;
  routeQuery?: Record<string, unknown>;
  specId?: string;
}) {
  const { t } = useTranslation();
  const [pageHost, setPageHost] = useState<HTMLDivElement | null>(null);
  const [selectedSpecId, setSelectedSpecId] = useState(specId ?? "");
  const page = usePlanDetailPage({
    projectId,
    planId,
    routeName,
    routeQuery,
    specId,
    pageHost,
  });
  const selectedTask = useMemo(() => {
    if (!page.routeTaskId || !page.rollout) {
      return undefined;
    }
    for (const stage of page.rollout.stages) {
      const task = stage.tasks.find((item) =>
        item.name.endsWith(`/${page.routeTaskId}`)
      );
      if (task) {
        return task;
      }
    }
    return undefined;
  }, [page.rollout, page.routeTaskId]);
  const supportsInlineDetailPanel =
    page.layoutMode === "DESKTOP" &&
    page.containerWidth >= INLINE_TASK_PANEL_BREAKPOINT_PX;
  const showDesktopDetail = supportsInlineDetailPanel && !!selectedTask;
  const showTaskDrawer =
    !!selectedTask && !showDesktopDetail && page.layoutMode !== "NONE";

  const desktopLayoutStyle = useMemo<CSSProperties>(() => {
    const baseStyle: CSSProperties = {
      minHeight: "calc(100vh - 4rem)",
    };

    if (showDesktopDetail) {
      return {
        ...baseStyle,
        gridTemplateColumns: "minmax(0, 1fr) minmax(0, 50%)",
      };
    }

    return baseStyle;
  }, [showDesktopDetail]);
  const isGitOpsPlan = useMemo(
    () => isReleaseBackedPlan(page.plan.specs),
    [page.plan.specs]
  );

  const phaseConfigs = useMemo(() => {
    const hasIssue = !!page.issue;
    const hasRollout = !!page.rollout;
    const isIssueClosed =
      page.issue?.status === IssueStatus.CANCELED ||
      page.issue?.status === IssueStatus.DONE;
    const allTasks = page.rollout?.stages.flatMap((stage) => stage.tasks) ?? [];
    const allDone =
      allTasks.length > 0 &&
      allTasks.every(
        (task) =>
          task.status === Task_Status.DONE ||
          task.status === Task_Status.SKIPPED
      );

    let review: "completed" | "closed" | "active" | "future" = "future";
    if (hasIssue) {
      if (page.issue?.status === IssueStatus.CANCELED) {
        review = "closed";
      } else {
        review = hasRollout || isIssueClosed ? "completed" : "active";
      }
    }

    const changesStatus: "completed" | "closed" | "active" | "future" =
      page.isCreating || (!isGitOpsPlan && !hasIssue && !hasRollout)
        ? "active"
        : "completed";
    const deployStatus: "completed" | "closed" | "active" | "future" =
      hasRollout ? (allDone ? "completed" : "active") : "future";

    const changesBadge =
      changesStatus === "active" && !page.isCreating
        ? { label: t("common.draft"), variant: "default" as const }
        : undefined;
    const rawReviewBadge = getReviewBadge({
      hasIssue,
      issueStatus: page.issue?.status,
      hasRollout,
      approvalStatus: page.issue?.approvalStatus,
    });
    const reviewBadge = rawReviewBadge
      ? {
          label: t(rawReviewBadge.labelKey),
          variant: rawReviewBadge.variant,
        }
      : undefined;
    const deployBadge = (() => {
      if (deployStatus !== "active" || !page.rollout) return undefined;
      const hasCompletedTasks = allTasks.some(
        (task) =>
          task.status === Task_Status.DONE ||
          task.status === Task_Status.SKIPPED
      );
      if (allTasks.some((task) => task.status === Task_Status.FAILED)) {
        return { label: t("common.failed"), variant: "destructive" as const };
      }
      if (
        allTasks.some(
          (task) =>
            task.status === Task_Status.RUNNING ||
            task.status === Task_Status.PENDING
        ) ||
        hasCompletedTasks
      ) {
        return {
          label: t("common.in-progress"),
          variant: "secondary" as const,
        };
      }
      return { label: t("common.not-started"), variant: "default" as const };
    })();

    const lineClass = (
      from: "completed" | "closed" | "active" | "future",
      to: "completed" | "closed" | "active" | "future"
    ) => {
      if (from === "closed" || to === "closed")
        return "border-l-2 border-dashed border-control-border";
      if (from === "completed" && to === "completed")
        return "border-l-2 border-success";
      if (from === "completed" && to === "active")
        return "border-l-2 border-success";
      if (from === "active") return "border-l-2 border-dashed border-accent";
      return "border-l-2 border-dashed border-control-border";
    };

    return {
      changes: {
        status: changesStatus,
        badge: changesBadge,
        lineClass: lineClass(
          changesStatus,
          isGitOpsPlan ? deployStatus : review
        ),
      },
      review: {
        status: review,
        badge: reviewBadge,
        lineClass: lineClass(review, deployStatus),
      },
      deploy: {
        status: deployStatus,
        badge: deployBadge,
        lineClass: "",
      },
    };
  }, [isGitOpsPlan, page.isCreating, page.issue, page.rollout, t]);

  // Mirror the URL specId into local state. We deliberately don't include
  // selectedSpecId in the deps — children (e.g. PlanDetailChangesBranch) may
  // set selectedSpecId to a draft spec that has no URL yet, and snapping it
  // back to specId here would defeat the selection.
  useEffect(() => {
    if (!page.isCreating && specId) {
      setSelectedSpecId(specId);
    }
  }, [page.isCreating, specId]);

  // Default to the first spec when nothing is selected.
  useEffect(() => {
    if (!selectedSpecId && page.plan.specs.length > 0) {
      setSelectedSpecId(page.plan.specs[0].id);
    }
  }, [page.plan.specs, selectedSpecId]);

  return (
    <PlanDetailProvider value={page}>
      <div
        ref={setPageHost}
        className="relative h-full overflow-x-hidden bg-gray-50"
      >
        <div
          className={cn(
            "flex min-h-full flex-col",
            page.ready ? "" : "invisible pointer-events-none"
          )}
        >
          <header className="shrink-0 border-b bg-white">
            <PlanDetailHeader />
          </header>

          <div
            className="min-h-0 flex flex-1 flex-col lg:grid"
            style={desktopLayoutStyle}
          >
            <main className="min-w-0 flex-1">
              <div className="flex min-w-0 flex-col gap-y-3 pb-6 pl-2 pr-4 pt-4 xl:pr-8 2xl:pr-12">
                <PhaseSection
                  badge={phaseConfigs.changes.badge}
                  expanded={page.activePhases.has("changes")}
                  icon={<Code2 className="h-4 w-4 text-white" />}
                  lineClass={phaseConfigs.changes.lineClass}
                  label={t("plan.navigator.changes")}
                  onSelect={() => page.expandPhase("changes")}
                  status={phaseConfigs.changes.status}
                  onToggle={() => page.togglePhase("changes")}
                  summary={buildChangesSummary(page.plan, t)}
                >
                  <PlanDetailChangesBranch
                    onSelectedSpecIdChange={setSelectedSpecId}
                    selectedSpecId={selectedSpecId}
                  />
                </PhaseSection>

                {!isGitOpsPlan && (
                  <PhaseSection
                    badge={phaseConfigs.review.badge}
                    expanded={page.activePhases.has("review")}
                    icon={<MessageSquareMore className="h-4 w-4 text-white" />}
                    lineClass={phaseConfigs.review.lineClass}
                    label={t("plan.navigator.review")}
                    onSelect={() => page.expandPhase("review")}
                    status={phaseConfigs.review.status}
                    onToggle={() => page.togglePhase("review")}
                    summary={buildReviewSummary(page.issue, t)}
                    future={
                      <p className="mt-0.5 text-sm text-control-placeholder">
                        {t("plan.phase.review-description")}
                      </p>
                    }
                  >
                    <ReviewBranch />
                  </PhaseSection>
                )}

                <PhaseSection
                  badge={phaseConfigs.deploy.badge}
                  expanded={page.activePhases.has("deploy")}
                  icon={<Rocket className="h-4 w-4 text-white" />}
                  isLast
                  label={t("plan.navigator.deploy")}
                  status={phaseConfigs.deploy.status}
                  onSelect={() => page.expandPhase("deploy")}
                  onToggle={() => page.togglePhase("deploy")}
                  summary={buildDeploySummary(page.rollout, t)}
                  future={<PlanDetailDeployFuture />}
                >
                  <DeployBranch
                    selectedTask={selectedTask}
                    onCloseTaskPanel={page.closeTaskPanel}
                  />
                </PhaseSection>
              </div>
            </main>

            {showDesktopDetail && selectedTask && (
              <DesktopColumn
                header={
                  <div className="flex items-center justify-between border-b bg-white px-4 py-2">
                    <span className="textinfolabel">{t("common.detail")}</span>
                    <Button
                      size="xs"
                      variant="ghost"
                      onClick={page.closeTaskPanel}
                    >
                      <X className="h-4 w-4" />
                      {t("common.close")}
                    </Button>
                  </div>
                }
              >
                <DeployTaskDetailPanel task={selectedTask} />
              </DesktopColumn>
            )}
          </div>
        </div>

        {!page.ready && (
          <div className="absolute inset-0 flex flex-col items-center justify-center gap-y-3 bg-white">
            <Loader2 className="h-8 w-8 animate-spin text-accent" />
            <div className="text-sm text-control-light">
              {t("common.loading")}
            </div>
          </div>
        )}

        <Sheet
          onOpenChange={(open) => {
            if (!open) {
              page.closeTaskPanel();
            }
          }}
          open={showTaskDrawer}
        >
          <SheetContent
            className="w-[40rem] max-w-[calc(100vw-2rem)]"
            width="wide"
          >
            <SheetHeader>
              <SheetTitle>{t("common.detail")}</SheetTitle>
            </SheetHeader>
            <SheetBody className="px-0 py-0">
              {selectedTask ? (
                <DeployTaskDetailPanel task={selectedTask} />
              ) : null}
            </SheetBody>
          </SheetContent>
        </Sheet>

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
                variant="outline"
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
    </PlanDetailProvider>
  );
}

function PhaseSection({
  badge,
  label,
  lineClass,
  icon,
  future,
  isLast = false,
  expanded,
  status,
  summary,
  children,
  onToggle,
  onSelect,
}: {
  badge?: {
    label: string;
    variant: "default" | "secondary" | "warning" | "success" | "destructive";
  };
  label: string;
  lineClass?: string;
  icon: ReactNode;
  future?: ReactNode;
  isLast?: boolean;
  expanded: boolean;
  status: "completed" | "closed" | "active" | "future";
  summary?: string;
  children: ReactNode;
  onToggle: () => void;
  onSelect: () => void;
}) {
  const { t } = useTranslation();
  const dotClass =
    status === "completed"
      ? "bg-success"
      : status === "closed"
        ? "bg-control-placeholder"
        : status === "active"
          ? "bg-accent ring-[3px] ring-accent/20"
          : "border-2 border-dashed border-control-border";

  return (
    <div className={cn("flex", isLast && "mb-48")}>
      <div className="flex w-10 shrink-0 flex-col items-center md:w-16">
        <div
          className="mt-0.5 flex h-5 w-5 shrink-0 cursor-pointer items-center justify-center md:h-7 md:w-7"
          onClick={onSelect}
        >
          <div
            className={cn(
              "flex h-5 w-5 items-center justify-center rounded-full md:h-7 md:w-7",
              dotClass
            )}
          >
            {status !== "future" ? icon : null}
          </div>
        </div>
        {!isLast && <div className={cn("flex-1 min-h-[16px]", lineClass)} />}
      </div>

      <div className="min-w-0 flex-1 pb-4">
        {status === "future" ? (
          <div className="py-0.5">
            <span className="textlabel uppercase text-control-placeholder">
              {label}
            </span>
            {future ?? (
              <div className="mt-0.5 text-sm text-control-placeholder">
                {summary}
              </div>
            )}
          </div>
        ) : !expanded ? (
          <div className="cursor-pointer py-0.5" onClick={onToggle}>
            <div className="flex items-center gap-2">
              <span className="textlabel uppercase">{label}</span>
              {badge && <Badge variant={badge.variant}>{badge.label}</Badge>}
              <div className="flex-1" />
              <span className="shrink-0 text-[11px] text-control-placeholder">
                {summary ? t("plan.phase.show-details") : ""}
              </span>
            </div>
            {summary && (
              <div className="mt-0.5 text-sm text-control">{summary}</div>
            )}
          </div>
        ) : (
          <div className="flex flex-col">
            <div
              className="flex items-center gap-2 py-0.5 cursor-pointer"
              onClick={onToggle}
            >
              <span className="textlabel uppercase text-accent">{label}</span>
              {badge && <Badge variant={badge.variant}>{badge.label}</Badge>}
              <div className="flex-1" />
              <span className="shrink-0 text-[11px] text-control-placeholder">
                {t("plan.phase.hide-details")}
              </span>
            </div>
            <div className="mt-1 overflow-hidden rounded-lg border bg-white">
              {children}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function ReviewBranch() {
  const { t } = useTranslation();
  const page = usePlanDetailContext();

  if (!page.issue) {
    return (
      <div className="p-4 text-sm text-control-light">
        {t("common.no-data")}
      </div>
    );
  }

  return (
    <div className="flex flex-col">
      <PlanDetailReviewApprovalFlow />
    </div>
  );
}

function DesktopColumn({
  children,
  header,
}: {
  children: ReactNode;
  header?: ReactNode;
}) {
  return (
    <div className="min-w-0 border-l bg-white">
      <div className="sticky top-0 flex max-h-[calc(100vh-4rem)] min-h-[calc(100vh-4rem)] flex-col overflow-hidden">
        {header ? <div className="shrink-0">{header}</div> : null}
        <div className="min-h-0 flex-1 overflow-y-auto">{children}</div>
      </div>
    </div>
  );
}
