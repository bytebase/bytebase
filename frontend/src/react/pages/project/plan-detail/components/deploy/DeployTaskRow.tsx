import { X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import { type Task, Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { DeployTaskStatus } from "./DeployTaskStatus";

export function DeployTaskRow({
  active,
  onCloseTaskPanel,
  stageId,
  task,
}: {
  active: boolean;
  onCloseTaskPanel: () => void;
  stageId: string;
  task: Task;
}) {
  const { t } = useTranslation();
  const taskId = task.name.split("/").pop();

  return (
    <button
      className={cn(
        "flex w-full items-center justify-between rounded border px-3 py-2 text-left text-sm",
        active
          ? "border-accent bg-accent/5"
          : "border-control-border hover:bg-gray-50"
      )}
      onClick={() => {
        void router.push({
          query: {
            phase: "deploy",
            stageId,
            taskId,
          },
        });
      }}
      type="button"
    >
      <div className="flex min-w-0 items-center gap-2">
        <DeployTaskStatus size="small" status={task.status} />
        <div className="min-w-0">
          <div className="truncate text-main">{task.name}</div>
          <div className="text-xs text-control-light">
            {Task_Status[task.status]}
          </div>
        </div>
      </div>
      {active && (
        <Button
          aria-label={t("common.close")}
          size="sm"
          variant="ghost"
          onClick={(event) => {
            event.stopPropagation();
            onCloseTaskPanel();
          }}
        >
          <X className="h-4 w-4" />
        </Button>
      )}
    </button>
  );
}
