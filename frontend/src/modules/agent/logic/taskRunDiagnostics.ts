import {
  completedOneTimeError,
  groupEntriesByReleaseFile,
  parseReleaseFileError,
} from "@/components/task-run-log/model";
import type { FetchStatus } from "@/components/task-run-log/useTaskRunLogData";
import {
  type Task,
  type TaskRun,
  TaskRun_Status,
  type TaskRunLogEntry,
  TaskRunLogEntry_Type,
} from "@/types/proto-es/v1/rollout_service_pb";
import { normalizeAgentDiagnosticCapability } from "./diagnosticCapability";
import type { AgentDiagnosticContext } from "./diagnosticHelp";
import type { AgentResourceReference } from "./types";

export interface AgentTaskRunDiagnosticInput {
  resource: AgentResourceReference;
  /** For an Issue, this is the Plan linked by the Issue record. */
  planName: string;
  run: Pick<TaskRun, "name" | "status" | "detail">;
  provenance: "latest" | "explicit-history";
  task?: Task;
  log?: {
    entries: readonly TaskRunLogEntry[];
    fetchStatus: FetchStatus;
    snapshot?: { taskRunName: string; status?: TaskRun_Status };
  };
}

const RUN_NAME =
  /^(?<task>(?<plan>(?<project>projects\/[^/]+)\/plans\/[1-9]\d*)\/rollout\/stages\/(?<stage>[a-z](?:[a-z0-9-]*[a-z0-9])?|[1-9]\d*)\/tasks\/[1-9]\d*)\/taskRuns\/[1-9]\d*$/;

function findRecordedError(
  entries: readonly TaskRunLogEntry[],
  run: AgentTaskRunDiagnosticInput["run"]
) {
  const errors = groupEntriesByReleaseFile([...entries]).flatMap(
    ({ file, entries }) =>
      entries.flatMap((entry) => {
        const command =
          run.status === TaskRun_Status.FAILED &&
          entry.type === TaskRunLogEntry_Type.COMMAND_EXECUTE
            ? entry.commandExecute
            : undefined;
        const error = command?.response?.error || completedOneTimeError(entry);
        return error?.trim()
          ? [{ entry, error, file, range: command?.range }]
          : [];
      })
  );
  if (run.status === TaskRun_Status.DONE) return errors[0];

  // Do not infer that the last log error caused the FAILED result. In
  // particular, errors from earlier driver attempts may have been superseded.
  const detail = run.detail.trim();
  const releaseError = parseReleaseFileError(detail);
  const candidates = errors.filter(({ error, file, entry }) => {
    if (releaseError) {
      return (
        file?.filePath === releaseError.path &&
        file.version === releaseError.version &&
        error.trim() === releaseError.message.trim()
      );
    }
    if (detail) return error.trim() === detail;
    return entry.type !== TaskRunLogEntry_Type.COMMAND_EXECUTE;
  });
  return candidates.length === 1 ? candidates[0] : undefined;
}

export function getTaskRunDiagnostic({
  resource,
  planName,
  run,
  provenance,
  task,
  log,
}: AgentTaskRunDiagnosticInput): AgentDiagnosticContext | undefined {
  if (
    run.status !== TaskRun_Status.FAILED &&
    run.status !== TaskRun_Status.DONE
  )
    return undefined;
  const match = RUN_NAME.exec(run.name);
  if (
    !match ||
    match[0] !== run.name ||
    match.groups?.plan !== planName ||
    (task && task.name !== match.groups.task)
  )
    return undefined;

  const capability = normalizeAgentDiagnosticCapability({
    kind: "diagnostic",
    resource,
  });
  const bound = capability?.resource;
  if (!bound || bound.project !== match.groups.project) return undefined;
  if (bound.kind === "plan") {
    if (bound.name !== planName) return undefined;
  } else if (
    bound.kind !== "issue" ||
    task?.payload.case !== "databaseCreate"
  ) {
    return undefined;
  }

  const finalLog =
    log?.fetchStatus === "success" &&
    log.snapshot?.taskRunName === run.name &&
    log.snapshot.status === run.status;
  // Empty legacy IDs are a stream too; the viewer's fallback display label
  // would merge that stream with a replica literally named "unknown".
  const unambiguous =
    finalLog &&
    new Set(log.entries.map(({ replicaId }) => replicaId)).size <= 1;
  const recorded = unambiguous
    ? findRecordedError(log.entries, run)
    : undefined;
  if (run.status === TaskRun_Status.DONE && !recorded) return undefined;

  const status = TaskRun_Status[run.status];
  const facts: { label: string; value: string }[] = [
    { label: "Status", value: status },
    { label: "Provenance", value: provenance },
    {
      label: "Logs",
      value:
        !finalLog || log.entries.length === 0
          ? "Unavailable"
          : unambiguous
            ? "Final snapshot available"
            : "Final snapshot available; replica attribution ambiguous",
    },
  ];
  if (task?.target)
    facts.push({ label: "Recorded target", value: task.target });
  if (recorded) {
    facts.push({
      label: "Recorded error type",
      value: TaskRunLogEntry_Type[recorded.entry.type],
    });
    if (recorded.entry.replicaId)
      facts.push({
        label: "Recorded replica",
        value: recorded.entry.replicaId,
      });
    if (recorded.file?.filePath)
      facts.push({ label: "Recorded file", value: recorded.file.filePath });
    if (recorded.file?.version)
      facts.push({ label: "Recorded version", value: recorded.file.version });
    if (
      recorded.range &&
      recorded.range.start >= 0 &&
      recorded.range.end > recorded.range.start
    )
      facts.push({
        label: "Recorded byte range",
        value: `[${recorded.range.start}, ${recorded.range.end})`,
      });
  }
  return {
    kind: "task-run",
    sourceId: run.name,
    resource: bound,
    // Optional details may use up the prompt budget before facts are added.
    summary: `Recorded Task Run status: ${status}; selection: ${provenance}.${
      run.status === TaskRun_Status.DONE ? " A step logged an error." : ""
    }`,
    detail:
      run.status === TaskRun_Status.DONE
        ? recorded?.error
        : run.detail.trim() || recorded?.error,
    facts,
  };
}
