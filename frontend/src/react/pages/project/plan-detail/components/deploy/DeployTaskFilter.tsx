import { useTranslation } from "react-i18next";
import type { Stage } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { stringifyTaskStatus } from "@/utils";
import { DeployTaskStatus } from "./DeployTaskStatus";

const TASK_STATUS_FILTERS = [
  Task_Status.NOT_STARTED,
  Task_Status.PENDING,
  Task_Status.RUNNING,
  Task_Status.DONE,
  Task_Status.FAILED,
  Task_Status.CANCELED,
  Task_Status.SKIPPED,
];

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
      {TASK_STATUS_FILTERS.map((status) => {
        const count = getTaskCount(status);
        if (count <= 0) return null;
        const checked = selectedStatuses.includes(status);
        return (
          <button
            key={status}
            className={[
              "inline-flex items-center gap-1 rounded-full border px-2 py-1",
              checked
                ? "border-accent bg-accent/10"
                : "border-control-border bg-white",
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
            <DeployTaskStatus size="small" status={status} />
            <span className="select-none text-sm">
              {stringifyTaskStatus(status, t)}
            </span>
            <span className="select-none text-sm font-medium">{count}</span>
          </button>
        );
      })}
    </div>
  );
}
