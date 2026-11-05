import type { Timestamp } from "@bufbuild/protobuf/wkt";
import type { TFunction } from "i18next";
import {
  firstLineOf,
  parseReleaseFileError,
} from "@/components/task-run-log/model";
import { StatementType } from "@/types/proto-es/v1/common_pb";
import {
  type PlanCheckRun,
  PlanCheckRun_Result_Type,
} from "@/types/proto-es/v1/plan_service_pb";
import { Release_Type } from "@/types/proto-es/v1/release_service_pb";
import {
  type Task,
  Task_Status,
  type TaskRun,
} from "@/types/proto-es/v1/rollout_service_pb";

export interface FoldedReleaseInfo {
  type: Release_Type;
  fileCount: number;
}

// From the plan check's statement summary report for the task's target.
export interface FoldedStatementSummary {
  statementTypes: StatementType[];
  affectedRows: bigint;
}

export type FoldedTone = "success" | "info" | "error" | "muted";

export interface FoldedDescription {
  // The failed version, shown ahead of the error.
  version?: string;
  text: string;
  tone: FoldedTone;
}

// The statement summary report for one target, when the plan check has one.
export const statementSummaryForTarget = (
  runs: readonly PlanCheckRun[],
  target: string
): FoldedStatementSummary | undefined => {
  for (const run of runs) {
    for (const result of run.results) {
      if (
        result.type === PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT &&
        result.target === target &&
        result.report.case === "sqlSummaryReport"
      ) {
        return {
          statementTypes: result.report.value.statementTypes,
          affectedRows: result.report.value.affectedRows,
        };
      }
    }
  }
  return undefined;
};

const DML_TYPES: ReadonlySet<StatementType> = new Set([
  StatementType.INSERT,
  StatementType.UPDATE,
  StatementType.DELETE,
  StatementType.MERGE,
]);

// DDL, DML or Mixed, from the statement types the plan check found.
export const statementKindOf = (
  types: readonly StatementType[]
): "DDL" | "DML" | "Mixed" | undefined => {
  const known = types.filter(
    (t) => t !== StatementType.STATEMENT_TYPE_UNSPECIFIED
  );
  if (known.length === 0) return undefined;
  const dml = known.some((t) => DML_TYPES.has(t));
  const ddl = known.some((t) => !DML_TYPES.has(t));
  if (dml && ddl) return "Mixed";
  return dml ? "DML" : "DDL";
};

// The clock time for today, the date and time for any other day, so an old
// cancel or a run stuck since yesterday does not read as today.
const clockOf = (
  timestamp: Timestamp | undefined,
  locale: string,
  nowMs: number
) => {
  if (!timestamp) return "";
  const date = new Date(Number(timestamp.seconds) * 1000);
  const time: Intl.DateTimeFormatOptions = {
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
  };
  return date.toDateString() === new Date(nowMs).toDateString()
    ? date.toLocaleTimeString(locale, time)
    : date.toLocaleString(locale, { month: "short", day: "numeric", ...time });
};

// DDL, DML or Mixed, as the reader's locale writes it.
export const statementKindLabel = (
  kind: "DDL" | "DML" | "Mixed",
  t: TFunction
): string => {
  switch (kind) {
    case "DDL":
      return t("task.folded.kind-ddl");
    case "DML":
      return t("task.folded.kind-dml");
    case "Mixed":
      return t("task.folded.kind-mixed");
  }
};

const join = (...parts: (string | undefined)[]) =>
  parts.filter((part) => part).join(" · ");

// The folded task card's one line (D3): what happened, or will happen, to
// the task's database, from the task, its latest run, the state stored at
// rollout creation and the plan check. It never reads the run's log.
export function foldedTaskDescription({
  task,
  latestTaskRun,
  release,
  statementSummary,
  duration,
  skipperName,
  t,
  locale,
  nowMs = Date.now(),
}: {
  task: Task;
  latestTaskRun?: TaskRun;
  release?: FoldedReleaseInfo;
  statementSummary?: FoldedStatementSummary;
  // The latest run's duration, as the card formats it.
  duration: string;
  // The skipper's display name; the email when the user has not loaded.
  skipperName?: string;
  nowMs?: number;
  t: TFunction;
  locale: string;
}): FoldedDescription {
  const isRelease =
    task.payload.case === "databaseUpdate" &&
    task.payload.value.source.case === "release";
  const versioned = isRelease && release?.type === Release_Type.VERSIONED;
  const state =
    task.payload.case === "databaseUpdate"
      ? task.payload.value.releaseState
      : undefined;
  const pendingCount = state?.pendingCount ?? 0;
  const appliedEarlier =
    state && t("task.folded.already-applied", { count: state.appliedCount });
  const statementKind = statementSummary
    ? statementKindOf(statementSummary.statementTypes)
    : undefined;
  const kind = statementKind ? statementKindLabel(statementKind, t) : undefined;
  const started = latestTaskRun
    ? t("task.folded.started", {
        time: clockOf(
          latestTaskRun.startTime ?? latestTaskRun.createTime,
          locale,
          nowMs
        ),
      })
    : undefined;
  const statusLabel = (status: Task_Status) =>
    t(`task.status.${Task_Status[status].toLowerCase().replaceAll("_", "-")}`);

  switch (task.status) {
    case Task_Status.FAILED: {
      const error = latestTaskRun?.detail ?? "";
      const fileError = isRelease ? parseReleaseFileError(error) : undefined;
      return {
        version: fileError?.version,
        text:
          firstLineOf(fileError?.message ?? error) || statusLabel(task.status),
        tone: "error",
      };
    }
    case Task_Status.SKIPPED:
      return {
        text: join(
          task.skipper
            ? t("task.folded.skipped-by", {
                user: skipperName || task.skipper.replace(/^users\//, ""),
              })
            : statusLabel(task.status),
          task.skippedReason
        ),
        tone: "muted",
      };
    case Task_Status.CANCELED:
      return {
        text: join(
          statusLabel(task.status),
          clockOf(latestTaskRun?.updateTime, locale, nowMs)
        ),
        tone: "muted",
      };
    case Task_Status.RUNNING:
      return {
        text: join(
          versioned && state && pendingCount > 0
            ? t("task.folded.applying", { count: pendingCount })
            : statusLabel(task.status),
          started
        ),
        tone: "info",
      };
    case Task_Status.DONE: {
      // A statement about the database: the stored list is only a preview.
      if (versioned && release) {
        return {
          text: join(
            t("task.folded.all-applied", { count: release.fileCount }),
            duration
          ),
          tone: "success",
        };
      }
      return {
        text: join(isRelease ? statusLabel(task.status) : kind, duration),
        tone: "success",
      };
    }
    default: {
      if (versioned && state) {
        return {
          text:
            pendingCount > 0
              ? join(
                  t("task.folded.to-apply", { count: pendingCount }),
                  appliedEarlier
                )
              : join(t("task.folded.nothing-to-apply"), appliedEarlier),
          tone: "muted",
        };
      }
      if (!isRelease && kind) {
        return {
          text: join(
            kind,
            statementSummary && statementSummary.affectedRows > 0n
              ? t("task.folded.about-rows", {
                  count: Number(statementSummary.affectedRows),
                  rows: Number(statementSummary.affectedRows).toLocaleString(
                    locale
                  ),
                })
              : undefined
          ),
          tone: "muted",
        };
      }
      return { text: statusLabel(task.status), tone: "muted" };
    }
  }
}
