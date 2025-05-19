<template>
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
            currentTab?.mode !== 'ADMIN' &&
            data.length === editorStore.resultRowsLimit
          "
          class="ml-2 whitespace-nowrap text-sm text-gray-500"
        >
          <span>-</span>
          <span class="ml-2">{{ $t("sql-editor.rows-upper-limit") }}</span>
        </span>
      </div>
      <div class="flex justify-between items-center shrink-0 gap-x-3">
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
        <template v-if="!disallowExportQueryData">
          <DataExportButton
            v-if="result.allowExport"
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
            size="small"
            @click="handleRequestExport"
          >
            {{ $t("quick-action.request-export-data") }}
            <ExternalLinkIcon class="w-4 h-auto ml-1 opacity-80" />
          </NButton>
        </template>
      </div>
      <SelectionCopyTooltips />
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
        :column-type-names="props.result.columnTypeNames"
      />
    </div>

    <div
      class="w-full flex items-center justify-between text-xs mt-1 gap-x-4 text-control-light"
    >
      <div class="flex-1 truncate">
        {{ result.statement }}
      </div>
      <div class="shrink-0 space-x-2">
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
import type { ColumnDef } from "@tanstack/vue-table";
import {
  getCoreRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useVueTable,
} from "@tanstack/vue-table";
import { useDebounceFn, useLocalStorage } from "@vueuse/core";
import dayjs from "dayjs";
import { isEmpty } from "lodash-es";
import { ExternalLinkIcon } from "lucide-vue-next";
import {
  NButton,
  NInput,
  NSwitch,
  NPagination,
  NTooltip,
  type SelectOption,
  NSelect,
} from "naive-ui";
import type { BinaryLike } from "node:crypto";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBAttention } from "@/bbkit";
import type { ExportOption } from "@/components/DataExportButton.vue";
import DataExportButton from "@/components/DataExportButton.vue";
import { DISMISS_PLACEHOLDER } from "@/plugins/ai/components/state";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  useSQLEditorTabStore,
  featureToRef,
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorStore,
  useAppFeature,
  pushNotification,
  usePolicyByParentAndType,
  useStorageStore,
  useSQLStore,
} from "@/store";
import type { ComposedDatabase, SQLEditorQueryParams } from "@/types";
import {
  DEBOUNCE_SEARCH_DELAY,
  isValidDatabaseName,
  isValidInstanceName,
} from "@/types";
import { ExportFormat } from "@/types/proto/v1/common";
import { Engine } from "@/types/proto/v1/common";
import { PolicyType } from "@/types/proto/v1/org_policy_service";
import { DatabaseChangeMode } from "@/types/proto/v1/setting_service";
import type {
  QueryResult,
  QueryRow,
  RowValue,
} from "@/types/proto/v1/sql_service";
import {
  compareQueryRowValues,
  createExplainToken,
  extractProjectResourceName,
  extractSQLRowValuePlain,
  generateIssueTitle,
  hasPermissionToCreateRequestGrantIssue,
  instanceV1HasStructuredQueryResult,
  isNullOrUndefined,
} from "@/utils";
import DataBlock from "./DataBlock.vue";
import DataTable from "./DataTable";
import { provideSelectionContext } from "./DataTable/common/selection-logic";
import EmptyView from "./EmptyView.vue";
import ErrorView from "./ErrorView";
import SelectionCopyTooltips from "./SelectionCopyTooltips.vue";
import { useSQLResultViewContext } from "./context";

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
  database?: ComposedDatabase;
  result: QueryResult;
  setIndex: number;
}>();

const state = reactive<LocalState>({
  search: "",
  vertical: false,
});

const { t } = useI18n();
const router = useRouter();
const { dark, keyword } = useSQLResultViewContext();
const tabStore = useSQLEditorTabStore();
const editorStore = useSQLEditorStore();
const appFeatureDisallowExport = useAppFeature(
  "bb.feature.sql-editor.disallow-export-query-data"
);
const databaseChangeMode = useAppFeature("bb.feature.database-change-mode");
const currentTab = computed(() => tabStore.currentTab);
const { instance: connectedInstance } = useConnectionOfCurrentSQLEditorTab();

const { policy: exportDataPolicy } = usePolicyByParentAndType(
  computed(() => ({
    parentPath: "",
    policyType: PolicyType.DATA_EXPORT,
  }))
);

const disallowExportQueryData = computed(() => {
  const disableDataExport =
    exportDataPolicy.value?.exportDataPolicy?.disable ?? false;

  return disableDataExport || appFeatureDisallowExport.value;
});

const viewMode = computed((): ViewMode => {
  const { result } = props;
  if (result.error && data.value.length === 0) {
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
  if (!isValidInstanceName(connectedInstance.value.name)) {
    return false;
  }
  return instanceV1HasStructuredQueryResult(connectedInstance.value);
});

const allowToRequestExportData = computed(() => {
  const { database } = props;
  if (!database) {
    return false;
  }

  // The current plan doesn't have access control feature.
  // Developers can not self-helped to request export.
  if (!featureToRef("bb.feature.access-control").value) {
    return false;
  }

  // SQL Editor Mode has no issues
  // So we cannot self-helped to request export either.
  if (databaseChangeMode.value === DatabaseChangeMode.EDITOR) {
    return false;
  }

  if (!isValidDatabaseName(database.name)) {
    return false;
  }

  return hasPermissionToCreateRequestGrantIssue(database);
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
      const columnType = props.result.columnTypeNames[index] as string;

      return {
        id: `${columnName}@${index}`,
        accessorFn: (item) => item.values[index],
        header: columnName,
        meta: {
          // Store column type in meta for easy access by other components
          columnType: columnType,
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

const data = computed(() => {
  const data = props.result.rows;
  const search = keyword.value.trim().toLowerCase();
  let temp = data;
  if (search) {
    temp = data.filter((item) => {
      return item.values.some((col) => {
        const value = extractSQLRowValuePlain(col);
        if (isNullOrUndefined(value)) {
          return false;
        }
        return String(value).toLowerCase().includes(search);
      });
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

const handleExportBtnClick = async (
  options: ExportOption,
  callback: (content: BinaryLike | Blob, filename: string) => void
) => {
  // If props.database is specified and it's not unknown database
  // the query is executed on database level
  // otherwise the query is executed on instance level, we should use the
  // `instanceId` from the tab's connection attributes
  const database =
    props.database && isValidDatabaseName(props.database.name)
      ? props.database.name
      : "";
  // use props.params.statement which is the "snapshot" of the query statement
  // not using props.result.statement because it might be rewritten by Query() API
  const statement = props.params.statement;
  const admin = tabStore.currentTab?.mode === "ADMIN";
  const limit = options.limit ?? (admin ? 0 : editorStore.resultRowsLimit);

  try {
    const content = await useSQLStore().exportData({
      name: database,
      // TODO(lj): support data source id similar to queries.
      dataSourceId: "",
      format: options.format,
      statement,
      limit,
      admin,
      password: options.password,
    });

    callback(
      content,
      `export-data.${dayjs(new Date()).format("YYYY-MM-DDTHH-mm-ss")}`
    );
  } catch (e) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.error"),
      description: String(e),
    });
  }
};

const handleRequestExport = async () => {
  if (!props.database) {
    return;
  }

  const database = props.database;
  const project = database.projectEntity;
  const issueType = "bb.issue.database.data.export";
  const sqlStorageKey = `bb.issues.sql.${uuidv4()}`;
  useStorageStore().put(sqlStorageKey, props.result.statement);
  const query: Record<string, any> = {
    template: issueType,
    name: generateIssueTitle(issueType, [database.databaseName]),
    databaseList: database.name,
    sqlStorageKey,
  };
  const route = router.resolve({
    name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
    params: {
      projectId: extractProjectResourceName(project.name),
      issueSlug: "create",
    },
    query,
  });
  window.open(route.fullPath, "_blank");
};

const showVisualizeButton = computed((): boolean => {
  return (
    connectedInstance.value.engine === Engine.POSTGRES && props.params.explain
  );
});

const visualizeExplain = () => {
  try {
    const { params, result } = props;
    const { statement } = params;
    if (!statement) return;

    const lines = result.rows.map((row) =>
      row.values.map((value) => String(extractSQLRowValuePlain(value)))
    );
    const explain = lines.map((line) => line[0]).join("\n");
    if (!explain) return;

    const token = createExplainToken(statement, explain);

    window.open(`/explain-visualizer.html?token=${token}`, "_blank");
  } catch {
    // nothing
  }
};

const handleChangePage = (page: number) => {
  table.setPageIndex(page - 1);
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
