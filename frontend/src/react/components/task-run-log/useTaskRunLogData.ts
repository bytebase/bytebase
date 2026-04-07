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
            error: "Task cannot be resolved from rollout metadata",
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
      if (!taskRunName) {
        setSheetFetchState({ status: "idle", source: "none" });
      } else if (metadataFetchState.status === "loading") {
        setSheetFetchState({ status: "loading", source: "none" });
      } else if (metadataFetchState.status === "error") {
        setSheetFetchState({
          status: "error",
          source: "none",
          error:
            metadataFetchState.error ??
            "Task cannot be resolved from rollout metadata",
        });
      } else {
        setSheetFetchState({
          status: "error",
          source: "none",
          error: "Task cannot be resolved from rollout metadata",
        });
      }
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

          const nextSheetsMap = new Map<string, Sheet>();
          const failedReleaseVersions: string[] = [];
          for (const item of fileSheets) {
            if (item?.sheet) {
              nextSheetsMap.set(item.version, item.sheet);
            } else if (item?.version) {
              failedReleaseVersions.push(item.version);
            }
          }
          setSheetsMap(nextSheetsMap);
          if (failedReleaseVersions.length === 0) {
            setSheetFetchState({ status: "success", source: "release" });
            return;
          }
          if (nextSheetsMap.size === 0) {
            setSheetFetchState({
              status: "error",
              source: "release",
              error: "Failed to fetch all release sheets",
              failedReleaseVersions,
            });
            return;
          }
          setSheetFetchState({
            status: "partial",
            source: "release",
            error: "Failed to fetch some release sheets",
            failedReleaseVersions,
          });
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
    metadataFetchState.error,
    metadataFetchState.status,
    releaseStore,
    task,
    taskRunName,
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
