<template>
  <div
    class="relative w-full h-full flex flex-col justify-start items-start z-10 overflow-x-hidden"
  >
    <template v-if="loading">
      <div
        class="w-full h-full flex flex-col justify-center items-center text-sm gap-y-1 bg-white/80 dark:bg-black/80"
      >
        <div class="flex items-center gap-x-1">
          <BBSpin :size="20" class="mr-1" />
          <span>{{ $t("sql-editor.executing-query") }}</span>
          <span>-</span>
          <!-- use mono font to prevent the UI jitters frequently -->
          <span class="font-mono">{{ queryElapsedTime }}</span>
        </div>
        <div>
          <NButton size="small" @click="cancelQuery">
            {{ $t("common.cancel") }}
          </NButton>
        </div>
      </div>
    </template>
    <template v-else>
      <BatchQuerySelect v-model:selected-database="selectedDatabase" />
      <template v-if="!selectedResultSet">
        <div
          class="w-full h-full flex flex-col justify-center items-center text-sm"
        >
          <span>{{ $t("sql-editor.table-empty-placeholder") }}</span>
        </div>
      </template>
      <ResultViewV1
        v-else
        class="w-full h-auto grow"
        :execute-params="executeParams"
        :database="selectedDatabase"
        :result-set="selectedResultSet"
      />
    </template>
  </div>
</template>

<script lang="ts" setup>
import { useTimestamp } from "@vueuse/core";
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { BBSpin } from "@/bbkit";
import { useSQLEditorTabStore } from "@/store";
import type { ComposedDatabase } from "@/types";
import { ResultViewV1 } from "../../EditorCommon/";
import BatchQuerySelect from "./BatchQuerySelect.vue";

const tabStore = useSQLEditorTabStore();
const selectedDatabase = ref<ComposedDatabase>();

const selectedResultSet = computed(() => {
  return tabStore.currentTab?.queryContext?.results.get(
    selectedDatabase.value?.name || ""
  );
});
const executeParams = computed(() => tabStore.currentTab?.queryContext?.params);
const loading = computed(
  () => tabStore.currentTab?.queryContext?.status === "EXECUTING"
);
const currentTimestampMS = useTimestamp();
const queryElapsedTime = computed(() => {
  if (!loading.value) return "";
  const tab = tabStore.currentTab;
  if (!tab) return "";
  const { queryContext } = tab;
  if (!queryContext) return "";
  const beginMS = queryContext.beginTimestampMS;
  const elapsedMS = currentTimestampMS.value - beginMS;
  return `${(elapsedMS / 1000).toFixed(1)}s`;
});

const cancelQuery = () => {
  tabStore.currentTab?.queryContext?.abortController.abort();
};
</script>
