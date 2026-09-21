import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { DatabaseTargetDisplay } from "@/components/DatabaseTargetDisplay";
import { HumanizeTs } from "@/components/HumanizeTs";
import { TaskRunStatusIcon } from "@/components/TaskRunStatusIcon";
import { TIMESTAMP_COLUMN } from "@/components/timestampColumn";
import { EllipsisText } from "@/components/ui/ellipsis-text";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useProjectByName } from "@/hooks/useProjectByName";
import {
  executionDurationOfTaskRun,
  getTaskRunComment,
  sortTaskRunsNewestFirst,
} from "@/lib/taskRun";
import { useAppStore } from "@/stores/app";
import { projectNamePrefix } from "@/stores/modules/v1/common";
import { getTimeForPbTimestampProtoEs } from "@/types";
import type { Task, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { databaseForTask, extractTaskUID, humanizeDurationV1 } from "@/utils";
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
              <TableHead className="sticky top-0 z-10 w-9 bg-control-bg/50 px-2" />
              {showDatabaseColumn && (
                <TableHead className="sticky top-0 z-10 w-64 bg-control-bg/50">
                  {t("common.database")}
                </TableHead>
              )}
              <TableHead className="sticky top-0 z-10 bg-control-bg/50">
                {t("common.detail")}
              </TableHead>
              {/* Sized to the form, since the detail column is the one
                  that fills; a narrower column breaks a date in two. */}
              <TableHead
                className="sticky top-0 z-10 bg-control-bg/50"
                style={{ width: TIMESTAMP_COLUMN.compact.width }}
              >
                {t("task.created")}
              </TableHead>
              <TableHead
                className="sticky top-0 z-10 bg-control-bg/50"
                style={{ width: TIMESTAMP_COLUMN.compact.width }}
              >
                {t("task.started")}
              </TableHead>
              <TableHead className="sticky top-0 z-10 w-28 bg-control-bg/50 pr-6 whitespace-nowrap text-sm">
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
                        <DatabaseTargetDisplay
                          database={database}
                          showEnvironment
                        />
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
    <div className="flex flex-col gap-y-1 xl:flex-row xl:items-center xl:gap-x-1">
      <div className="min-w-0 flex-1">
        <EllipsisText className="line-clamp-1" text={comment} />
      </div>
    </div>
  );
}

function IssueDetailTaskRunDateCell({ date }: { date?: Timestamp }) {
  if (!date) {
    return <span className="text-control-light">-</span>;
  }
  return (
    <HumanizeTs
      mode="compact"
      tsMs={getTimeForPbTimestampProtoEs(date)}
      className="block truncate text-sm text-control"
    />
  );
}
