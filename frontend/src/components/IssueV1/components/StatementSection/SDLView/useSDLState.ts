import axios from "axios";
import Emittery from "emittery";
import { computed, reactive, watch } from "vue";
import { sqlServiceClient } from "@/grpcweb";
import { useSilentRequest } from "@/plugins/silent-request";
import { useChangeHistoryStore, useDatabaseV1Store } from "@/store";
import { engineToJSON } from "@/types/proto/v1/common";
import {
  Task,
  TaskRun_Status,
  Task_Status,
} from "@/types/proto/v1/rollout_service";
import { extractTaskUID } from "@/utils";
import { useIssueContext } from "../../../logic";
import { useTaskSheet } from "../useTaskSheet";

export type SDLDetail = {
  error: string;
  previousSDL: string;
  prettyExpectedSDL: string;
  expectedSDL: string;
  diffDDL: string;
};

export type SDLState = {
  task: Task;
  loading: boolean;
  detail: SDLDetail | undefined;
};

export type SDLEvents = Emittery<{
  error: { message: string };
}>;

export const useSDLState = () => {
  const { issue, selectedTask } = useIssueContext();
  const { sheetStatement } = useTaskSheet();
  const databaseStore = useDatabaseV1Store();
  const events: SDLEvents = new Emittery();

  const emptyState = (task: Task): SDLState => {
    return {
      task,
      loading: true,
      detail: undefined,
    };
  };

  const map = reactive(new Map<string, SDLState>());

  const findLatestChangeHistoryName = (task: Task) => {
    if (task.status !== Task_Status.DONE) return undefined;
    const taskRunList = issue.value.rolloutTaskRunList.filter((taskRun) => {
      return extractTaskUID(taskRun.name) === task.uid;
    });
    for (let i = taskRunList.length - 1; i >= 0; i--) {
      const taskRun = taskRunList[i];
      if (taskRun.status === TaskRun_Status.DONE) {
        return taskRun.changeHistory;
      }
    }
    return undefined;
  };

  const changeHistoryName = computed(() => {
    const task = selectedTask.value;
    return findLatestChangeHistoryName(task);
  });

  const fetchOngoingSDLDetail = async (
    task: Task,
    statement: string
  ): Promise<SDLDetail | undefined> => {
    if (!task.target) return undefined;
    const database = await databaseStore.getOrFetchDatabaseByName(task.target);
    const previousSDL = (
      await databaseStore.fetchDatabaseSchema(
        `${database.name}/schema`,
        true // fetch SDL format
      )
    ).schema;
    const expectedSDL = statement;

    const getSchemaDiff = async () => {
      const { data } = await axios.post("/v1/sql/schema/diff", {
        engineType: engineToJSON(database.instanceEntity.engine),
        sourceSchema: previousSDL ?? "",
        targetSchema: expectedSDL ?? "",
      });
      return data ?? "";
    };
    const diffDDL = await useSilentRequest(getSchemaDiff);

    const { currentSchema, expectedSchema } = await sqlServiceClient.pretty({
      engine: database.instanceEntity.engine,
      currentSchema: previousSDL ?? "",
      expectedSchema: expectedSDL ?? "",
    });

    if (task.status === Task_Status.DONE) {
      throw new Error();
    }

    return {
      error: "",
      previousSDL: currentSchema,
      prettyExpectedSDL: expectedSchema,
      expectedSDL,
      diffDDL,
    };
  };

  const fetchSDLDetailFromChangeHistory = async (
    task: Task,
    changeHistoryName: string | undefined
  ): Promise<SDLDetail | undefined> => {
    if (!changeHistoryName) {
      return undefined;
    }
    const history = await useChangeHistoryStore().fetchChangeHistory({
      name: changeHistoryName,
      sdlFormat: true,
    });
    // The latestChangeHistoryName might change during fetching the
    // ChangeHistory.
    // Should give up the result.
    const latestChangeHistoryName = findLatestChangeHistoryName(task);
    if (history.name !== latestChangeHistoryName) {
      throw new Error();
    }
    return {
      error: "",
      previousSDL: history.prevSchema,
      prettyExpectedSDL: history.schema,
      expectedSDL: history.schema,
      diffDDL: history.statement,
    };
  };

  watch(
    [() => selectedTask.value.name, sheetStatement, changeHistoryName],
    async ([taskName, statement, changeHistoryName]) => {
      const task = selectedTask.value;
      if (!map.has(taskName)) {
        map.set(taskName, emptyState(task));
      }
      const finish = (detail?: SDLState["detail"]) => {
        const state = map.get(taskName);
        if (!state) return;
        state.loading = false;
        state.detail = detail;
      };
      try {
        if (task.status === Task_Status.DONE) {
          const detail = await fetchSDLDetailFromChangeHistory(
            task,
            changeHistoryName
          );
          finish(detail);
        } else {
          const detail = await fetchOngoingSDLDetail(task, statement);
          finish(detail);
        }
      } catch (error: any) {
        // The task has been changed during the fetch
        // The result is meaningless.
        const message =
          error.response?.data?.message ??
          error.details ??
          "Internal server error";

        events.emit("error", { message });
        finish({
          error: message,
          diffDDL: "",
          expectedSDL: statement,
          previousSDL: "",
          prettyExpectedSDL: "",
        });
      }
    },
    { immediate: true }
  );

  const state = computed(() => {
    const task = selectedTask.value as Task;
    return map.get(task.name) ?? emptyState(task);
  });

  return { state, events };
};
