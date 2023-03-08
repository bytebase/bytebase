<template>
  <div v-if="!hasFeature('bb.feature.sql-review')">
    <div class="flex space-x-4 flex-1">
      <button
        type="button"
        class="btn-small py-0.5 inline-flex items-center text-accent"
        @click.prevent="state.showFeatureModal = true"
      >
        🎈{{ $t("sql-review.unlock-full-feature") }}
      </button>
    </div>
  </div>
  <div class="w-full">
    <div
      v-if="sdlState.loading"
      class="h-20 flex flex-col items-center justify-center"
    >
      <BBSpin />
    </div>
    <template v-else-if="sdlState.detail">
      <NTabs v-model:value="state.tab" class="mb-1">
        <NTab name="diff" :disabled="!!sdlState.detail.error">
          <div class="flex items-center gap-x-1">
            {{ $t("issue.sdl.schema-change") }}
            <NTooltip>
              <template #trigger>
                <heroicons:question-mark-circle class="w-4 h-4" />
              </template>
              <div class="whitespace-nowrap">
                <span>{{ $t("issue.sdl.left-schema-may-change") }}</span>
                <LearnMoreLink
                  url="https://www.bytebase.com/docs/change-database/state-based-migration?source=console#caveats"
                  color="light"
                  class="ml-1"
                />
              </div>
            </NTooltip>
          </div>
        </NTab>
        <NTab name="statement" :disabled="!!sdlState.detail.error">
          {{ $t("issue.sdl.generated-ddl-statements") }}
        </NTab>
        <NTab name="schema">
          {{ $t("issue.sdl.full-schema") }}
        </NTab>
      </NTabs>

      <div
        v-if="sdlState.detail.error"
        class="flex text-error gap-x-1 items-start text-sm mb-1"
      >
        <heroicons:exclamation-circle class="w-4 h-4 shrink-0 mt-0.5" />
        <div>{{ sdlState.detail.error }}</div>
      </div>

      <CodeDiff
        v-if="state.tab === 'diff'"
        :old-string="sdlState.detail.previousSDL"
        :new-string="sdlState.detail.prettyExpectedSDL"
        output-format="side-by-side"
        data-label="bb-change-detail-code-diff-block"
      />
      <MonacoEditor
        v-if="state.tab === 'statement'"
        ref="editorRef"
        class="w-full border h-auto max-h-[360px]"
        data-label="bb-issue-sql-editor"
        :value="sdlState.detail.diffDDL"
        :readonly="true"
        :auto-focus="false"
        language="sql"
        @ready="handleMonacoEditorReady"
      />
      <MonacoEditor
        v-if="state.tab === 'schema'"
        ref="editorRef"
        class="w-full border h-auto max-h-[360px]"
        data-label="bb-issue-sql-editor"
        :value="sdlState.detail.expectedSDL"
        :readonly="true"
        :auto-focus="false"
        :advices="markers"
        language="sql"
        @ready="handleMonacoEditorReady"
      />
    </template>
  </div>
  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.sql-review"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { NTabs, NTab, NTooltip } from "naive-ui";
import { reactive, watch, computed, ref } from "vue";
import { CodeDiff } from "v-code-diff";
import axios from "axios";

import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { hasFeature, useDatabaseStore, useInstanceStore } from "@/store";
import { useIssueLogic } from "./logic";
import { Task, TaskDatabaseSchemaUpdateSDLPayload, TaskId } from "@/types";
import MonacoEditor from "../MonacoEditor";
import { sqlClient } from "@/grpcweb";
import { convertEngineType } from "@/types";
import { useSQLAdviceMarkers } from "./logic/useSQLAdviceMarkers";
import { useSilentRequest } from "@/plugins/silent-request";

type TabView = "diff" | "statement" | "schema";

type SDLDetail = {
  error: string;
  previousSDL: string;
  prettyExpectedSDL: string;
  expectedSDL: string;
  diffDDL: string;
};

type SDLState = {
  task: Task;
  loading: boolean;
  detail: SDLDetail | undefined;
};

interface LocalState {
  showFeatureModal: boolean;
  tab: TabView;
}

const { selectedTask } = useIssueLogic();

const state = reactive<LocalState>({
  showFeatureModal: false,
  tab: "diff",
});
const editorRef = ref<InstanceType<typeof MonacoEditor>>();

const useSDLState = () => {
  const emptyState = (task: Task): SDLState => {
    return {
      task,
      loading: true,
      detail: undefined,
    };
  };

  const map = reactive(new Map<TaskId, SDLState>());

  const findLatestMigrationId = (task: Task) => {
    if (task.status !== "DONE") return undefined;
    const list = task.taskRunList;
    for (let i = list.length - 1; i >= 0; i--) {
      const taskRun = list[i];
      if (taskRun.status === "DONE") {
        return taskRun.result.migrationId;
      }
    }
    return undefined;
  };

  const migrationId = computed(() => {
    const task = selectedTask.value as Task;
    return findLatestMigrationId(task);
  });

  const fetchOngoingSDLDetail = async (
    task: Task
  ): Promise<SDLDetail | undefined> => {
    const database = task.database;
    if (!database) return undefined;
    const previousSDL = await useDatabaseStore().fetchDatabaseSchemaById(
      task.database!.id,
      true // fetch SDL format
    );
    const payload = task.payload as TaskDatabaseSchemaUpdateSDLPayload;
    if (!payload) return undefined;
    const expectedSDL = payload.statement;

    const getSchemaDiff = async () => {
      const { data } = await axios.post("/v1/sql/schema/diff", {
        engineType: database.instance.engine,
        sourceSchema: previousSDL ?? "",
        targetSchema: expectedSDL ?? "",
      });
      return data ?? "";
    };
    const diffDDL = await useSilentRequest(getSchemaDiff);

    const { currentSchema, expectedSchema } = await sqlClient.pretty({
      engine: convertEngineType(database.instance.engine),
      currentSchema: previousSDL ?? "",
      expectedSchema: expectedSDL ?? "",
    });

    if (task.status === "DONE") {
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

  const fetchSDLDetailFromMigrationHistory = async (
    task: Task,
    migrationId: string | undefined
  ): Promise<SDLDetail | undefined> => {
    if (!migrationId) {
      return undefined;
    }
    const history = await useInstanceStore().fetchMigrationHistoryById({
      instanceId: task.instance.id,
      migrationHistoryId: migrationId,
      sdl: true,
    });
    // The latestMigrationId might change during fetching the
    // migrationHistory.
    // Should give up the result.
    const latestMigrationId = findLatestMigrationId(task);
    if (history.id !== latestMigrationId) {
      throw new Error();
    }
    return {
      error: "",
      previousSDL: history.schemaPrev,
      prettyExpectedSDL: history.schema,
      expectedSDL: history.schema,
      diffDDL: history.statement,
    };
  };

  watch(
    [
      () => (selectedTask.value as Task).id,
      () => (selectedTask.value as Task).status,
      migrationId,
    ],
    async ([taskId, taskStatus, migrationId]) => {
      const task = selectedTask.value as Task;
      if (!map.has(taskId)) {
        map.set(taskId, emptyState(task));
      }
      const finish = (detail?: SDLState["detail"]) => {
        const state = map.get(taskId)!;
        state.loading = false;
        state.detail = detail;
      };
      try {
        if (taskStatus === "DONE") {
          const detail = await fetchSDLDetailFromMigrationHistory(
            task,
            migrationId
          );
          finish(detail);
        } else {
          const detail = await fetchOngoingSDLDetail(task);
          finish(detail);
        }
      } catch (err: any) {
        // The task has been changed during the fetch
        // The result is meaningless.
        state.tab = "schema";
        const payload = task.payload as TaskDatabaseSchemaUpdateSDLPayload;
        const message =
          err.response?.data?.message ?? err.details ?? "Internal server error";

        finish({
          error: message,
          diffDDL: "",
          expectedSDL: payload?.statement,
          previousSDL: "",
          prettyExpectedSDL: "",
        });
      }
    },
    { immediate: true }
  );

  return computed(() => {
    const task = selectedTask.value as Task;
    return map.get(task.id) ?? emptyState(task);
  });
};

const sdlState = useSDLState();

const updateEditorHeight = () => {
  const contentHeight =
    editorRef.value?.editorInstance?.getContentHeight() as number;
  editorRef.value?.setEditorContentHeight(contentHeight);
};

const handleMonacoEditorReady = () => {
  updateEditorHeight();
};

const { markers } = useSQLAdviceMarkers();
</script>
