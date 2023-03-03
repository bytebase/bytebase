<template>
  <NConfigProvider
    v-bind="naiveUIConfig"
    class="relative flex flex-col justify-start items-start p-2"
    :class="dark && 'dark bg-dark-bg'"
  >
    <template v-if="viewMode === 'result'">
      <div
        class="w-full shrink-0 flex flex-row justify-between items-center mb-2 overflow-x-auto"
      >
        <div class="flex flex-row justify-start items-center mr-2 shrink-0">
          <NInput
            v-if="showSearchFeature"
            v-model:value="state.search"
            class="!max-w-[8rem] sm:!max-w-xs"
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
          <span
            v-if="data.length === RESULT_ROWS_LIMIT"
            class="ml-2 whitespace-nowrap text-sm text-gray-500"
          >
            <span>-</span>
            <span class="ml-2">{{ $t("sql-editor.rows-upper-limit") }}</span>
          </span>
        </div>
        <div class="flex justify-between items-center gap-x-3">
          <NPagination
            v-if="showPagination"
            :item-count="table.getCoreRowModel().rows.length"
            :page="table.getState().pagination.pageIndex + 1"
            :page-size="table.getState().pagination.pageSize"
            :show-quick-jumper="true"
            :show-size-picker="true"
            :page-sizes="[20, 50, 100]"
            @update-page="handleChangePage"
            @update-page-size="(ps) => table.setPageSize(ps)"
          />
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

      <div class="flex-1 w-full flex flex-col overflow-y-auto">
        <DataTable
          v-show="!showPlaceholder"
          ref="dataTable"
          :table="table"
          :columns="columns"
          :data="data"
          :sensitive="sensitive"
        />
      </div>
    </template>
    <template v-else-if="viewMode === 'affected-rows'">
      <div
        class="text-md font-normal flex items-center gap-x-1"
        :class="[
          dark
            ? 'text-[var(--color-matrix-green-hover)]'
            : 'text-control-light',
        ]"
      >
        <span>{{ queryResult?.data[2][0][0] }}</span>
        <span>rows affected</span>
      </div>
    </template>
    <template v-else-if="viewMode === 'empty'">
      <div
        class="text-md font-normal"
        :class="[
          dark
            ? 'text-[var(--color-matrix-green-hover)]'
            : 'text-control-light',
        ]"
      >
        {{ $t("sql-editor.no-rows-found") }}
      </div>
    </template>
    <template v-else-if="viewMode === 'error'">
      <div
        class="text-md font-normal"
        :class="[
          dark
            ? 'text-[var(--color-matrix-green-hover)]'
            : 'text-control-light',
        ]"
      >
        {{ queryResult?.error }}
      </div>
    </template>

    <div
      v-if="showPlaceholder"
      class="absolute inset-0 flex flex-col justify-center items-center z-10"
      :class="loading && 'bg-white/80 dark:bg-black/80'"
    >
      <template v-if="loading">
        <BBSpin />
        {{ $t("sql-editor.loading-data") }}
      </template>
      <template v-else-if="!queryResult">
        {{ $t("sql-editor.table-empty-placeholder") }}
      </template>
    </div>
  </NConfigProvider>
</template>

<script lang="ts" setup>
import { computed, PropType, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { debouncedRef } from "@vueuse/core";
import { unparse } from "papaparse";
import { isEmpty } from "lodash-es";
import dayjs from "dayjs";
import { darkTheme, NConfigProvider, NPagination } from "naive-ui";

import { darkThemeOverrides } from "@/../naive-ui.config";
import { useTabStore, useInstanceStore } from "@/store";
import { createExplainToken } from "@/utils";
import DataTable from "./DataTable.vue";
import { RESULT_ROWS_LIMIT } from "@/store";
import {
  ColumnDef,
  getCoreRowModel,
  getPaginationRowModel,
  useVueTable,
} from "@tanstack/vue-table";
import { SingleSQLResult } from "@/types";

type ViewMode = "result" | "empty" | "affected-rows" | "error";

interface State {
  search: string;
}

const props = defineProps({
  queryResult: {
    type: Object as PropType<SingleSQLResult>,
    default: undefined,
  },
  loading: {
    type: Boolean,
    default: false,
  },
  dark: {
    type: Boolean,
    default: false,
  },
});

const PAGE_SIZES = [20, 50, 100];
const DEFAULT_PAGE_SIZE = 50;

const { t } = useI18n();
const tabStore = useTabStore();
const instanceStore = useInstanceStore();

const state = reactive<State>({
  search: "",
});

const viewMode = computed((): ViewMode => {
  const { queryResult } = props;
  if (!queryResult) {
    return "empty";
  }
  if (queryResult.error) {
    return "error";
  }
  if (
    queryResult.data?.[0].length === 1 &&
    queryResult.data[0][0] === "Affected Rows"
  ) {
    return "affected-rows";
  }
  return "result";
});

const dataTable = ref<InstanceType<typeof DataTable>>();

const showSearchFeature = computed(() => {
  const instance = instanceStore.getInstanceById(
    tabStore.currentTab.connection.instanceId
  );
  return instance.engine !== "MONGODB";
});

const naiveUIConfig = computed(() => {
  if (props.dark) {
    return { theme: darkTheme, themeOverrides: darkThemeOverrides.value };
  }
  return {};
});

// use a debounced value to improve performance when typing rapidly
const keyword = debouncedRef(
  computed(() => state.search),
  200
);

const columns = computed(() => {
  if (!props.queryResult?.data) {
    return [];
  }
  const columns = props.queryResult.data[0];
  return columns.map<ColumnDef<string[]>>((col, index) => ({
    id: `${col}@${index}`,
    accessorFn: (item) => item[index],
    header: col,
  }));
});

const data = computed(() => {
  if (!props.queryResult?.data) {
    return [];
  }

  const data = props.queryResult.data[2];
  const search = keyword.value;
  let temp = data;
  if (search) {
    temp = data.filter((item) => {
      return item.some((col) => String(col).includes(search));
    });
  }
  return temp;
});

const sensitive = computed(() => {
  if (!props.queryResult?.data) {
    return [];
  }
  return props.queryResult.data[3] ?? [];
});

const table = useVueTable<string[]>({
  get data() {
    return data.value;
  },
  get columns() {
    return columns.value;
  },
  getCoreRowModel: getCoreRowModel(),
  getPaginationRowModel: getPaginationRowModel(),
});

table.setPageSize(DEFAULT_PAGE_SIZE);

const showPlaceholder = computed(() => {
  if (!props.queryResult?.data) return true;
  if (props.loading) return true;
  return false;
});

const exportDropdownOptions = computed(() => [
  {
    label: t("sql-editor.download-as-csv"),
    key: "csv",
    disabled: props.queryResult === null || isEmpty(props.queryResult),
  },
  {
    label: t("sql-editor.download-as-json"),
    key: "json",
    disabled: props.queryResult === null || isEmpty(props.queryResult),
  },
]);

const handleExportBtnClick = (format: "csv" | "json") => {
  let rawText = "";

  if (format === "csv") {
    const csvFields = columns.value.map((col) => col.header as string);
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

    const lines: string[][] = queryResult.data[2];
    const explain = lines.map((line) => line[0]).join("\n");
    if (!explain) return;

    const token = createExplainToken(statement, explain);

    window.open(`/explain-visualizer.html?token=${token}`, "_blank");
  } catch {
    // nothing
  }
};

const showPagination = computed(() => data.value.length > PAGE_SIZES[0]);

const handleChangePage = (page: number) => {
  table.setPageIndex(page - 1);
  dataTable.value?.scrollTo(0, 0);
};
</script>
