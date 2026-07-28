import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { Loader2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { rolloutServiceClientConnect } from "@/api";
import { TaskStatusIcon } from "@/components/TaskStatusIcon";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Textarea } from "@/components/ui/textarea";
import { Tooltip } from "@/components/ui/tooltip";
import { useCurrentUser } from "@/hooks/useAppState";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { projectNamePrefix } from "@/stores/modules/v1/common";
import { Issue_Type } from "@/types/proto-es/v1/issue_service_pb";
import type { Stage, Task } from "@/types/proto-es/v1/rollout_service_pb";
import {
  BatchCancelTaskRunsRequestSchema,
  BatchRunTasksRequestSchema,
  BatchSkipTasksRequestSchema,
  ListTaskRunsRequestSchema,
  Task_Type,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import { extractStageUID } from "@/utils";
import {
  CANCELABLE_TASK_STATUSES,
  canRolloutTasks,
  preloadRolloutPermissionContext,
  RUNNABLE_TASK_STATUSES,
} from "../../issue-detail/utils/rollout";
import {
  ScheduledRunTimeInput,
  TASK_ROLLOUT_ACTION_SHEET_WIDTH,
} from "../../utils/taskRolloutActionPanel";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import { PlanTargetDisplay } from "./PlanTargetDisplay";

const DEFAULT_RUN_DELAY_MS = 60 * 60 * 1000;

export type PlanDetailTaskRolloutAction = "RUN" | "SKIP" | "CANCEL";

export type PlanDetailTaskRolloutTarget = {
  type: "tasks";
  tasks?: Task[];
  stage?: Stage;
};

const pluralizedLabel = (raw: string, count: number): string => {
  const [singular, plural] = raw.split("|").map((part) => part.trim());
  if (!plural) return singular;
  return count === 1 ? singular : plural;
};

export function PlanDetailTaskRolloutActionPanel({
  action,
  onConfirm,
  onOpenChange,
  open,
  target,
}: {
  action?: PlanDetailTaskRolloutAction;
  onConfirm?: () => Promise<void> | void;
  onOpenChange: (open: boolean) => void;
  open: boolean;
  target: PlanDetailTaskRolloutTarget;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const currentUser = useCurrentUser();
  const projectName = `${projectNamePrefix}${page.projectId}`;
  const projectsByName = useAppStore((s) => s.projectsByName);
  const project = useMemo(
    () => useAppStore.getState().getProjectByName(projectName),
    [projectsByName, projectName]
  );
  const [loading, setLoading] = useState(false);
  const [permissionLoading, setPermissionLoading] = useState(false);
  const [canRun, setCanRun] = useState(true);
  const [comment, setComment] = useState("");
  const [runTimeInMS, setRunTimeInMS] = useState<number | undefined>();
  const [skipPriorBackup, setSkipPriorBackup] = useState(false);

  const allRolloutTasks = useMemo(
    () => page.rollout?.stages.flatMap((stage) => stage.tasks) ?? [],
    [page.rollout]
  );
  const rolloutType = useMemo(() => {
    if (
      allRolloutTasks.every((task) => task.type === Task_Type.DATABASE_CREATE)
    ) {
      return "DATABASE_CREATE";
    }
    if (
      allRolloutTasks.every((task) => task.type === Task_Type.DATABASE_EXPORT)
    ) {
      return "DATABASE_EXPORT";
    }
    return "DATABASE_CHANGE";
  }, [allRolloutTasks]);
  const isDatabaseCreateOrExport =
    rolloutType === "DATABASE_CREATE" || rolloutType === "DATABASE_EXPORT";
  const baseTasks = useMemo(() => {
    if (target.tasks) {
      return target.tasks;
    }
    if (isDatabaseCreateOrExport) {
      return allRolloutTasks;
    }
    return target.stage?.tasks ?? [];
  }, [allRolloutTasks, isDatabaseCreateOrExport, target.stage, target.tasks]);
  const eligibleTasks = useMemo(() => {
    if (action === "RUN" || action === "SKIP") {
      return baseTasks.filter((task) =>
        RUNNABLE_TASK_STATUSES.includes(task.status)
      );
    }
    if (action === "CANCEL") {
      return baseTasks.filter((task) =>
        CANCELABLE_TASK_STATUSES.includes(task.status)
      );
    }
    return [];
  }, [action, baseTasks]);
  const hasPriorBackupTasks = useMemo(() => {
    return eligibleTasks.some((task) => {
      const spec = page.plan.specs.find((item) => item.id === task.specId);
      return (
        spec?.config.case === "changeDatabaseConfig" &&
        spec.config.value.enablePriorBackup
      );
    });
  }, [eligibleTasks, page.plan.specs]);
  const showStageInfo = !isDatabaseCreateOrExport;
  const showTaskInfo = rolloutType !== "DATABASE_CREATE";
  const taskCountSuffix = useMemo(() => {
    if (
      eligibleTasks.length <= 1 ||
      !showStageInfo ||
      !target.stage?.tasks.length
    ) {
      return "";
    }
    const total = target.stage.tasks.length;
    return eligibleTasks.length === total
      ? String(eligibleTasks.length)
      : `${eligibleTasks.length} / ${total}`;
  }, [eligibleTasks.length, showStageInfo, target.stage?.tasks.length]);

  useEffect(() => {
    if (!open) {
      setComment("");
      setRunTimeInMS(undefined);
      setSkipPriorBackup(false);
      return;
    }

    let canceled = false;
    const prepare = async () => {
      setPermissionLoading(true);
      setCanRun(false);
      await preloadRolloutPermissionContext({
        environment: target.stage?.environment,
        projectName: project?.name ?? "",
        tasks: eligibleTasks,
      });
      if (canceled) return;
      setCanRun(
        project
          ? canRolloutTasks({
              currentUser,
              environment: target.stage?.environment,
              issue: page.issue,
              project,
              tasks: eligibleTasks,
            })
          : false
      );
      setPermissionLoading(false);
    };

    void prepare();
    return () => {
      canceled = true;
    };
  }, [
    currentUser,
    eligibleTasks,
    open,
    page.issue,
    project,
    target.stage?.environment,
  ]);

  useEffect(() => {
    if (!open) return;
    const runTimes = new Set(
      eligibleTasks.map((task) => task.runTime).filter(Boolean)
    );
    if (runTimes.size !== 1) return;
    const runTime = [...runTimes][0];
    if (!runTime) return;
    setRunTimeInMS(Number(runTime.seconds) * 1000 + runTime.nanos / 1000000);
  }, [eligibleTasks, open]);

  const validationErrors = useMemo(() => {
    const errors: string[] = [];
    if (eligibleTasks.length === 0) {
      errors.push(t("common.no-data"));
    }
    if (!canRun) {
      errors.push(
        page.issue?.type === Issue_Type.DATABASE_EXPORT
          ? t("task.data-export-creator-only")
          : t("task.no-permission")
      );
    }
    if (
      action === "RUN" &&
      runTimeInMS !== undefined &&
      runTimeInMS <= Date.now()
    ) {
      errors.push(t("task.error.scheduled-time-must-be-in-the-future"));
    }
    return errors;
  }, [action, canRun, eligibleTasks.length, page.issue?.type, runTimeInMS, t]);

  const handleConfirm = useCallback(async () => {
    if (!page.rollout || !action || validationErrors.length > 0) {
      return;
    }

    try {
      setLoading(true);
      if (action === "RUN") {
        await runTasks({
          rolloutName: page.rollout.name,
          runTimeInMS,
          skipPriorBackup,
          tasks: eligibleTasks,
        });
      } else if (action === "SKIP") {
        await skipTasks({
          comment,
          rolloutName: page.rollout.name,
          tasks: eligibleTasks,
        });
      } else {
        await cancelTasks({
          rolloutName: page.rollout.name,
          tasks: eligibleTasks,
        });
      }
      await onConfirm?.();
      onOpenChange(false);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: String(error),
      });
    } finally {
      setLoading(false);
    }
  }, [
    action,
    comment,
    eligibleTasks,
    onConfirm,
    onOpenChange,
    page.rollout,
    runTimeInMS,
    skipPriorBackup,
    t,
    validationErrors.length,
  ]);

  if (!action) {
    return null;
  }

  const title =
    action === "RUN"
      ? pluralizedLabel(t("task.run-task"), eligibleTasks.length)
      : action === "SKIP"
        ? pluralizedLabel(t("task.skip-task"), eligibleTasks.length)
        : pluralizedLabel(t("task.cancel-task"), eligibleTasks.length);
  const confirmButtonText =
    action === "RUN"
      ? t("common.run")
      : action === "SKIP"
        ? t("common.skip")
        : t("common.cancel");

  return (
    <Sheet onOpenChange={onOpenChange} open={open}>
      <SheetContent width={TASK_ROLLOUT_ACTION_SHEET_WIDTH}>
        <SheetHeader>
          <SheetTitle>{title}</SheetTitle>
        </SheetHeader>
        <SheetBody className="relative">
          <div className="flex h-full flex-col gap-y-4 px-1">
            {validationErrors.length > 0 && (
              <div className="rounded-xs border border-error/30 bg-error/5 px-4 py-3">
                <div className="mb-1 font-medium text-error">
                  {t("rollout.task-execution-errors")}
                </div>
                <ul className="flex list-inside list-disc flex-col gap-y-1 text-sm text-error">
                  {validationErrors.map((error) => (
                    <li key={error}>{error}</li>
                  ))}
                </ul>
              </div>
            )}

            {showStageInfo && target.stage?.environment && (
              <div className="flex shrink-0 flex-row items-center justify-start gap-x-2 overflow-y-hidden">
                <label className="font-medium text-control">
                  {t("common.stage")}
                </label>
                <PlanDetailStageEnvironment
                  environmentName={target.stage.environment}
                />
              </div>
            )}

            {showTaskInfo && (
              <div className="flex shrink-0 flex-col gap-y-1">
                <label className="text-control">
                  <span className="font-medium">{t("common.task")}</span>
                  {taskCountSuffix && (
                    <span className="opacity-80"> ({taskCountSuffix})</span>
                  )}
                </label>
                <div className="max-h-64 overflow-y-auto">
                  <ul className="flex flex-col gap-y-2 text-sm">
                    {eligibleTasks.map((task) => (
                      <li
                        className="flex h-8 items-center gap-x-2"
                        key={task.name}
                      >
                        <TaskStatusIcon size="small" status={task.status} />
                        <PlanDetailTaskDatabaseName task={task} />
                      </li>
                    ))}
                  </ul>
                </div>
              </div>
            )}

            {action === "CANCEL" && (
              <p className="text-sm text-control-light">
                {t("task.cancel-task-description")}
              </p>
            )}

            {action === "RUN" && (
              <div className="flex flex-col">
                <h3 className="mb-1 font-medium text-control">
                  {t("task.execution-time")}
                </h3>
                <RadioGroup
                  className="flex! flex-col gap-2 sm:flex-row sm:gap-4"
                  onValueChange={(value) =>
                    setRunTimeInMS(
                      value === "immediate"
                        ? undefined
                        : Date.now() + DEFAULT_RUN_DELAY_MS
                    )
                  }
                  value={runTimeInMS === undefined ? "immediate" : "scheduled"}
                >
                  <RadioGroupItem value="immediate">
                    {t("task.run-immediately.self")}
                  </RadioGroupItem>
                  <RadioGroupItem value="scheduled">
                    {t("task.schedule-for-later.self")}
                  </RadioGroupItem>
                </RadioGroup>
                <div className="mt-1 text-sm text-control-light">
                  {runTimeInMS === undefined
                    ? t("task.run-immediately.description")
                    : t("task.schedule-for-later.description")}
                </div>
                {runTimeInMS !== undefined && (
                  <ScheduledRunTimeInput
                    className="mt-2"
                    onChange={setRunTimeInMS}
                    placeholder={t("task.select-scheduled-time")}
                    value={runTimeInMS}
                  />
                )}
              </div>
            )}

            {action === "RUN" && hasPriorBackupTasks && (
              <div className="flex flex-col">
                <label className="inline-flex items-center gap-x-2 text-sm">
                  <Checkbox
                    checked={skipPriorBackup}
                    onCheckedChange={(checked) => setSkipPriorBackup(checked)}
                  />
                  <span>{t("task.skip-prior-backup")}</span>
                </label>
                <div className="ml-6 text-sm text-control-light">
                  {t("task.skip-prior-backup-description")}
                </div>
              </div>
            )}

            {action === "SKIP" && page.issue && (
              <div className="flex shrink-0 flex-col gap-y-1">
                <p className="font-medium text-control">{t("common.reason")}</p>
                <Textarea
                  className="min-h-28"
                  maxLength={1000}
                  onChange={(event) => setComment(event.target.value)}
                  placeholder={t("issue.leave-a-comment")}
                  value={comment}
                />
              </div>
            )}
          </div>

          {(loading || permissionLoading) && (
            <div className="absolute inset-0 flex items-center justify-center bg-white/50">
              <Loader2 className="h-6 w-6 animate-spin text-control" />
            </div>
          )}
        </SheetBody>
        <SheetFooter>
          <Button onClick={() => onOpenChange(false)} appearance="secondary">
            {t("common.close")}
          </Button>
          <Tooltip
            content={
              validationErrors.length > 0 ? (
                <ul className="flex list-inside list-disc flex-col gap-y-1">
                  {validationErrors.map((error) => (
                    <li key={error}>{error}</li>
                  ))}
                </ul>
              ) : null
            }
          >
            <div>
              <Button
                disabled={permissionLoading || validationErrors.length > 0}
                onClick={() => void handleConfirm()}
              >
                {loading && <Loader2 className="h-4 w-4 animate-spin" />}
                {confirmButtonText}
              </Button>
            </div>
          </Tooltip>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

async function runTasks({
  rolloutName,
  runTimeInMS,
  skipPriorBackup,
  tasks,
}: {
  rolloutName: string;
  runTimeInMS?: number;
  skipPriorBackup: boolean;
  tasks: Task[];
}) {
  const tasksByStage = groupTasksByStage(tasks);
  for (const [stageId, stageTasks] of tasksByStage) {
    const request = create(BatchRunTasksRequestSchema, {
      parent: `${rolloutName}/stages/${stageId}`,
      tasks: stageTasks.map((task) => task.name),
    });
    if (runTimeInMS !== undefined) {
      request.runTime = create(TimestampSchema, {
        seconds: BigInt(Math.floor(runTimeInMS / 1000)),
        nanos: (runTimeInMS % 1000) * 1000000,
      });
    }
    request.skipPriorBackup = skipPriorBackup;
    await rolloutServiceClientConnect.batchRunTasks(request);
  }
}

async function skipTasks({
  comment,
  rolloutName,
  tasks,
}: {
  comment: string;
  rolloutName: string;
  tasks: Task[];
}) {
  const tasksByStage = groupTasksByStage(tasks);
  for (const [stageId, stageTasks] of tasksByStage) {
    await rolloutServiceClientConnect.batchSkipTasks(
      create(BatchSkipTasksRequestSchema, {
        parent: `${rolloutName}/stages/${stageId}`,
        reason: comment,
        tasks: stageTasks.map((task) => task.name),
      })
    );
  }
}

async function cancelTasks({
  rolloutName,
  tasks,
}: {
  rolloutName: string;
  tasks: Task[];
}) {
  const tasksByStage = groupTasksByStage(tasks);
  const cancelableRuns = new Map<string, string[]>();
  for (const [stageId, stageTasks] of tasksByStage) {
    const taskNames = new Set(stageTasks.map((task) => task.name));
    const response = await rolloutServiceClientConnect.listTaskRuns(
      create(ListTaskRunsRequestSchema, {
        parent: `${rolloutName}/stages/${stageId}/tasks/-`,
      })
    );
    const runs = response.taskRuns
      .filter((run) => {
        const taskName = run.name.split("/taskRuns/")[0];
        return (
          taskNames.has(taskName) &&
          (run.status === TaskRun_Status.PENDING ||
            run.status === TaskRun_Status.AVAILABLE ||
            run.status === TaskRun_Status.RUNNING)
        );
      })
      .map((run) => run.name);
    if (runs.length > 0) {
      cancelableRuns.set(`${rolloutName}/stages/${stageId}`, runs);
    }
  }

  await Promise.all(
    [...cancelableRuns.entries()].map(([stageName, taskRuns]) =>
      rolloutServiceClientConnect.batchCancelTaskRuns(
        create(BatchCancelTaskRunsRequestSchema, {
          parent: `${stageName}/tasks/-`,
          taskRuns,
        })
      )
    )
  );
}

function groupTasksByStage(tasks: Task[]) {
  const grouped = new Map<string, Task[]>();
  for (const task of tasks) {
    const stageId = extractStageUID(task.name);
    if (!stageId) continue;
    if (!grouped.has(stageId)) {
      grouped.set(stageId, []);
    }
    grouped.get(stageId)?.push(task);
  }
  return grouped;
}

function PlanDetailTaskDatabaseName({ task }: { task: Task }) {
  if (task.target) {
    return <PlanTargetDisplay showEnvironment target={task.target} />;
  }
  return <span className="truncate">{task.name.split("/").at(-1)}</span>;
}

function PlanDetailStageEnvironment({
  environmentName,
}: {
  environmentName: string;
}) {
  const environmentList = useAppStore((s) => s.environmentList);
  const environment = useMemo(
    () => useAppStore.getState().getEnvironmentByName(environmentName),
    [environmentList, environmentName]
  );

  return (
    <span className="text-sm text-control">
      {environment.title || environmentName.split("/").at(-1)}
    </span>
  );
}
