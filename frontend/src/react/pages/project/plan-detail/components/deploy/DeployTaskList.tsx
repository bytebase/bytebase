import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { useOnKeyChange } from "@/react/hooks/useOnKeyChange";
import { router } from "@/react/router";
import {
  type Stage,
  type Task,
  Task_Status,
  type TaskRun,
} from "@/types/proto-es/v1/rollout_service_pb";
import { extractStageUID, extractTaskUID } from "@/utils/v1/issue/rollout";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import { DeployTaskItem } from "./DeployTaskItem";
import { DeployTaskToolbar } from "./DeployTaskToolbar";
import { isDeployTaskSelectable } from "./taskActionState";

const DEFAULT_PAGE_SIZE = 20;
const EMPTY_TASK_RUNS: TaskRun[] = [];

// The task worth opening by default: the first one that needs attention
// (running or failed), falling back to the first task.
const autoExpandTaskName = (tasks: Task[]): string | undefined =>
  (
    tasks.find(
      (task) =>
        task.status === Task_Status.RUNNING ||
        task.status === Task_Status.FAILED
    ) ?? tasks[0]
  )?.name;

// The shareable ?stageId&taskId= shape for a task.
const deployTaskQuery = (stageName: string, taskName: string) => ({
  phase: "deploy",
  stageId: extractStageUID(stageName),
  taskId: extractTaskUID(taskName),
});

const initialExpandedNames = (
  tasks: Task[],
  selectedTaskName?: string
): Set<string> => {
  const names = new Set<string>();
  const autoName = autoExpandTaskName(tasks);
  if (autoName) names.add(autoName);
  // A deep-linked (?taskId=) task starts expanded alongside the auto pick.
  if (
    selectedTaskName &&
    tasks.some((task) => task.name === selectedTaskName)
  ) {
    names.add(selectedTaskName);
  }
  return names;
};

// Grow the page so a deep-linked task beyond the first page is visible.
const initialDisplayedCount = (
  tasks: Task[],
  selectedTaskName?: string
): number => {
  const index = selectedTaskName
    ? tasks.findIndex((task) => task.name === selectedTaskName)
    : -1;
  return Math.max(DEFAULT_PAGE_SIZE, index + 1);
};

export function DeployTaskList({
  stage,
  readonly = false,
  active = true,
}: {
  stage: Stage;
  readonly?: boolean;
  // Every stage's list stays mounted (visibility is the parent's concern);
  // `active` marks the one the user is looking at. The default card expansion
  // (Monaco + log fetch) is deferred to the first activation so loading the
  // page doesn't mount every stage's editors at once.
  active?: boolean;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  // Full task resource name resolved from the ?taskId= deep link.
  const selectedTaskName = page.selectedTaskName;
  const filteredTasks = stage.tasks;
  const isTaskInStage = (name: string | undefined): name is string =>
    !!name && filteredTasks.some((task) => task.name === name);

  const [activated, setActivated] = useState(active);
  const [displayedTaskCount, setDisplayedTaskCount] = useState(() =>
    initialDisplayedCount(filteredTasks, selectedTaskName)
  );
  const [expandedTaskNames, setExpandedTaskNames] = useState<Set<string>>(() =>
    active ? initialExpandedNames(filteredTasks, selectedTaskName) : new Set()
  );
  const [selectedTaskNames, setSelectedTaskNames] = useState<Set<string>>(
    new Set()
  );
  // The focused task of this stage — the last card the user opened, defaulting
  // to the auto-expand pick or an incoming deep link. It is mirrored into
  // ?stageId&taskId= while this stage is active (the effect below), so the
  // address bar is a shareable link to this stage + task, and switching back to
  // a visited stage restores it.
  const [focusedTaskName, setFocusedTaskName] = useState<string | undefined>(
    () =>
      (isTaskInStage(selectedTaskName) ? selectedTaskName : undefined) ??
      autoExpandTaskName(filteredTasks)
  );
  // A card scrolls into view only for an EXTERNAL arrival (a shared link or
  // back/forward), never for a ?taskId= this list wrote itself. This ref marks
  // our own writes; comparing against it only when the route has settled (not
  // every render) is what avoids the intermittent scroll-to-the-wrong-card
  // jump (BYT-9765).
  const selfWroteRef = useRef<string | undefined>(undefined);
  const [arrivalTaskName, setArrivalTaskName] = useState<string | undefined>(
    () =>
      isTaskInStage(selectedTaskName) &&
      selectedTaskName !== selfWroteRef.current
        ? selectedTaskName
        : undefined
  );
  // Latest expansion, read by the stable toggle callback without being a dep.
  const expandedTaskNamesRef = useRef(expandedTaskNames);
  expandedTaskNamesRef.current = expandedTaskNames;

  // filteredTasks identity is poll-stable (snapshot gate), so the join runs
  // only when the stage's task set actually changed.
  const taskNamesKey = useMemo(
    () => filteredTasks.map((task) => task.name).join(","),
    [filteredTasks]
  );

  // First activation of a lazily-mounted list: seed the default expansion
  // during render (same BYT-9763 idiom as below) so the stage's first visible
  // paint already shows the right card open.
  if (active && !activated) {
    setActivated(true);
    setDisplayedTaskCount(
      initialDisplayedCount(filteredTasks, selectedTaskName)
    );
    setExpandedTaskNames(initialExpandedNames(filteredTasks, selectedTaskName));
  }

  // When the visible task set changes (a plan edit, a filter change), re-derive
  // the list state during render (useOnKeyChange / BYT-9763), so the first paint
  // after the change already shows the right task expanded. Keyed on task
  // names, not statuses, so polling-driven status changes never reset the
  // user's expansion mid-view.
  useOnKeyChange(taskNamesKey, () => {
    setDisplayedTaskCount(
      initialDisplayedCount(filteredTasks, selectedTaskName)
    );
    // A never-activated (still hidden) list keeps everything collapsed — its
    // default expansion is seeded on first activation above.
    setExpandedTaskNames(
      activated
        ? initialExpandedNames(filteredTasks, selectedTaskName)
        : new Set()
    );
    setSelectedTaskNames((prev) => {
      const remaining = [...prev].filter((taskName) =>
        filteredTasks.some((task) => task.name === taskName)
      );
      return remaining.length === prev.size ? prev : new Set(remaining);
    });
    // Keep focus valid: fall back to the default if the focused task is gone.
    setFocusedTaskName((prev) =>
      isTaskInStage(prev) ? prev : autoExpandTaskName(filteredTasks)
    );
  });

  // When ?taskId= settles to a task in this stage, focus + reveal it. Scroll
  // only when it is an external arrival (not a ?taskId= we just wrote) — decided
  // here, once the route has settled, so the transient render during our own
  // write can't be mistaken for an arrival.
  useOnKeyChange(selectedTaskName ?? "", () => {
    if (isTaskInStage(selectedTaskName)) {
      const index = filteredTasks.findIndex(
        (task) => task.name === selectedTaskName
      );
      setExpandedTaskNames((prev) =>
        prev.has(selectedTaskName) ? prev : new Set(prev).add(selectedTaskName)
      );
      setDisplayedTaskCount((count) => Math.max(count, index + 1));
      setFocusedTaskName(selectedTaskName);
      setArrivalTaskName(
        selectedTaskName === selfWroteRef.current ? undefined : selectedTaskName
      );
    } else {
      setArrivalTaskName(undefined);
    }
  });

  // Mirror the focused task into ?stageId&taskId= while this stage is active,
  // so the URL is a shareable link. Only the active stage writes (a hidden
  // keep-alive list must not hijack the address bar); marking the write in
  // selfWroteRef keeps it from being read back as an external arrival.
  useEffect(() => {
    if (
      readonly ||
      !active ||
      !focusedTaskName ||
      selectedTaskName === focusedTaskName
    ) {
      return;
    }
    selfWroteRef.current = focusedTaskName;
    void router.replace({
      query: deployTaskQuery(stage.name, focusedTaskName),
    });
  }, [active, readonly, focusedTaskName, selectedTaskName, stage.name]);

  const visibleTasks = filteredTasks.slice(0, displayedTaskCount);
  const hasMoreTasks = filteredTasks.length > displayedTaskCount;
  const remainingTasksCount = filteredTasks.length - displayedTaskCount;
  const selectedTasks = useMemo(
    () => filteredTasks.filter((task) => selectedTaskNames.has(task.name)),
    [filteredTasks, selectedTaskNames]
  );

  // One control per card: toggle it open or closed (local state; keep-alive
  // preserves it across stage switches). Opening focuses the task, which the
  // effect above mirrors into the URL. Stable identities so the memoized cards
  // don't re-render when the list does.
  const toggleTask = useCallback((task: Task) => {
    const willExpand = !expandedTaskNamesRef.current.has(task.name);
    setExpandedTaskNames((prev) => {
      const next = new Set(prev);
      if (next.has(task.name)) next.delete(task.name);
      else next.add(task.name);
      return next;
    });
    if (willExpand) {
      setFocusedTaskName(task.name);
    }
  }, []);
  const toggleSelect = useCallback((task: Task) => {
    setSelectedTaskNames((prev) => {
      const next = new Set(prev);
      if (next.has(task.name)) next.delete(task.name);
      else next.add(task.name);
      return next;
    });
  }, []);

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
            currentUser={page.currentUser}
            deepLinked={arrivalTaskName === task.name}
            key={task.name}
            isExpanded={expandedTaskNames.has(task.name)}
            isSelected={selectedTaskNames.has(task.name)}
            isSelectable={!readonly && isDeployTaskSelectable(task)}
            issue={page.issue}
            onRefresh={page.refreshState}
            onToggleExpand={toggleTask}
            onToggleSelect={toggleSelect}
            plan={page.plan}
            project={page.project}
            rolloutName={page.rollout?.name ?? ""}
            stage={stage}
            task={task}
            taskRuns={page.taskRunsByTaskName.get(task.name) ?? EMPTY_TASK_RUNS}
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
              appearance="secondary"
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
