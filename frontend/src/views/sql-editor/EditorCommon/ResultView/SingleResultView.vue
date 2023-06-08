<template>
  <template v-if="viewMode === 'RESULT'">
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
          :simple="true"
          :item-count="table.getCoreRowModel().rows.length"
          :page="table.getState().pagination.pageIndex + 1"
          :page-size="table.getState().pagination.pageSize"
          @update-page="handleChangePage"
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
          v-if="showExportButton"
          trigger="hover"
          :options="exportDropdownOptions"
          @select="handleExportBtnClick"
        >
          <NButton :loading="isExportingData" :disabled="isExportingData">
            <template #icon>
              <heroicons-outline:download class="h-5 w-5" />
            </template>
            {{ t("common.export") }}
          </NButton>
        </NDropdown>
        <NButton
          v-if="showRequestExportButton"
          @click="handleGotoRequestExportPage"
        >
          {{ $t("quick-action.request-export") }}
        </NButton>
      </div>
    </div>

    <div class="flex-1 w-full flex flex-col overflow-y-auto">
      <DataTable
        ref="dataTable"
        :table="table"
        :columns="columns"
        :data="data"
        :sensitive="sensitive"
        :keyword="state.search"
      />
    </div>
  </template>
  <template v-else-if="viewMode === 'AFFECTED-ROWS'">
    <div
      class="text-md font-normal flex items-center gap-x-1"
      :class="[
        dark ? 'text-[var(--color-matrix-green-hover)]' : 'text-control-light',
      ]"
    >
      <span>{{ result.data[2][0][0] }}</span>
      <span>rows affected</span>
    </div>
  </template>
  <template v-else-if="viewMode === 'EMPTY'">
    <EmptyView />
  </template>
  <template v-else-if="viewMode === 'ERROR'">
    <ErrorView :error="result.error" />
  </template>
</template>

<script lang="ts" setup>
import { computed, reactive, ref } from "vue";
import { NPagination } from "naive-ui";
import { useI18n } from "vue-i18n";
import { debouncedRef } from "@vueuse/core";
import {
  ColumnDef,
  getCoreRowModel,
  getPaginationRowModel,
  useVueTable,
} from "@tanstack/vue-table";
import { isEmpty } from "lodash-es";

import {
  ExecuteConfig,
  ExecuteOption,
  SingleSQLResult,
  TabMode,
  UNKNOWN_ID,
} from "@/types";
import {
  createExplainToken,
  hasWorkspacePermissionV1,
  instanceV1HasStructuredQueryResult,
} from "@/utils";
import {
  useInstanceV1Store,
  useTabStore,
  RESULT_ROWS_LIMIT,
  useCurrentUserIamPolicy,
  pushNotification,
  useDatabaseV1Store,
  useCurrentUserV1,
  hasFeature,
} from "@/store";
import DataTable from "./DataTable";
import EmptyView from "./EmptyView.vue";
import ErrorView from "./ErrorView.vue";
import { useSQLResultViewContext } from "./context";
import { Engine } from "@/types/proto/v1/common";
import { useRouter } from "vue-router";
import { useExportData } from "./useExportData";

type LocalState = {
  search: string;
};
type ViewMode = "RESULT" | "EMPTY" | "AFFECTED-ROWS" | "ERROR";

const PAGE_SIZES = [20, 50, 100];
const DEFAULT_PAGE_SIZE = 50;

const props = defineProps<{
  params: {
    query: string;
    config: ExecuteConfig;
    option?: Partial<ExecuteOption> | undefined;
  };
  result: SingleSQLResult;
}>();

const state = reactive<LocalState>({
  search: "",
});

const { dark } = useSQLResultViewContext();

const { t } = useI18n();
const router = useRouter();
const tabStore = useTabStore();
const instanceStore = useInstanceV1Store();
const databaseStore = useDatabaseV1Store();
const currentUserV1 = useCurrentUserV1();
const dataTable = ref<InstanceType<typeof DataTable>>();
const { isExportingData, exportData } = useExportData();

const viewMode = computed((): ViewMode => {
  const { result } = props;
  if (result.error) {
    return "ERROR";
  }
  const columnNames = result.data?.[0];
  if (columnNames?.length === 0) {
    return "EMPTY";
  }
  if (columnNames?.length === 1 && columnNames[0] === "Affected Rows") {
    return "AFFECTED-ROWS";
  }
  return "RESULT";
});

const showSearchFeature = computed(() => {
  const instance = instanceStore.getInstanceByUID(
    tabStore.currentTab.connection.instanceId
  );
  return instanceV1HasStructuredQueryResult(instance);
});

const allowToExportData = computed(() => {
  const database = databaseStore.getDatabaseByUID(
    tabStore.currentTab.connection.databaseId
  );
  return useCurrentUserIamPolicy().allowToExportDatabaseV1(database);
});

// use a debounced value to improve performance when typing rapidly
const keyword = debouncedRef(
  computed(() => state.search),
  200
);

const columns = computed(() => {
  if (!props.result.data) {
    return [];
  }
  const columns = props.result.data[0];
  return columns.map<ColumnDef<string[]>>((col, index) => ({
    id: `${col}@${index}`,
    accessorFn: (item) => item[index],
    header: col,
  }));
});

const data = computed(() => {
  if (!props.result.data) {
    return [];
  }

  const data = props.result.data[2];
  const search = keyword.value.trim().toLowerCase();
  let temp = data;
  if (search) {
    temp = data.filter((item) => {
      return item.some((col) => String(col).toLowerCase().includes(search));
    });
  }
  return temp;
});

const sensitive = computed(() => {
  if (!props.result.data) {
    return [];
  }
  return props.result.data[3] ?? [];
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

const exportDropdownOptions = computed(() => [
  {
    label: t("sql-editor.download-as-csv"),
    key: "CSV",
    disabled: props.result === null || isEmpty(props.result),
  },
  {
    label: t("sql-editor.download-as-json"),
    key: "JSON",
    disabled: props.result === null || isEmpty(props.result),
  },
]);

const showExportButton = computed(() => {
  if (!hasFeature("bb.feature.custom-role")) {
    return true;
  }
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-database",
    currentUserV1.value.userRole
  );
});

const showRequestExportButton = computed(() => {
  return hasFeature("bb.feature.custom-role") && !showExportButton.value;
});

const handleExportBtnClick = (format: "CSV" | "JSON") => {
  if (!allowToExportData.value) {
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: "You don't have permission to export data.",
    });
    return;
  }

  const { instanceId, databaseId } = tabStore.currentTab.connection;
  const instance = instanceStore.getInstanceByUID(instanceId).name;
  const database =
    databaseId === String(UNKNOWN_ID)
      ? ""
      : databaseStore.getDatabaseByUID(databaseId).name;
  const statement = props.params.query;
  const limit =
    tabStore.currentTab.mode === TabMode.Admin ? 0 : RESULT_ROWS_LIMIT;
  exportData({
    database,
    instance,
    format,
    statement,
    limit,
  });
};

const handleGotoRequestExportPage = () => {
  const routeInfo = {
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query: {
      template: "bb.issue.grant.request",
      role: "EXPORTER",
      name: "New grant exporter request",
    },
  };

  const currentTab = tabStore.currentTab;
  if (String(currentTab.connection.databaseId) !== String(UNKNOWN_ID)) {
    const database = databaseStore.getDatabaseByUID(
      currentTab.connection.databaseId
    );
    (routeInfo.query as any).project = database.projectEntity.uid;
    (routeInfo.query as any).databaseList = database.uid;
    (routeInfo.query as any).sql =
      currentTab.selectedStatement || currentTab.statement;
  }

  router.push(routeInfo);
};

const showVisualizeButton = computed((): boolean => {
  const instance = instanceStore.getInstanceByUID(
    tabStore.currentTab.connection.instanceId
  );
  const databaseType = instance.engine;
  const { executeParams } = tabStore.currentTab;
  return databaseType === Engine.POSTGRES && !!executeParams?.option?.explain;
});

const visualizeExplain = () => {
  try {
    const { executeParams, queryResult } = tabStore.currentTab;
    if (!executeParams || !queryResult) return;

    const statement = executeParams.query || "";
    if (!statement) return;

    const lines: string[][] = queryResult.resultList[0].data[2];
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
