import {
  ChevronDown,
  ChevronRight,
  LoaderCircle,
  Play,
  SkipForward,
  X,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { HumanizeTs } from "@/components/HumanizeTs";
import { TaskStatusIcon } from "@/components/TaskStatusIcon";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { cn } from "@/lib/utils";
import { getTimeForPbTimestampProtoEs } from "@/types";
import type { Task, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import type { PlanDetailTaskRolloutAction } from "../PlanDetailTaskRolloutActionPanel";
import { PlanTargetDisplay } from "../PlanTargetDisplay";

type ActionItem = { key: PlanDetailTaskRolloutAction; label: string };

// The card header: a bulk-select checkbox, the task identity (which toggles the
// card open/closed), the per-card actions when expanded, and a caret. Purely
// presentational — DeployTaskItem owns the derived data and handlers.
export function DeployTaskHeader({
  actionItems,
  collapsedContextInfo,
  collapsedStatusText,
  isExpanded,
  isSelectable,
  isSelected,
  latestTaskRun,
  onAction,
  onRollback,
  onToggleExpand,
  onToggleSelect,
  scheduledTimeTs,
  showRollback,
  task,
  timingDisplay,
}: {
  actionItems: ActionItem[];
  collapsedContextInfo: string;
  collapsedStatusText: string;
  isExpanded: boolean;
  isSelectable: boolean;
  isSelected: boolean;
  latestTaskRun?: TaskRun;
  onAction: (action: PlanDetailTaskRolloutAction) => void;
  onRollback: () => void;
  onToggleExpand: () => void;
  onToggleSelect: () => void;
  scheduledTimeTs: number;
  showRollback: boolean;
  task: Task;
  timingDisplay: string;
}) {
  const { t } = useTranslation();
  const showActions = actionItems.length > 0 || showRollback;

  return (
    <>
      <div className="flex items-center gap-x-2">
        <Checkbox
          aria-label={t("task.select-task")}
          checked={isSelected}
          className="shrink-0"
          disabled={!isSelectable}
          onCheckedChange={() => onToggleSelect()}
          onClick={(event) => event.stopPropagation()}
        />
        {/* The identity is the toggle. It hugs its content, not the full row,
            so the hover and hit target track the target — not the empty space
            out to the caret. */}
        <button
          aria-expanded={isExpanded}
          className="flex min-w-0 cursor-pointer items-center gap-x-2 rounded-md border-0 bg-transparent p-0 text-left outline-none hover:bg-control-bg/50 focus-visible:ring-2 focus-visible:ring-accent"
          onClick={onToggleExpand}
          type="button"
        >
          <TaskStatusIcon size="small" status={task.status} />
          <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
            <PlanTargetDisplay size="md" target={task.target} />
            {isExpanded && scheduledTimeTs > 0 && (
              <span className="flex shrink-0 items-center gap-x-1 rounded-full bg-info/10 px-2 py-0.5 text-xs text-info">
                <LoaderCircle className="size-3 animate-spin motion-reduce:animate-none" />
                <HumanizeTs ts={scheduledTimeTs} />
              </span>
            )}
          </div>
        </button>

        {/* Right cluster, pushed to the row's edge: the collapsed duration or
            the expanded actions, then the caret. Actions live in the header —
            consistent with the app, reachable without scrolling the body. */}
        <div className="ml-auto flex shrink-0 items-center gap-x-2">
          {!isExpanded && (
            <span className="text-xs tabular-nums text-control-light">
              {scheduledTimeTs > 0 ||
              (task.status === Task_Status.RUNNING && timingDisplay) ? (
                <span className="flex items-center gap-x-1 text-info">
                  <LoaderCircle className="size-3 animate-spin motion-reduce:animate-none" />
                  {scheduledTimeTs > 0 ? (
                    <HumanizeTs ts={scheduledTimeTs} />
                  ) : (
                    timingDisplay
                  )}
                </span>
              ) : collapsedContextInfo ? (
                <span>{collapsedContextInfo}</span>
              ) : null}
            </span>
          )}
          {isExpanded && showActions && (
            <div className="flex flex-wrap items-center justify-end gap-2">
              {actionItems.map((item) => (
                <Button
                  key={item.key}
                  onClick={() => onAction(item.key)}
                  size="xs"
                  appearance={item.key === "RUN" ? "solid" : "outline"}
                >
                  {item.key === "RUN" && <Play className="size-3" />}
                  {item.key === "SKIP" && <SkipForward className="size-3" />}
                  {item.key === "CANCEL" && <X className="size-3" />}
                  {item.label}
                </Button>
              ))}
              {showRollback && (
                <Button onClick={onRollback} size="xs" appearance="outline">
                  {t("common.rollback")}
                </Button>
              )}
            </div>
          )}
          {/* Redundant mouse affordance for the toggle; the identity button is
              the keyboard-accessible control, so keep this out of the tab
              order. */}
          <button
            aria-hidden="true"
            className="shrink-0 rounded-md p-1 text-control-light outline-none hover:bg-control-bg/50"
            onClick={onToggleExpand}
            tabIndex={-1}
            type="button"
          >
            {isExpanded ? (
              <ChevronDown className="size-4" />
            ) : (
              <ChevronRight className="size-4" />
            )}
          </button>
        </div>
      </div>

      {!isExpanded && collapsedStatusText && (
        <div className="mt-1 flex items-center gap-x-2 text-xs">
          {latestTaskRun?.createTime && (
            <span className="rounded-full border bg-control-bg px-2 py-0.5 text-control-light">
              <HumanizeTs
                ts={
                  getTimeForPbTimestampProtoEs(latestTaskRun.createTime, 0) /
                  1000
                }
              />
            </span>
          )}
          <span
            className={cn(
              "min-w-0 truncate",
              task.status === Task_Status.FAILED
                ? "text-error"
                : "italic text-control-light"
            )}
          >
            {collapsedStatusText}
          </span>
        </div>
      )}
    </>
  );
}
