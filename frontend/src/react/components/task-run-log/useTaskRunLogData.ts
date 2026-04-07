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

export interface UseTaskRunLogDataResult {
  entries: TaskRunLogEntry[];
  sheet: Sheet | undefined;
  sheetsMap: Map<string, Sheet>;
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

export const useTaskRunLogData = (
  taskRunName?: string
): UseTaskRunLogDataResult => {
  const rolloutStore = useRolloutStore();
  const releaseStore = useReleaseStore();

  const [entries, setEntries] = useState<TaskRunLogEntry[]>([]);
  const [sheet, setSheet] = useState<Sheet | undefined>(undefined);
  const [sheetsMap, setSheetsMap] = useState<Map<string, Sheet>>(new Map());
  const logFetchVersion = useRef(0);
  const sheetFetchVersion = useRef(0);

  const rolloutName = useMemo(() => {
    if (!taskRunName) return "";
    return extractRolloutNameFromTaskRunName(taskRunName);
  }, [taskRunName]);

  useEffect(() => {
    if (!rolloutName) return;
    void rolloutStore.fetchRolloutByName(rolloutName, true).catch(() => {
      // Ignore rollout fetch errors and keep best-effort cached value.
    });
  }, [rolloutName, rolloutStore]);

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

    if (!taskRunName) {
      setEntries([]);
      return;
    }

    const request = create(GetTaskRunLogRequestSchema, {
      parent: taskRunName,
    });
    void rolloutServiceClientConnect
      .getTaskRunLog(request)
      .then((response) => {
        if (version !== logFetchVersion.current) return;
        setEntries(response.entries);
      })
      .catch(() => {
        if (version !== logFetchVersion.current) return;
        setEntries([]);
      });
  }, [taskRunName]);

  useEffect(() => {
    const version = ++sheetFetchVersion.current;

    if (!task) {
      setSheet(undefined);
      setSheetsMap(new Map());
      return;
    }

    if (isReleaseBasedTask(task)) {
      setSheet(undefined);
      const releaseName = releaseNameOfTaskV1(task);
      if (!releaseName) {
        setSheetsMap(new Map());
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
                  const fetchedSheet = await sheetServiceClientConnect.getSheet({
                    name: file.sheet,
                    raw: true,
                  });
                  return { version: file.version, sheet: fetchedSheet };
                } catch {
                  return undefined;
                }
              })
          );
          if (version !== sheetFetchVersion.current) return;

          const nextSheetsMap = new Map<string, Sheet>();
          for (const item of fileSheets) {
            if (item?.sheet) {
              nextSheetsMap.set(item.version, item.sheet);
            }
          }
          setSheetsMap(nextSheetsMap);
        })
        .catch(() => {
          if (version !== sheetFetchVersion.current) return;
          setSheetsMap(new Map());
        });

      return;
    }

    setSheetsMap(new Map());
    const sheetName = sheetNameOfTaskV1(task);
    if (!sheetName) {
      setSheet(undefined);
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
      })
      .catch(() => {
        if (version !== sheetFetchVersion.current) return;
        setSheet(undefined);
      });
  }, [releaseStore, task]);

  return { entries, sheet, sheetsMap };
};
