import Emittery from "emittery";
import { computed, reactive, watch } from "vue";
import { sqlServiceClient } from "@/grpcweb";
import { useSilentRequest } from "@/plugins/silent-request";
import { useChangelogStore, useDatabaseV1Store } from "@/store";
import { ChangelogView } from "@/types/proto/v1/database_service";
import type { Task } from "@/types/proto/v1/rollout_service";
import { TaskRun_Status, Task_Status } from "@/types/proto/v1/rollout_service";
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
  const { sheetStatement, sheetReady } = useTaskSheet();
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

  const findLatestChangelogName = (task: Task) => {
    if (task.status !== Task_Status.DONE) return undefined;
    const taskRunList = issue.value.rolloutTaskRunList.filter((taskRun) => {
      return extractTaskUID(taskRun.name) === extractTaskUID(task.name);
    });
    for (let i = taskRunList.length - 1; i >= 0; i--) {
      const taskRun = taskRunList[i];
      if (taskRun.status === TaskRun_Status.DONE) {
        return taskRun.changelog;
      }
    }
    return undefined;
  };

  const changelogName = computed(() => {
    const task = selectedTask.value;
    return findLatestChangelogName(task);
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
      const { diff } = await databaseStore.diffSchema({
        name: database.name,
        schema: expectedSDL,
        sdlFormat: true,
      });
      return diff ?? "";
    };
    const diffDDL = await useSilentRequest(getSchemaDiff);

    const { currentSchema, expectedSchema } = await sqlServiceClient.pretty(
      {
        engine: database.instanceResource.engine,
        currentSchema: previousSDL ?? "",
        expectedSchema: expectedSDL ?? "",
      },
      {
        silent: true,
      }
    );

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

  const fetchSDLDetailFromChangelog = async (
    task: Task,
    changelogName: string | undefined
  ): Promise<SDLDetail | undefined> => {
    if (!changelogName) {
      return undefined;
    }
    const changelog = await useChangelogStore().fetchChangelog({
      name: changelogName,
      sdlFormat: true,
      view: ChangelogView.CHANGELOG_VIEW_FULL,
    });
    // The latestChangelogName might change during fetching the
    // Changelog.
    // Should give up the result.
    const latestChangelogName = findLatestChangelogName(task);
    if (changelog.name !== latestChangelogName) {
      throw new Error();
    }
    return {
      error: "",
      previousSDL: changelog.prevSchema,
      prettyExpectedSDL: changelog.schema,
      expectedSDL: changelog.schema,
      diffDDL: changelog.statement,
    };
  };

  watch(
    [() => selectedTask.value.name, sheetStatement, sheetReady, changelogName],
    async ([taskName, statement, sheetReady, changelogName]) => {
      if (!sheetReady) return;

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
          const detail = await fetchSDLDetailFromChangelog(task, changelogName);
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
