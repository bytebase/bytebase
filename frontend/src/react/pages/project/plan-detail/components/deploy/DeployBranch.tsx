import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { useSeededState } from "@/react/hooks/useSeededState";
import { router } from "@/react/router";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Rollout, Task } from "@/types/proto-es/v1/rollout_service_pb";
import { extractStageUID } from "@/utils/v1/issue/rollout";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import { generateRolloutPreview } from "../../utils/rolloutPreview";
import { getFrontierStage } from "../lifecycle/frontierStage";
import { DeployPendingTasksSection } from "./DeployPendingTasksSection";
import { DeployStageList } from "./DeployStageCard";
import { DeployStageContentView } from "./DeployStageContentView";
import { autoExpandTaskName, deployTaskQuery } from "./DeployTaskList";
import type { PendingTaskGroup } from "./types";

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

export function DeployBranch() {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  // Full task resource name resolved from the ?taskId= deep link.
  const selectedTaskName = page.selectedTaskName;
  const projectName = `projects/${page.projectId}`;
  const [pendingOpen, setPendingOpen] = useState(false);
  const [pendingGroups, setPendingGroups] = useState<PendingTaskGroup[]>([]);
  // Content fingerprints for the pending-groups effect. rollout/plan
  // identities are poll-stable (snapshot gate), so the O(tasks) string builds
  // run only when content actually changed.
  const rolloutKey = useMemo(
    () =>
      page.rollout?.stages
        .map(
          (stage) =>
            `${stage.name}:${stage.tasks.map((task) => `${task.target}:${task.specId}`).join(",")}`
        )
        .join("|") ?? "",
    [page.rollout]
  );
  const planKey = useMemo(
    () =>
      `${page.plan.name}:${page.plan.specs.map((spec) => spec.id).join(",")}`,
    [page.plan]
  );

  // Stage selection is optimistic: a tab click paints in the same commit,
  // while router.push merely reflects it into the URL. Waiting for the route
  // round trip (which re-renders the whole plan page) reads as a dead click
  // followed by an all-at-once swap. useSeededState drops the override when
  // the URL catches up — or changes externally (back/forward).
  const [optimisticStageName, setOptimisticStageName] = useSeededState<
    string | undefined
  >(page.routeStageId ?? "", () => undefined);

  // Per-stage memory of the task mirrored in the URL (default pick, honored
  // deep link, or the last card the user opened) — restored into the URL when
  // switching back to a visited stage.
  const lastOpenedTaskByStage = useRef(new Map<string, string>());
  // Shared with the lists: ?taskId= values this page wrote itself are not
  // arrivals and must never scroll a card into view.
  const selfWrittenTaskRef = useRef<string | undefined>(undefined);
  const handleOpenedTaskChange = useCallback(
    (stageName: string, taskName: string) => {
      lastOpenedTaskByStage.current.set(stageName, taskName);
    },
    []
  );

  const selectedStage = useMemo(() => {
    if (!page.rollout?.stages.length) return undefined;
    // The optimistic pick wins over the (still-stale) URL selections.
    if (optimisticStageName) {
      const optimisticStage = page.rollout.stages.find(
        (stage) => stage.name === optimisticStageName
      );
      if (optimisticStage) {
        return optimisticStage;
      }
    }
    if (selectedTaskName) {
      const stageOfTask = page.rollout.stages.find((stage) =>
        stage.tasks.some((task) => task.name === selectedTaskName)
      );
      if (stageOfTask) {
        return stageOfTask;
      }
    }
    if (page.routeStageId) {
      return page.rollout.stages.find((stage) =>
        stage.name.endsWith(`/${page.routeStageId}`)
      );
    }
    // Default to the frontier (first non-complete) stage, falling back to the
    // first stage once every stage is complete.
    return getFrontierStage(page.rollout) ?? page.rollout.stages[0];
  }, [page.rollout, page.routeStageId, selectedTaskName, optimisticStageName]);

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
  const canCreateRollout = !page.readonly && page.projectCanCreateRollout;

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
          setOptimisticStageName(stage.name);
          // One URL write carrying the stage AND its task — the stage's
          // remembered task if visited, else its default pick. A stage-only
          // write would be followed ~a frame later by the list's default
          // ?taskId= mirror, costing a second page-wide render pass.
          const targetTask =
            lastOpenedTaskByStage.current.get(stage.name) ??
            autoExpandTaskName(stage.tasks);
          if (targetTask) {
            selfWrittenTaskRef.current = targetTask;
            void router.push({
              query: deployTaskQuery(stage.name, targetTask),
            });
          } else {
            void router.push({
              query: {
                phase: "deploy",
                stageId: extractStageUID(stage.name),
              },
            });
          }
        }}
        rollout={page.rollout}
        selectedStageId={selectedStage.name}
      />

      {/* Every stage's content stays mounted; only the selected one is
          visible. A stage switch just flips visibility, so card state —
          expansion, Monaco editors, fetched logs — survives instead of
          remounting (the flicker). Heavy per-stage work is deferred to first
          activation inside DeployTaskList. */}
      {page.rollout.stages.map((stage) => (
        <div
          className={stage.name === selectedStage.name ? undefined : "hidden"}
          key={stage.name}
        >
          <DeployStageContentView
            active={stage.name === selectedStage.name}
            onOpenedTaskChange={handleOpenedTaskChange}
            selfWrittenTaskRef={selfWrittenTaskRef}
            stage={stage}
          />
        </div>
      ))}

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
