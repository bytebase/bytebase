import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { Check, FastForward, Loader2, Pause, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { rolloutServiceClientConnect } from "@/connect";
import { EngineIcon } from "@/react/components/EngineIcon";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import { Input } from "@/react/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/react/components/ui/radio-group";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { Textarea } from "@/react/components/ui/textarea";
import { Tooltip } from "@/react/components/ui/tooltip";
import { cn } from "@/react/lib/utils";
import {
  pushNotification,
  useCurrentUserV1,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useProjectV1Store,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { Issue_Type } from "@/types/proto-es/v1/issue_service_pb";
import type { Stage, Task } from "@/types/proto-es/v1/rollout_service_pb";
import {
  BatchCancelTaskRunsRequestSchema,
  BatchRunTasksRequestSchema,
  BatchSkipTasksRequestSchema,
  ListTaskRunsRequestSchema,
  Task_Status,
  Task_Type,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractDatabaseResourceName,
  extractInstanceResourceName,
  extractStageUID,
} from "@/utils";
import {
  CANCELABLE_TASK_STATUSES,
  canRolloutTasks,
  preloadRolloutPermissionContext,
  RUNNABLE_TASK_STATUSES,
} from "../../issue-detail/utils/rollout";
import { usePlanDetailContext } from "../context/PlanDetailContext";

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
  const projectStore = useProjectV1Store();
  const currentUser = useCurrentUserV1().value;
  const projectName = `${projectNamePrefix}${page.projectId}`;
  const project = projectStore.getProjectByName(projectName);
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
      <SheetContent width="wide">
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
                        <PlanDetailTaskStatus status={task.status} />
                        <PlanDetailTaskDatabaseName task={task} />
                      </li>
                    ))}
                  </ul>
                </div>
              </div>
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
                  <Input
                    className="mt-2"
                    onChange={(event) =>
                      setRunTimeInMS(
                        parseDatetimeLocalValue(event.target.value)
                      )
                    }
                    placeholder={t("task.select-scheduled-time")}
                    type="datetime-local"
                    value={formatDatetimeLocalValue(runTimeInMS)}
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
          <Button onClick={() => onOpenChange(false)} variant="ghost">
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

function formatDatetimeLocalValue(value?: number) {
  if (value === undefined) return "";
  const date = new Date(value);
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");
  return `${year}-${month}-${day}T${hours}:${minutes}`;
}

function parseDatetimeLocalValue(value: string) {
  return value ? new Date(value).getTime() : undefined;
}

function PlanDetailTaskDatabaseName({ task }: { task: Task }) {
  if (task.target) {
    return <PlanDetailDatabaseTarget target={task.target} />;
  }
  return <span className="truncate">{task.name.split("/").at(-1)}</span>;
}

function PlanDetailDatabaseTarget({ target }: { target: string }) {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();
  const environmentStore = useEnvironmentV1Store();
  const database = databaseStore.getDatabaseByName(target);
  const environment = environmentStore.getEnvironmentByName(
    database.effectiveEnvironment ??
      database.instanceResource?.environment ??
      ""
  );
  const instance = database.instanceResource;
  const { databaseName } = extractDatabaseResourceName(target);
  const instanceTitle =
    instance?.title ||
    extractInstanceResourceName(target) ||
    t("common.unknown");

  return (
    <div className="flex min-w-0 items-center truncate text-sm">
      {instance && (
        <EngineIcon
          engine={instance.engine}
          className="mr-1 inline-block h-4 w-4"
        />
      )}
      <span className="mr-1 truncate text-gray-400">{environment.title}</span>
      <span className="truncate text-gray-600">{instanceTitle}</span>
      <span className="mx-1 shrink-0 text-gray-500 opacity-60">›</span>
      <span className="truncate text-gray-800">{databaseName}</span>
    </div>
  );
}

function PlanDetailStageEnvironment({
  environmentName,
}: {
  environmentName: string;
}) {
  const environmentStore = useEnvironmentV1Store();
  const environment = environmentStore.getEnvironmentByName(environmentName);

  return (
    <span className="text-sm text-control">
      {environment.title || environmentName.split("/").at(-1)}
    </span>
  );
}

function PlanDetailTaskStatus({ status }: { status: Task_Status }) {
  const { t } = useTranslation();
  const label = taskStatusLabel(t, status);
  return (
    <Tooltip content={label}>
      <div
        aria-label={label}
        className={cn(
          "relative flex h-5 w-5 shrink-0 select-none items-center justify-center overflow-hidden rounded-full",
          taskStatusClassName(status)
        )}
      >
        {status === Task_Status.NOT_STARTED && (
          <span className="h-1/2 w-1/2 rounded-full bg-control" />
        )}
        {status === Task_Status.PENDING && <Pause className="h-3/4 w-3/4" />}
        {status === Task_Status.RUNNING && (
          <div className="relative flex h-1/2 w-1/2 overflow-visible">
            <span
              className="absolute z-0 h-full w-full animate-ping-slow rounded-full"
              style={{ backgroundColor: "rgba(37, 99, 235, 0.5)" }}
            />
            <span className="z-1 h-full w-full rounded-full bg-info" />
          </div>
        )}
        {status === Task_Status.SKIPPED && (
          <FastForward className="h-3/4 w-3/4" />
        )}
        {status === Task_Status.DONE && <Check className="h-3/4 w-3/4" />}
        {status === Task_Status.FAILED && (
          <span className="rounded-full text-base font-medium">!</span>
        )}
        {status === Task_Status.CANCELED && (
          <span className="text-base leading-none">-</span>
        )}
        {status !== Task_Status.CANCELED &&
          status !== Task_Status.FAILED &&
          status !== Task_Status.DONE &&
          status !== Task_Status.SKIPPED &&
          status !== Task_Status.RUNNING &&
          status !== Task_Status.PENDING &&
          status !== Task_Status.NOT_STARTED && <X className="h-3/4 w-3/4" />}
      </div>
    </Tooltip>
  );
}

function taskStatusClassName(status: Task_Status) {
  switch (status) {
    case Task_Status.NOT_STARTED:
      return "border-2 border-control bg-white";
    case Task_Status.PENDING:
    case Task_Status.RUNNING:
      return "border-2 border-info bg-white text-info";
    case Task_Status.SKIPPED:
      return "border-2 border-control-light bg-white text-gray-600";
    case Task_Status.DONE:
      return "bg-success text-white";
    case Task_Status.FAILED:
      return "bg-error text-white";
    case Task_Status.CANCELED:
      return "border-2 border-control-light bg-white text-control-light";
    default:
      return "border border-dashed border-control bg-white";
  }
}

function taskStatusLabel(t: (key: string) => string, status: Task_Status) {
  switch (status) {
    case Task_Status.NOT_STARTED:
      return t("task.status.not-started");
    case Task_Status.PENDING:
      return t("task.status.pending");
    case Task_Status.RUNNING:
      return t("task.status.running");
    case Task_Status.SKIPPED:
      return t("task.status.skipped");
    case Task_Status.DONE:
      return t("task.status.done");
    case Task_Status.FAILED:
      return t("task.status.failed");
    case Task_Status.CANCELED:
      return t("task.status.canceled");
    default:
      return t("task.status.not-started");
  }
}
