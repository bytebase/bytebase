import { create } from "@bufbuild/protobuf";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { planServiceClientConnect } from "@/api";
import { useCurrentUser } from "@/hooks/useAppState";
import { useProjectByName } from "@/hooks/useProjectByName";
import { projectNamePrefix, pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { extractUserEmail } from "@/stores/modules/v1/common";
import { RunPlanChecksRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { usePlanDetailContext } from "../shell/PlanDetailContext";

// The running flag lives on the page context so concurrent triggers of
// runPlanChecks on the same page are disabled together.
export function usePlanCheckActions() {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const { isRunningChecks, refreshState, setIsRunningChecks } = page;
  // subscribe to re-render on project cache change
  const projectsByName = useAppStore((s) => s.projectsByName);
  void projectsByName;
  const currentUser = useCurrentUser();
  const projectName = `${projectNamePrefix}${page.projectId}`;
  const project = useProjectByName(projectName);

  const allowRunChecks = useMemo(() => {
    // A closed/done issue (or deleted plan) is read-only — running checks is
    // rejected by the backend too, so hide the action.
    if (page.readonly) return false;
    if (page.plan.hasRollout) return false;
    if (extractUserEmail(page.plan.creator) === currentUser.email) return true;
    return hasProjectPermissionV2(project, "bb.planCheckRuns.run");
  }, [
    currentUser.email,
    page.plan.creator,
    page.plan.hasRollout,
    page.readonly,
    project,
  ]);

  const runChecks = useCallback(async () => {
    try {
      setIsRunningChecks(true);
      await planServiceClientConnect.runPlanChecks(
        create(RunPlanChecksRequestSchema, { name: page.plan.name })
      );
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("plan.checks.started"),
      });
      await refreshState().catch(() => undefined);
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
  }, [page.plan.name, refreshState, setIsRunningChecks, t]);

  return {
    allowRunChecks,
    isRunningChecks,
    refreshChecks: refreshState,
    runChecks,
  };
}
