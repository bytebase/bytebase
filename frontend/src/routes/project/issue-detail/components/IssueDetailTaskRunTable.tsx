import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { DatabaseTargetDisplay } from "@/components/DatabaseTargetDisplay";
import { HumanizeTs } from "@/components/HumanizeTs";
import { TaskRunStatusIcon } from "@/components/TaskRunStatusIcon";
import { TIMESTAMP_COLUMN_WIDTH } from "@/components/timestampColumn";
import { EllipsisText } from "@/components/ui/ellipsis-text";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useMediaQuery } from "@/hooks/useMediaQuery";
import { useProjectByName } from "@/hooks/useProjectByName";
import {
  executionDurationOfTaskRun,
  getTaskRunComment,
  sortTaskRunsNewestFirst,
} from "@/lib/taskRun";
import {
  AgentDiagnosticHelpButton,
  useAgentDiagnosticHelpAvailable,
} from "@/modules/agent/components/AgentDiagnosticHelpButton";
import { getTaskRunDiagnostic } from "@/modules/agent/logic/taskRunDiagnostics";
import { useAppStore } from "@/stores/app";
import { projectNamePrefix } from "@/stores/modules/v1/common";
import { getTimeForPbTimestampProtoEs } from "@/types";
import {
  type Task,
  type TaskRun,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  databaseForTask,
  extractTaskNameFromTaskRunName,
  extractTaskUID,
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
  const agentHelpAvailable = useAgentDiagnosticHelpAvailable();
  const compactHelp = useMediaQuery("(max-width: 767px)");
  const projectName = `${projectNamePrefix}${page.projectId}`;
  // subscribe to re-render on project cache change
  const projectsByName = useAppStore((s) => s.projectsByName);
  void projectsByName;
  const project = useProjectByName(projectName);

  const { taskByUID, taskByName } = useMemo(() => {
    const taskByUID = new Map<string, Task>();
    const taskByName = new Map<string, Task>();
    for (const stage of page.rollout?.stages ?? []) {
      for (const task of stage.tasks) {
        taskByUID.set(extractTaskUID(task.name), task);
        taskByName.set(task.name, task);
      }
    }
    return { taskByUID, taskByName };
  }, [page.rollout?.stages]);

  const sortedTaskRuns = useMemo(
    () => sortTaskRunsNewestFirst(taskRuns),
    [taskRuns]
  );
  const latestRunByTask = useMemo(() => {
    const latest = new Map<string, TaskRun>();
    for (const run of sortedTaskRuns) {
      const taskName = extractTaskNameFromTaskRunName(run.name);
      if (!latest.has(taskName) && /\/taskRuns\/[1-9]\d*$/.test(run.name)) {
        latest.set(taskName, run);
      }
    }
    return latest;
  }, [sortedTaskRuns]);

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

  const getDiagnosticForTaskRun = (run: TaskRun) => {
    const planName = page.plan?.name;
    const issue = page.issue;
    if (
      run.status !== TaskRun_Status.FAILED ||
      page.isCreating ||
      !planName ||
      issue?.plan !== planName
    ) {
      return undefined;
    }

    const taskName = extractTaskNameFromTaskRunName(run.name);
    return getTaskRunDiagnostic({
      resource: {
        kind: "issue",
        name: issue.name,
        project: projectName,
        label: issue.name,
      },
      planName,
      run,
      task: taskByName.get(taskName),
      provenance:
        latestRunByTask.get(taskName) === run ? "latest" : "explicit-history",
    });
  };

  const rows = sortedTaskRuns.map((taskRun) => ({
    taskRun,
    diagnostic: agentHelpAvailable
      ? getDiagnosticForTaskRun(taskRun)
      : undefined,
  }));
  const hasDiagnostic = rows.some(({ diagnostic }) => diagnostic);
  // Keep 14rem for Detail after the fixed status (w-9), duration (w-28),
  // optional database (w-64), and timestamp columns take their widths.
  const diagnosticTableMinWidth = `calc(${2 * TIMESTAMP_COLUMN_WIDTH.compact}px + ${(9 + 28 + (showDatabaseColumn ? 64 : 0)) / 4 + 14}rem)`;

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
        <Table
          className="table-fixed"
          style={
            hasDiagnostic ? { minWidth: diagnosticTableMinWidth } : undefined
          }
        >
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
              {/* Sized to the form; the detail column is the one left to
                  take the rest. */}
              <TableHead
                className="sticky top-0 z-10 bg-control-bg/50"
                style={{ width: TIMESTAMP_COLUMN_WIDTH.compact }}
              >
                {t("task.created")}
              </TableHead>
              <TableHead
                className="sticky top-0 z-10 bg-control-bg/50"
                style={{ width: TIMESTAMP_COLUMN_WIDTH.compact }}
              >
                {t("task.started")}
              </TableHead>
              <TableHead className="sticky top-0 z-10 w-28 bg-control-bg/50 pr-6 whitespace-nowrap text-sm">
                {t("task.execution-time")}
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map(({ taskRun, diagnostic }) => {
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
                    <div className="flex min-w-0 items-center gap-2">
                      <div className="min-w-0 flex-1">
                        <IssueDetailTaskRunComment taskRun={taskRun} />
                      </div>
                      {diagnostic && (
                        <div
                          className="-my-1 flex shrink-0"
                          data-testid="task-run-diagnostic-help"
                          data-task-run-name={taskRun.name}
                        >
                          <AgentDiagnosticHelpButton
                            context={diagnostic}
                            iconOnly={compactHelp}
                            className="relative after:absolute after:-inset-2"
                          />
                        </div>
                      )}
                    </div>
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
      className="text-sm text-control"
      mode="compact"
      truncate
      tsMs={getTimeForPbTimestampProtoEs(date)}
    />
  );
}
