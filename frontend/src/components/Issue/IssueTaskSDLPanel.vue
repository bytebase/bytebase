<template>
  <div
    class="flex flex-col md:flex-row md:justify-between md:items-center gap-2 md:gap-4"
  >
    <div class="flex space-x-4 flex-1">
      <div class="py-2 text-sm font-medium text-control">
        {{ $t("common.sql") }}
        <button
          v-if="!hasFeature('bb.feature.sql-review')"
          type="button"
          class="ml-1 btn-small py-0.5 inline-flex items-center text-accent"
          @click.prevent="state.showFeatureModal = true"
        >
          🎈{{ $t("sql-review.unlock-full-feature") }}
        </button>
      </div>
    </div>
  </div>
  <div class="w-full">
    <div
      v-if="changeHistory.loading"
      class="h-20 flex flex-col items-center justify-center"
    >
      <BBSpin />
    </div>
    <template v-else>
      <NTabs
        v-if="changeHistory.history"
        v-model:value="state.tab"
        pane-style="padding-top: 0.25rem"
      >
        <NTabPane name="diff" :tab="$t('issue.sdl.schema-change')">
          <CodeDiff
            :old-string="changeHistory.history.schemaPrev"
            :new-string="changeHistory.history.schema"
            output-format="side-by-side"
            data-label="bb-change-history-code-diff-block"
        /></NTabPane>
        <NTabPane
          name="statement"
          :tab="$t('issue.sdl.generated-ddl-statements')"
        >
          <HighlightCodeBlock
            class="border px-2 whitespace-pre-wrap"
            :code="changeHistory.history.statement"
          />
        </NTabPane>
        <NTabPane name="schema" :tab="$t('issue.sdl.full-schema')">
          <HighlightCodeBlock
            class="border px-2 whitespace-pre-wrap"
            :code="changeHistory.history.schema"
          />
        </NTabPane>
      </NTabs>
    </template>
  </div>
  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.sql-review"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { NTabs, NTabPane } from "naive-ui";
import { reactive, watch, computed } from "vue";
import { CodeDiff } from "v-code-diff";

import { hasFeature, useInstanceStore } from "@/store";
import { useIssueLogic } from "./logic";
import { MigrationHistory, Task, TaskId } from "@/types";
import HighlightCodeBlock from "../HighlightCodeBlock";

type TabView = "diff" | "statement" | "schema";

interface LocalState {
  showFeatureModal: boolean;
  tab: TabView;
}

const { selectedTask } = useIssueLogic();

const state = reactive<LocalState>({
  showFeatureModal: false,
  tab: "diff",
});

const useChangeHistory = () => {
  type ChangeHistoryState = {
    task: Task;
    loading: boolean;
    history: MigrationHistory | undefined;
  };
  const emptyState = (task: Task): ChangeHistoryState => {
    return {
      task,
      loading: true,
      history: undefined,
    };
  };

  const map = reactive(new Map<TaskId, ChangeHistoryState>());

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

  watch(
    [() => (selectedTask.value as Task).id, migrationId],
    ([taskId, migrationId]) => {
      const task = selectedTask.value as Task;
      if (!map.has(taskId)) {
        map.set(taskId, emptyState(task));
      }
      const finish = (ch?: MigrationHistory) => {
        const state = map.get(taskId)!;
        state.loading = false;
        state.history = ch;
      };

      if (!migrationId) return finish();
      useInstanceStore()
        .fetchMigrationHistoryById({
          instanceId: task.instance.id,
          migrationHistoryId: migrationId,
        })
        .then((history) => {
          // The latestMigrationId might change during fetching the
          // migrationHistory.
          // Should give up the result.
          const latestMigrationId = findLatestMigrationId(task);
          if (history.id !== latestMigrationId) return;
          finish(history);
        });
    },
    { immediate: true }
  );

  return computed(() => {
    const task = selectedTask.value as Task;
    return map.get(task.id) ?? emptyState(task);
  });
};

const changeHistory = useChangeHistory();
</script>
