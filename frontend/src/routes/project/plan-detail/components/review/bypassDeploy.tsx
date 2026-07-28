// Shared "bypass and deploy" state machine + confirm sheet. Both the review
// readiness footer and the header's Plan status panel surface this manual-deploy
// override, so the gate/warning logic and the confirmation UI live here once
// (BYT-9722). The backend only enforces bb.rollouts.create on CreateRollout; the
// project "require issue approval" / "require no failed checks" settings are
// client-side gates that this module evaluates before allowing a bypass.
import { create } from "@bufbuild/protobuf";
import { Loader2 } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { rolloutServiceClientConnect } from "@/api";
import { PlanCheckSection } from "@/components/plan-check/PlanCheckSection";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { getPlanCheckSummaryWithFallback } from "@/lib/plan/check";
import { pushNotification } from "@/stores";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import { focusPlanPhase } from "../../shell/focusPhase";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import type { PlanCheckSummary } from "../../utils/phaseSummary";
import { getPlanCheckSummary } from "../../utils/phaseSummary";
import { ReviewApprovalFlow } from "./ReviewApprovalFlow";
import {
  type BypassActionWeight,
  computeBypassActionWeight,
  computeReadinessFooterState,
  type ReadinessFooterState,
} from "./readinessFooterState";

export interface BypassDeploy {
  state: ReadinessFooterState;
  checks: PlanCheckSummary;
  weight: BypassActionWeight;
  creating: boolean;
  triggerDeploy: () => void;
  createRollout: () => Promise<void>;
  confirmOpen: boolean;
  setConfirmOpen: (open: boolean) => void;
  acknowledge: boolean;
  setAcknowledge: (value: boolean) => void;
  gates: { label: string; met: boolean }[];
  gatesBlocked: boolean;
  warnings: string[];
  needAck: boolean;
  confirmDisabled: boolean;
  planCheckSummary: PlanCheckSummary;
}

export function useBypassDeploy(issue: Issue, plan: Plan): BypassDeploy {
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
  const planCheckSummary = useMemo(
    () =>
      getPlanCheckSummaryWithFallback(
        page.planCheckRuns,
        plan.planCheckRunStatusCount
      ),
    [page.planCheckRuns, plan.planCheckRunStatusCount]
  );

  const openConfirm = () => {
    setAcknowledge(false);
    setConfirmOpen(true);
  };

  // When the review has passed and every plan check is green there is nothing to
  // bypass or confirm — deploy straight away. Any other case (failed/running
  // checks, or bypassing review) opens the confirm sheet.
  const triggerDeploy = () => {
    if (state.kind === "all-gates-passed" && checks.running === 0) {
      void createRollout();
    } else {
      openConfirm();
    }
  };

  const createRollout = async () => {
    if (creating) return;
    try {
      setCreating(true);
      await rolloutServiceClientConnect.createRollout(
        create(CreateRolloutRequestSchema, { parent: plan.name })
      );
      setConfirmOpen(false);
      await page.refreshState();
      // Land the user on the freshly created rollout.
      focusPlanPhase("deploy", page.expandPhase);
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

  return {
    state,
    checks,
    weight,
    creating,
    triggerDeploy,
    createRollout,
    confirmOpen,
    setConfirmOpen,
    acknowledge,
    setAcknowledge,
    gates,
    gatesBlocked,
    warnings,
    needAck,
    confirmDisabled,
    planCheckSummary,
  };
}

export function BypassAndDeploySheet({
  issue,
  bypass,
}: {
  issue: Issue;
  bypass: BypassDeploy;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const {
    gates,
    gatesBlocked,
    warnings,
    needAck,
    acknowledge,
    setAcknowledge,
    confirmOpen,
    setConfirmOpen,
    confirmDisabled,
    creating,
    planCheckSummary,
  } = bypass;

  return (
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
            <Button
              onClick={() => setConfirmOpen(false)}
              appearance="secondary"
            >
              {t("common.cancel")}
            </Button>
            <Button
              disabled={confirmDisabled}
              onClick={() => void bypass.createRollout()}
            >
              {creating && <Loader2 className="size-4 animate-spin" />}
              {t("plan.navigator.deploy")}
            </Button>
          </div>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
