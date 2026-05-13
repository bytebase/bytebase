import { useEffect, useMemo, useState } from "react";
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
  const [expandedTaskNames, setExpandedTaskNames] = useState<Set<string>>(
    new Set()
  );
  const [selectedTaskNames, setSelectedTaskNames] = useState<Set<string>>(
    new Set()
  );
  const filteredTasks = stage.tasks;
  const visibleTasks = filteredTasks.slice(0, displayedTaskCount);
  const hasMoreTasks = filteredTasks.length > displayedTaskCount;
  const remainingTasksCount = filteredTasks.length - displayedTaskCount;
  const selectedTasks = useMemo(
    () => filteredTasks.filter((task) => selectedTaskNames.has(task.name)),
    [filteredTasks, selectedTaskNames]
  );
  const taskNamesKey = filteredTasks.map((task) => task.name).join(",");

  useEffect(() => {
    setDisplayedTaskCount(DEFAULT_PAGE_SIZE);
  }, [taskNamesKey]);

  useEffect(() => {
    setExpandedTaskNames(() => {
      if (filteredTasks.length === 0) {
        return new Set();
      }
      return new Set([filteredTasks[0].name]);
    });
  }, [taskNamesKey]);

  useEffect(() => {
    setSelectedTaskNames((prev) => {
      const remaining = [...prev].filter((taskName) =>
        filteredTasks.some((task) => task.name === taskName)
      );
      if (remaining.length === prev.size) {
        return prev;
      }
      return new Set(remaining);
    });
  }, [taskNamesKey]);

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
