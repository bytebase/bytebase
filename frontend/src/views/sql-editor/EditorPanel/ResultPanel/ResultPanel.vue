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
      <template v-if="selectedResults.length === 0">
        <div
          class="w-full h-full flex flex-col justify-center items-center text-sm"
        >
          <span>{{ $t("sql-editor.table-empty-placeholder") }}</span>
        </div>
      </template>
      <NTabs
        v-else
        type="card"
        size="small"
        class="flex-1 flex flex-col overflow-hidden px-2"
        :class="isBatchQuery ? 'pt-0' : 'pt-2'"
        style="--n-tab-padding: 4px 12px"
        v-model:value="selectedTab"
      >
        <NTabPane
          v-for="(result, i) in selectedResults"
          :key="i"
          :name="result.beginTimestampMS"
          class="flex-1 flex flex-col overflow-hidden"
        >
          <template #tab>
            <span>{{ tabName(result.beginTimestampMS) }}</span>
            <Info
              v-if="result.resultSet.error"
              class="ml-2 text-yellow-600 w-4 h-auto"
            />
          </template>
          <ResultViewV1
            class="w-full h-auto grow"
            :execute-params="result.params"
            :database="selectedDatabase"
            :result-set="result.resultSet"
          />
        </NTabPane>
      </NTabs>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { useTimestamp } from "@vueuse/core";
import dayjs from "dayjs";
import { head } from "lodash-es";
import { Info } from "lucide-vue-next";
import { NButton, NTabs, NTabPane } from "naive-ui";
import { computed, ref, watch } from "vue";
import { BBSpin } from "@/bbkit";
import { useSQLEditorTabStore } from "@/store";
import type { ComposedDatabase } from "@/types";
import { ResultViewV1 } from "../../EditorCommon/";
import BatchQuerySelect from "./BatchQuerySelect.vue";

const tabStore = useSQLEditorTabStore();
const selectedDatabase = ref<ComposedDatabase>();
const selectedTab = ref<number>();

const selectedResults = computed(() => {
  return (
    tabStore.currentTab?.queryContext?.results.get(
      selectedDatabase.value?.name || ""
    ) ?? []
  );
});

const tabName = (beginTimestampMS: number) => {
  return dayjs(beginTimestampMS).format("YYYY-MM-DD HH:mm:ss");
};

watch(
  () => selectedResults.value,
  (results) => {
    selectedTab.value = head(results)?.beginTimestampMS;
  },
  { deep: true }
);

const loading = computed(
  () => tabStore.currentTab?.queryContext?.status === "EXECUTING"
);
const currentTimestampMS = useTimestamp();
const queryElapsedTime = computed(() => {
  if (!loading.value) {
    return "";
  }
  const tab = tabStore.currentTab;
  if (!tab) {
    return "";
  }
  const beginMS = tabStore.currentTab?.queryContext?.beginTimestampMS;
  if (!beginMS) {
    return "";
  }
  const elapsedMS = currentTimestampMS.value - beginMS;
  return `${(elapsedMS / 1000).toFixed(1)}s`;
});

const isBatchQuery = computed(
  () =>
    Array.from(tabStore.currentTab?.queryContext?.results.keys() || []).length >
    1
);

const cancelQuery = () => {
  tabStore.currentTab?.queryContext?.abortController.abort();
};
</script>
