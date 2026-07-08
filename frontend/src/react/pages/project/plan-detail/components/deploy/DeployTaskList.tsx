import {
  type MutableRefObject,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
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
export const autoExpandTaskName = (tasks: Task[]): string | undefined =>
  (
    tasks.find(
      (task) =>
        task.status === Task_Status.RUNNING ||
        task.status === Task_Status.FAILED
    ) ?? tasks[0]
  )?.name;

// The one place a stage's ?taskId= deep-link shape is built.
export const deployTaskQuery = (stageName: string, taskName: string) => ({
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
  onOpenedTaskChange,
  selfWrittenTaskRef,
}: {
  stage: Stage;
  readonly?: boolean;
  // Every stage's list stays mounted (visibility is the parent's concern);
  // `active` marks the one the user is looking at. Heavy work — the default
  // card expansion (Monaco + log fetch) and the ?taskId= URL mirror — is
  // deferred to the first activation so loading the page doesn't mount every
  // stage's editors at once, and hidden stages never write the URL.
  active?: boolean;
  // Reports this stage's currently-mirrored task (default pick, honored deep
  // link, or a card the user opened) so the parent can restore it in the URL
  // when switching back to this stage.
  onOpenedTaskChange?: (stageName: string, taskName: string) => void;
  // Shared "we wrote this ?taskId= ourselves" marker. A self-written task is
  // not an arrival: it must not scroll its card into view. Only a taskId the
  // page arrived with from outside (shared link, back/forward) scrolls.
  selfWrittenTaskRef?: MutableRefObject<string | undefined>;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  // Full task resource name resolved from the ?taskId= deep link.
  const selectedTaskName = page.selectedTaskName;
  const filteredTasks = stage.tasks;
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
  const selfWrittenLocalRef = useRef<string | undefined>(undefined);
  const selfWritten = selfWrittenTaskRef ?? selfWrittenLocalRef;
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
  // the list state — pagination, the auto-expanded task, and the still-valid
  // selection — during render (useOnKeyChange / BYT-9763), so the first paint
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
  });

  // When the deep-linked task changes on the same task set (e.g. following a
  // second ?taskId= link), expand it and reveal its page — also during render
  // so the card is in the DOM before its scroll-into-view effect fires. A
  // self-written taskId (a stage-switch restore) is not a link being followed:
  // it must not re-open a card the user deliberately closed.
  useOnKeyChange(selectedTaskName ?? "", () => {
    if (selectedTaskName && selectedTaskName !== selfWritten.current) {
      const index = filteredTasks.findIndex(
        (task) => task.name === selectedTaskName
      );
      if (index >= 0) {
        setExpandedTaskNames((prev) =>
          prev.has(selectedTaskName)
            ? prev
            : new Set(prev).add(selectedTaskName)
        );
        setDisplayedTaskCount((count) => Math.max(count, index + 1));
      }
    }
  });

  const visibleTasks = filteredTasks.slice(0, displayedTaskCount);
  const hasMoreTasks = filteredTasks.length > displayedTaskCount;
  const remainingTasksCount = filteredTasks.length - displayedTaskCount;
  const selectedTasks = useMemo(
    () => filteredTasks.filter((task) => selectedTaskNames.has(task.name)),
    [filteredTasks, selectedTaskNames]
  );

  // Callbacks are identity-stable (state is read through refs) so the memoized
  // cards don't re-render when the list does.
  const expandedTaskNamesRef = useRef(expandedTaskNames);
  expandedTaskNamesRef.current = expandedTaskNames;

  // One control per card: toggle it open or closed. Opening mirrors the task
  // into the URL (?taskId= expands + scrolls on arrival) so the address bar is
  // the shareable link without a separate affordance; `replace`, not `push` —
  // it's state reflection, not navigation. Closing leaves the URL alone.
  // Preview stages (readonly) have no shareable task, so they just toggle.
  const toggleTask = useCallback(
    (task: Task) => {
      const willExpand = !expandedTaskNamesRef.current.has(task.name);
      setExpandedTaskNames((prev) => {
        const next = new Set(prev);
        if (next.has(task.name)) next.delete(task.name);
        else next.add(task.name);
        return next;
      });
      if (!readonly && willExpand) {
        onOpenedTaskChange?.(stage.name, task.name);
        selfWritten.current = task.name;
        void router.replace({ query: deployTaskQuery(stage.name, task.name) });
      }
    },
    [readonly, stage.name, onOpenedTaskChange, selfWritten]
  );
  const toggleSelect = useCallback((task: Task) => {
    setSelectedTaskNames((prev) => {
      const next = new Set(prev);
      if (next.has(task.name)) next.delete(task.name);
      else next.add(task.name);
      return next;
    });
  }, []);

  // Default-select: on first landing on a stage with no ?taskId=, mirror the
  // auto-opened task into the URL so the address bar reflects what's open from
  // the start (and is shareable). Runs once per instance (each instance is
  // permanently bound to one stage); the written name is marked self-written
  // so it isn't treated as an arrival — a normal load must not auto-scroll,
  // only an explicit shared link does.
  const hasDefaultSelectedRef = useRef(false);
  useEffect(() => {
    // Only the stage the user is looking at may mirror its default task into
    // the URL — a hidden keep-alive list writing ?taskId= would hijack the
    // address bar (and the active stage derivation) from offscreen.
    if (readonly || !active || hasDefaultSelectedRef.current) {
      return;
    }
    // Only a deep link into THIS stage counts — right after an (optimistic)
    // stage switch the URL can still carry the previous stage's taskId, which
    // must not suppress this stage's default write.
    if (
      selectedTaskName &&
      filteredTasks.some((task) => task.name === selectedTaskName)
    ) {
      hasDefaultSelectedRef.current = true;
      // Remember the honored link as this stage's mirrored task so switching
      // away and back restores it.
      onOpenedTaskChange?.(stage.name, selectedTaskName);
      return;
    }
    const autoName = autoExpandTaskName(filteredTasks);
    if (!autoName) {
      return;
    }
    hasDefaultSelectedRef.current = true;
    onOpenedTaskChange?.(stage.name, autoName);
    selfWritten.current = autoName;
    void router.replace({ query: deployTaskQuery(stage.name, autoName) });
  }, [
    stage.name,
    readonly,
    active,
    selectedTaskName,
    filteredTasks,
    onOpenedTaskChange,
    selfWritten,
  ]);

  // A card scrolls into view only for a genuine arrival (shared link,
  // back/forward) — never for a taskId this page wrote into the URL itself
  // (default pick, stage-switch restore, or a card the user opened).
  const deepLinkedTaskName =
    selectedTaskName && selectedTaskName !== selfWritten.current
      ? selectedTaskName
      : undefined;

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
            deepLinked={deepLinkedTaskName === task.name}
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
