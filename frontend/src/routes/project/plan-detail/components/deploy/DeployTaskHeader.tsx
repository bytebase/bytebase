import {
  ChevronDown,
  ChevronRight,
  LoaderCircle,
  Play,
  SkipForward,
  X,
} from "lucide-react";
import type { MouseEvent } from "react";
import { useTranslation } from "react-i18next";
import { HumanizeTs } from "@/components/HumanizeTs";
import { TaskStatusIcon } from "@/components/TaskStatusIcon";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { useMediaQuery } from "@/hooks/useMediaQuery";
import { getTaskRunWaitingMessage } from "@/lib/taskRun";
import { cn } from "@/lib/utils";
import type { Task, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import type { PlanDetailTaskRolloutAction } from "../PlanDetailTaskRolloutActionPanel";
import { PlanTargetDisplay } from "../PlanTargetDisplay";
import { useUserDisplayName } from "../useUserDisplayName";
import {
  type FoldedReleaseInfo,
  type FoldedStatementSummary,
  type FoldedTone,
  foldedTaskDescription,
} from "./foldedTaskDescription";

type ActionItem = { key: PlanDetailTaskRolloutAction; label: string };

const TONE_CLASS: Record<FoldedTone, string> = {
  success: "text-success",
  info: "text-info",
  error: "text-error",
  muted: "text-control-light",
};

// Below Tailwind's `sm` breakpoint the folded description moves under the
// database.
const NARROW_MEDIA_QUERY = "(max-width: 639px)";

// The card header: a bulk-select checkbox, the task identity, the folded card's
// one-line description, the per-card actions when expanded, and a caret. The
// whole line, the card's padding around it included, toggles the card;
// DeployTaskItem owns the handlers.
export function DeployTaskHeader({
  actionItems,
  isExpanded,
  isSelectable,
  isSelected,
  latestTaskRun,
  onAction,
  onRollback,
  onToggleExpand,
  onToggleSelect,
  release,
  scheduledTimeMs,
  showRollback,
  statementSummary,
  task,
  timingDisplay,
}: {
  actionItems: ActionItem[];
  isExpanded: boolean;
  isSelectable: boolean;
  isSelected: boolean;
  latestTaskRun?: TaskRun;
  onAction: (action: PlanDetailTaskRolloutAction) => void;
  onRollback: () => void;
  onToggleExpand: () => void;
  onToggleSelect: () => void;
  // The task's release, for a release-backed task.
  release?: FoldedReleaseInfo;
  scheduledTimeMs: number | undefined;
  showRollback: boolean;
  // The plan check's statement summary for the task's target.
  statementSummary?: FoldedStatementSummary;
  task: Task;
  timingDisplay: string;
}) {
  const { t, i18n } = useTranslation();
  const narrow = useMediaQuery(NARROW_MEDIA_QUERY);
  const skipperName = useUserDisplayName(task.skipper || undefined);
  const showActions = actionItems.length > 0 || showRollback;
  const folded = isExpanded
    ? undefined
    : foldedTaskDescription({
        task,
        latestTaskRun,
        release,
        statementSummary,
        duration: timingDisplay,
        skipperName,
        t,
        locale: i18n.language,
      });
  const waitingCause =
    folded &&
    latestTaskRun &&
    (task.status === Task_Status.NOT_STARTED ||
      task.status === Task_Status.PENDING)
      ? getTaskRunWaitingMessage(latestTaskRun, t)
      : undefined;

  // Buttons and the checkbox, and the margins around them marked
  // data-fold-guard, keep their own click, so a near miss does not toggle.
  const handleLineClick = (event: MouseEvent<HTMLDivElement>) => {
    const target = event.target as Element;
    if (target.closest("button, [role='checkbox'], [data-fold-guard]")) return;
    // A click that ends a drag, e.g. a selection swept across the description.
    if (!(window.getSelection()?.isCollapsed ?? true)) return;
    onToggleExpand();
  };

  return (
    <>
      <div
        className="flex cursor-pointer items-center gap-x-2 py-2.5 pr-4 pl-3"
        data-testid="deploy-task-header"
        onClick={handleLineClick}
      >
        {/* Reaches into the card's padding, so the guard spans the line's
            height and the card's left edge without moving the checkbox. */}
        <span
          className="-my-2.5 -mr-1.5 -ml-3 flex cursor-default items-center self-stretch py-2.5 pr-1.5 pl-3"
          data-fold-guard
        >
          <Checkbox
            aria-label={t("task.select-task")}
            checked={isSelected}
            className="shrink-0"
            disabled={!isSelectable}
            onCheckedChange={() => onToggleSelect()}
            onClick={(event) => event.stopPropagation()}
          />
        </span>
        {/* The keyboard toggle; the line around it is the mouse target. */}
        <Button
          appearance="secondary"
          size="md"
          aria-expanded={isExpanded}
          className="h-auto min-w-0 cursor-pointer items-center justify-start gap-x-2 rounded-sm border-0 bg-transparent p-0 text-left whitespace-normal outline-none hover:bg-control-bg/50 focus-visible:ring-2 focus-visible:ring-accent"
          onClick={onToggleExpand}
          type="button"
        >
          <TaskStatusIcon size="small" status={task.status} />
          <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
            <PlanTargetDisplay size="md" target={task.target} />
            {folded && (
              <span
                className={cn(
                  "min-w-0 truncate text-sm",
                  TONE_CLASS[folded.tone],
                  narrow && "basis-full"
                )}
                data-testid="folded-description"
              >
                {folded.version ? (
                  <span className="tabular-nums">{folded.version} </span>
                ) : null}
                {folded.text}
                {scheduledTimeMs !== undefined ? (
                  <>
                    {" · "}
                    {t("task.folded.scheduled")}{" "}
                    <HumanizeTs mode="operational" tsMs={scheduledTimeMs} />
                  </>
                ) : waitingCause ? (
                  ` · ${waitingCause}`
                ) : null}
              </span>
            )}
            {isExpanded && scheduledTimeMs !== undefined && (
              <span className="flex shrink-0 items-center gap-x-1 rounded-xs border border-transparent bg-info/10 px-1.5 py-0.5 text-xs text-info">
                <LoaderCircle className="size-3 animate-spin motion-reduce:animate-none" />
                <HumanizeTs mode="operational" tsMs={scheduledTimeMs} />
              </span>
            )}
          </div>
        </Button>

        {/* Right cluster, pushed to the row's edge: the collapsed duration or
            the expanded actions, then the caret. Actions live in the header —
            consistent with the app, reachable without scrolling the body. */}
        <div className="ml-auto flex shrink-0 items-center gap-x-2 self-stretch">
          {isExpanded && showActions && (
            <div
              className="-my-2.5 -mr-1 -ml-2 flex cursor-default flex-wrap items-center justify-end gap-2 self-stretch py-2.5 pr-1 pl-2"
              data-fold-guard
            >
              {actionItems.map((item) => (
                <Button
                  key={item.key}
                  onClick={() => onAction(item.key)}
                  size="xs"
                  appearance="outline"
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
          <Button
            appearance="secondary"
            size="xs"
            aria-hidden="true"
            className="shrink-0 rounded-sm p-1 text-control-light outline-none hover:bg-control-bg/50"
            onClick={onToggleExpand}
            tabIndex={-1}
            type="button"
          >
            {isExpanded ? (
              <ChevronDown className="size-4" />
            ) : (
              <ChevronRight className="size-4" />
            )}
          </Button>
        </div>
      </div>
    </>
  );
}
