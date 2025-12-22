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
      class="result-toolbar relative w-full shrink-0 flex flex-row gap-x-4 justify-between items-center mb-2 hide-scrollbar"
    >
      <div class="flex flex-row justify-start items-center mr-2 flex-1">
        <AdvancedSearch
          v-model:params="state.searchParams"
          placeholder=""
          :scope-options="searchScopeOptions"
          :cache-query="false"
          @keyup:enter="scrollToNextCandidate"
        />
        <span class="ml-2 whitespace-nowrap text-sm text-gray-500">{{
          resultRowsText
        }}</span>
        <span
          v-if="
            currentTab?.mode !== 'ADMIN' &&
            rows.length === editorStore.resultRowsLimit
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

    <div class="flex-1 w-full flex flex-col overflow-y-auto relative">
      <VirtualDataBlock
        v-if="state.vertical"
        ref="dataTableRef"
        :rows="rows"
        :columns="columns"
        :set-index="setIndex"
        :is-sensitive-column="isSensitiveColumn"
        :get-masking-reason="getMaskingReason"
        :database="database"
        :active-row-index="activeRowIndex"
        :search="state.searchParams"
      />
      <VirtualDataTable
        v-else
        ref="dataTableRef"
        :rows="rows"
        :columns="columns"
        :set-index="setIndex"
        :is-sensitive-column="isSensitiveColumn"
        :get-masking-reason="getMaskingReason"
        :database="database"
        :sort-state="state.sortState"
        :active-row-index="activeRowIndex"
        :search="state.searchParams"
        @toggle-sort="toggleSort"
      />

      <div class="absolute bottom-2 right-4 flex items-end gap-x-2">
        <div v-if="state.searchCandidateRowIndexs.length > 0" class="flex flex-row gap-x-2 border shadow rounded bg-white py-1 px-2">
          <NButton
            quaternary
            size="small"
            :disabled="state.searchCandidateActiveIndex <= 0"
            @click="scrollToPreviousCandidate"
          >
            <template #icon>
              <ArrowUpIcon class="x-4" />
            </template>
            {{ $t("sql-editor.previous-row") }}
          </NButton>
          <NButton
            quaternary
            size="small"
            :disabled="state.searchCandidateActiveIndex >= (state.searchCandidateRowIndexs.length - 1)"
            @click="scrollToNextCandidate"
          >
            <template #icon>
              <ArrowDownIcon class="x-4" />
            </template>
            {{ $t("sql-editor.next-row") }}
          </NButton>
          <NButton quaternary size="small" style="--n-padding: 2px" @click="clearSearchCandidate">
            <template #icon>
              <XIcon class="x-4" />
            </template>
          </NButton>
        </div>

        <div class="flex flex-col gap-y-2">
          <NTooltip>
            <template #trigger>
              <NButton
                circle
                size="medium"
                class="shadow"
                @click="() => scrollToRow(0)"
              >
                <template #icon>
                  <ArrowUpIcon class="x-4" />
                </template>
              </NButton>
            </template>
            {{ $t("sql-editor.scroll-to-top") }}
          </NTooltip>
          <NTooltip>
            <template #trigger>
              <NButton
                circle
                size="medium"
                class="shadow"
                @click="() => scrollToRow(rows.length - 1)"
              >
                <template #icon>
                  <ArrowDownIcon class="x-4" />
                </template>
              </NButton>
            </template>
            {{ $t("sql-editor.scroll-to-bottom") }}
          </NTooltip>
        </div>
      </div>
    </div>

    <div
      class="w-full flex items-center justify-between text-xs mt-1 gap-x-4 text-control-light"
    >
      <div class="flex items-center gap-x-2">
        <RichDatabaseName :database="database" />
        <div class="flex items-center gap-x-1">
          <EllipsisText
            :line-clamp="1"
          >
            {{ result.statement }}
          </EllipsisText>
          <CopyButton :size="'tiny'" :content="result.statement" />
        </div>
      </div>
      <div class="flex shrink-0 items-center gap-x-2">
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
import { isEmpty } from "lodash-es";
import { ArrowDownIcon, ArrowUpIcon, XIcon } from "lucide-vue-next";
import { NButton, NFormItem, NSwitch, NTooltip } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, nextTick, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBAttention } from "@/bbkit";
import AdvancedSearch from "@/components/AdvancedSearch";
import type { ScopeOption } from "@/components/AdvancedSearch/types";
import DatabaseInfo from "@/components/DatabaseInfo.vue";
import type {
  DownloadContent,
  ExportOption,
} from "@/components/DataExportButton.vue";
import DataExportButton from "@/components/DataExportButton.vue";
import EllipsisText from "@/components/EllipsisText.vue";
import { CopyButton, RichDatabaseName } from "@/components/v2";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { DISMISS_PLACEHOLDER } from "@/plugins/ai/components/state";
import { useSQLEditorStore, useSQLEditorTabStore } from "@/store";
import type {
  ComposedDatabase,
  SQLEditorDatabaseQueryContext,
  SQLEditorQueryParams,
} from "@/types";
import { Engine, ExportFormat } from "@/types/proto-es/v1/common_pb";
import {
  QueryOption_MSSQLExplainFormat,
  QueryOptionSchema,
  type QueryResult,
  type RowValue,
} from "@/types/proto-es/v1/sql_service_pb";
import {
  compareQueryRowValues,
  createExplainToken,
  extractSQLRowValuePlain,
  isNullOrUndefined,
  type SearchParams,
} from "@/utils";
import { useSQLResultViewContext } from "./context";
import { provideSelectionContext } from "./DataTable/common/selection-logic";
import type {
  ResultTableColumn,
  ResultTableRow,
  SortState,
} from "./DataTable/common/types";
import VirtualDataTable from "./DataTable/VirtualDataTable.vue";
import EmptyView from "./EmptyView.vue";
import ErrorView from "./ErrorView";
import SelectionCopyTooltips from "./SelectionCopyTooltips.vue";
import VirtualDataBlock from "./VirtualDataBlock.vue";

type LocalState = {
  vertical: boolean;
  sortState: SortState | undefined;
  searchParams: SearchParams;
  searchCandidateActiveIndex: number;
  searchCandidateRowIndexs: number[];
};

type ViewMode = "RESULT" | "EMPTY" | "AFFECTED-ROWS" | "ERROR";

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
      reject: (reason?: unknown) => void;
      options: ExportOption;
      statement: string;
    }
  ): Promise<void>;
}>();

const state = reactive<LocalState>({
  vertical: false,
  sortState: undefined,
  searchParams: {
    query: "",
    scopes: [],
  },
  searchCandidateActiveIndex: -1,
  searchCandidateRowIndexs: [],
});

const dataTableRef =
  ref<InstanceType<typeof VirtualDataTable | typeof VirtualDataBlock>>();

const { t } = useI18n();
const { dark } = useSQLResultViewContext();
const tabStore = useSQLEditorTabStore();
const editorStore = useSQLEditorStore();
const currentTab = computed(() => tabStore.currentTab);
const { runQuery } = useExecuteSQL();

const viewMode = computed((): ViewMode => {
  const { result } = props;
  if (result.error && rows.value.length === 0) {
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

const columns = computed((): ResultTableColumn[] => {
  return props.result.columnNames.map<ResultTableColumn>(
    (columnName, index) => {
      const columnType = props.result.columnTypeNames[index];
      return {
        id: columnName,
        name: columnName,
        columnType,
      };
    }
  );
});

const searchScopeOptions = computed((): ScopeOption[] => {
  const options: ScopeOption[] = [
    {
      id: "row-number",
      title: t("sql-editor.search-scope-row-number-title"),
      description: t("sql-editor.search-scope-row-number-description"),
    },
  ];

  for (const column of columns.value) {
    options.push({
      id: column.id,
      title: column.name,
      description: t("sql-editor.search-scope-column-description", {
        type: column.columnType,
      }),
    });
  }
  return options;
});

const activeRowIndex = computed(() => {
  return state.searchCandidateRowIndexs[state.searchCandidateActiveIndex] ?? -1;
});

watch(
  () => state.searchParams,
  (params) => {
    const next = getNextCandidateRowIndex(0, params);
    const indexs = [];
    if (next >= 0) {
      indexs.push(next);
      const another = getNextCandidateRowIndex(next + 1, params);
      if (another >= 0) {
        indexs.push(another);
      }
    }
    state.searchCandidateRowIndexs = indexs;
    state.searchCandidateActiveIndex = 0;
  },
  { deep: true }
);

watch(
  () => activeRowIndex.value,
  () => {
    scrollToCurrentCandidate();
  }
);

watch(
  () => state.vertical,
  () => {
    scrollToCurrentCandidate();
  }
);

const scrollToNextCandidate = () => {
  if (
    state.searchCandidateActiveIndex >=
    state.searchCandidateRowIndexs.length - 1
  ) {
    return;
  }
  state.searchCandidateActiveIndex++;
  // Append next candidate if reaches the last
  if (
    state.searchCandidateActiveIndex ===
    state.searchCandidateRowIndexs.length - 1
  ) {
    const currentRowIndex =
      state.searchCandidateRowIndexs[state.searchCandidateActiveIndex];
    const next = getNextCandidateRowIndex(
      currentRowIndex + 1,
      state.searchParams
    );
    if (next >= 0) {
      state.searchCandidateRowIndexs.push(next);
    }
  }
};

const scrollToPreviousCandidate = () => {
  if (state.searchCandidateActiveIndex <= 0) {
    return;
  }
  state.searchCandidateActiveIndex--;
};

const scrollToCurrentCandidate = () => {
  scrollToRow(activeRowIndex.value);
};

const scrollToRow = (index: number | undefined) => {
  if (index === undefined || index < 0) {
    return;
  }
  nextTick(() => {
    if (index >= 0) {
      dataTableRef.value?.scrollTo(index);
    }
  });
};

const clearSearchCandidate = () => {
  state.searchParams = {
    query: "",
    scopes: [],
  };
};

const cellValueMatches = (cell: RowValue, query: string) => {
  const value = extractSQLRowValuePlain(cell);
  if (isNullOrUndefined(value)) {
    return false;
  }
  return String(value).toLowerCase().includes(query.toLowerCase());
};

const getNextCandidateRowIndex = (from: number, params: SearchParams) => {
  if (params.scopes.length === 0 && !params.query) {
    return -1;
  }

  for (let i = from; i < rows.value.length; i++) {
    const row = rows.value[i];

    let checked = params.scopes.every((scope) => {
      if (!scope.value) {
        return false;
      }
      if (scope.id === "row-number") {
        return i + 1 === Number.parseInt(scope.value, 10);
      }
      const columnIndex = columns.value.findIndex(
        (column) => column.name === scope.id
      );
      if (columnIndex < 0) {
        return false;
      }
      const cell = row.item.values[columnIndex];
      return cellValueMatches(cell, scope.value);
    });
    if (!checked) {
      continue;
    }

    if (params.query) {
      checked = row.item.values.some((cell) => {
        return cellValueMatches(cell, params.query);
      });
    }
    if (checked) {
      return i;
    }
  }

  return -1;
};

const toggleSort = (columnIndex: number) => {
  const currentSort = state.sortState;

  if (!currentSort || currentSort.columnIndex !== columnIndex) {
    // New column or no current sort - start with descending
    state.sortState = { columnIndex, direction: "desc" };
  } else if (currentSort.direction === "desc") {
    // Currently descending - switch to ascending
    state.sortState = { columnIndex, direction: "asc" };
  } else {
    // Currently ascending - clear sort
    state.sortState = undefined;
  }
};

// Apply sorting to filtered rows
const rows = computed((): ResultTableRow[] => {
  const sortState = state.sortState;

  if (!sortState || !sortState.direction) {
    return props.result.rows.map((item, index) => ({
      key: index,
      item,
    }));
  }

  const { columnIndex, direction } = sortState;
  const columnType = columns.value[columnIndex]?.columnType ?? "";

  return props.result.rows
    .map((item, index) => ({
      key: index,
      item,
    }))
    .sort((a, b) => {
      const result = compareQueryRowValues(
        columnType,
        a.item.values[columnIndex],
        b.item.values[columnIndex]
      );
      return direction === "asc" ? result : -result;
    });
});

// Computed property for result rows text to avoid accessing data.length twice
const resultRowsText = computed(() => {
  const length = rows.value.length;
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

provideSelectionContext({
  columns: columns,
  rows: rows,
});

const engine = computed(() => props.database.instanceResource.engine);

const showVisualizeButton = computed((): boolean => {
  return (
    (engine.value === Engine.POSTGRES ||
      engine.value === Engine.MSSQL ||
      engine.value === Engine.SPANNER) &&
    props.params.explain
  );
});

const visualizeExplain = async () => {
  let token: string | undefined;
  try {
    if (engine.value === Engine.POSTGRES || engine.value === Engine.SPANNER) {
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

.result-toolbar {
  scrollbar-width: thin;
  scrollbar-color: rgba(0, 0, 0, 0.2) transparent;
}

.result-toolbar::-webkit-scrollbar {
  width: 6px;
}

.result-toolbar::-webkit-scrollbar-track {
  background: transparent;
}

.result-toolbar::-webkit-scrollbar-thumb {
  background-color: rgba(0, 0, 0, 0.2);
  border-radius: 3px;
}

.result-toolbar::-webkit-scrollbar-thumb:hover {
  background-color: rgba(0, 0, 0, 0.3);
}
</style>
