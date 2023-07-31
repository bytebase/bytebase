<template>
  <template v-if="viewMode === 'RESULT'">
    <div
      class="w-full shrink-0 flex flex-row justify-between items-center mb-2 overflow-x-auto"
    >
      <div class="flex flex-row justify-start items-center mr-2 shrink-0">
        <NInput
          v-if="showSearchFeature"
          :value="state.search"
          class="!max-w-[10rem]"
          size="small"
          type="text"
          :placeholder="t('sql-editor.search-results')"
          @update:value="debouncedUpdateKeyword"
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
      <div class="flex justify-between items-center gap-x-3 overflow-y-hidden">
        <NPagination
          v-if="showPagination"
          :simple="true"
          :item-count="table.getCoreRowModel().rows.length"
          :page="pageIndex + 1"
          :page-size="pageSize"
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
          <NButton
            size="small"
            :loading="isExportingData"
            :disabled="isExportingData"
          >
            <template #icon>
              <heroicons-outline:download class="h-5 w-5" />
            </template>
            {{ t("common.export") }}
          </NButton>
        </NDropdown>
        <NButton v-else @click="state.showRequestExportPanel = true">
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
        :masked="props.result.masked"
        :sensitive="props.result.sensitive"
        :set-index="setIndex"
        :offset="pageIndex * pageSize"
      />
    </div>

    <div
      class="w-full flex items-center justify-between text-xs mt-1 gap-x-4 text-control-light"
    >
      <div class="flex-1 truncate">
        {{ result.statement }}
      </div>
      <div class="shrink-0">
        {{ $t("sql-editor.query-time") }}: {{ queryTime }}
      </div>
    </div>
  </template>
  <template v-else-if="viewMode === 'AFFECTED-ROWS'">
    <div
      class="text-md font-normal flex items-center gap-x-1"
      :class="[
        dark ? 'text-[var(--color-matrix-green-hover)]' : 'text-control-light',
      ]"
    >
      <span>{{ extractSQLRowValue(result.rows[0].values[0]) }}</span>
      <span>rows affected</span>
    </div>
  </template>
  <template v-else-if="viewMode === 'EMPTY'">
    <EmptyView />
  </template>
  <template v-else-if="viewMode === 'ERROR'">
    <ErrorView :error="result.error" />
  </template>

  <RequestExportPanel
    v-if="state.showRequestExportPanel"
    :database-id="currentTab.connection.databaseId"
    :statement="currentTab.statement"
    @close="state.showRequestExportPanel = false"
  />
</template>

<script lang="ts" setup>
import { computed, reactive, ref } from "vue";
import { NInput, NPagination } from "naive-ui";
import { useI18n } from "vue-i18n";
import { useDebounceFn } from "@vueuse/core";
import {
  ColumnDef,
  getCoreRowModel,
  getPaginationRowModel,
  useVueTable,
} from "@tanstack/vue-table";
import { isEmpty } from "lodash-es";

import {
  createExplainToken,
  extractEnvironmentNameListFromExpr,
  extractSQLRowValue,
  hasWorkspacePermissionV1,
  instanceV1HasStructuredQueryResult,
} from "@/utils";
import {
  useInstanceV1Store,
  useTabStore,
  RESULT_ROWS_LIMIT,
  featureToRef,
  useCurrentUserIamPolicy,
  useDatabaseV1Store,
  useCurrentUserV1,
  usePolicyV1Store,
  useDatabaseV1ByUID,
} from "@/store";
import DataTable from "./DataTable";
import EmptyView from "./EmptyView.vue";
import ErrorView from "./ErrorView.vue";
import { useSQLResultViewContext } from "./context";
import { Engine } from "@/types/proto/v1/common";
import { QueryResult } from "@/types/proto/v1/sql_service";
import {
  ExecuteConfig,
  ExecuteOption,
  SQLResultSetV1,
  TabMode,
  UNKNOWN_ID,
} from "@/types";
import { useExportData } from "./useExportData";
import RequestExportPanel from "@/components/Issue/panel/RequestExportPanel/index.vue";
import { resolveCELExpr } from "@/plugins/cel";
import { Expr } from "@/types/proto/google/api/expr/v1alpha1/syntax";

type LocalState = {
  search: string;
  showRequestExportPanel: boolean;
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
  result: QueryResult;
  setIndex: number;
}>();

const state = reactive<LocalState>({
  search: "",
  showRequestExportPanel: false,
});

const { dark, keyword } = useSQLResultViewContext();

const { t } = useI18n();
const tabStore = useTabStore();
const instanceStore = useInstanceV1Store();
const databaseStore = useDatabaseV1Store();
const currentUserV1 = useCurrentUserV1();
const dataTable = ref<InstanceType<typeof DataTable>>();
const { isExportingData, exportData } = useExportData();
const currentTab = computed(() => tabStore.currentTab);
const { database } = useDatabaseV1ByUID(
  computed(() => currentTab.value.connection.databaseId)
);

const viewMode = computed((): ViewMode => {
  const { result } = props;
  if (result.error) {
    return "ERROR";
  }
  const columnNames = result.columnNames;
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

const showExportButton = computed(() => {
  if (!featureToRef("bb.feature.access-control").value) {
    // The current plan doesn't have access control feature.
    // Fallback to true.
    return true;
  }

  return allowToExportData.value;
});

const allowToExportData = computed(() => {
  if (!featureToRef("bb.feature.access-control").value) {
    return true;
  }

  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-access-control",
      currentUserV1.value.userRole
    )
  ) {
    return true;
  }

  const policy = usePolicyV1Store().getPolicyByName("policies/WORKSPACE_IAM");
  if (database.value && policy) {
    const bindings = policy.workspaceIamPolicy?.bindings;
    if (bindings) {
      const querierBinding = bindings.find(
        (binding) => binding.role === "roles/EXPORTER"
      );
      if (querierBinding) {
        const simpleExpr = resolveCELExpr(
          querierBinding.parsedExpr?.expr || Expr.fromPartial({})
        );
        const envNameList = extractEnvironmentNameListFromExpr(simpleExpr);
        if (envNameList.includes(database.value.instanceEntity.environment)) {
          return true;
        }
      }
    }
  }

  return useCurrentUserIamPolicy().allowToExportDatabaseV1(database.value);
});

// use a debounced value to improve performance when typing rapidly
const debouncedUpdateKeyword = useDebounceFn((value: string) => {
  keyword.value = value;
}, 200);

const columns = computed(() => {
  const columns = props.result.columnNames;
  return columns.map<ColumnDef<string[]>>((col, index) => ({
    id: `${col}@${index}`,
    accessorFn: (item) => item[index],
    header: col,
  }));
});

const convertedData = computed(() => {
  const rows = props.result.rows;
  return rows.map((row) => {
    return row.values.map((value) => extractSQLRowValue(value));
  });
});

const data = computed(() => {
  const data = convertedData.value;
  const search = keyword.value.trim().toLowerCase();
  let temp = data;
  if (search) {
    temp = data.filter((item) => {
      return item.some((col) => String(col).toLowerCase().includes(search));
    });
  }
  return temp;
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

const pageIndex = computed(() => {
  return table.getState().pagination.pageIndex;
});
const pageSize = computed(() => {
  return table.getState().pagination.pageSize;
});

const exportDropdownOptions = computed(() => [
  {
    label: t("sql-editor.download-as-file", { file: "CSV" }),
    key: "CSV",
    disabled: props.result === null || isEmpty(props.result),
  },
  {
    label: t("sql-editor.download-as-file", { file: "JSON" }),
    key: "JSON",
    disabled: props.result === null || isEmpty(props.result),
  },
  {
    label: t("sql-editor.download-as-file", { file: "SQL" }),
    key: "SQL",
    disabled: props.result === null || isEmpty(props.result),
  },
  {
    label: t("sql-editor.download-as-file", { file: "XLSX" }),
    key: "XLSX",
    disabled: props.result === null || isEmpty(props.result),
  },
]);

const handleExportBtnClick = (format: "CSV" | "JSON" | "SQL" | "XLSX") => {
  const { instanceId, databaseId } = tabStore.currentTab.connection;
  const instance = instanceStore.getInstanceByUID(instanceId).name;
  const database =
    databaseId === String(UNKNOWN_ID)
      ? ""
      : databaseStore.getDatabaseByUID(databaseId).name;
  const statement = props.result.statement;
  const admin = tabStore.currentTab.mode === TabMode.Admin;
  const limit = admin ? 0 : RESULT_ROWS_LIMIT;
  exportData({
    database,
    instance,
    format,
    statement,
    limit,
    admin,
  });
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
    const { executeParams, sqlResultSet } = tabStore.currentTab;
    if (!executeParams || !sqlResultSet) return;

    const statement = executeParams.query || "";
    if (!statement) return;

    const explain = explainFromSQLResultSetV1(sqlResultSet);
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

const explainFromSQLResultSetV1 = (resultSet: SQLResultSetV1 | undefined) => {
  if (!resultSet) return "";
  const lines = resultSet.results[0].rows.map((row) =>
    row.values.map((value) => String(extractSQLRowValue(value)))
  );
  const explain = lines.map((line) => line[0]).join("\n");
  return explain;
};

const queryTime = computed(() => {
  const { latency } = props.result;
  if (!latency) return "-";

  const { seconds, nanos } = latency;
  const totalSeconds = seconds + nanos / 1e9;
  if (totalSeconds < 1) {
    const totalMS = Math.round(totalSeconds * 1000);
    return `${totalMS} ms`;
  }
  return `${totalSeconds.toFixed(2)} s`;
});
</script>
