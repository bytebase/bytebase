import { create } from "@bufbuild/protobuf";
import { useEffect, useMemo, useRef, useState } from "react";
import {
  rolloutServiceClientConnect,
  sheetServiceClientConnect,
} from "@/connect";
import { useVueState } from "@/react/hooks/useVueState";
import { useReleaseStore, useRolloutStore } from "@/store";
import {
  GetTaskRunLogRequestSchema,
  type Task,
  type TaskRunLogEntry,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import {
  extractRolloutNameFromTaskRunName,
  extractTaskNameFromTaskRunName,
  isReleaseBasedTask,
  releaseNameOfTaskV1,
  sheetNameOfTaskV1,
} from "@/utils";

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

export const useTaskRunLogData = (
  taskRunName?: string
): UseTaskRunLogDataResult => {
  const rolloutStore = useRolloutStore();
  const releaseStore = useReleaseStore();

  const [entries, setEntries] = useState<TaskRunLogEntry[]>([]);
  const [sheet, setSheet] = useState<Sheet | undefined>(undefined);
  const [sheetsMap, setSheetsMap] = useState<Map<string, Sheet>>(new Map());
  const [metadataFetchState, setMetadataFetchState] = useState<FetchState>({
    status: "idle",
  });
  const [logFetchState, setLogFetchState] = useState<FetchState>({
    status: "idle",
  });
  const [sheetFetchState, setSheetFetchState] = useState<SheetFetchState>({
    status: "idle",
    source: "none",
  });
  const metadataFetchVersion = useRef(0);
  const logFetchVersion = useRef(0);
  const sheetFetchVersion = useRef(0);

  const rolloutName = useMemo(() => {
    if (!taskRunName) return "";
    return extractRolloutNameFromTaskRunName(taskRunName);
  }, [taskRunName]);

  useEffect(() => {
    const version = ++metadataFetchVersion.current;
    if (!rolloutName || !taskRunName) {
      setMetadataFetchState({ status: "idle" });
      return;
    }

    setMetadataFetchState({ status: "loading" });
    void rolloutStore
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
  }, [rolloutName, rolloutStore, taskRunName]);

  const rollout = useVueState(() => {
    if (!rolloutName) return undefined;
    return rolloutStore.getRolloutByName(rolloutName);
  });

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

  useEffect(() => {
    const version = ++logFetchVersion.current;
    setEntries([]);

    if (!taskRunName) {
      setLogFetchState({ status: "idle" });
      return;
    }

    setLogFetchState({ status: "loading" });
    const request = create(GetTaskRunLogRequestSchema, {
      parent: taskRunName,
    });
    void rolloutServiceClientConnect
      .getTaskRunLog(request)
      .then((response) => {
        if (version !== logFetchVersion.current) return;
        setEntries(response.entries);
        setLogFetchState({ status: "success" });
      })
      .catch((error: unknown) => {
        if (version !== logFetchVersion.current) return;
        setEntries([]);
        setLogFetchState({
          status: "error",
          error: getErrorMessage(error),
        });
      });
  }, [taskRunName]);

  useEffect(() => {
    const version = ++sheetFetchVersion.current;

    // Always clear both sources when task changes, so previous-task data
    // cannot be displayed while the new task's requests are in flight.
    setSheet(undefined);
    setSheetsMap(new Map());

    if (!task) {
      setSheetFetchState(
        buildSheetFetchStateForMissingTask(taskRunName, metadataFetchState)
      );
      return;
    }

    if (isReleaseBasedTask(task)) {
      setSheetFetchState({ status: "loading", source: "release" });
      const releaseName = releaseNameOfTaskV1(task);
      if (!releaseName) {
        setSheetFetchState({ status: "success", source: "release" });
        return;
      }

      void releaseStore
        .fetchReleaseByName(releaseName, true)
        .then(async (release) => {
          const fileSheets = await Promise.all(
            release.files
              .filter((file) => file.sheet && file.version)
              .map(async (file) => {
                try {
                  const fetchedSheet = await sheetServiceClientConnect.getSheet(
                    {
                      name: file.sheet,
                      raw: true,
                    }
                  );
                  return { version: file.version, sheet: fetchedSheet };
                } catch (error: unknown) {
                  return {
                    version: file.version,
                    error: getErrorMessage(error),
                  };
                }
              })
          );
          if (version !== sheetFetchVersion.current) return;

          const releaseResult = buildReleaseSheetFetchResult(
            fileSheets
              .filter((item) => item?.version)
              .map((item) => ({
                version: item.version,
                sheet: item.sheet,
              }))
          );
          setSheetsMap(releaseResult.sheetsMap);
          setSheetFetchState(releaseResult.state);
        })
        .catch((error: unknown) => {
          if (version !== sheetFetchVersion.current) return;
          setSheetsMap(new Map());
          setSheetFetchState({
            status: "error",
            source: "release",
            error: getErrorMessage(error),
          });
        });

      return;
    }

    setSheetFetchState({ status: "loading", source: "sheet" });
    const sheetName = sheetNameOfTaskV1(task);
    if (!sheetName) {
      setSheetFetchState({ status: "success", source: "sheet" });
      return;
    }

    void sheetServiceClientConnect
      .getSheet({
        name: sheetName,
        raw: true,
      })
      .then((fetchedSheet) => {
        if (version !== sheetFetchVersion.current) return;
        setSheet(fetchedSheet);
        setSheetFetchState({ status: "success", source: "sheet" });
      })
      .catch((error: unknown) => {
        if (version !== sheetFetchVersion.current) return;
        setSheet(undefined);
        setSheetFetchState({
          status: "error",
          source: "sheet",
          error: getErrorMessage(error),
        });
      });
  }, [
    releaseStore,
    task,
    taskRunName,
    sheetFetchTaskKey,
    unresolvedTaskMetadataStateKey,
  ]);

  return {
    entries,
    sheet,
    sheetsMap,
    metadataFetch: metadataFetchState,
    logFetch: logFetchState,
    sheetFetch: sheetFetchState,
  };
};
