import { create } from "@bufbuild/protobuf";
import {
  CheckCircle2,
  CircleAlert,
  Clock3,
  Loader2,
  MinusCircle,
} from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { rolloutServiceClientConnect } from "@/connect";
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
import { router } from "@/router";
import { buildPlanDeployRouteFromRolloutName } from "@/router/dashboard/projectV1RouteHelpers";
import { pushNotification } from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import { isApprovalCompleted } from "../../issue-detail/utils/approval";
import { usePlanDetailContext } from "../context/PlanDetailContext";
import { getPlanCheckSummary } from "../utils/phaseSummary";

export function PlanDetailDeployFuture() {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const [creatingRollout, setCreatingRollout] = useState(false);
  const [rolloutConfirmOpen, setRolloutConfirmOpen] = useState(false);
  const [bypassWarnings, setBypassWarnings] = useState(false);

  const planCheckSummary = useMemo(
    () => getPlanCheckSummary(page.plan),
    [page.plan]
  );
  const planChecksFailed = planCheckSummary.error > 0;
  const planChecksRunning = planCheckSummary.running > 0;
  const hasAnyPlanCheckRun = planCheckSummary.total > 0;
  const issueApproved = isApprovalCompleted(page.issue);
  const canCreateRollout = Boolean(
    page.issue &&
      !page.plan.hasRollout &&
      page.plan.state === State.ACTIVE &&
      page.issue.status === IssueStatus.OPEN &&
      page.projectCanCreateRollout
  );
  const showManualCreateRolloutHint = Boolean(
    page.issue &&
      !page.plan.hasRollout &&
      page.plan.state === State.ACTIVE &&
      page.issue.status === IssueStatus.OPEN &&
      !(page.projectRequireIssueApproval && !issueApproved) &&
      !(page.projectRequirePlanCheckNoError && planChecksFailed) &&
      (!page.projectRequireIssueApproval ||
        !page.projectRequirePlanCheckNoError)
  );
  const manualCreateRolloutDescription = canCreateRollout
    ? t("plan.phase.deploy-manual-create-description")
    : t("plan.phase.deploy-manual-create-description-readonly");
  const errorMessages = useMemo(() => {
    const messages: string[] = [];
    if (!page.projectCanCreateRollout) {
      messages.push(
        t("common.missing-required-permission", {
          permissions: "bb.rollouts.create",
        })
      );
    }
    if (page.projectRequireIssueApproval && !issueApproved) {
      messages.push(
        t("project.settings.issue-related.require-issue-approval.description")
      );
    }
    if (page.projectRequirePlanCheckNoError && planChecksFailed) {
      messages.push(
        t(
          "project.settings.issue-related.require-plan-check-no-error.description"
        )
      );
    }
    return messages;
  }, [
    issueApproved,
    page.projectCanCreateRollout,
    page.projectRequireIssueApproval,
    page.projectRequirePlanCheckNoError,
    planChecksFailed,
    t,
  ]);
  const warningMessages = useMemo(() => {
    const messages: string[] = [];
    if (!page.projectRequireIssueApproval && !issueApproved) {
      messages.push(
        t("project.settings.issue-related.require-issue-approval.description")
      );
    }
    if (planChecksRunning) {
      messages.push(
        t(
          "custom-approval.issue-review.disallow-approve-reason.some-task-checks-are-still-running"
        )
      );
    } else if (!page.projectRequirePlanCheckNoError && planChecksFailed) {
      messages.push(
        t(
          "project.settings.issue-related.require-plan-check-no-error.description"
        )
      );
    }
    return messages;
  }, [
    issueApproved,
    page.projectRequireIssueApproval,
    page.projectRequirePlanCheckNoError,
    planChecksFailed,
    planChecksRunning,
    t,
  ]);
  const createRolloutDisabled =
    creatingRollout ||
    errorMessages.length > 0 ||
    (warningMessages.length > 0 && !bypassWarnings);

  const createRollout = async () => {
    if (creatingRollout) return;
    try {
      setCreatingRollout(true);
      const createdRollout = await rolloutServiceClientConnect.createRollout(
        create(CreateRolloutRequestSchema, {
          parent: page.plan.name,
        })
      );
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.created"),
      });
      await page.refreshState();
      setRolloutConfirmOpen(false);
      void router.push(
        buildPlanDeployRouteFromRolloutName(createdRollout.name)
      );
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    } finally {
      setCreatingRollout(false);
    }
  };

  const statePresets = useMemo(
    () => ({
      done: {
        icon: CheckCircle2,
        iconClass: "text-success/80",
        statusClass: "text-success",
        tagLabel: t("common.done"),
      },
      failed: {
        icon: CircleAlert,
        iconClass: "text-error/80",
        statusClass: "text-error",
        tagLabel: t("common.failed"),
      },
      running: {
        icon: Clock3,
        iconClass: "text-warning/80",
        statusClass: "text-warning",
        tagLabel: t("common.in-progress"),
      },
      pending: {
        icon: Clock3,
        iconClass: "text-warning/80",
        statusClass: "text-warning",
        tagLabel: t("common.pending"),
      },
      optional: {
        icon: MinusCircle,
        iconClass: "text-control-placeholder",
        statusClass: "text-control-placeholder",
        tagLabel: t("common.optional"),
      },
    }),
    [t]
  );
  const checksItem = useMemo(() => {
    const base = {
      key: "checks",
      label: t("plan.phase.deploy-checks-must-pass"),
    };
    if (planChecksRunning) {
      return {
        ...base,
        ...statePresets.running,
        description: t("plan.phase.deploy-checks-running"),
      };
    }
    if (planChecksFailed) {
      return {
        ...base,
        ...statePresets.failed,
        description: t("plan.phase.deploy-checks-blocked"),
      };
    }
    if (hasAnyPlanCheckRun || page.projectRequirePlanCheckNoError) {
      return {
        ...base,
        ...statePresets.done,
        description: t("plan.phase.deploy-checks-ready"),
      };
    }
    return {
      ...base,
      ...statePresets.optional,
      description: t("plan.phase.deploy-checks-optional"),
    };
  }, [
    t,
    statePresets,
    planChecksRunning,
    planChecksFailed,
    hasAnyPlanCheckRun,
    page.projectRequirePlanCheckNoError,
  ]);
  const approvalItem = useMemo(() => {
    const base = {
      key: "approval",
      label: t("plan.phase.deploy-approval-must-complete"),
    };
    if (issueApproved) {
      return {
        ...base,
        ...statePresets.done,
        description: t("plan.phase.deploy-approval-ready"),
      };
    }
    if (page.projectRequireIssueApproval) {
      return {
        ...base,
        ...statePresets.pending,
        description: t("plan.phase.deploy-approval-pending"),
      };
    }
    return {
      ...base,
      ...statePresets.optional,
      description: t("plan.phase.deploy-approval-optional"),
    };
  }, [t, statePresets, issueApproved, page.projectRequireIssueApproval]);
  const requirementItems = [checksItem, approvalItem];

  return (
    <div className="mt-1.5 flex flex-col gap-y-3">
      <p className="text-sm text-control-placeholder">
        {t("plan.phase.deploy-description")}
      </p>

      {page.issue && (
        <ul className="flex max-w-[28rem] flex-col gap-y-1">
          {requirementItems.map((item) => {
            const Icon = item.icon;
            return (
              <li
                key={item.key}
                className="flex min-w-0 items-start gap-x-2 py-1"
              >
                <Icon className={`mt-0.5 size-4 shrink-0 ${item.iconClass}`} />
                <div className="flex min-w-0 flex-col gap-y-0.5">
                  <div className="flex flex-wrap items-center gap-x-2">
                    <span className="text-sm font-medium text-control">
                      {item.label}
                    </span>
                    <span className={`text-sm ${item.statusClass}`}>
                      {item.tagLabel}
                    </span>
                  </div>
                  <p className="text-xs text-control-placeholder">
                    {item.description}
                  </p>
                </div>
              </li>
            );
          })}
        </ul>
      )}

      {showManualCreateRolloutHint && (
        <div className="flex flex-col items-start gap-y-2 border-t border-control-border pt-3">
          <p className="text-sm text-control-light">
            {manualCreateRolloutDescription}
          </p>
          {canCreateRollout && (
            <Button
              disabled={creatingRollout}
              onClick={() => setRolloutConfirmOpen(true)}
            >
              {t("plan.phase.create-rollout-action")}
            </Button>
          )}
        </div>
      )}

      <Sheet
        onOpenChange={(open) => {
          setRolloutConfirmOpen(open);
          if (!open) {
            setBypassWarnings(false);
          }
        }}
        open={rolloutConfirmOpen}
      >
        <SheetContent
          className="w-[28rem] max-w-[calc(100vw-2rem)]"
          width="standard"
        >
          <SheetHeader>
            <SheetTitle>{t("issue.create-rollout")}</SheetTitle>
          </SheetHeader>
          <SheetBody className="gap-y-4">
            {errorMessages.length > 0 ? (
              <Alert
                variant="error"
                title={t("common.error")}
                description={
                  <ul className="list-inside list-disc text-sm">
                    {errorMessages.map((message) => (
                      <li key={message}>{message}</li>
                    ))}
                  </ul>
                }
              />
            ) : warningMessages.length > 0 ? (
              <Alert
                variant="warning"
                title={t("common.warning")}
                description={
                  <ul className="list-inside list-disc text-sm">
                    {warningMessages.map((message) => (
                      <li key={message}>{message}</li>
                    ))}
                  </ul>
                }
              />
            ) : null}

            {page.issue && (
              <div className="flex flex-col gap-y-2">
                <span className="font-medium text-control">
                  {t("plan.navigator.review")}
                </span>
                <span className="text-sm text-control-light">
                  {issueApproved
                    ? t("plan.phase.deploy-approval-ready")
                    : t("plan.phase.deploy-approval-pending")}
                </span>
              </div>
            )}

            <PlanCheckStatusSummary
              planChecksFailed={planChecksFailed}
              planChecksRunning={planChecksRunning}
            />
          </SheetBody>
          <SheetFooter className="justify-between">
            {warningMessages.length > 0 && errorMessages.length === 0 ? (
              <label className="flex items-center gap-x-2 text-sm text-control">
                <Checkbox
                  checked={bypassWarnings}
                  disabled={creatingRollout}
                  onCheckedChange={(checked) => setBypassWarnings(checked)}
                />
                <span>{t("rollout.bypass-stage-requirements")}</span>
              </label>
            ) : (
              <div />
            )}
            <div className="flex items-center gap-x-2">
              <Button
                onClick={() => setRolloutConfirmOpen(false)}
                variant="ghost"
              >
                {t("common.cancel")}
              </Button>
              <Button
                disabled={createRolloutDisabled}
                onClick={() => void createRollout()}
              >
                {creatingRollout && (
                  <Loader2 className="h-4 w-4 animate-spin" />
                )}
                {t("common.confirm")}
              </Button>
            </div>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </div>
  );
}

function PlanCheckStatusSummary({
  planChecksFailed,
  planChecksRunning,
}: {
  planChecksFailed: boolean;
  planChecksRunning: boolean;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const statusCount = page.plan.planCheckRunStatusCount ?? {};
  const hasAnyChecks = Object.values(statusCount).some((count) => count > 0);

  return (
    <div className="flex flex-col gap-y-2">
      <span className="font-medium text-control">
        {t("plan.navigator.checks")}
      </span>
      {hasAnyChecks ? (
        <div className="flex flex-wrap items-center gap-2 text-sm text-control-light">
          {planChecksRunning && <span>{t("common.running")}</span>}
          {planChecksFailed ? (
            <span className="text-error">{t("common.failed")}</span>
          ) : (
            <span className="text-success">{t("common.done")}</span>
          )}
        </div>
      ) : (
        <span className="text-sm text-control-placeholder">
          {t("plan.overview.no-checks")}
        </span>
      )}
    </div>
  );
}
