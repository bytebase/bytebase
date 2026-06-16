import { create } from "@bufbuild/protobuf";
import { Ban, CircleCheck, CircleX, Clock3, Loader2 } from "lucide-react";
import { Fragment, type ReactNode, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { rolloutServiceClientConnect } from "@/connect";
import { PlanCheckSection } from "@/react/components/plan-check/PlanCheckSection";
import { Alert } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { pushNotification } from "@/store";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import { getPlanCheckSummary } from "../../utils/phaseSummary";
import { getPlanCheckSummaryWithFallback } from "../../utils/planCheck";
import { ReviewApprovalFlow } from "./ReviewApprovalFlow";
import {
  computeBypassActionWeight,
  computeReadinessFooterState,
} from "./readinessFooterState";

export function ReviewReadinessFooter({
  issue,
  plan,
}: {
  issue: Issue;
  plan: Plan;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const [creating, setCreating] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [acknowledge, setAcknowledge] = useState(false);

  const checks = useMemo(() => getPlanCheckSummary(plan), [plan]);
  const state = useMemo(
    () =>
      computeReadinessFooterState({
        hasRollout: plan.hasRollout,
        issueStatus: issue.status,
        approvalStatus: issue.approvalStatus,
        checks,
      }),
    [checks, issue.approvalStatus, issue.status, plan.hasRollout]
  );
  const weight = computeBypassActionWeight({
    state: state.kind,
    canCreateRollout: page.projectCanCreateRollout && !page.readonly,
    requireIssueApproval: page.projectRequireIssueApproval,
  });
  // Must run before the "hidden" early return below — otherwise the hook count
  // drops when a deploy flips the footer to hidden, and React throws "rendered
  // fewer hooks than expected" on the success path.
  const planCheckSummary = useMemo(
    () =>
      getPlanCheckSummaryWithFallback(
        page.planCheckRuns,
        plan.planCheckRunStatusCount
      ),
    [page.planCheckRuns, plan.planCheckRunStatusCount]
  );

  if (state.kind === "hidden") return null;

  const openConfirm = () => {
    setAcknowledge(false);
    setConfirmOpen(true);
  };

  // When the review has passed and every plan check is green there is nothing
  // to bypass or confirm — deploy straight away. Any other case (failed/running
  // checks, or bypassing review) opens the confirm sheet.
  const triggerDeploy = () => {
    if (state.kind === "all-gates-passed" && checks.running === 0) {
      void bypass();
    } else {
      openConfirm();
    }
  };

  const bypass = async () => {
    if (creating) return;
    try {
      setCreating(true);
      await rolloutServiceClientConnect.createRollout(
        create(CreateRolloutRequestSchema, { parent: plan.name })
      );
      setConfirmOpen(false);
      await page.refreshState();
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    } finally {
      setCreating(false);
    }
  };

  // The project's enforced gates and whether each is satisfied. Enforced gates
  // are HARD: if any is unmet, bypass-and-deploy is blocked entirely — you can
  // only bypass the project's *optional* requirements, never its mandatory ones.
  const reviewApproved =
    state.kind === "all-gates-passed" ||
    state.kind === "approved-checks-failed";
  const gates: { label: string; met: boolean }[] = [];
  if (page.projectRequireIssueApproval) {
    gates.push({
      label: t("plan.review.footer.gate-approval"),
      met: reviewApproved,
    });
  }
  if (page.projectRequirePlanCheckNoError) {
    gates.push({
      label: t("plan.review.footer.gate-no-failed-checks"),
      met: checks.error === 0,
    });
  }
  const gatesBlocked = gates.some((gate) => !gate.met);

  // Soft items this bypass would skip (NOT enforced by project policy) — shown
  // as warnings the user must acknowledge. Moot when a hard gate blocks deploy.
  const warnings: string[] = [];
  if (!page.projectRequireIssueApproval) {
    if (state.kind === "rejected") {
      warnings.push(t("plan.review.footer.confirm-review-rejected"));
    } else if (state.kind === "waiting-review") {
      warnings.push(t("plan.review.footer.confirm-review-not-approved"));
    }
  }
  if (!page.projectRequirePlanCheckNoError && checks.error > 0) {
    warnings.push(
      t("plan.review.footer.confirm-checks-failed", { count: checks.error })
    );
  }
  if (checks.running > 0) {
    warnings.push(t("plan.review.footer.confirm-checks-running"));
  }
  const needAck = warnings.length > 0;
  const confirmDisabled = creating || gatesBlocked || (needAck && !acknowledge);

  // Each non-zero count is its own "·"-separated segment with a leading dot, so
  // empty buckets (e.g. "0 checks passed") are dropped rather than shown.
  const checkSegments: { key: string; node: ReactNode }[] = [];
  if (checks.error > 0) {
    checkSegments.push({
      key: "failed",
      node: (
        <span className="whitespace-nowrap text-error">
          {t("plan.review.footer.n-checks-failed", { count: checks.error })}
        </span>
      ),
    });
  }
  if (checks.success > 0) {
    checkSegments.push({
      key: "passed",
      node: (
        <span className="whitespace-nowrap">
          {t("plan.review.footer.n-checks-passed", { count: checks.success })}
        </span>
      ),
    });
  }
  const checkCounts = checkSegments.map((segment) => (
    <Fragment key={segment.key}>
      <span>·</span>
      {segment.node}
    </Fragment>
  ));

  return (
    <div
      className={
        state.kind === "approved-checks-failed"
          ? "flex flex-wrap items-center gap-x-2 gap-y-1 border-t px-4 py-2.5"
          : "flex flex-wrap items-center gap-x-2 gap-y-1 border-t px-4 py-2.5 text-sm text-control-placeholder"
      }
    >
      {state.kind === "waiting-review" && (
        <>
          <Clock3 className="size-4 shrink-0" />
          <span className="whitespace-nowrap font-medium text-control">
            {t("plan.review.footer.waiting-on-review")}
          </span>
          <span className="hidden sm:inline">·</span>
          <span className="hidden whitespace-nowrap sm:inline">
            {checks.error > 0
              ? t("plan.review.footer.rollout-blocked-by-failed-checks")
              : t("plan.review.footer.auto-rollout-after-approval")}
          </span>
          {checkCounts}
        </>
      )}
      {state.kind === "all-gates-passed" && (
        <>
          <CircleCheck className="size-4 shrink-0 text-success" />
          <span className="whitespace-nowrap font-medium text-control">
            {t("plan.review.footer.all-gates-passed")}
          </span>
          <span className="hidden sm:inline">·</span>
          <span className="hidden whitespace-nowrap sm:inline">
            {t("plan.review.footer.creating-rollout-automatically")}
          </span>
          {checkCounts}
        </>
      )}
      {state.kind === "approved-checks-failed" && (
        <>
          <CircleX className="size-6 shrink-0 text-error" />
          <div className="min-w-0 flex-1">
            <div className="text-sm font-semibold text-main">
              {t("plan.review.footer.approved-but-checks-failed")}
            </div>
            <div className="text-xs text-control-placeholder">
              {t("plan.review.footer.errors-passed-not-created", {
                count: checks.error,
                passed: checks.success,
              })}
            </div>
          </div>
        </>
      )}
      {state.kind === "rejected" && (
        <>
          <Ban className="size-4 shrink-0" />
          <span className="min-w-0 truncate">
            {t("plan.review.footer.blocked-by-rejection")}
          </span>
        </>
      )}

      {weight === "link" && (
        <button
          className="ml-auto inline-flex shrink-0 items-center gap-x-1 text-xs text-control-placeholder underline hover:text-control disabled:opacity-60"
          disabled={creating}
          onClick={triggerDeploy}
          type="button"
        >
          {creating && <Loader2 className="size-3 animate-spin" />}
          {t("plan.review.footer.bypass-and-deploy")}
        </button>
      )}
      {weight === "button" && (
        <Button
          className="ml-auto shrink-0"
          disabled={creating}
          onClick={triggerDeploy}
          variant="outline"
        >
          {creating && <Loader2 className="size-4 animate-spin" />}
          {t("plan.review.footer.bypass-and-deploy")}
        </Button>
      )}

      <Sheet
        onOpenChange={(open) => {
          setConfirmOpen(open);
          if (!open) setAcknowledge(false);
        }}
        open={confirmOpen}
      >
        <SheetContent
          className="w-[28rem] max-w-[calc(100vw-2rem)]"
          width="standard"
        >
          <SheetHeader>
            <SheetTitle>{t("plan.review.footer.bypass-and-deploy")}</SheetTitle>
          </SheetHeader>
          <SheetBody className="gap-y-4">
            {gatesBlocked ? (
              <Alert
                description={
                  <ul className="list-inside list-disc text-sm">
                    {gates
                      .filter((gate) => !gate.met)
                      .map((gate) => (
                        <li key={gate.label}>{gate.label}</li>
                      ))}
                  </ul>
                }
                title={t("plan.review.footer.gates-blocked")}
                variant="error"
              />
            ) : (
              needAck && (
                <Alert
                  variant="warning"
                  description={
                    <ul className="list-inside list-disc text-sm">
                      {warnings.map((warning) => (
                        <li key={warning}>{warning}</li>
                      ))}
                    </ul>
                  }
                  title={t("plan.review.footer.confirm-bypass-title")}
                />
              )
            )}

            <div className="flex flex-col gap-y-1">
              <h3 className="textlabel uppercase">
                {t("plan.navigator.review")}
              </h3>
              <div className="rounded-md border">
                <ReviewApprovalFlow issue={issue} />
              </div>
            </div>

            <PlanCheckSection
              canRun={false}
              includeRunFailure
              isRunning={false}
              onRun={() => {}}
              planCheckRuns={page.planCheckRuns}
              summaryOverride={planCheckSummary}
            />
          </SheetBody>
          <SheetFooter className="justify-between">
            {!gatesBlocked && needAck ? (
              <label className="flex items-center gap-x-2 text-sm text-control">
                <Checkbox
                  checked={acknowledge}
                  disabled={creating}
                  onCheckedChange={setAcknowledge}
                />
                <span>{t("plan.review.footer.confirm-acknowledge")}</span>
              </label>
            ) : (
              <div />
            )}
            <div className="flex items-center gap-x-2">
              <Button onClick={() => setConfirmOpen(false)} variant="ghost">
                {t("common.cancel")}
              </Button>
              <Button disabled={confirmDisabled} onClick={() => void bypass()}>
                {creating && <Loader2 className="size-4 animate-spin" />}
                {t("plan.navigator.deploy")}
              </Button>
            </div>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </div>
  );
}
