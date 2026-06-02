import { create } from "@bufbuild/protobuf";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { planServiceClientConnect } from "@/connect";
import { PlanCheckSection } from "@/react/components/plan-check/PlanCheckSection";
import { useCurrentUser } from "@/react/hooks/useAppState";
import { useVueState } from "@/react/hooks/useVueState";
import { useAppStore } from "@/react/stores/app";
import { pushNotification } from "@/store";
import { extractUserEmail, projectNamePrefix } from "@/store/modules/v1/common";
import {
  GetPlanCheckRunRequestSchema,
  GetPlanRequestSchema,
  type PlanCheckRun,
  RunPlanChecksRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { getPlanCheckSummaryWithFallback } from "../../plan-detail/utils/planCheck";
import { useIssueDetailContext } from "../context/IssueDetailContext";

export function IssueDetailChecks() {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  // subscribe to re-render on project cache change
  const projectsByName = useAppStore((s) => s.projectsByName);
  const currentUser = useCurrentUser();
  const [isRunningChecks, setIsRunningChecks] = useState(false);
  const projectName = `${projectNamePrefix}${page.projectId}`;
  const project = useVueState(() =>
    useAppStore.getState().getProjectByName(projectName)
  );
  void projectsByName;

  const summary = useMemo(
    () =>
      getPlanCheckSummaryWithFallback(
        page.planCheckRuns,
        page.plan?.planCheckRunStatusCount
      ),
    [page.planCheckRuns, page.plan?.planCheckRunStatusCount]
  );

  const allowRunChecks = useMemo(() => {
    if (!page.plan) return false;
    // Once a rollout exists the plan is frozen — re-running checks would only
    // produce the same results and misleads the user.
    if (page.plan.hasRollout) return false;
    if (extractUserEmail(page.plan.creator) === currentUser.email) return true;
    return hasProjectPermissionV2(project, "bb.planCheckRuns.run");
  }, [currentUser.email, page.plan, project]);

  const refreshChecks = useCallback(async (): Promise<PlanCheckRun[]> => {
    const planName = page.plan?.name;
    if (!planName) return [];
    const [nextPlan, runOrNull] = await Promise.all([
      planServiceClientConnect.getPlan(
        create(GetPlanRequestSchema, { name: planName })
      ),
      planServiceClientConnect
        .getPlanCheckRun(
          create(GetPlanCheckRunRequestSchema, {
            name: `${planName}/planCheckRun`,
          })
        )
        .catch(() => null),
    ]);
    const nextPlanCheckRuns = runOrNull ? [runOrNull] : [];
    page.patchState({ plan: nextPlan, planCheckRuns: nextPlanCheckRuns });
    return nextPlanCheckRuns;
  }, [page.plan?.name, page.patchState]);

  const runChecks = useCallback(async () => {
    if (!page.plan?.name) return;
    try {
      setIsRunningChecks(true);
      await planServiceClientConnect.runPlanChecks(
        create(RunPlanChecksRequestSchema, { name: page.plan.name })
      );
      await refreshChecks();
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("plan.checks.started"),
      });
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("plan.checks.failed-to-run"),
        description: String(error),
      });
    } finally {
      setIsRunningChecks(false);
    }
  }, [page.plan?.name, refreshChecks, t]);

  if (!page.plan) return null;

  return (
    <PlanCheckSection
      canRun={allowRunChecks}
      headingClassName="textlabel"
      includeRunFailure
      isRunning={isRunningChecks}
      onRefreshOnOpen={refreshChecks}
      onRun={runChecks}
      planCheckRuns={page.planCheckRuns}
      summaryOverride={summary}
    />
  );
}
