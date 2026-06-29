import { CircleDotDashed, FastForward, Pause } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Tooltip } from "@/react/components/ui/tooltip";
import { cn } from "@/react/lib/utils";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { stringifyTaskStatus } from "@/utils";

/**
 * Canonical status icon for a rollout task/stage. Drive the size with the
 * `size` prop. Use this everywhere a `Task_Status` is shown so the plan table,
 * plan detail task rows, and stage tabs stay visually consistent.
 */
const SIZE_CLASSES = {
  tiny: "h-4 w-4",
  small: "h-5 w-5",
  medium: "h-6 w-6",
  large: "h-7 w-7",
} as const;

export function TaskStatusIcon({
  status,
  size = "small",
}: {
  status: Task_Status;
  size?: keyof typeof SIZE_CLASSES;
}) {
  const { t } = useTranslation();
  const classes = SIZE_CLASSES[size];

  const statusLabel = stringifyTaskStatus(status, t);

  return (
    <Tooltip content={statusLabel}>
      <div
        className={cn(
          "relative inline-flex shrink-0 items-center justify-center overflow-hidden rounded-full select-none",
          classes,
          status === Task_Status.NOT_STARTED &&
            "border-2 border-control bg-white",
          (status === Task_Status.PENDING || status === Task_Status.RUNNING) &&
            "border-2 border-accent bg-white text-accent",
          status === Task_Status.SKIPPED &&
            "border-2 border-control-light bg-white text-control-light",
          status === Task_Status.DONE && "bg-success text-white",
          status === Task_Status.FAILED && "bg-error text-white",
          status === Task_Status.CANCELED &&
            "border-2 border-control-light bg-white text-control-light"
        )}
      >
        {status === Task_Status.STATUS_UNSPECIFIED && (
          <CircleDotDashed className="h-full w-full" />
        )}
        {status === Task_Status.NOT_STARTED && (
          <span className="h-1/2 w-1/2 rounded-full bg-control" />
        )}
        {status === Task_Status.PENDING && <Pause className="h-3/4 w-3/4" />}
        {status === Task_Status.RUNNING && (
          <div className="relative flex h-1/2 w-1/2 overflow-visible">
            <span
              aria-hidden="true"
              className="absolute z-0 h-full w-full animate-ping rounded-full"
              style={{ backgroundColor: "rgba(37, 99, 235, 0.5)" }}
            />
            <span
              aria-hidden="true"
              className="z-1 h-full w-full rounded-full bg-accent"
            />
          </div>
        )}
        {status === Task_Status.SKIPPED && (
          <FastForward className="h-3/4 w-3/4" />
        )}
        {status === Task_Status.DONE && <span className="text-sm">✓</span>}
        {status === Task_Status.FAILED && (
          <span className="text-base font-medium">!</span>
        )}
        {status === Task_Status.CANCELED && (
          <span className="text-base">−</span>
        )}
      </div>
    </Tooltip>
  );
}
