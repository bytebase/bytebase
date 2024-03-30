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
          @update:value="updateKeyword"
        >
          <template #prefix>
            <heroicons-outline:search class="h-5 w-5 text-gray-300" />
          </template>
        </NInput>
        <span class="ml-2 whitespace-nowrap text-sm text-gray-500">{{
          `${data.length} ${t("sql-editor.rows", data.length)}`
        }}</span>
        <span
          v-if="
            currentTab?.mode !== 'ADMIN' && data.length === RESULT_ROWS_LIMIT
          "
          class="ml-2 whitespace-nowrap text-sm text-gray-500"
        >
          <span>-</span>
          <span class="ml-2">{{ $t("sql-editor.rows-upper-limit") }}</span>
        </span>
      </div>
      <div class="flex justify-between items-center gap-x-3 overflow-y-hidden">
        <div class="flex items-center">
          <NSwitch v-model:value="state.vertical" size="small" />
          <span class="ml-1 whitespace-nowrap text-sm text-gray-500">
            {{ $t("sql-editor.vertical-display") }}
          </span>
        </div>
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
        <NTooltip v-if="DISMISS_PLACEHOLDER">
          <template #trigger>
            <NButton
              size="small"
              style="--n-padding: 0 8px"
              @click="DISMISS_PLACEHOLDER = false"
            >
              <template #icon>
                <heroicons:academic-cap class="w-4 h-4" />
              </template>
            </NButton>
          </template>
          <template #default>
            {{ $t("plugin.ai.chat-sql") }}
          </template>
        </NTooltip>
        <template v-if="showExportButton">
          <DataExportButton
            v-if="allowToExportData"
            size="small"
            :file-type="'zip'"
            :disabled="props.result === null || isEmpty(props.result)"
            :support-formats="[
              ExportFormat.CSV,
              ExportFormat.JSON,
              ExportFormat.SQL,
              ExportFormat.XLSX,
            ]"
            :allow-specify-row-count="true"
            @export="handleExportBtnClick"
          />
          <NButton
            v-else-if="allowToRequestExportData"
            @click="state.showRequestExportPanel = true"
          >
            {{ $t("quick-action.request-export-data") }}
          </NButton>
        </template>
      </div>
    </div>

    <div class="flex-1 w-full flex flex-col overflow-y-auto">
      <DataBlock
        v-if="state.vertical"
        :table="table"
        :set-index="setIndex"
        :offset="pageIndex * pageSize"
        :is-sensitive-column="isSensitiveColumn"
        :is-column-missing-sensitive="isColumnMissingSensitive"
      />
      <DataTable
        v-else
        :table="table"
        :set-index="setIndex"
        :offset="pageIndex * pageSize"
        :is-sensitive-column="isSensitiveColumn"
        :is-column-missing-sensitive="isColumnMissingSensitive"
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
      :class="[dark ? 'text-matrix-green-hover' : 'text-control-light']"
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
    v-if="database && state.showRequestExportPanel"
    :database-id="database.uid"
    :statement="result.statement"
    :statement-only="true"
    :redirect-to-issue-page="pageMode === 'BUNDLED'"
    @close="state.showRequestExportPanel = false"
  />
</template>

<script lang="ts" setup>
import type { ColumnDef } from "@tanstack/vue-table";
import {
  getCoreRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useVueTable,
} from "@tanstack/vue-table";
import { useDebounceFn } from "@vueuse/core";
import { isEmpty } from "lodash-es";
import { NInput, NSwitch, NPagination, NTooltip } from "naive-ui";
import type { BinaryLike } from "node:crypto";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import type { ExportOption } from "@/components/DataExportButton.vue";
import RequestExportPanel from "@/components/Issue/panel/RequestExportPanel/index.vue";
import { DISMISS_PLACEHOLDER } from "@/plugins/ai/components/state";
import {
  useSQLEditorTabStore,
  RESULT_ROWS_LIMIT,
  featureToRef,
  useCurrentUserV1,
  usePageMode,
  useActuatorV1Store,
  useConnectionOfCurrentSQLEditorTab,
} from "@/store";
import { useExportData } from "@/store/modules/export";
import type {
  ComposedDatabase,
  SQLEditorQueryParams,
  SQLResultSetV1,
} from "@/types";
import { UNKNOWN_ID } from "@/types";
import { ExportFormat } from "@/types/proto/v1/common";
import { Engine } from "@/types/proto/v1/common";
import type {
  QueryResult,
  QueryRow,
  RowValue,
} from "@/types/proto/v1/sql_service";
import {
  compareQueryRowValues,
  createExplainToken,
  extractSQLRowValue,
  hasPermissionToCreateRequestGrantIssue,
  hasWorkspacePermissionV2,
  instanceV1HasStructuredQueryResult,
} from "@/utils";
import DataBlock from "./DataBlock.vue";
import DataTable from "./DataTable";
import EmptyView from "./EmptyView.vue";
import ErrorView from "./ErrorView.vue";
import { useSQLResultViewContext } from "./context";

type LocalState = {
  search: string;
  vertical: boolean;
  showRequestExportPanel: boolean;
};
type ViewMode = "RESULT" | "EMPTY" | "AFFECTED-ROWS" | "ERROR";

const PAGE_SIZES = [20, 50, 100];
const DEFAULT_PAGE_SIZE = 50;

const props = defineProps<{
  params: SQLEditorQueryParams;
  database?: ComposedDatabase;
  sqlResultSet: SQLResultSetV1;
  result: QueryResult;
  setIndex: number;
}>();

const state = reactive<LocalState>({
  search: "",
  vertical: false,
  showRequestExportPanel: false,
});

const { dark, keyword } = useSQLResultViewContext();

const actuatorStore = useActuatorV1Store();
const { t } = useI18n();
const tabStore = useSQLEditorTabStore();
const currentUserV1 = useCurrentUserV1();
const { exportData } = useExportData();
const currentTab = computed(() => tabStore.currentTab);
const pageMode = usePageMode();
const { instance: connectedInstance } = useConnectionOfCurrentSQLEditorTab();

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
  if (connectedInstance.value.uid === String(UNKNOWN_ID)) {
    return false;
  }
  return instanceV1HasStructuredQueryResult(connectedInstance.value);
});

const showExportButton = computed(() => {
  return actuatorStore.customTheme !== "lixiang";
});

const allowToExportData = computed(() => {
  // The current plan doesn't have access control feature.
  // Fallback to true.
  if (!featureToRef("bb.feature.access-control").value) {
    return true;
  }

  if (hasWorkspacePermissionV2(currentUserV1.value, "bb.policies.update")) {
    return true;
  }

  return props.sqlResultSet?.allowExport || false;
});

const allowToRequestExportData = computed(() => {
  const { database } = props;
  if (!database) {
    return false;
  }
  if (database.uid === String(UNKNOWN_ID)) {
    return false;
  }

  return hasPermissionToCreateRequestGrantIssue(database, currentUserV1.value);
});

// use a debounced value to improve performance when typing rapidly
const debouncedUpdateKeyword = useDebounceFn((value: string) => {
  keyword.value = value;
}, 200);
const updateKeyword = (value: string) => {
  state.search = value;
  debouncedUpdateKeyword(value);
};

const columns = computed(() => {
  return props.result.columnNames.map<ColumnDef<QueryRow, RowValue>>(
    (columnName, index) => {
      const columnType = props.result.columnTypeNames[index] as string;
      return {
        id: `${columnName}@${index}`,
        accessorFn: (item) => item.values[index],
        header: columnName,
        sortingFn: (rowA, rowB) => {
          return compareQueryRowValues(
            columnType,
            rowA.original.values[index],
            rowB.original.values[index]
          );
        },
      };
    }
  );
});

const data = computed(() => {
  const data = props.result.rows;
  const search = keyword.value.trim().toLowerCase();
  let temp = data;
  if (search) {
    temp = data.filter((item) => {
      return item.values.some((col) =>
        String(extractSQLRowValue(col)).toLowerCase().includes(search)
      );
    });
  }
  return temp;
});

const isSensitiveColumn = (columnIndex: number): boolean => {
  return props.result.masked[columnIndex] ?? false;
};

const isColumnMissingSensitive = (columnIndex: number): boolean => {
  return (
    (props.result.sensitive[columnIndex] ?? false) &&
    !isSensitiveColumn(columnIndex)
  );
};

const table = useVueTable<QueryRow>({
  get data() {
    return data.value;
  },
  get columns() {
    return columns.value;
  },
  getCoreRowModel: getCoreRowModel(),
  getSortedRowModel: getSortedRowModel(),
  getPaginationRowModel: getPaginationRowModel(),
});

table.setPageSize(DEFAULT_PAGE_SIZE);

const pageIndex = computed(() => {
  return table.getState().pagination.pageIndex;
});
const pageSize = computed(() => {
  return table.getState().pagination.pageSize;
});

const handleExportBtnClick = async (
  options: ExportOption,
  callback: (content: BinaryLike | Blob, options: ExportOption) => void
) => {
  // If props.database is specified and it's not unknown database
  // the query is executed on database level
  // otherwise the query is executed on instance level, we should use the
  // `instanceId` from the tab's connection attributes
  const database =
    props.database && props.database.uid !== String(UNKNOWN_ID)
      ? props.database.name
      : "";
  const instance =
    props.database && props.database.uid !== String(UNKNOWN_ID)
      ? props.database.instance
      : connectedInstance.value.name;
  const statement = props.result.statement;
  const admin = tabStore.currentTab?.mode === "ADMIN";
  const limit = options.limit ?? (admin ? 0 : RESULT_ROWS_LIMIT);

  const content = await exportData({
    database,
    instance,
    format: options.format,
    statement,
    limit,
    admin,
    password: options.password,
  });

  callback(content, options);
};

const showVisualizeButton = computed((): boolean => {
  return (
    connectedInstance.value.engine === Engine.POSTGRES && props.params.explain
  );
});

const visualizeExplain = () => {
  try {
    const { params, sqlResultSet } = props;
    if (!sqlResultSet) return;

    const { statement } = params;
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
  const totalSeconds = seconds.toNumber() + nanos / 1e9;
  if (totalSeconds < 1) {
    const totalMS = Math.round(totalSeconds * 1000);
    return `${totalMS} ms`;
  }
  return `${totalSeconds.toFixed(2)} s`;
});
</script>
