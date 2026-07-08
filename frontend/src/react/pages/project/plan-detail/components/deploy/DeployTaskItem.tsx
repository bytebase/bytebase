import { memo, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { formatTaskRunDuration } from "@/react/lib/taskRun";
import { cn } from "@/react/lib/utils";
import { getTimeForPbTimestampProtoEs } from "@/types";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  type Stage,
  type Task,
  Task_Status,
  type TaskRun,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { databaseForTask } from "@/utils";
import { isReleaseBasedTask } from "@/utils/v1/issue/rollout";
import { isRerunnableTaskStatus } from "../lifecycle/frontierStage";
import { PlanDetailRollbackSheet } from "../PlanDetailRollbackSheet";
import {
  type PlanDetailTaskRolloutAction,
  PlanDetailTaskRolloutActionPanel,
} from "../PlanDetailTaskRolloutActionPanel";
import { DeployTaskBody } from "./DeployTaskBody";
import { DeployTaskHeader } from "./DeployTaskHeader";
import { DeployTaskRunHistorySheet } from "./DeployTaskRunHistorySheet";
import { useDeployTaskActions } from "./taskActions";
import { useDeployTaskStatement } from "./useDeployTaskStatement";

// Orchestrates a single deploy task card: derives the display state, owns the
// action/rollback/history overlays, and composes the presentational
// DeployTaskHeader + DeployTaskBody. The card is one disclosure toggle.
//
// memo + props only (no page context): every input is identity-stable across
// poll ticks thanks to the snapshot gate's structural sharing, so a tick that
// changes one task re-renders that one card — the other mounted cards (across
// every kept-alive stage) skip entirely.
export const DeployTaskItem = memo(function DeployTaskItem({
  currentUser,
  deepLinked,
  isExpanded,
  isSelected,
  isSelectable,
  issue,
  onRefresh,
  onToggleExpand,
  onToggleSelect,
  plan,
  project,
  rolloutName,
  stage,
  task,
  taskRuns,
}: {
  currentUser: User;
  deepLinked: boolean;
  isExpanded: boolean;
  isSelected: boolean;
  isSelectable: boolean;
  issue?: Issue;
  onRefresh: () => Promise<void>;
  onToggleExpand: (task: Task) => void;
  onToggleSelect: (task: Task) => void;
  plan: Plan;
  project: Project;
  rolloutName: string;
  stage: Stage;
  task: Task;
  // This task's runs, newest first (grouped upstream with stable identities).
  taskRuns: TaskRun[];
}) {
  const { t } = useTranslation();
  const latestTaskRun = taskRuns[0];
  const [action, setAction] = useState<
    PlanDetailTaskRolloutAction | undefined
  >();
  const [actionOpen, setActionOpen] = useState(false);
  const [rollbackOpen, setRollbackOpen] = useState(false);
  const [historyOpen, setHistoryOpen] = useState(false);
  // A deep link (?taskId=) brings its card into view once. `deepLinked` only
  // changes with the route, so polling re-renders don't re-trigger the scroll.
  // Skip when the card top is already on screen — opening a card writes its own
  // ?taskId=, and the card the user just clicked must not yank the viewport.
  const hostRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!deepLinked) {
      return;
    }
    const host = hostRef.current;
    if (!host) {
      return;
    }
    const { top } = host.getBoundingClientRect();
    const alreadyVisible = top >= 0 && top <= window.innerHeight * 0.8;
    if (!alreadyVisible) {
      host.scrollIntoView({ block: "center" });
    }
  }, [deepLinked]);
  const isReleaseTask = isReleaseBasedTask(task);
  const {
    isLoading: isStatementLoading,
    isTruncated,
    statement,
  } = useDeployTaskStatement({
    enabled: isExpanded && !isReleaseTask,
    task,
  });
  const { canCancel, canRun, canSkip } = useDeployTaskActions({
    currentUser,
    issue,
    project,
    stage,
    task,
  });
  const rollbackableTaskRun =
    latestTaskRun &&
    latestTaskRun.status === TaskRun_Status.DONE &&
    latestTaskRun.hasPriorBackup
      ? latestTaskRun
      : undefined;
  const scheduledTimeTs =
    task.runTime && task.status === Task_Status.PENDING
      ? getTimeForPbTimestampProtoEs(task.runTime, 0) / 1000
      : 0;
  // Same duration source as the history sheet and the expanded body, so all
  // three render identical strings for the same run.
  const timingDisplay = latestTaskRun
    ? formatTaskRunDuration(latestTaskRun)
    : "";
  const collapsedContextInfo =
    task.status === Task_Status.DONE ? timingDisplay : "";
  const collapsedStatusText =
    latestTaskRun?.detail ||
    task.skippedReason ||
    (task.status === Task_Status.FAILED
      ? t("task.status.failed")
      : task.status === Task_Status.SKIPPED
        ? t("task.status.skipped")
        : "");
  const actionItems = [
    {
      key: "RUN" as const,
      // Match the stage/plan-header verb: re-executing a failed or canceled
      // task is "Rerun"; a fresh execution is "Run".
      label: isRerunnableTaskStatus(task.status)
        ? t("plan.lifecycle.rerun")
        : t("plan.lifecycle.run"),
      visible: canRun,
    },
    {
      key: "SKIP" as const,
      label: t("common.skip"),
      visible: canSkip,
    },
    {
      key: "CANCEL" as const,
      label: t("common.cancel"),
      visible: canCancel,
    },
  ].filter((item) => item.visible);
  // Read only in the expanded branch; databaseForTask does store lookups, so
  // don't pay for it on collapsed cards.
  const databaseEngine =
    isExpanded && latestTaskRun
      ? databaseForTask(project, task, plan).instanceResource?.engine
      : undefined;

  return (
    <>
      <div className="group rounded-lg border bg-white" ref={hostRef}>
        {/* Same padding in both modes: the header row is button-height (h-6)
            either way, so expanding only reveals content below — nothing
            above the fold moves. */}
        <div
          className={cn(
            "py-2.5 pl-3 pr-4",
            isExpanded && "flex flex-col gap-3"
          )}
        >
          <DeployTaskHeader
            actionItems={actionItems}
            collapsedContextInfo={collapsedContextInfo}
            collapsedStatusText={collapsedStatusText}
            isExpanded={isExpanded}
            isSelectable={isSelectable}
            isSelected={isSelected}
            latestTaskRun={latestTaskRun}
            onAction={(nextAction) => {
              setAction(nextAction);
              setActionOpen(true);
            }}
            onRollback={() => setRollbackOpen(true)}
            onToggleExpand={() => onToggleExpand(task)}
            onToggleSelect={() => onToggleSelect(task)}
            scheduledTimeTs={scheduledTimeTs}
            showRollback={Boolean(rollbackableTaskRun)}
            task={task}
            timingDisplay={timingDisplay}
          />
          {isExpanded && (
            <DeployTaskBody
              databaseEngine={databaseEngine}
              historyCount={taskRuns.length}
              isStatementLoading={isStatementLoading}
              isTruncated={isTruncated}
              latestTaskRun={latestTaskRun}
              onShowHistory={() => setHistoryOpen(true)}
              statement={statement}
              task={task}
              timingDisplay={timingDisplay}
            />
          )}
        </div>
      </div>

      {action && (
        <PlanDetailTaskRolloutActionPanel
          action={action}
          onConfirm={async () => {
            await onRefresh();
          }}
          onOpenChange={(open) => {
            setActionOpen(open);
            if (!open) {
              setAction(undefined);
            }
          }}
          open={actionOpen}
          target={{ type: "tasks", tasks: [task], stage }}
        />
      )}

      {rollbackableTaskRun && (
        <PlanDetailRollbackSheet
          items={[{ task, taskRun: rollbackableTaskRun }]}
          onOpenChange={setRollbackOpen}
          open={rollbackOpen}
          projectName={project.name}
          rolloutName={rolloutName}
        />
      )}

      {/* Only openable when the History (n) control renders (>1 run); run
          lists only grow, so this never unmounts while open. */}
      {taskRuns.length > 1 && (
        <DeployTaskRunHistorySheet
          onOpenChange={setHistoryOpen}
          open={historyOpen}
          taskRuns={taskRuns}
        />
      )}
    </>
  );
});
