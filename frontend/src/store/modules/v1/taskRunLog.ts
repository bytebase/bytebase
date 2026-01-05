import { create } from "@bufbuild/protobuf";
import { defineStore } from "pinia";
import { reactive } from "vue";
import { rolloutServiceClientConnect } from "@/connect";
import type { TaskRunLog } from "@/types/proto-es/v1/rollout_service_pb";
import {
  GetTaskRunLogRequestSchema,
  TaskRunLogSchema,
} from "@/types/proto-es/v1/rollout_service_pb";

export const useTaskRunLogStore = defineStore("taskRunLog", () => {
  // Map from taskRun name to TaskRunLog
  const taskRunLogByName = reactive(new Map<string, TaskRunLog>());

  const getTaskRunLog = (taskRunName: string): TaskRunLog => {
    const existing = taskRunLogByName.get(taskRunName);
    if (existing) {
      return existing;
    }

    // Return empty log if not found
    const emptyLog = create(TaskRunLogSchema, {});
    return emptyLog;
  };

  const fetchTaskRunLog = async (
    taskRunName: string,
    options?: { skipCache?: boolean }
  ): Promise<TaskRunLog> => {
    if (!options?.skipCache) {
      const existing = taskRunLogByName.get(taskRunName);
      if (existing) {
        return existing;
      }
    }

    try {
      const request = create(GetTaskRunLogRequestSchema, {
        parent: taskRunName,
      });
      const response = await rolloutServiceClientConnect.getTaskRunLog(request);
      taskRunLogByName.set(taskRunName, response);
      return response;
    } catch (error) {
      console.error(`Failed to fetch task run log for ${taskRunName}:`, error);
      const emptyLog = create(TaskRunLogSchema, {});
      taskRunLogByName.set(taskRunName, emptyLog);
      return emptyLog;
    }
  };

  const clearCache = () => {
    taskRunLogByName.clear();
  };

  return {
    getTaskRunLog,
    fetchTaskRunLog,
    clearCache,
  };
});
