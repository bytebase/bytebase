import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { EngineIcon } from "@/react/components/EngineIcon";
import { HumanizeTs } from "@/react/components/HumanizeTs";
import { TaskRunStatusIcon } from "@/react/components/TaskRunStatusIcon";
import { EllipsisText } from "@/react/components/ui/ellipsis-text";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import { useProjectByName } from "@/react/hooks/useProjectByName";
import {
  executionDurationOfTaskRun,
  getTaskRunComment,
  sortTaskRunsNewestFirst,
} from "@/react/lib/taskRun";
import { useAppStore } from "@/react/stores/app";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { getDateForPbTimestampProtoEs } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Task, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import {
  databaseForTask,
  extractDatabaseResourceName,
  extractInstanceResourceName,
  extractTaskUID,
  formatAbsoluteDateTime,
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
  const projectName = `${projectNamePrefix}${page.projectId}`;
  // subscribe to re-render on project cache change
  const projectsByName = useAppStore((s) => s.projectsByName);
  void projectsByName;
  const project = useProjectByName(projectName);

  const taskByUID = useMemo(() => {
    const map = new Map<string, Task>();
    for (const stage of page.rollout?.stages ?? []) {
      for (const task of stage.tasks) {
        map.set(extractTaskUID(task.name), task);
      }
    }
    return map;
  }, [page.rollout?.stages]);

  const sortedTaskRuns = useMemo(
    () => sortTaskRunsNewestFirst(taskRuns),
    [taskRuns]
  );

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
      void useAppStore.getState().batchGetOrFetchDatabases(targets);
    }
  }, [showDatabaseColumn, taskByUID, taskRuns]);

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
              const duration = executionDurationOfTaskRun(taskRun);
              return (
                <TableRow key={taskRun.name}>
                  <TableCell className="px-2">
                    <TaskRunStatusIcon status={taskRun.status} />
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
                      {duration ? humanizeDurationV1(duration) : "-"}
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

function IssueDetailTaskRunComment({ taskRun }: { taskRun: TaskRun }) {
  const { t } = useTranslation();
  const comment = getTaskRunComment(taskRun, t);

  return (
    <div className="flex flex-col gap-y-0.5 xl:flex-row xl:items-center xl:gap-x-1">
      <div className="min-w-0 flex-1">
        <EllipsisText className="line-clamp-1" text={comment} />
      </div>
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
  if (format === "absolute") {
    return (
      <span className="text-sm text-control">
        {formatAbsoluteDateTime(parsedDate.getTime())}
      </span>
    );
  }

  return (
    <HumanizeTs
      ts={parsedDate.getTime() / 1000}
      className="text-sm text-control"
    />
  );
}

function IssueDetailTaskRunDatabaseCell({ database }: { database: Database }) {
  const { t } = useTranslation();
  const environmentList = useAppStore((s) => s.environmentList);
  const environmentName =
    database.effectiveEnvironment ??
    database.instanceResource?.environment ??
    "";
  const environment = useMemo(
    () => useAppStore.getState().getEnvironmentByName(environmentName),
    [environmentList, environmentName]
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
