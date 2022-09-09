<template>
  <div class="relative flex flex-col justify-start items-start h-full p-2">
    <div
      v-show="queryResult !== null"
      class="w-full flex flex-row justify-between items-center mb-2"
    >
      <div class="flex flex-row justify-start items-center mr-2">
        <NInput
          v-model:value="state.search"
          class="max-w-xs"
          type="text"
          :placeholder="t('sql-editor.search-results')"
        >
          <template #prefix>
            <heroicons-outline:search class="h-5 w-5 text-gray-300" />
          </template>
        </NInput>
        <span class="ml-2 whitespace-nowrap text-sm text-gray-500">{{
          `${data.length} ${t("sql-editor.rows", data.length)}`
        }}</span>
      </div>
      <div class="flex justify-between items-center gap-x-3">
        <NButton
          v-if="showVisualizeButton"
          text
          type="primary"
          @click="visualizeExplain"
        >
          {{ $t("sql-editor.visualize-explain") }}
        </NButton>
        <NDropdown
          trigger="hover"
          :options="exportDropdownOptions"
          @select="handleExportBtnClick"
        >
          <NButton>
            <template #icon>
              <heroicons-outline:download class="h-5 w-5" />
            </template>
            {{ t("common.export") }}
          </NButton>
        </NDropdown>
      </div>
    </div>

    <DataTable v-show="!showPlaceholder" :columns="columns" :data="data" />

    <div
      v-if="showPlaceholder"
      class="absolute inset-0 flex flex-col justify-center items-center z-10"
      :class="sqlEditorStore.isExecuting && 'bg-white/80'"
    >
      <template v-if="sqlEditorStore.isExecuting">
        <BBSpin />
        {{ $t("sql-editor.loading-data") }}
      </template>
      <template v-else-if="!queryResult">
        {{ $t("sql-editor.table-empty-placeholder") }}
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { debouncedRef } from "@vueuse/core";
import { unparse } from "papaparse";
import { isEmpty } from "lodash-es";
import dayjs from "dayjs";
import { useTabStore, useSQLEditorStore, useInstanceStore } from "@/store";
import { createExplainToken } from "@/utils";
import DataTable from "./DataTable.vue";

interface State {
  search: string;
}

const { t } = useI18n();
const tabStore = useTabStore();
const instanceStore = useInstanceStore();
const sqlEditorStore = useSQLEditorStore();

const queryResult = computed(() => tabStore.currentTab.queryResult || null);

const state = reactive<State>({
  search: "",
});

// use a debounced value to improve performance when typing rapidly
const keyword = debouncedRef(
  computed(() => state.search),
  200
);

const columns = computed(() => {
  if (!queryResult.value) {
    return [];
  }

  const columns = queryResult.value[0];
  return columns.map((d) => {
    return {
      title: d,
      key: d,
    };
  });
});

const data = computed(() => {
  if (!queryResult.value) {
    return [];
  }

  const data = queryResult.value[2];
  const search = keyword.value;
  let temp = data;
  if (search) {
    temp = data.filter((item) => {
      return item.some((col) => String(col).includes(search));
    });
  }
  return temp;
});

const showPlaceholder = computed(() => {
  if (!queryResult.value) return true;
  if (sqlEditorStore.isExecuting) return true;
  return false;
});

const exportDropdownOptions = computed(() => [
  {
    label: t("sql-editor.download-as-csv"),
    key: "csv",
    disabled: queryResult.value === null || isEmpty(queryResult.value),
  },
  {
    label: t("sql-editor.download-as-json"),
    key: "json",
    disabled: queryResult.value === null || isEmpty(queryResult.value),
  },
]);

const handleExportBtnClick = (format: "csv" | "json") => {
  let rawText = "";

  if (format === "csv") {
    const csvFields = columns.value.map((item) => item.key);
    const csvData = data.value.map((d) => {
      const temp: any[] = [];
      for (const k in d) {
        temp.push(d[k]);
      }
      return temp;
    });

    rawText = unparse({
      fields: csvFields,
      data: csvData,
    });
  } else {
    rawText = JSON.stringify(data.value);
  }

  const encodedUri = encodeURI(`data:text/${format};charset=utf-8,${rawText}`);
  const formattedDateString = dayjs(new Date()).format("YYYY-MM-DDTHH-mm-ss");
  // Example filename: `mysheet-2022-03-23T09-54-21.json`
  const filename = `${tabStore.currentTab.name}-${formattedDateString}`;
  const link = document.createElement("a");

  link.download = `${filename}.${format}`;
  link.href = encodedUri;
  link.click();
};

const showVisualizeButton = computed((): boolean => {
  const instance = instanceStore.getInstanceById(
    tabStore.currentTab.connection.instanceId
  );
  const databaseType = instance.engine;
  const { executeParams } = tabStore.currentTab;
  return databaseType === "POSTGRES" && !!executeParams?.option?.explain;
});

const visualizeExplain = () => {
  try {
    const { executeParams, queryResult } = tabStore.currentTab;
    if (!executeParams || !queryResult) return;

    const statement = executeParams.query || "";
    if (!statement) return;

    const lines: string[][] = queryResult[2];
    const explain = lines.map((line) => line[0]).join("\n");
    if (!explain) return;

    const token = createExplainToken(statement, explain);

    window.open(`/explain-visualizer.html?token=${token}`, "_blank");
  } catch {
    // nothing
  }
};
</script>
