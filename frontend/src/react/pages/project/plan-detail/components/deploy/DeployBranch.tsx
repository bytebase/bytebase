import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { router } from "@/router";
import { useProjectV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Rollout, Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import { generateRolloutPreview } from "../../utils/rolloutPreview";
import { DeployPendingTasksSection } from "./DeployPendingTasksSection";
import { DeployStageList } from "./DeployStageCard";
import { DeployStageContentView } from "./DeployStageContentView";
import type { DeployBranchProps, PendingTaskGroup } from "./types";

async function loadPendingGroups(
  plan: Plan,
  rollout: Rollout | undefined,
  projectName: string
): Promise<PendingTaskGroup[]> {
  const preview = await generateRolloutPreview(plan, projectName);
  const existing = new Set<string>();
  for (const stage of rollout?.stages ?? []) {
    for (const task of stage.tasks) {
      existing.add(`${task.target}:${task.specId}`);
    }
  }
  const groups = new Map<string, Task[]>();
  for (const stage of preview.stages) {
    for (const task of stage.tasks) {
      const key = `${task.target}:${task.specId}`;
      if (existing.has(key)) continue;
      const tasks = groups.get(stage.environment) ?? [];
      tasks.push(task);
      groups.set(stage.environment, tasks);
    }
  }
  return Array.from(groups.entries()).map(([environment, tasks]) => ({
    environment,
    tasks,
  }));
}

export function DeployBranch({
  selectedTask,
  onCloseTaskPanel: _onCloseTaskPanel,
}: DeployBranchProps) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const projectStore = useProjectV1Store();
  const project = useMemo(
    () =>
      projectStore.getProjectByName(`${projectNamePrefix}${page.projectId}`),
    [page.projectId, projectStore]
  );
  const projectName = `projects/${page.projectId}`;
  const [pendingOpen, setPendingOpen] = useState(false);
  const [pendingGroups, setPendingGroups] = useState<PendingTaskGroup[]>([]);
  const rolloutKey =
    page.rollout?.stages
      .map(
        (stage) =>
          `${stage.name}:${stage.tasks.map((task) => `${task.target}:${task.specId}`).join(",")}`
      )
      .join("|") ?? "";
  const planKey = `${page.plan.name}:${page.plan.specs.map((spec) => spec.id).join(",")}`;
  const selectedStage = useMemo(() => {
    if (!page.rollout?.stages.length) return undefined;
    if (selectedTask) {
      return page.rollout.stages.find((stage) =>
        stage.tasks.some((task) => task.name === selectedTask.name)
      );
    }
    if (page.routeStageId) {
      return page.rollout.stages.find((stage) =>
        stage.name.endsWith(`/${page.routeStageId}`)
      );
    }
    const firstIncomplete = page.rollout.stages.find((stage) =>
      stage.tasks.some(
        (task) =>
          task.status !== Task_Status.DONE &&
          task.status !== Task_Status.SKIPPED
      )
    );
    return firstIncomplete ?? page.rollout.stages[0];
  }, [page.rollout?.stages, page.routeStageId, selectedTask]);

  useEffect(() => {
    let canceled = false;
    const load = async () => {
      try {
        const groups = await loadPendingGroups(
          page.plan,
          page.rollout,
          projectName
        );
        if (!canceled) setPendingGroups(groups);
      } catch {
        if (!canceled) setPendingGroups([]);
      }
    };
    void load();
    return () => {
      canceled = true;
    };
  }, [planKey, projectName, rolloutKey]);

  const hasPendingTasks = pendingGroups.length > 0;
  const canCreateRollout =
    !page.readonly && hasProjectPermissionV2(project, "bb.rollouts.create");

  if (!page.rollout) {
    return null;
  }

  if (page.rollout.stages.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-4 py-10">
        <p className="text-control-placeholder">
          {t("rollout.no-tasks-created")}
        </p>
        {hasPendingTasks && canCreateRollout && (
          <Button onClick={() => setPendingOpen(true)} size="sm">
            {t("rollout.pending-tasks-preview.action")}
          </Button>
        )}
        <DeployPendingTasksSection
          onCreated={async () => {
            await page.refreshState();
          }}
          onOpenChange={setPendingOpen}
          open={pendingOpen}
          plan={page.plan}
          projectName={projectName}
          rollout={page.rollout}
        />
      </div>
    );
  }

  if (!selectedStage) {
    return (
      <div className="flex items-center justify-center py-12">
        <p className="text-gray-500">{t("rollout.no-stages")}</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col">
      <DeployStageList
        hasPendingTasks={hasPendingTasks}
        onOpenPreview={() => setPendingOpen(true)}
        onSelectStage={(stage) => {
          void router.push({
            query: {
              phase: "deploy",
              stageId: stage.name.split("/").pop(),
            },
          });
        }}
        rollout={page.rollout}
        selectedStageId={selectedStage.name}
      />

      <DeployStageContentView
        selectedTaskName={selectedTask?.name}
        stage={selectedStage}
      />

      <DeployPendingTasksSection
        onCreated={async () => {
          await page.refreshState();
        }}
        onOpenChange={setPendingOpen}
        open={pendingOpen}
        plan={page.plan}
        projectName={projectName}
        rollout={page.rollout}
      />
    </div>
  );
}
