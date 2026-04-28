import { create } from "@bufbuild/protobuf";
import { DurationSchema } from "@bufbuild/protobuf/wkt";
import { Check, Minus } from "lucide-react";
import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { EngineIcon } from "@/react/components/EngineIcon";
import { EllipsisText } from "@/react/components/ui/ellipsis-text";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import {
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useProjectV1Store,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  getDateForPbTimestampProtoEs,
  getTimeForPbTimestampProtoEs,
} from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  type Task,
  type TaskRun,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  databaseForTask,
  extractDatabaseResourceName,
  extractInstanceResourceName,
  extractTaskUID,
  formatAbsoluteDateTime,
  humanizeDate,
  humanizeDurationV1,
} from "@/utils";
import { useIssueDetailContext } from "../context/IssueDetailContext";

export function IssueDetailTaskRunTable({
  maxHeight,
  showDatabaseColumn = false,
  taskRuns,
}: {
  maxHeight?: string | number;
  showDatabaseColumn?: boolean;
  taskRuns: TaskRun[];
}) {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const projectStore = useProjectV1Store();
  const databaseStore = useDatabaseV1Store();
  const projectName = `${projectNamePrefix}${page.projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));

  const taskByUID = useMemo(() => {
    const map = new Map<string, Task>();
    for (const stage of page.rollout?.stages ?? []) {
      for (const task of stage.tasks) {
        map.set(extractTaskUID(task.name), task);
      }
    }
    return map;
  }, [page.rollout?.stages]);

  const sortedTaskRuns = useMemo(() => {
    return [...taskRuns].sort((left, right) => {
      const leftTime = left.createTime ? Number(left.createTime.seconds) : 0;
      const rightTime = right.createTime ? Number(right.createTime.seconds) : 0;
      return rightTime - leftTime;
    });
  }, [taskRuns]);

  useEffect(() => {
    if (!showDatabaseColumn) {
      return;
    }

    const targets = [
      ...new Set(
        taskRuns
          .map((taskRun) => taskByUID.get(extractTaskUID(taskRun.name))?.target)
          .filter((target): target is string => Boolean(target))
      ),
    ];
    if (targets.length > 0) {
      void databaseStore.batchGetOrFetchDatabases(targets);
    }
  }, [databaseStore, showDatabaseColumn, taskByUID, taskRuns]);

  const getTaskForTaskRun = (taskRun: TaskRun) => {
    return taskByUID.get(extractTaskUID(taskRun.name));
  };

  const getDatabaseForTaskRun = (taskRun: TaskRun) => {
    const task = getTaskForTaskRun(taskRun);
    return task ? databaseForTask(project, task, page.plan) : undefined;
  };

  const wrapperStyle =
    maxHeight !== undefined
      ? {
          maxHeight:
            typeof maxHeight === "number" ? `${maxHeight}px` : maxHeight,
        }
      : undefined;

  return (
    <>
      <div className="overflow-auto rounded-sm border" style={wrapperStyle}>
        <Table className="table-fixed">
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <TableHead className="sticky top-0 z-10 w-9 bg-gray-50 px-2" />
              {showDatabaseColumn && (
                <TableHead className="sticky top-0 z-10 w-64 bg-gray-50">
                  {t("common.database")}
                </TableHead>
              )}
              <TableHead className="sticky top-0 z-10 bg-gray-50">
                {t("common.detail")}
              </TableHead>
              <TableHead className="sticky top-0 z-10 w-36 bg-gray-50">
                {t("task.created")}
              </TableHead>
              <TableHead className="sticky top-0 z-10 w-36 bg-gray-50">
                {t("task.started")}
              </TableHead>
              <TableHead className="sticky top-0 z-10 w-28 bg-gray-50 pr-6 whitespace-nowrap text-sm">
                {t("task.execution-time")}
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {sortedTaskRuns.map((taskRun) => {
              const database = getDatabaseForTaskRun(taskRun);
              return (
                <TableRow key={taskRun.name}>
                  <TableCell className="px-2">
                    <IssueDetailTaskRunStatusIcon status={taskRun.status} />
                  </TableCell>
                  {showDatabaseColumn && (
                    <TableCell>
                      {database ? (
                        <IssueDetailTaskRunDatabaseCell database={database} />
                      ) : (
                        <span className="text-control-light">-</span>
                      )}
                    </TableCell>
                  )}
                  <TableCell className="min-w-0 pr-2">
                    <IssueDetailTaskRunComment taskRun={taskRun} />
                  </TableCell>
                  <TableCell>
                    <IssueDetailTaskRunDateCell date={taskRun.createTime} />
                  </TableCell>
                  <TableCell>
                    <IssueDetailTaskRunDateCell date={taskRun.startTime} />
                  </TableCell>
                  <TableCell className="pr-6 whitespace-nowrap">
                    <span className="whitespace-nowrap text-sm text-control">
                      {executionDurationOfTaskRun(taskRun)
                        ? humanizeDurationV1(
                            executionDurationOfTaskRun(taskRun)
                          )
                        : "-"}
                    </span>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </div>
    </>
  );
}

function IssueDetailTaskRunStatusIcon({ status }: { status: TaskRun_Status }) {
  const { t } = useTranslation();
  const classes = (() => {
    switch (status) {
      case TaskRun_Status.PENDING:
        return "bg-white border-2 border-info text-info";
      case TaskRun_Status.RUNNING:
        return "bg-white border-2 border-info text-info";
      case TaskRun_Status.DONE:
        return "bg-success text-white";
      case TaskRun_Status.FAILED:
        return "bg-error text-white";
      case TaskRun_Status.CANCELED:
        return "bg-white border-2 border-gray-400 text-gray-400";
      default:
        return "";
    }
  })();

  return (
    <Tooltip content={taskRunStatusLabel(t, status)}>
      <div
        className={cn(
          "relative flex h-5 w-5 shrink-0 items-center justify-center rounded-full select-none",
          classes
        )}
      >
        {status === TaskRun_Status.PENDING && (
          <span className="h-1.5 w-1.5 rounded-full bg-info" />
        )}
        {status === TaskRun_Status.RUNNING && (
          <div className="relative flex h-2.5 w-2.5 overflow-visible">
            <span className="absolute z-0 h-full w-full animate-ping-slow rounded-full bg-blue-600/50" />
            <span className="z-1 h-full w-full rounded-full bg-info" />
          </div>
        )}
        {status === TaskRun_Status.DONE && <Check className="h-4 w-4" />}
        {status === TaskRun_Status.FAILED && (
          <span className="text-base font-medium text-white">!</span>
        )}
        {status === TaskRun_Status.CANCELED && <Minus className="h-4 w-4" />}
      </div>
    </Tooltip>
  );
}

function IssueDetailTaskRunComment({ taskRun }: { taskRun: TaskRun }) {
  const { t } = useTranslation();
  const earliestAllowedTime = taskRun.runTime
    ? getTimeForPbTimestampProtoEs(taskRun.runTime)
    : null;

  const comment = (() => {
    if (taskRun.status === TaskRun_Status.PENDING) {
      if (earliestAllowedTime) {
        return t("task-run.status.enqueued-with-rollout-time", {
          time: formatAbsoluteDateTime(earliestAllowedTime),
        });
      }
      return t("task-run.status.enqueued");
    }
    if (taskRun.status === TaskRun_Status.RUNNING && taskRun.schedulerInfo) {
      const cause = taskRun.schedulerInfo.waitingCause;
      if (cause?.cause?.case === "parallelTasksLimit") {
        return t("task-run.status.waiting-max-tasks-per-rollout", {
          time: formatAbsoluteDateTime(
            getTimeForPbTimestampProtoEs(taskRun.schedulerInfo.reportTime)
          ),
        });
      }
    }
    return taskRun.detail || "-";
  })();

  const commentLink =
    taskRun.status === TaskRun_Status.FAILED && comment.includes("version")
      ? {
          link: "https://docs.bytebase.com/change-database/troubleshoot/?source=console#duplicate-version",
          title: t("common.troubleshoot"),
        }
      : undefined;

  return (
    <div className="flex flex-col gap-y-0.5 xl:flex-row xl:items-center xl:gap-x-1">
      <div className="min-w-0 flex-1">
        <EllipsisText className="line-clamp-1" text={comment} />
      </div>
      {commentLink && (
        <a
          className="normal-link shrink-0"
          href={commentLink.link}
          rel="noopener noreferrer"
          target="_blank"
        >
          {commentLink.title}
        </a>
      )}
    </div>
  );
}

function IssueDetailTaskRunDateCell({
  date,
  format = "humanized",
}: {
  date?: Parameters<typeof getDateForPbTimestampProtoEs>[0];
  format?: "absolute" | "humanized";
}) {
  if (!date) {
    return <span className="text-control-light">-</span>;
  }

  const parsedDate = getDateForPbTimestampProtoEs(date);
  if (!parsedDate) {
    return <span className="text-control-light">-</span>;
  }
  const text =
    format === "absolute"
      ? formatAbsoluteDateTime(parsedDate.getTime())
      : humanizeDate(parsedDate);

  return (
    <Tooltip content={formatAbsoluteDateTime(parsedDate.getTime())}>
      <span className="text-sm text-control">{text}</span>
    </Tooltip>
  );
}

function IssueDetailTaskRunDatabaseCell({ database }: { database: Database }) {
  const { t } = useTranslation();
  const environmentStore = useEnvironmentV1Store();
  const environment = useVueState(() =>
    environmentStore.getEnvironmentByName(
      database.effectiveEnvironment ??
        database.instanceResource?.environment ??
        ""
    )
  );
  const instance = database.instanceResource;
  const { databaseName } = extractDatabaseResourceName(database.name);
  const instanceTitle =
    instance?.title ||
    extractInstanceResourceName(database.name) ||
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
      <span className="mx-1 shrink-0 text-gray-500 opacity-60">/</span>
      <span className="truncate text-gray-800">{databaseName}</span>
    </div>
  );
}

function executionDurationOfTaskRun(taskRun: TaskRun) {
  const { startTime, updateTime } = taskRun;
  if (!startTime || !updateTime) {
    return undefined;
  }
  if (Number(startTime.seconds) === 0) {
    return undefined;
  }
  if (taskRun.status === TaskRun_Status.RUNNING) {
    const elapsedMS = Date.now() - getTimeForPbTimestampProtoEs(startTime);
    return create(DurationSchema, {
      nanos: (elapsedMS % 1000) * 1e6,
      seconds: BigInt(Math.floor(elapsedMS / 1000)),
    });
  }
  const elapsedMS =
    getTimeForPbTimestampProtoEs(updateTime) -
    getTimeForPbTimestampProtoEs(startTime);
  return create(DurationSchema, {
    nanos: (elapsedMS % 1000) * 1e6,
    seconds: BigInt(Math.floor(elapsedMS / 1000)),
  });
}

function taskRunStatusLabel(
  t: ReturnType<typeof useTranslation>["t"],
  status: TaskRun_Status
) {
  switch (status) {
    case TaskRun_Status.PENDING:
      return t("task.status.pending");
    case TaskRun_Status.RUNNING:
      return t("task.status.running");
    case TaskRun_Status.DONE:
      return t("task.status.done");
    case TaskRun_Status.FAILED:
      return t("task.status.failed");
    case TaskRun_Status.CANCELED:
      return t("task.status.canceled");
    default:
      return String(status);
  }
}
