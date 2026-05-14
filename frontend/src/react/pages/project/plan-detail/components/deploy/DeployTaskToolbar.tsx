import { Play, SkipForward, X } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import { Tooltip } from "@/react/components/ui/tooltip";
import { Issue_Type } from "@/types/proto-es/v1/issue_service_pb";
import type { Stage, Task } from "@/types/proto-es/v1/rollout_service_pb";
import {
  CANCELABLE_TASK_STATUSES,
  canRolloutTasks,
  preloadRolloutPermissionContext,
  RUNNABLE_TASK_STATUSES,
} from "../../../issue-detail/utils/rollout";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import {
  type PlanDetailTaskRolloutAction,
  PlanDetailTaskRolloutActionPanel,
} from "../PlanDetailTaskRolloutActionPanel";
import { isDeployTaskSelectable } from "./taskActionState";

export function DeployTaskToolbar({
  allTasks,
  selectedTasks,
  stage,
  onActionComplete,
  onClearSelection,
  onSelectAll,
}: {
  allTasks: Task[];
  selectedTasks: Task[];
  stage: Stage;
  onActionComplete: () => Promise<void> | void;
  onClearSelection: () => void;
  onSelectAll: () => void;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const currentUser = page.currentUser;
  const project = page.project;
  const [showActionPanel, setShowActionPanel] = useState(false);
  const [pendingAction, setPendingAction] = useState<
    PlanDetailTaskRolloutAction | undefined
  >(undefined);
  const [permissionReady, setPermissionReady] = useState(false);
  useEffect(() => {
    let canceled = false;
    const load = async () => {
      setPermissionReady(false);
      await preloadRolloutPermissionContext({
        environment: stage.environment,
        projectName: project.name,
        tasks: allTasks,
      });
      if (!canceled) setPermissionReady(true);
    };
    void load();
    return () => {
      canceled = true;
    };
  }, [allTasks, project.name, stage.environment]);

  const selectableTasks = allTasks.filter(isDeployTaskSelectable);
  const canPerformTaskActions = permissionReady
    ? canRolloutTasks({
        currentUser,
        environment: stage.environment,
        issue: page.issue,
        project,
        tasks: selectedTasks,
      })
    : false;

  const hasRunnableTasks = selectedTasks.some((task) =>
    RUNNABLE_TASK_STATUSES.includes(task.status)
  );
  const hasSkippableTasks = selectedTasks.some((task) =>
    RUNNABLE_TASK_STATUSES.includes(task.status)
  );
  const hasCancellableTasks = selectedTasks.some((task) =>
    CANCELABLE_TASK_STATUSES.includes(task.status)
  );

  const getBaseDisabledTooltip = () => {
    if (selectedTasks.length === 0) {
      return t("task.no-tasks-selected");
    }
    if (!permissionReady) {
      return "";
    }
    if (!canPerformTaskActions) {
      if (
        page.issue &&
        page.issue.type === Issue_Type.DATABASE_EXPORT &&
        page.issue.creator !== currentUser.name
      ) {
        return t("task.data-export-creator-only");
      }
      return t("task.no-permission");
    }
    return "";
  };

  const selectionCountText = t("rollout.task.selected-count", {
    count: selectedTasks.length,
  });

  return (
    <>
      <div className="sticky top-0 z-10 px-4">
        <div className="flex items-center justify-between rounded-lg border border-blue-200 bg-blue-100 px-3 py-1">
          <div className="flex items-center gap-x-3">
            <Tooltip
              content={
                selectableTasks.length === 0
                  ? t("task.no-selectable-tasks")
                  : ""
              }
            >
              <div>
                <Checkbox
                  checked={
                    selectedTasks.length > 0 &&
                    selectedTasks.length < selectableTasks.length
                      ? "indeterminate"
                      : selectedTasks.length > 0 &&
                        selectedTasks.length === selectableTasks.length
                  }
                  disabled={selectableTasks.length === 0}
                  onCheckedChange={(checked) => {
                    if (checked) onSelectAll();
                    else onClearSelection();
                  }}
                />
              </div>
            </Tooltip>
            <span className="text-sm text-blue-900">{selectionCountText}</span>
            <div className="flex items-center gap-x-0.5">
              <Tooltip
                content={
                  !hasRunnableTasks
                    ? t("task.no-runnable-tasks")
                    : getBaseDisabledTooltip()
                }
              >
                <div>
                  <Button
                    disabled={!hasRunnableTasks || !canPerformTaskActions}
                    onClick={() => {
                      setPendingAction("RUN");
                      setShowActionPanel(true);
                    }}
                    size="sm"
                    variant="ghost"
                    className="text-accent hover:bg-transparent"
                  >
                    <Play className="h-3.5 w-3.5" />
                    {t("common.run")}
                  </Button>
                </div>
              </Tooltip>
              <Tooltip
                content={
                  !hasSkippableTasks
                    ? t("task.no-skippable-tasks")
                    : getBaseDisabledTooltip()
                }
              >
                <div>
                  <Button
                    disabled={!hasSkippableTasks || !canPerformTaskActions}
                    onClick={() => {
                      setPendingAction("SKIP");
                      setShowActionPanel(true);
                    }}
                    size="sm"
                    variant="ghost"
                    className="text-accent hover:bg-transparent"
                  >
                    <SkipForward className="h-3.5 w-3.5" />
                    {t("common.skip")}
                  </Button>
                </div>
              </Tooltip>
              <Tooltip
                content={
                  !hasCancellableTasks
                    ? t("task.no-cancelable-tasks")
                    : getBaseDisabledTooltip()
                }
              >
                <div>
                  <Button
                    disabled={!hasCancellableTasks || !canPerformTaskActions}
                    onClick={() => {
                      setPendingAction("CANCEL");
                      setShowActionPanel(true);
                    }}
                    size="sm"
                    variant="ghost"
                    className="text-accent hover:bg-transparent"
                  >
                    <X className="h-3.5 w-3.5" />
                    {t("common.cancel")}
                  </Button>
                </div>
              </Tooltip>
            </div>
          </div>
        </div>
      </div>

      {pendingAction && (
        <PlanDetailTaskRolloutActionPanel
          action={pendingAction}
          onConfirm={async () => {
            await onActionComplete();
            onClearSelection();
          }}
          onOpenChange={(open) => {
            setShowActionPanel(open);
            if (!open) {
              setPendingAction(undefined);
            }
          }}
          open={showActionPanel}
          target={{ type: "tasks", tasks: selectedTasks, stage }}
        />
      )}
    </>
  );
}
