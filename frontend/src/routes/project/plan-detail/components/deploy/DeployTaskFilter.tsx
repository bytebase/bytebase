import { useTranslation } from "react-i18next";
import { TaskStatusIcon } from "@/components/TaskStatusIcon";
import { Button } from "@/components/ui/button";
import type { Stage } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { stringifyTaskStatus, TASK_STATUS_PRIORITY } from "@/utils";

export function DeployTaskFilter({
  selectedStatuses,
  onChange,
  stage,
}: {
  selectedStatuses: Task_Status[];
  onChange: (statuses: Task_Status[]) => void;
  stage: Stage;
}) {
  const { t } = useTranslation();
  const getTaskCount = (status: Task_Status) =>
    stage.tasks.filter((task) => task.status === status).length;

  return (
    <div className="flex flex-row items-center gap-1">
      {TASK_STATUS_PRIORITY.map((status) => {
        const count = getTaskCount(status);
        if (count <= 0) return null;
        const checked = selectedStatuses.includes(status);
        return (
          <Button
            appearance="secondary"
            size="md"
            key={status}
            className={[
              "h-auto inline-flex items-center gap-1 rounded-xs border px-2 py-1",
              checked
                ? "border-accent bg-accent/10"
                : "border-control-border bg-background",
            ].join(" ")}
            onClick={() => {
              onChange(
                checked
                  ? selectedStatuses.filter((item) => item !== status)
                  : [...selectedStatuses, status]
              );
            }}
            type="button"
          >
            <TaskStatusIcon size="small" status={status} />
            <span className="select-none text-sm">
              {stringifyTaskStatus(status, t)}
            </span>
            <span className="select-none text-sm font-medium">{count}</span>
          </Button>
        );
      })}
    </div>
  );
}
