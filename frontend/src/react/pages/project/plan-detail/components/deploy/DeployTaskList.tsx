import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import type { Stage, Task } from "@/types/proto-es/v1/rollout_service_pb";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import { DeployTaskItem } from "./DeployTaskItem";
import { DeployTaskToolbar } from "./DeployTaskToolbar";
import { isDeployTaskSelectable } from "./taskActionState";

const DEFAULT_PAGE_SIZE = 20;

export function DeployTaskList({
  selectedTaskName,
  stage,
  readonly = false,
}: {
  selectedTaskName?: string;
  stage: Stage;
  readonly?: boolean;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const [displayedTaskCount, setDisplayedTaskCount] =
    useState(DEFAULT_PAGE_SIZE);
  const filteredTasks = stage.tasks;
  const [expandedTaskNames, setExpandedTaskNames] = useState<Set<string>>(() =>
    filteredTasks.length > 0 ? new Set([filteredTasks[0].name]) : new Set()
  );
  const [selectedTaskNames, setSelectedTaskNames] = useState<Set<string>>(
    new Set()
  );
  const taskNamesKey = filteredTasks.map((task) => task.name).join(",");

  // When the visible task set changes — most importantly when switching to a
  // different stage — re-derive the per-stage list state (pagination, the
  // auto-expanded first task, and the still-valid selection) during render
  // rather than in a post-paint effect. Doing it here means the new stage's
  // first paint already shows the first task expanded, instead of painting a
  // collapsed list and expanding it a frame later, which read as a flash
  // (BYT-9763).
  const [trackedTaskNamesKey, setTrackedTaskNamesKey] = useState(taskNamesKey);
  if (taskNamesKey !== trackedTaskNamesKey) {
    setTrackedTaskNamesKey(taskNamesKey);
    setDisplayedTaskCount(DEFAULT_PAGE_SIZE);
    setExpandedTaskNames(
      filteredTasks.length > 0 ? new Set([filteredTasks[0].name]) : new Set()
    );
    setSelectedTaskNames((prev) => {
      const remaining = [...prev].filter((taskName) =>
        filteredTasks.some((task) => task.name === taskName)
      );
      return remaining.length === prev.size ? prev : new Set(remaining);
    });
  }

  const visibleTasks = filteredTasks.slice(0, displayedTaskCount);
  const hasMoreTasks = filteredTasks.length > displayedTaskCount;
  const remainingTasksCount = filteredTasks.length - displayedTaskCount;
  const selectedTasks = useMemo(
    () => filteredTasks.filter((task) => selectedTaskNames.has(task.name)),
    [filteredTasks, selectedTaskNames]
  );

  const toggleExpand = (task: Task) => {
    setExpandedTaskNames((prev) => {
      const next = new Set(prev);
      if (next.has(task.name)) next.delete(task.name);
      else next.add(task.name);
      return next;
    });
  };
  const toggleSelect = (task: Task) => {
    setSelectedTaskNames((prev) => {
      const next = new Set(prev);
      if (next.has(task.name)) next.delete(task.name);
      else next.add(task.name);
      return next;
    });
  };

  return (
    <div className="w-full">
      {!readonly && (
        <DeployTaskToolbar
          allTasks={filteredTasks}
          onActionComplete={async () => {
            await page.refreshState();
          }}
          onClearSelection={() => setSelectedTaskNames(new Set())}
          onSelectAll={() =>
            setSelectedTaskNames(
              new Set(
                filteredTasks
                  .filter((task) => isDeployTaskSelectable(task))
                  .map((task) => task.name)
              )
            )
          }
          selectedTasks={selectedTasks}
          stage={stage}
        />
      )}

      <div className="task-list space-y-3 px-4 py-3">
        {visibleTasks.map((task) => (
          <DeployTaskItem
            active={selectedTaskName === task.name}
            key={task.name}
            isExpanded={expandedTaskNames.has(task.name)}
            isSelected={selectedTaskNames.has(task.name)}
            isSelectable={!readonly && isDeployTaskSelectable(task)}
            onToggleExpand={() => toggleExpand(task)}
            onToggleSelect={() => toggleSelect(task)}
            readonly={readonly}
            stageId={stage.name.split("/").pop() ?? ""}
            stage={stage}
            task={task}
          />
        ))}

        {filteredTasks.length === 0 && (
          <div className="py-8 text-center text-gray-500">
            {t("rollout.task.no-tasks")}
          </div>
        )}

        {hasMoreTasks && (
          <div className="flex justify-start">
            <Button
              onClick={() =>
                setDisplayedTaskCount((count) =>
                  Math.min(count + DEFAULT_PAGE_SIZE, filteredTasks.length)
                )
              }
              size="xs"
              variant="ghost"
            >
              {t("common.show-more")} ({remainingTasksCount}{" "}
              {t("common.remaining")})
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
