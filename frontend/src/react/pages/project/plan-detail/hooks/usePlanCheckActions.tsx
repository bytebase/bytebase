import { create } from "@bufbuild/protobuf";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { planServiceClientConnect } from "@/connect";
import { useVueState } from "@/react/hooks/useVueState";
import {
  projectNamePrefix,
  pushNotification,
  useCurrentUserV1,
  useProjectV1Store,
} from "@/store";
import { extractUserEmail } from "@/store/modules/v1/common";
import {
  GetPlanCheckRunRequestSchema,
  GetPlanRequestSchema,
  type PlanCheckRun,
  RunPlanChecksRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { usePlanDetailContext } from "../shell/PlanDetailContext";

// Wraps the run / refresh logic shared between PlanDetailChecks (per-spec
// section in the body) and the plan-level summary in PlanDetailMetadataSidebar.
// The running flag lives on the page context so both surfaces disable their
// Run buttons together — otherwise the user could trigger two concurrent
// runPlanChecks calls from the same page.
export function usePlanCheckActions() {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const { patchState, isRunningChecks, setIsRunningChecks } = page;
  const projectStore = useProjectV1Store();
  const currentUser = useCurrentUserV1().value;
  const projectName = `${projectNamePrefix}${page.projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));

  const allowRunChecks = useMemo(() => {
    if (page.plan.hasRollout) return false;
    if (extractUserEmail(page.plan.creator) === currentUser.email) return true;
    return hasProjectPermissionV2(project, "bb.planCheckRuns.run");
  }, [currentUser.email, page.plan.creator, page.plan.hasRollout, project]);

  const refreshChecks = useCallback(async (): Promise<PlanCheckRun[]> => {
    const [nextPlan, runOrNull] = await Promise.all([
      planServiceClientConnect.getPlan(
        create(GetPlanRequestSchema, { name: page.plan.name })
      ),
      planServiceClientConnect
        .getPlanCheckRun(
          create(GetPlanCheckRunRequestSchema, {
            name: `${page.plan.name}/planCheckRun`,
          })
        )
        .catch(() => null),
    ]);
    const nextPlanCheckRuns = runOrNull ? [runOrNull] : [];
    patchState({ plan: nextPlan, planCheckRuns: nextPlanCheckRuns });
    return nextPlanCheckRuns;
  }, [page.plan.name, patchState]);

  const runChecks = useCallback(async () => {
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
  }, [page.plan.name, refreshChecks, t]);

  return {
    allowRunChecks,
    isRunningChecks,
    refreshChecks,
    runChecks,
  };
}
