import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { useCallback, useEffect, useMemo, useRef } from "react";
import { rolloutServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { useLatestRef } from "@/react/hooks/useLatestRef";
import { useLivePoll } from "@/react/hooks/useLivePoll";
import { useSeededState } from "@/react/hooks/useSeededState";
import { sameMessageList } from "@/react/lib/protoIdentity";
import { useAppStore } from "@/react/stores/app";
import { unknownRollout } from "@/types";
import {
  GetTaskRunLogRequestSchema,
  type Task,
  TaskRun_Status,
  type TaskRunLogEntry,
  TaskRunLogEntrySchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import {
  extractRolloutNameFromTaskRunName,
  extractTaskNameFromTaskRunName,
  isReleaseBasedTask,
  releaseNameOfTaskV1,
  sheetNameOfTaskV1,
} from "@/utils";
import { isSheetContentComplete } from "@/utils/v1/sheet";

export type FetchStatus = "idle" | "loading" | "success" | "partial" | "error";
export type SheetSource = "none" | "sheet" | "release";

export interface FetchState {
  status: FetchStatus;
  error?: string;
}

export interface SheetFetchState extends FetchState {
  source: SheetSource;
  failedReleaseVersions?: string[];
}

export interface UseTaskRunLogDataResult {
  entries: TaskRunLogEntry[];
  sheet: Sheet | undefined;
  sheetsMap: Map<string, Sheet>;
  metadataFetch: FetchState;
  logFetch: FetchState;
  sheetFetch: SheetFetchState;
}

interface ReleaseSheetFetchResult {
  version: string;
  sheet?: Sheet;
}

const TASK_RESOLUTION_ERROR = "Task cannot be resolved from rollout metadata";

// The viewer remounts on every stage switch (and on the card's refresh key
// bump), so all three data slices below seed synchronously from caches during
// render — a remount paints fully formed instead of flashing empty states
// while effects and requests settle (same pattern as useSheetStatement /
// BYT-9763).

type LogCacheEntry = {
  entries: TaskRunLogEntry[];
  // The run's status when the entries were fetched. Once a run is terminal
  // its log is immutable, so a matching cached copy needs no revalidation.
  status?: TaskRun_Status;
};
const taskRunLogEntriesCache = new Map<string, LogCacheEntry>();
// Keep the session-lifetime cache bounded; Map iteration order gives FIFO.
const LOG_CACHE_MAX_ENTRIES = 100;
const cacheLogEntries = (taskRunName: string, entry: LogCacheEntry) => {
  taskRunLogEntriesCache.delete(taskRunName);
  taskRunLogEntriesCache.set(taskRunName, entry);
  if (taskRunLogEntriesCache.size > LOG_CACHE_MAX_ENTRIES) {
    const oldest = taskRunLogEntriesCache.keys().next().value;
    if (oldest !== undefined) {
      taskRunLogEntriesCache.delete(oldest);
    }
  }
};

const TERMINAL_TASK_RUN_STATUSES: ReadonlySet<TaskRun_Status> = new Set([
  TaskRun_Status.DONE,
  TaskRun_Status.FAILED,
  TaskRun_Status.CANCELED,
]);

const isLogCacheFresh = (
  cached: LogCacheEntry | undefined,
  taskRunStatus?: TaskRun_Status
): boolean =>
  !!cached &&
  taskRunStatus !== undefined &&
  TERMINAL_TASK_RUN_STATUSES.has(taskRunStatus) &&
  cached.status === taskRunStatus;

// A running task appends log entries as it executes; poll on this cadence so
// the log grows live instead of freezing until the run's status flips.
const LIVE_LOG_POLL_INTERVAL_MS = 5000;

type LogSlice = { entries: TaskRunLogEntry[]; state: FetchState };

const seedLogSlice = (taskRunName?: string): LogSlice => {
  if (!taskRunName) {
    return { entries: [], state: { status: "idle" } };
  }
  const cached = taskRunLogEntriesCache.get(taskRunName);
  if (cached) {
    return { entries: cached.entries, state: { status: "success" } };
  }
  return { entries: [], state: { status: "loading" } };
};

// A cached sheet satisfies a raw consumer only when it holds the full content;
// the store's `fetchSheet(name, raw=true)` applies the same rule, so the async
// path just delegates to it and this sync probe backs the render-time seed.
const getCachedCompleteSheet = (name: string): Sheet | undefined => {
  const cached = useAppStore.getState().getSheetByName(name);
  return cached && isSheetContentComplete(cached) ? cached : undefined;
};

const getTaskFromRollout = (
  taskRunName: string | undefined,
  rollout: { stages: Array<{ tasks: Task[] }> } | undefined
): Task | undefined => {
  if (!taskRunName || !rollout) return undefined;

  const taskName = extractTaskNameFromTaskRunName(taskRunName);
  if (!taskName) return undefined;

  for (const stage of rollout.stages) {
    const task = stage.tasks.find((candidate) => candidate.name === taskName);
    if (task) {
      return task;
    }
  }

  return undefined;
};

const getErrorMessage = (error: unknown): string => {
  if (error instanceof Error) return error.message;
  return String(error);
};

export const buildSheetFetchStateForMissingTask = (
  taskRunName: string | undefined,
  metadataFetchState: FetchState
): SheetFetchState => {
  if (!taskRunName) {
    return { status: "idle", source: "none" };
  }
  if (
    metadataFetchState.status === "idle" ||
    metadataFetchState.status === "loading"
  ) {
    return { status: "loading", source: "none" };
  }
  if (metadataFetchState.status === "error") {
    return {
      status: "error",
      source: "none",
      error: metadataFetchState.error ?? TASK_RESOLUTION_ERROR,
    };
  }
  return {
    status: "error",
    source: "none",
    error: TASK_RESOLUTION_ERROR,
  };
};

export const getUnresolvedTaskMetadataStateKey = (
  hasResolvedTask: boolean,
  metadataFetchState: FetchState
): string => {
  if (hasResolvedTask) {
    return "resolved";
  }
  return `${metadataFetchState.status}:${metadataFetchState.error ?? ""}`;
};

export const buildReleaseSheetFetchResult = (
  fileSheets: ReleaseSheetFetchResult[]
): { sheetsMap: Map<string, Sheet>; state: SheetFetchState } => {
  const sheetsMap = new Map<string, Sheet>();
  const failedReleaseVersions: string[] = [];

  for (const item of fileSheets) {
    if (item.sheet) {
      sheetsMap.set(item.version, item.sheet);
    } else {
      failedReleaseVersions.push(item.version);
    }
  }

  if (failedReleaseVersions.length === 0) {
    return {
      sheetsMap,
      state: { status: "success", source: "release" },
    };
  }
  if (sheetsMap.size === 0) {
    return {
      sheetsMap,
      state: {
        status: "error",
        source: "release",
        error: "Failed to fetch all release sheets",
        failedReleaseVersions,
      },
    };
  }
  return {
    sheetsMap,
    state: {
      status: "partial",
      source: "release",
      error: "Failed to fetch some release sheets",
      failedReleaseVersions,
    },
  };
};

const seedMetadataState = (
  rolloutName: string,
  taskRunName?: string
): FetchState => {
  if (!rolloutName || !taskRunName) {
    return { status: "idle" };
  }
  // The cached rollout usually already carries the task (the surrounding page
  // loaded it); the effect only hits the network when it can't be resolved.
  const cachedRollout = useAppStore.getState().rolloutsByName[rolloutName];
  if (getTaskFromRollout(taskRunName, cachedRollout)) {
    return { status: "success" };
  }
  return { status: "loading" };
};

const EMPTY_SHEETS_MAP = new Map<string, Sheet>();

type SheetSlice = {
  sheet: Sheet | undefined;
  sheetsMap: Map<string, Sheet>;
  state: SheetFetchState;
};

const sheetlessSlice = (state: SheetFetchState): SheetSlice => ({
  sheet: undefined,
  sheetsMap: EMPTY_SHEETS_MAP,
  state,
});

const seedSheetSlice = (
  task: Task | undefined,
  taskRunName: string | undefined,
  metadataFetchState: FetchState
): SheetSlice => {
  if (!task) {
    return sheetlessSlice(
      buildSheetFetchStateForMissingTask(taskRunName, metadataFetchState)
    );
  }
  if (isReleaseBasedTask(task)) {
    return sheetlessSlice(
      releaseNameOfTaskV1(task)
        ? { status: "loading", source: "release" }
        : { status: "success", source: "release" }
    );
  }
  const sheetName = sheetNameOfTaskV1(task);
  if (!sheetName) {
    return sheetlessSlice({ status: "success", source: "sheet" });
  }
  const cached = getCachedCompleteSheet(sheetName);
  if (cached) {
    return {
      sheet: cached,
      sheetsMap: EMPTY_SHEETS_MAP,
      state: { status: "success", source: "sheet" },
    };
  }
  return sheetlessSlice({ status: "loading", source: "sheet" });
};

export const useTaskRunLogData = (
  taskRunName?: string,
  // When provided, a terminal status lets a cached log skip revalidation
  // entirely — a finished run's log never changes.
  taskRunStatus?: TaskRun_Status
): UseTaskRunLogDataResult => {
  const fetchRelease = useAppStore((state) => state.fetchRelease);

  const metadataFetchVersion = useRef(0);
  const logFetchSeq = useRef(0);
  const sheetFetchVersion = useRef(0);

  const rolloutName = useMemo(() => {
    if (!taskRunName) return "";
    return extractRolloutNameFromTaskRunName(taskRunName);
  }, [taskRunName]);

  // ---- Task metadata (rollout) ----

  // rolloutName is derived from taskRunName, so the run name alone keys both.
  const [metadataFetchState, setMetadataFetchState] =
    useSeededState<FetchState>(taskRunName ?? "", () =>
      seedMetadataState(rolloutName, taskRunName)
    );

  useEffect(() => {
    const version = ++metadataFetchVersion.current;
    if (!rolloutName || !taskRunName) {
      return;
    }
    if (seedMetadataState(rolloutName, taskRunName).status === "success") {
      // The cached rollout resolves the task; seeded as success during render.
      return;
    }

    void useAppStore
      .getState()
      .fetchRolloutByName(rolloutName, true)
      .then((fetchedRollout) => {
        if (version !== metadataFetchVersion.current) return;
        const resolvedTask = getTaskFromRollout(taskRunName, fetchedRollout);
        if (!resolvedTask) {
          setMetadataFetchState({
            status: "error",
            error: TASK_RESOLUTION_ERROR,
          });
          return;
        }
        setMetadataFetchState({ status: "success" });
      })
      .catch((error: unknown) => {
        if (version !== metadataFetchVersion.current) return;
        setMetadataFetchState({
          status: "error",
          error: getErrorMessage(error),
        });
      });
  }, [rolloutName, taskRunName]);

  // Subscribe to the cached entry directly (stable ref) and derive the unknown
  // fallback outside the selector — a selector returning `unknownRollout()`
  // would yield a fresh object each call and loop forever.
  const cachedRollout = useAppStore((state) =>
    rolloutName ? state.rolloutsByName[rolloutName] : undefined
  );
  const rollout = useMemo(
    () => (rolloutName ? (cachedRollout ?? unknownRollout()) : undefined),
    [rolloutName, cachedRollout]
  );

  const task = useMemo(
    () => getTaskFromRollout(taskRunName, rollout),
    [rollout, taskRunName]
  );
  const sheetFetchTaskKey = useMemo(() => {
    if (!task) return "";
    if (isReleaseBasedTask(task)) {
      return `release:${task.name}:${releaseNameOfTaskV1(task) ?? ""}`;
    }
    return `sheet:${task.name}:${sheetNameOfTaskV1(task) ?? ""}`;
  }, [task]);
  const unresolvedTaskMetadataStateKey = useMemo(
    () => getUnresolvedTaskMetadataStateKey(Boolean(task), metadataFetchState),
    [metadataFetchState, task]
  );

  // ---- Log entries ----

  const [logSlice, setLogSlice] = useSeededState<LogSlice>(
    taskRunName ?? "",
    () => seedLogSlice(taskRunName)
  );

  // A monotonic per-fetch sequence: initial revalidation and every live-poll
  // tick take a number, and a response whose number is no longer the latest is
  // dropped entirely — no cache write, no state write. This keeps overlapping
  // fetches (a slow one for a RUNNING run resolving after a newer one) from
  // rewinding the log or clobbering the terminal-status cache.
  const fetchLog = useCallback(() => {
    if (!taskRunName) {
      return;
    }
    const seq = ++logFetchSeq.current;
    void rolloutServiceClientConnect
      .getTaskRunLog(
        create(GetTaskRunLogRequestSchema, { parent: taskRunName }),
        { contextValues: createContextValues().set(silentContextKey, true) }
      )
      .then((response) => {
        if (seq !== logFetchSeq.current) return;
        cacheLogEntries(taskRunName, {
          entries: response.entries,
          status: taskRunStatus,
        });
        // Identity-stable across live-poll ticks: unchanged entries keep the
        // previous slice so the viewer doesn't re-render for nothing.
        setLogSlice((prev) =>
          prev.state.status === "success" &&
          sameMessageList(TaskRunLogEntrySchema, prev.entries, response.entries)
            ? prev
            : { entries: response.entries, state: { status: "success" } }
        );
      })
      .catch((error: unknown) => {
        if (seq !== logFetchSeq.current) return;
        // Keep showing stale cached entries over an empty error state.
        setLogSlice((prev) => ({
          entries: prev.entries,
          state: { status: "error", error: getErrorMessage(error) },
        }));
      });
  }, [taskRunName, taskRunStatus, setLogSlice]);

  // Cached entries were painted during render; revalidate on mount / status
  // change only when the cached copy could still change (run not finished).
  useEffect(() => {
    if (
      taskRunName &&
      !isLogCacheFresh(taskRunLogEntriesCache.get(taskRunName), taskRunStatus)
    ) {
      fetchLog();
    }
  }, [taskRunName, taskRunStatus, fetchLog]);

  // A running task appends log lines as it executes; poll while it runs.
  useLivePoll(
    Boolean(taskRunName) && taskRunStatus === TaskRun_Status.RUNNING,
    LIVE_LOG_POLL_INTERVAL_MS,
    fetchLog
  );

  // Invalidate any in-flight log request on unmount. A status flip (RUNNING→DONE)
  // remounts the viewer under a new key, so a newer instance may already have
  // cached the terminal log; without this, a late RUNNING-era response from this
  // now-unmounted instance would still pass its own per-instance seq guard and
  // rewind that shared cache to an incomplete, stale snapshot. The seq guard is
  // per-instance, so only bumping it here closes the cross-instance window.
  useEffect(
    () => () => {
      logFetchSeq.current++;
    },
    []
  );

  // ---- Sheet(s) referenced by the log ----

  const sheetSeedKey = `${sheetFetchTaskKey}:${unresolvedTaskMetadataStateKey}:${taskRunName ?? ""}`;
  const [sheetSlice, setSheetSlice] = useSeededState<SheetSlice>(
    sheetSeedKey,
    () => seedSheetSlice(task, taskRunName, metadataFetchState)
  );

  // The effect below is keyed on derived stable strings and reads the latest
  // task through this ref, so a task identity change alone doesn't re-run it
  // (and refetch).
  const taskRef = useLatestRef(task);

  useEffect(() => {
    const version = ++sheetFetchVersion.current;
    const currentTask = taskRef.current;
    if (!currentTask) {
      // The missing-task state was seeded during render.
      return;
    }

    if (isReleaseBasedTask(currentTask)) {
      const releaseName = releaseNameOfTaskV1(currentTask);
      if (!releaseName) {
        return;
      }

      void fetchRelease(releaseName, true)
        .then(async (release) => {
          const fileSheets = await Promise.all(
            (release?.files ?? [])
              .filter((file) => file.sheet && file.version)
              .map(async (file) => ({
                version: file.version,
                // Resolves undefined on failure; the result builder counts
                // sheetless versions as failed.
                sheet: await useAppStore
                  .getState()
                  .fetchSheet(file.sheet, true),
              }))
          );
          if (version !== sheetFetchVersion.current) return;

          const releaseResult = buildReleaseSheetFetchResult(fileSheets);
          setSheetSlice({
            sheet: undefined,
            sheetsMap: releaseResult.sheetsMap,
            state: releaseResult.state,
          });
        })
        .catch((error: unknown) => {
          if (version !== sheetFetchVersion.current) return;
          setSheetSlice(
            sheetlessSlice({
              status: "error",
              source: "release",
              error: getErrorMessage(error),
            })
          );
        });

      return;
    }

    const sheetName = sheetNameOfTaskV1(currentTask);
    if (!sheetName || getCachedCompleteSheet(sheetName)) {
      // Seeded as success (empty or from cache) during render.
      return;
    }

    void useAppStore
      .getState()
      .fetchSheet(sheetName, true)
      .then((fetchedSheet) => {
        if (version !== sheetFetchVersion.current) return;
        if (!fetchedSheet) {
          setSheetSlice(
            sheetlessSlice({
              status: "error",
              source: "sheet",
              error: `Failed to fetch sheet ${sheetName}`,
            })
          );
          return;
        }
        setSheetSlice({
          sheet: fetchedSheet,
          sheetsMap: EMPTY_SHEETS_MAP,
          state: { status: "success", source: "sheet" },
        });
      })
      .catch((error: unknown) => {
        // Symmetric with the release branch: without this a rejecting fetch
        // would leave the slice stuck on "loading" and float an unhandled
        // rejection.
        if (version !== sheetFetchVersion.current) return;
        setSheetSlice(
          sheetlessSlice({
            status: "error",
            source: "sheet",
            error: getErrorMessage(error),
          })
        );
      });
  }, [
    fetchRelease,
    taskRunName,
    sheetFetchTaskKey,
    unresolvedTaskMetadataStateKey,
  ]);

  return {
    entries: logSlice.entries,
    sheet: sheetSlice.sheet,
    sheetsMap: sheetSlice.sheetsMap,
    metadataFetch: metadataFetchState,
    logFetch: logSlice.state,
    sheetFetch: sheetSlice.state,
  };
};
