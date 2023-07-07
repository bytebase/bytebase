import { computed, reactive, watch } from "vue";
import axios from "axios";

import { Task, Task_Status } from "@/types/proto/v1/rollout_service";
import { useIssueContext } from "../../../logic";
import { useTaskSheet } from "../useTaskSheet";
import { useChangeHistoryStore, useDatabaseV1Store } from "@/store";
import { sqlServiceClient } from "@/grpcweb";
import { useSilentRequest } from "@/plugins/silent-request";
import { engineToJSON } from "@/types/proto/v1/common";
import Emittery from "emittery";

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
  const { selectedTask } = useIssueContext();
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

  const findLatestChangeHistoryId = (task: Task) => {
    if (task.status !== Task_Status.DONE) return undefined;
    // TODO: find changeHistory id for task from latest taskRun
    // const list = task.taskRunList;
    // for (let i = list.length - 1; i >= 0; i--) {
    //   const taskRun = list[i];
    //   if (taskRun.status === "DONE") {
    //     return taskRun.result.migrationId;
    //   }
    // }
    return undefined;
  };

  const changeHistoryId = computed(() => {
    const task = selectedTask.value as Task;
    return findLatestChangeHistoryId(task);
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
    changeHistoryId: string | undefined
  ): Promise<SDLDetail | undefined> => {
    if (!changeHistoryId) {
      return undefined;
    }
    const database = await databaseStore.getOrFetchDatabaseByName(task.target);
    const history = await useChangeHistoryStore().fetchChangeHistory({
      name: `${database.name}/changeHistories/${changeHistoryId}`,
      sdlFormat: true,
    });
    // The latestChangeHistoryId might change during fetching the
    // ChangeHistory.
    // Should give up the result.
    const latestChangeHistoryId = findLatestChangeHistoryId(task);
    if (history.uid !== latestChangeHistoryId) {
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
    [() => selectedTask.value.name, sheetStatement, changeHistoryId],
    async ([taskName, statement, changeHistoryId]) => {
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
            changeHistoryId
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
