import { ChevronDown, ChevronRight, Clock3, User } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { HumanizeTs } from "@/components/HumanizeTs";
import { TaskRunStatusIcon } from "@/components/TaskRunStatusIcon";
import { TaskRunLogViewer } from "@/components/task-run-log";
import { EllipsisText } from "@/components/ui/ellipsis-text";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useOnKeyChange } from "@/hooks/useOnKeyChange";
import {
  executorEmailOfTaskRun,
  formatTaskRunDuration,
  getTaskRunComment,
} from "@/lib/taskRun";
import { getTimeForPbTimestampProtoEs } from "@/types";
import type { TaskRun } from "@/types/proto-es/v1/rollout_service_pb";

// With this many runs or more, only the newest run starts expanded; older
// runs collapse to header rows. Keeps the sheet scannable and avoids fetching
// every run's log up front.
const COLLAPSE_OLDER_RUNS_THRESHOLD = 4;

export function DeployTaskRunHistorySheet({
  onOpenChange,
  open,
  taskRuns,
}: {
  onOpenChange: (open: boolean) => void;
  open: boolean;
  // Sorted newest-first by the caller (the page's per-task run grouping).
  taskRuns: TaskRun[];
}) {
  const { t } = useTranslation();

  return (
    <Sheet onOpenChange={onOpenChange} open={open}>
      <SheetContent width="standard">
        <SheetHeader>
          <SheetTitle>{t("task-run.history")}</SheetTitle>
        </SheetHeader>
        {/* SheetContent unmounts its children after the close animation, so
            the run list's expansion state resets on every open. */}
        <SheetBody className="gap-y-2 overscroll-contain">
          <TaskRunHistoryList taskRuns={taskRuns} />
        </SheetBody>
      </SheetContent>
    </Sheet>
  );
}

function TaskRunHistoryList({ taskRuns }: { taskRuns: TaskRun[] }) {
  const [expandedRunNames, setExpandedRunNames] = useState<Set<string>>(() =>
    taskRuns.length >= COLLAPSE_OLDER_RUNS_THRESHOLD
      ? new Set([taskRuns[0].name])
      : new Set(taskRuns.map((taskRun) => taskRun.name))
  );
  // A rerun created while the sheet is open prepends a new newest run; expand
  // it so the latest is visible without reopening the sheet.
  useOnKeyChange(taskRuns[0]?.name ?? "", () => {
    const newest = taskRuns[0]?.name;
    if (newest) {
      setExpandedRunNames((prev) =>
        prev.has(newest) ? prev : new Set(prev).add(newest)
      );
    }
  });
  const toggle = (taskRun: TaskRun) => {
    setExpandedRunNames((prev) => {
      const next = new Set(prev);
      if (next.has(taskRun.name)) next.delete(taskRun.name);
      else next.add(taskRun.name);
      return next;
    });
  };

  return (
    <>
      {taskRuns.map((taskRun, index) => (
        <TaskRunHistoryItem
          isExpanded={expandedRunNames.has(taskRun.name)}
          key={taskRun.name}
          onToggle={() => toggle(taskRun)}
          runNumber={taskRuns.length - index}
          taskRun={taskRun}
        />
      ))}
    </>
  );
}

function TaskRunHistoryItem({
  isExpanded,
  onToggle,
  runNumber,
  taskRun,
}: {
  isExpanded: boolean;
  onToggle: () => void;
  runNumber: number;
  taskRun: TaskRun;
}) {
  const { t } = useTranslation();
  const startTs =
    getTimeForPbTimestampProtoEs(taskRun.startTime, 0) ||
    getTimeForPbTimestampProtoEs(taskRun.createTime, 0);
  const duration = formatTaskRunDuration(taskRun);
  const executorEmail = executorEmailOfTaskRun(taskRun);
  const comment = getTaskRunComment(taskRun, t);

  return (
    <div className="rounded-lg border">
      <button
        aria-expanded={isExpanded}
        className="flex w-full items-center gap-x-2 rounded-lg px-3 py-2 text-left hover:bg-control-bg focus-visible:ring-2 focus-visible:ring-accent"
        onClick={onToggle}
        type="button"
      >
        {isExpanded ? (
          <ChevronDown className="size-4 shrink-0 text-control-light" />
        ) : (
          <ChevronRight className="size-4 shrink-0 text-control-light" />
        )}
        <TaskRunStatusIcon status={taskRun.status} />
        <span className="shrink-0 text-sm font-medium text-control">
          {t("task-run.run-number", { number: runNumber })}
        </span>
        {/* Only when collapsed: the expanded body shows the full error, so a
            header teaser there would just duplicate it. */}
        <span className="min-w-0 flex-1 text-xs text-control-light">
          {!isExpanded && (
            <EllipsisText className="line-clamp-1" text={comment} />
          )}
        </span>
        <span className="flex shrink-0 items-center gap-x-3 text-xs text-control-light">
          {startTs > 0 && <HumanizeTs ts={startTs / 1000} />}
          {duration && (
            <span className="flex items-center gap-x-1">
              <Clock3 className="size-3" />
              {duration}
            </span>
          )}
          {executorEmail && (
            <span className="hidden items-center gap-x-1 lg:flex">
              <User className="size-3" />
              {executorEmail}
            </span>
          )}
        </span>
      </button>
      {isExpanded && (
        <div className="border-t p-3">
          {/* Status is part of the key (as in the latest-run panel): a
              RUNNING -> DONE flip remounts the viewer so useTaskRunLogData's
              unmount cleanup invalidates the in-flight RUNNING log request,
              which would otherwise cache an incomplete RUNNING log. */}
          <TaskRunLogViewer
            key={`logs-${taskRun.name}-${taskRun.status}`}
            taskRunName={taskRun.name}
            taskRunStatus={taskRun.status}
          />
        </div>
      )}
    </div>
  );
}
