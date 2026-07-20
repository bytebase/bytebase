// The header's read-only "Plan status" control, shown when the current user does
// not own the next action (observer, not-your-turn) or the plan is in a pre-deploy
// blocker state. The trigger names the blocker; clicking opens a details panel
// styled after GitHub's merge box — two collapsible gate rows (Review, Plan
// checks), each a status icon + headline + subtext. The detail is ours: Review
// expands to the (compact) approval flow; Plan checks expands to a severity
// breakdown with a "View Details" link into the existing plan-check drawer. The
// footer shows at most one state-dependent action: re-request review when
// rejected (mirroring the section's rejection banner), otherwise the demoted
// bypass-and-deploy override (BYT-9722).
import { create } from "@bufbuild/protobuf";
import {
  ChevronDown,
  ChevronRight,
  CircleCheck,
  CircleX,
  Clock3,
  Loader2,
} from "lucide-react";
import { type ReactNode, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { issueServiceClientConnect } from "@/connect";
import { PlanCheckResultsDrawer } from "@/react/components/plan-check/PlanCheckSection";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/react/components/ui/popover";
import { useCurrentUser } from "@/react/hooks/useAppState";
import { displayRoleTitleFromList } from "@/react/lib/role";
import { cn } from "@/react/lib/utils";
import { applyProjectDetailMutationResult } from "@/react/pages/project/applyProjectDetailMutationResult";
import { useAppStore } from "@/react/stores/app";
import { pushNotification } from "@/store";
import { userNamePrefix } from "@/store/modules/v1/common";
import { unknownUser } from "@/types";
import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { RequestIssueRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import type { PlanCheckSummary } from "../../utils/phaseSummary";
import { BypassAndDeploySheet, useBypassDeploy } from "../review/bypassDeploy";
import { deriveSteps, ReviewApprovalFlow } from "../review/ReviewApprovalFlow";
import type { PlanStatusReason } from "./planLifecycleHeaderState";
import { getPlanStatusTone, type PlanStatusTone } from "./planStatusLabel";

const TONE_TRIGGER: Record<PlanStatusTone, string> = {
  neutral: "border-control-border text-control",
  error: "border-error/40 text-error",
};

export function PlanStatusAction({
  issue,
  reason,
}: {
  issue: Issue;
  reason: PlanStatusReason;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const [open, setOpen] = useState(false);
  const [checksDrawerOpen, setChecksDrawerOpen] = useState(false);
  const bypass = useBypassDeploy(issue, page.plan);
  const tone = getPlanStatusTone(reason);
  // Only the Review gate expands (to the approval flow); Checks is a static row
  // with a View Details link. Open Review by default when it is the active gate.
  const [reviewOpen, setReviewOpen] = useState(
    reason === "in-review" || reason === "rejected"
  );

  // Literal t() calls (not t(dynamicKey)) so the i18n usage scanner sees them.
  const labelText = (() => {
    switch (reason) {
      case "checking":
        return t("plan.lifecycle.checking");
      case "in-review":
        // Reuse the shared review-status label so the header matches the plan
        // list / issue surfaces ("Under review"), not a header-only "In review".
        return t("common.under-review");
      case "rejected":
        return t("plan.lifecycle.rejected");
      case "checks-failing":
        return t("plan.lifecycle.checks-failing");
    }
  })();

  return (
    <>
      <Popover onOpenChange={setOpen} open={open}>
        <PopoverTrigger
          render={
            <button
              className={cn(
                "inline-flex h-9 shrink-0 items-center gap-x-1 rounded-full border px-3 text-sm hover:bg-control-bg/40",
                TONE_TRIGGER[tone]
              )}
              type="button"
            />
          }
        >
          {labelText}
          <ChevronDown className="size-4" />
        </PopoverTrigger>
        <PopoverContent
          align="end"
          className="w-[min(30rem,calc(100vw-2rem))] p-0"
        >
          <div className="flex max-h-[70vh] flex-col divide-y overflow-y-auto">
            <ReviewGateRow
              issue={issue}
              onToggle={() => setReviewOpen((v) => !v)}
              open={reviewOpen}
            />
            <ChecksGateRow
              onViewDetails={() => {
                setOpen(false);
                setChecksDrawerOpen(true);
              }}
              summary={bypass.planCheckSummary}
            />
          </div>
          <PlanStatusFooter
            bypass={bypass}
            issue={issue}
            onBypass={() => {
              setOpen(false);
              bypass.triggerDeploy();
            }}
            reason={reason}
          />
        </PopoverContent>
      </Popover>
      <BypassAndDeploySheet bypass={bypass} issue={issue} />
      {/* Rendered outside the popover so it survives the popover closing. */}
      <PlanCheckResultsDrawer
        includeRunFailure
        onOpenChange={setChecksDrawerOpen}
        open={checksDrawerOpen}
        planCheckRuns={page.planCheckRuns}
      />
    </>
  );
}

/* --------------------------- gate row shell ----------------------------- */

// Shared icon-adjacent label for the gate rows: a title with an optional
// subtitle beneath it.
function GateLabel({ title, subtitle }: { title: string; subtitle: string }) {
  return (
    <span className="min-w-0 flex-1">
      <span className="block text-sm font-medium leading-tight text-main">
        {title}
      </span>
      {subtitle && (
        <span className="mt-0.5 block truncate text-xs leading-tight text-control-light">
          {subtitle}
        </span>
      )}
    </span>
  );
}

function GateRow({
  icon,
  title,
  subtitle,
  open,
  onToggle,
  children,
}: {
  icon: ReactNode;
  title: string;
  subtitle: string;
  open: boolean;
  onToggle: () => void;
  children: ReactNode;
}) {
  return (
    <div>
      <button
        className="flex w-full items-center gap-3 px-4 py-3 text-left hover:bg-control-bg/40"
        onClick={onToggle}
        type="button"
      >
        {icon}
        <GateLabel subtitle={subtitle} title={title} />
        <ChevronDown
          className={cn(
            "size-4 shrink-0 text-control-light transition-transform",
            open && "rotate-180"
          )}
        />
      </button>
      {open && <div className="border-t bg-control-bg/20">{children}</div>}
    </div>
  );
}

/* ------------------------- review gate (ours) --------------------------- */

function ReviewGateRow({
  issue,
  open,
  onToggle,
}: {
  issue: Issue;
  open: boolean;
  onToggle: () => void;
}) {
  const { t } = useTranslation();
  const roleList = useAppStore((state) => state.roleList);
  const steps = useMemo(() => deriveSteps(issue), [issue]);
  const rejectedStep = steps.find((step) => step.status === "rejected");
  const currentStep = steps.find((step) => step.status === "current");
  const rejecter = useAppStore((state) =>
    rejectedStep?.approver
      ? state.getUserByIdentifier(rejectedStep.approver)
      : undefined
  );
  const approvedCount = steps.filter(
    (step) => step.status === "approved"
  ).length;

  let icon: ReactNode;
  let title: string;
  let subtitle: string;
  if (issue.approvalStatus === ApprovalStatus.REJECTED) {
    icon = <CircleX className="size-5 shrink-0 text-error" />;
    title = t("plan.lifecycle.gate-review-rejected");
    subtitle = t("plan.review.approval-flow.rejected-by", {
      user: (rejecter ?? unknownUser(rejectedStep?.approver ?? "")).title,
    });
  } else if (issue.approvalStatus === ApprovalStatus.APPROVED) {
    icon = <CircleCheck className="size-5 shrink-0 text-success" />;
    title = t("plan.lifecycle.gate-review-approved");
    subtitle = t("plan.summary.n-of-m-approved", {
      n: approvedCount,
      m: steps.length,
    });
  } else if (issue.approvalStatus === ApprovalStatus.SKIPPED) {
    icon = <CircleCheck className="size-5 shrink-0 text-success" />;
    title = t("custom-approval.approval-flow.skip");
    subtitle = "";
  } else {
    icon = <Clock3 className="size-5 shrink-0 text-control-light" />;
    title = t("plan.lifecycle.gate-review-pending");
    subtitle = currentStep
      ? t("plan.lifecycle.review-waiting", {
          role: displayRoleTitleFromList(currentStep.role, roleList),
        })
      : t("plan.summary.n-of-m-approved", {
          n: approvedCount,
          m: steps.length,
        });
  }

  return (
    <GateRow
      icon={icon}
      onToggle={onToggle}
      open={open}
      subtitle={subtitle}
      title={title}
    >
      <ReviewApprovalFlow compact issue={issue} />
    </GateRow>
  );
}

/* -------------------------- checks gate (ours) -------------------------- */

// Static (non-expandable) row: the header already summarizes the checks, so the
// only affordance is a View Details link into the full drawer.
function ChecksGateRow({
  summary,
  onViewDetails,
}: {
  summary: PlanCheckSummary;
  onViewDetails: () => void;
}) {
  const { t } = useTranslation();

  let icon: ReactNode;
  let title: string;
  let subtitle: string;
  if (summary.error > 0) {
    icon = <CircleX className="size-5 shrink-0 text-error" />;
    title = t("plan.lifecycle.gate-checks-failed");
    subtitle = t("plan.lifecycle.gate-checks-failed-sub", {
      failing: summary.error,
      passed: summary.success,
    });
  } else if (summary.running > 0) {
    icon = (
      <Loader2 className="size-5 shrink-0 animate-spin text-control-light" />
    );
    title = t("plan.lifecycle.gate-checks-running");
    subtitle = t("plan.lifecycle.gate-checks-running-sub", {
      running: summary.running,
      passed: summary.success,
    });
  } else {
    icon = <CircleCheck className="size-5 shrink-0 text-success" />;
    title = t("plan.lifecycle.gate-checks-passed");
    subtitle = t("plan.lifecycle.gate-checks-passed-sub", {
      count: summary.success,
    });
  }

  return (
    <div className="flex items-center gap-3 px-4 py-3">
      {icon}
      <GateLabel subtitle={subtitle} title={title} />
      {summary.total > 0 && (
        <button
          className="inline-flex shrink-0 items-center gap-0.5 text-xs text-accent hover:underline"
          onClick={onViewDetails}
          type="button"
        >
          {t("common.view-details")}
          <ChevronRight className="size-3.5" />
        </button>
      )}
    </div>
  );
}

/* --------------------------- footer (one action) ------------------------ */

function PlanStatusFooter({
  reason,
  issue,
  bypass,
  onBypass,
}: {
  reason: PlanStatusReason;
  issue: Issue;
  bypass: ReturnType<typeof useBypassDeploy>;
  onBypass: () => void;
}) {
  const { t } = useTranslation();
  if (reason === "rejected") {
    return (
      <div className="border-t px-4 py-2.5">
        <ReRequestGuidance issue={issue} />
      </div>
    );
  }
  if (bypass.weight === "none") {
    return null;
  }
  // Two-state note: the rollout either auto-creates once review approves (nothing
  // else blocking) or it wasn't created automatically (checks block it).
  const note =
    bypass.state.kind === "waiting-review" && bypass.checks.error === 0
      ? t("plan.review.footer.rollout-auto-creates")
      : t("plan.review.footer.rollout-not-created");
  return (
    <div className="flex items-center gap-x-3 border-t px-4 py-2.5">
      <span className="flex-1 text-xs text-control-placeholder">{note}</span>
      <BypassAction bypass={bypass} onClick={onBypass} />
    </div>
  );
}

// Mirrors ReviewRejectionBanner: guidance-prefix + accent inline link + suffix,
// gated on the issue creator (non-readonly); plain text otherwise.
function ReRequestGuidance({ issue }: { issue: Issue }) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const currentUser = useCurrentUser();
  const [reRequesting, setReRequesting] = useState(false);
  const canReRequest =
    issue.creator === `${userNamePrefix}${currentUser?.email ?? ""}` &&
    !page.readonly;

  const handleReRequest = async () => {
    if (reRequesting) return;
    try {
      setReRequesting(true);
      const response = await issueServiceClientConnect.requestIssue(
        create(RequestIssueRequestSchema, { name: issue.name })
      );
      applyProjectDetailMutationResult(page, { issue: response });
      await page.refreshState();
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    } finally {
      setReRequesting(false);
    }
  };

  return (
    <span className="text-xs leading-relaxed text-control-light">
      {t("plan.review.rejection.guidance-prefix")}{" "}
      {canReRequest ? (
        <button
          className="font-medium text-accent hover:underline disabled:opacity-60"
          disabled={reRequesting}
          onClick={() => void handleReRequest()}
          type="button"
        >
          {t("plan.review.rejection.re-request-review")}
        </button>
      ) : (
        <span>{t("plan.review.rejection.re-request-review")}</span>
      )}{" "}
      {t("plan.review.rejection.guidance-suffix")}
    </span>
  );
}

// The bypass override — always a quiet, muted link (a deliberate, demoted escape
// hatch). The confirm sheet still enforces mandatory project gates.
function BypassAction({
  bypass,
  onClick,
}: {
  bypass: ReturnType<typeof useBypassDeploy>;
  onClick: () => void;
}) {
  const { t } = useTranslation();
  return (
    <button
      className="inline-flex shrink-0 items-center gap-x-1 text-xs text-control-placeholder underline hover:text-control disabled:opacity-60"
      disabled={bypass.creating}
      onClick={onClick}
      type="button"
    >
      {bypass.creating && <Loader2 className="size-3 animate-spin" />}
      {t("plan.review.footer.bypass-and-deploy")}
    </button>
  );
}
