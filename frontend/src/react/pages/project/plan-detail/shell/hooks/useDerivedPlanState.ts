import { useMemo, useRef } from "react";
import { sortTaskRunsNewestFirst } from "@/react/lib/taskRun";
import type { TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { extractTaskNameFromTaskRunName } from "@/utils";
import type {
  PlanDetailPageSnapshot,
  PlanDetailPageState,
  PlanDetailPhase,
} from "./types";
import type { useEditingScopes } from "./useEditingScopes";
import type { usePhaseState } from "./usePhaseState";

type EditingScopes = ReturnType<typeof useEditingScopes>;
type PhaseState = ReturnType<typeof usePhaseState>;

export function useDerivedPlanState(params: {
  snapshot: PlanDetailPageSnapshot;
  creationIssueLabels: string[];
  setCreationIssueLabels: (labels: string[]) => void;
  isEditing: boolean;
  isRunningChecks: boolean;
  setIsRunningChecks: (running: boolean) => void;
  phase: PhaseState;
  editing: EditingScopes;
  routeName?: string;
  routePhase?: PlanDetailPhase;
  routeStageId?: string;
  routeTaskId?: string;
  patchState: (patch: Partial<PlanDetailPageSnapshot>) => void;
  refreshState: () => Promise<void>;
  resolveLeaveConfirm: (confirmed: boolean) => void;
}): PlanDetailPageState {
  const {
    snapshot,
    creationIssueLabels,
    setCreationIssueLabels,
    isEditing,
    isRunningChecks,
    setIsRunningChecks,
    phase,
    editing,
    routeName,
    routePhase,
    routeStageId,
    routeTaskId,
    patchState,
    refreshState,
    resolveLeaveConfirm,
  } = params;

  const selectedTaskName = useMemo(() => {
    if (!routeTaskId || !snapshot.rollout) {
      return undefined;
    }
    for (const stage of snapshot.rollout.stages) {
      const task = stage.tasks.find((item) =>
        item.name.endsWith(`/${routeTaskId}`)
      );
      if (task) {
        return task.name;
      }
    }
    return undefined;
  }, [routeTaskId, snapshot.rollout]);

  // Group the plan's task runs by task once for all consumers (every
  // kept-alive stage list reads from this map), reusing a task's previous
  // group array when all its runs kept their references — the snapshot gate's
  // per-run sharing makes that the common case — so memoized cards see stable
  // taskRuns/latestTaskRun props across poll ticks that only touched other
  // tasks.
  const taskRunsByTaskNameRef = useRef<Map<string, TaskRun[]>>(new Map());
  const taskRunsByTaskName = useMemo(() => {
    const prevGroups = taskRunsByTaskNameRef.current;
    const grouped = new Map<string, TaskRun[]>();
    for (const run of snapshot.taskRuns) {
      const taskName = extractTaskNameFromTaskRunName(run.name);
      const group = grouped.get(taskName);
      if (group) {
        group.push(run);
      } else {
        grouped.set(taskName, [run]);
      }
    }
    for (const [taskName, runs] of grouped) {
      const sorted = sortTaskRunsNewestFirst(runs);
      const prevRuns = prevGroups.get(taskName);
      grouped.set(
        taskName,
        prevRuns &&
          prevRuns.length === sorted.length &&
          sorted.every((run, index) => run === prevRuns[index])
          ? prevRuns
          : sorted
      );
    }
    taskRunsByTaskNameRef.current = grouped;
    return grouped;
  }, [snapshot.taskRuns]);

  return useMemo(
    () => ({
      ...snapshot,
      creationIssueLabels,
      setCreationIssueLabels,
      isEditing,
      isRunningChecks,
      setIsRunningChecks,
      activePhases: phase.activePhases,
      routeName,
      routePhase,
      routeStageId,
      selectedTaskName,
      taskRunsByTaskName,
      pendingLeaveConfirm: editing.pendingLeaveConfirm,
      bypassLeaveGuardOnce: editing.bypassLeaveGuardOnce,
      patchState,
      refreshState,
      setEditing: editing.setEditing,
      togglePhase: phase.togglePhase,
      expandPhase: phase.expandPhase,
      resolveLeaveConfirm,
    }),
    [
      editing,
      creationIssueLabels,
      isEditing,
      isRunningChecks,
      patchState,
      phase,
      refreshState,
      resolveLeaveConfirm,
      routeName,
      routePhase,
      routeStageId,
      setCreationIssueLabels,
      selectedTaskName,
      snapshot,
      taskRunsByTaskName,
    ]
  );
}
