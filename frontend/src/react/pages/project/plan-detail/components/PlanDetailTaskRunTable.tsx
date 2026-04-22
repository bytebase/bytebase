import { create } from "@bufbuild/protobuf";
import { DurationSchema } from "@bufbuild/protobuf/wkt";
import { Check, Minus } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { EllipsisText } from "@/react/components/ui/ellipsis-text";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useDatabaseV1Store } from "@/store";
import {
  getDateForPbTimestampProtoEs,
  getTimeForPbTimestampProtoEs,
} from "@/types";
import type { TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { TaskRun_Status } from "@/types/proto-es/v1/rollout_service_pb";
import {
  formatAbsoluteDateTime,
  humanizeDate,
  humanizeDurationV1,
} from "@/utils";
import { PlanDetailTaskRunDetail } from "./PlanDetailTaskRunDetail";

export function PlanDetailTaskRunTable({
  databaseName,
  taskRuns,
  selectedTaskRunName,
  onSelectTaskRun,
}: {
  databaseName?: string;
  taskRuns: TaskRun[];
  selectedTaskRunName?: string;
  onSelectTaskRun?: (taskRunName: string) => void;
}) {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();
  const [detailTaskRun, setDetailTaskRun] = useState<TaskRun | undefined>();
  const sortedTaskRuns = useMemo(() => {
    return [...taskRuns].sort((left, right) => {
      const leftTime = left.createTime ? Number(left.createTime.seconds) : 0;
      const rightTime = right.createTime ? Number(right.createTime.seconds) : 0;
      return rightTime - leftTime;
    });
  }, [taskRuns]);
  const database = databaseName
    ? databaseStore.getDatabaseByName(databaseName)
    : undefined;

  return (
    <>
      <div className="overflow-auto rounded-sm border">
        <Table className="table-fixed">
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <TableHead className="sticky top-0 z-10 w-9 bg-gray-50 px-2" />
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
              {!onSelectTaskRun && (
                <TableHead className="sticky top-0 z-10 w-24 bg-gray-50 pr-4" />
              )}
            </TableRow>
          </TableHeader>
          <TableBody>
            {sortedTaskRuns.map((taskRun) => (
              <TableRow
                key={taskRun.name}
                className={[
                  onSelectTaskRun ? "cursor-pointer" : "",
                  selectedTaskRunName === taskRun.name
                    ? "bg-accent/5 hover:bg-accent/5"
                    : undefined,
                ]
                  .filter(Boolean)
                  .join(" ")}
                onClick={() => onSelectTaskRun?.(taskRun.name)}
              >
                <TableCell className="px-2">
                  <TaskRunStatusIcon status={taskRun.status} />
                </TableCell>
                <TableCell className="min-w-0 pr-2">
                  <TaskRunComment taskRun={taskRun} />
                </TableCell>
                <TableCell>
                  <TaskRunDateCell date={taskRun.createTime} />
                </TableCell>
                <TableCell>
                  <TaskRunDateCell date={taskRun.startTime} />
                </TableCell>
                <TableCell className="pr-6 whitespace-nowrap">
                  <span className="whitespace-nowrap text-sm text-control">
                    {executionDurationOfTaskRun(taskRun)
                      ? humanizeDurationV1(executionDurationOfTaskRun(taskRun))
                      : "-"}
                  </span>
                </TableCell>
                {!onSelectTaskRun && (
                  <TableCell className="pr-4">
                    {shouldShowDetailButton(taskRun) ? (
                      <Button
                        size="xs"
                        variant="outline"
                        onClick={() => setDetailTaskRun(taskRun)}
                      >
                        {t("common.detail")}
                      </Button>
                    ) : null}
                  </TableCell>
                )}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <Sheet
        onOpenChange={(open) => {
          if (!open) {
            setDetailTaskRun(undefined);
          }
        }}
        open={Boolean(detailTaskRun)}
      >
        <SheetContent width="wide">
          <SheetHeader>
            <SheetTitle>{t("common.detail")}</SheetTitle>
          </SheetHeader>
          <SheetBody className="gap-y-4">
            {detailTaskRun && (
              <PlanDetailTaskRunDetail
                databaseEngine={database?.instanceResource?.engine}
                taskRun={detailTaskRun}
              />
            )}
          </SheetBody>
        </SheetContent>
      </Sheet>
    </>
  );
}

function TaskRunStatusIcon({ status }: { status: TaskRun_Status }) {
  const { t } = useTranslation();
  const classes = (() => {
    switch (status) {
      case TaskRun_Status.PENDING:
        return "border-2 border-info bg-white text-info";
      case TaskRun_Status.RUNNING:
        return "border-2 border-info bg-white text-info";
      case TaskRun_Status.DONE:
        return "bg-success text-white";
      case TaskRun_Status.FAILED:
        return "bg-error text-white";
      case TaskRun_Status.CANCELED:
        return "border-2 border-gray-400 bg-white text-gray-400";
      default:
        return "";
    }
  })();

  return (
    <Tooltip content={taskRunStatusLabel(t, status)}>
      <div
        className={[
          "relative flex h-5 w-5 shrink-0 items-center justify-center rounded-full select-none",
          classes,
        ].join(" ")}
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

function TaskRunComment({ taskRun }: { taskRun: TaskRun }) {
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

function TaskRunDateCell({
  date,
}: {
  date?: Parameters<typeof getDateForPbTimestampProtoEs>[0];
}) {
  if (!date) {
    return <span className="text-control-light">-</span>;
  }

  const parsedDate = getDateForPbTimestampProtoEs(date);
  if (!parsedDate) {
    return <span className="text-control-light">-</span>;
  }

  return (
    <Tooltip content={formatAbsoluteDateTime(parsedDate.getTime())}>
      <span className="text-sm text-control">{humanizeDate(parsedDate)}</span>
    </Tooltip>
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

function shouldShowDetailButton(taskRun: TaskRun) {
  return [
    TaskRun_Status.RUNNING,
    TaskRun_Status.DONE,
    TaskRun_Status.FAILED,
    TaskRun_Status.CANCELED,
  ].includes(taskRun.status);
}
