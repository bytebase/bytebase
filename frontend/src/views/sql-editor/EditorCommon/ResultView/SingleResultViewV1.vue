<template>
  <template v-if="result.messages.length > 0">
    <div
      v-for="(message, i) in result.messages"
      :key="`message-${i}`"
      :class="'text-control-light'"
    >
      <div>{{ `[${message.level}] ${message.content}` }}</div>
    </div>
  </template>
  <template v-if="viewMode === 'RESULT'">
    <BBAttention v-if="result.error" class="w-full mb-2" :type="'error'">
      <ErrorView :error="result.error" />
    </BBAttention>
    <div
      class="relative w-full shrink-0 flex flex-row justify-between items-center mb-2 overflow-x-auto hide-scrollbar"
    >
      <div class="flex flex-row justify-start items-center mr-2 shrink-0">
        <NInput
          v-if="showSearchFeature"
          :value="state.search"
          class="max-w-40!"
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
          resultRowsText
        }}</span>
        <span
          v-if="
            currentTab?.mode !== 'ADMIN' &&
            dataLength === editorStore.resultRowsLimit
          "
          class="ml-2 whitespace-nowrap text-sm text-gray-500"
        >
          <span>-</span>
          <span class="ml-2">{{ $t("sql-editor.rows-upper-limit") }}</span>
        </span>
      </div>
      <div class="flex justify-between items-center shrink-0 gap-x-2">
        <div class="flex items-center">
          <NSwitch v-model:value="state.vertical" size="small" />
          <span class="ml-1 whitespace-nowrap text-sm text-gray-500">
            {{ $t("sql-editor.vertical-display") }}
          </span>
        </div>
        <NPagination
          :simple="true"
          :item-count="table.getCoreRowModel().rows.length"
          :page="pageIndex + 1"
          :page-size="pageSize"
          class="pagination whitespace-nowrap"
          style="--n-input-width: 2.5rem"
          @update-page="handleChangePage"
        >
          <template #suffix>
            <NSelect
              v-model:value="pageSize"
              :options="pageSizeOptions"
              :consistent-menu-width="false"
              :show-arrow="false"
              class="pagesize-select"
              placement="bottom-end"
              size="small"
            />
          </template>
        </NPagination>
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
            {{ $t("plugin.ai.text2sql") }}
          </template>
        </NTooltip>
        <template v-if="showExport">
          <DataExportButton
            size="small"
            :disabled="props.result === null || isEmpty(props.result)"
            :support-formats="[
              ExportFormat.CSV,
              ExportFormat.JSON,
              ExportFormat.SQL,
              ExportFormat.XLSX,
            ]"
            :view-mode="'DRAWER'"
            :support-password="true"
            :maximum-export-count="maximumExportCount"
            @export="
              ($event) =>
                $emit('export', {
                  ...$event,
                  statement: result.statement,
                })
            "
          >
            <template #form>
              <NFormItem :label="$t('common.database')">
                <DatabaseInfo :database="database" />
              </NFormItem>
            </template>
          </DataExportButton>
        </template>
      </div>
      <SelectionCopyTooltips />
    </div>

    <div class="flex-1 w-full flex flex-col overflow-y-auto">
      <VirtualDataBlock
        v-if="state.vertical"
        :table="table"
        :set-index="setIndex"
        :offset="pageIndex * pageSize"
        :is-sensitive-column="isSensitiveColumn"
        :get-masking-reason="getMaskingReason"
        :database="database"
      />
      <VirtualDataTable
        v-else
        :table="table"
        :set-index="setIndex"
        :offset="pageIndex * pageSize"
        :is-sensitive-column="isSensitiveColumn"
        :get-masking-reason="getMaskingReason"
        :database="database"
      />
    </div>

    <div
      class="w-full flex items-center justify-between text-xs mt-1 gap-x-4 text-control-light"
    >
      <div class="flex flex-1 items-center gap-x-2">
        <RichDatabaseName :database="database" />
        <NTooltip :disabled="!isSupported">
          <template #trigger>
            <div
              class="truncate cursor-pointer hover:bg-gray-200"
              @click="copyStatement"
            >
              {{ result.statement }}
            </div>
          </template>
          {{ $t("common.click-to-copy") }}
        </NTooltip>
      </div>
      <div class="flex items-center shrink-0 gap-x-2">
        <NButton
          v-if="showVisualizeButton"
          text
          type="primary"
          @click="visualizeExplain"
          size="tiny"
        >
          {{ $t("sql-editor.visualize-explain") }}
        </NButton>
        <span>{{ $t("sql-editor.query-time") }}: {{ queryTime }}</span>
      </div>
    </div>
  </template>
  <template v-else-if="viewMode === 'AFFECTED-ROWS'">
    <div
      class="text-md font-normal flex items-center gap-x-1"
      :class="[dark ? 'text-matrix-green-hover' : 'text-control-light']"
    >
      <span>{{ extractSQLRowValuePlain(result.rows[0].values[0]) }}</span>
      <span>rows affected</span>
    </div>
  </template>
  <template v-else-if="viewMode === 'EMPTY'">
    <EmptyView />
  </template>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import type { ColumnDef } from "@tanstack/vue-table";
import {
  getCoreRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useVueTable,
} from "@tanstack/vue-table";
import { useClipboard, useDebounceFn, useLocalStorage } from "@vueuse/core";
import { isEmpty } from "lodash-es";
import {
  NButton,
  NFormItem,
  NInput,
  NPagination,
  NSelect,
  NSwitch,
  NTooltip,
  type SelectOption,
} from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { BBAttention } from "@/bbkit";
import DatabaseInfo from "@/components/DatabaseInfo.vue";
import type {
  DownloadContent,
  ExportOption,
} from "@/components/DataExportButton.vue";
import DataExportButton from "@/components/DataExportButton.vue";
import { RichDatabaseName } from "@/components/v2";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { DISMISS_PLACEHOLDER } from "@/plugins/ai/components/state";
import {
  pushNotification,
  useSQLEditorStore,
  useSQLEditorTabStore,
} from "@/store";
import type {
  ComposedDatabase,
  SQLEditorDatabaseQueryContext,
  SQLEditorQueryParams,
} from "@/types";
import { DEBOUNCE_SEARCH_DELAY, isValidInstanceName } from "@/types";
import { Engine, ExportFormat } from "@/types/proto-es/v1/common_pb";
import {
  QueryOption_MSSQLExplainFormat,
  QueryOptionSchema,
  type QueryResult,
  type QueryRow,
  type RowValue,
} from "@/types/proto-es/v1/sql_service_pb";
import {
  compareQueryRowValues,
  createExplainToken,
  extractSQLRowValuePlain,
  instanceV1HasStructuredQueryResult,
  isNullOrUndefined,
} from "@/utils";
import { useSQLResultViewContext } from "./context";
import { provideSelectionContext } from "./DataTable/common/selection-logic";
import VirtualDataTable from "./DataTable/VirtualDataTable.vue";
import EmptyView from "./EmptyView.vue";
import ErrorView from "./ErrorView";
import SelectionCopyTooltips from "./SelectionCopyTooltips.vue";
import VirtualDataBlock from "./VirtualDataBlock.vue";

// Using conversion function from common-conversions.ts

type LocalState = {
  search: string;
  vertical: boolean;
};
type ViewMode = "RESULT" | "EMPTY" | "AFFECTED-ROWS" | "ERROR";

const DEFAULT_PAGE_SIZE = 50;
const storedPageSize = useLocalStorage<number>(
  "bb.sql-editor.result-page-size",
  DEFAULT_PAGE_SIZE
);

const props = defineProps<{
  params: SQLEditorQueryParams;
  database: ComposedDatabase;
  result: QueryResult;
  setIndex: number;
  showExport: boolean;
  maximumExportCount?: number;
}>();

defineEmits<{
  (
    event: "export",
    option: {
      resolve: (content: DownloadContent[]) => void;
      reject: (reason?: any) => void;
      options: ExportOption;
      statement: string;
    }
  ): Promise<void>;
}>();

const state = reactive<LocalState>({
  search: "",
  vertical: false,
});

const { copy: copyTextToClipboard, isSupported } = useClipboard({
  legacy: true,
});

const copyStatement = () => {
  if (!isSupported.value) {
    return;
  }
  copyTextToClipboard(props.result.statement).then(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.copied"),
    });
  });
};

const { t } = useI18n();
const { dark, keyword } = useSQLResultViewContext();
const tabStore = useSQLEditorTabStore();
const editorStore = useSQLEditorStore();
const currentTab = computed(() => tabStore.currentTab);
const { runQuery } = useExecuteSQL();

const viewMode = computed((): ViewMode => {
  const { result } = props;
  if (result.error && dataLength.value === 0) {
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
  if (!isValidInstanceName(props.database.instance)) {
    return false;
  }
  return instanceV1HasStructuredQueryResult(props.database.instanceResource);
});

// use a debounced value to improve performance when typing rapidly
const debouncedUpdateKeyword = useDebounceFn((value: string) => {
  keyword.value = value;
}, DEBOUNCE_SEARCH_DELAY);

const updateKeyword = (value: string) => {
  state.search = value;
  debouncedUpdateKeyword(value);
};

const columns = computed(() => {
  return props.result.columnNames.map<ColumnDef<QueryRow, RowValue>>(
    (columnName, index) => {
      const columnType = props.result.columnTypeNames[index];

      return {
        id: `${columnName}@${index}`,
        accessorFn: (item) => item.values[index],
        header: columnName,
        meta: {
          // Store column type in meta for easy access by other components
          columnType,
        },
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

// Memoize the filtered data to avoid filtering on every render
const data = computed(() => {
  const search = keyword.value.trim().toLowerCase();

  // Return original rows if no search
  if (!search) {
    return props.result.rows;
  }

  // Only filter when there's a search term
  return props.result.rows.filter((item) => {
    return item.values.some((col) => {
      const value = extractSQLRowValuePlain(col);
      if (isNullOrUndefined(value)) {
        return false;
      }
      return String(value).toLowerCase().includes(search);
    });
  });
});

// Cache data length to avoid multiple accesses
const dataLength = computed(() => data.value.length);

// Computed property for result rows text to avoid accessing data.length twice
const resultRowsText = computed(() => {
  const length = dataLength.value;
  return `${length} ${t("sql-editor.rows", length)}`;
});

const isSensitiveColumn = (columnIndex: number): boolean => {
  const maskingReason = props.result.masked?.[columnIndex];
  // Check if maskingReason exists and has actual content (not empty object)
  return (
    maskingReason !== null &&
    maskingReason !== undefined &&
    maskingReason.semanticTypeId !== undefined &&
    maskingReason.semanticTypeId !== ""
  );
};

const getMaskingReason = (columnIndex: number) => {
  if (!props.result.masked || columnIndex >= props.result.masked.length) {
    return undefined;
  }
  const reason = props.result.masked[columnIndex];
  // Return undefined for empty masking reasons
  if (!reason || !reason.semanticTypeId) {
    return undefined;
  }
  return reason;
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

table.setPageSize(storedPageSize.value);

provideSelectionContext(computed(() => table));

const pageIndex = computed(() => {
  return table.getState().pagination.pageIndex;
});
const pageSize = computed({
  get() {
    return table.getState().pagination.pageSize;
  },
  set(ps) {
    table.setPageSize(ps);
    storedPageSize.value = ps;
  },
});
const pageSizeOptions = computed(() => {
  return [20, 50, 100, 200].map<SelectOption>((n) => ({
    label: t("sql-editor.n-per-page", { n }),
    value: n,
  }));
});

const engine = computed(() => props.database.instanceResource.engine);

const showVisualizeButton = computed((): boolean => {
  return (
    (engine.value === Engine.POSTGRES || engine.value === Engine.MSSQL) &&
    props.params.explain
  );
});

const visualizeExplain = async () => {
  let token: string | undefined;
  try {
    if (engine.value === Engine.POSTGRES) {
      token = getExplainToken(props.result);
    } else if (engine.value === Engine.MSSQL) {
      token = await getExplainTokenForMSSQL();
    }
    if (!token) return;

    window.open(`/explain-visualizer.html?token=${token}`, "_blank");
  } catch {
    // nothing
  }
};

const getExplainToken = (result: QueryResult) => {
  const { statement } = result;
  if (!statement) return;

  const lines = result.rows.map((row) =>
    row.values.map((value) => String(extractSQLRowValuePlain(value)))
  );
  const explain = lines.map((line) => line[0]).join("\n");
  if (!explain) return;

  return createExplainToken({
    statement,
    explain,
    engine: engine.value,
  });
};

const getExplainTokenForMSSQL = async () => {
  const context: SQLEditorDatabaseQueryContext = {
    id: uuidv4(),
    params: {
      ...props.params,
      queryOption: create(QueryOptionSchema, {
        mssqlExplainFormat:
          QueryOption_MSSQLExplainFormat.MSSQL_EXPLAIN_FORMAT_XML,
      }),
    },
    status: "PENDING",
  };
  await runQuery(props.database, context);

  const result = context.resultSet?.results[0];
  if (!result) {
    return;
  }
  return getExplainToken(result);
};

const handleChangePage = (page: number) => {
  table.setPageIndex(page - 1);
};

const queryTime = computed(() => {
  const { latency } = props.result;
  if (!latency) return "-";

  const { seconds, nanos } = latency;
  const totalSeconds = Number(seconds) + nanos / 1e9;
  if (totalSeconds < 1) {
    const totalMS = Math.round(totalSeconds * 1000);
    return `${totalMS} ms`;
  }
  return `${totalSeconds.toFixed(2)} s`;
});
</script>

<style scoped lang="postcss">
.pagination :deep(.n-input) {
  --n-padding-left: 6px !important;
  --n-padding-right: 6px !important;
}
.pagination :deep(.n-input__input-el) {
  text-align: right;
}

.pagesize-select :deep(.n-base-selection) {
  --n-padding-single-left: 8px !important;
  --n-padding-single-right: 8px !important;
}
</style>
