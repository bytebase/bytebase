<template>
  <NConfigProvider
    v-bind="naiveUIConfig"
    class="relative flex flex-col justify-start items-start pb-1 overflow-y-auto"
    :class="dark && 'dark bg-dark-bg'"
  >
    <template v-if="executeParams && resultSet && !showPlaceholder">
      <template v-if="viewMode === 'SINGLE-RESULT'">
        <ErrorView
          v-if="resultSet.results[0]?.error"
          :error="resultSet.results[0]?.error"
          :execute-params="executeParams"
          :result-set="resultSet"
          @execute="$emit('execute', $event)"
        >
          <template #suffix>
            <RequestQueryButton
              v-if="showRequestQueryButton"
              :database-resources="missingResources"
            />
          </template>
        </ErrorView>
        <SingleResultViewV1
          v-else
          :params="executeParams"
          :database="database"
          :result="resultSet.results[0]"
          :set-index="0"
          :show-export="!effectiveQueryDataPolicy.disableExport"
          :maximum-export-count="effectiveQueryDataPolicy.maximumResultRows"
          @export="handleExportBtnClick"
        />
      </template>
      <template v-else-if="viewMode === 'MULTI-RESULT'">
        <NTabs
          type="line"
          size="small"
          class="flex-1 flex flex-col overflow-hidden"
          style="--n-tab-padding: 4px 12px"
        >
          <NTabPane
            v-for="(result, i) in filteredResults"
            :key="i"
            :name="tabName(i)"
            class="flex-1 flex flex-col overflow-hidden"
          >
            <template #tab>
              <NTooltip>
                <template #trigger>
                  <div class="flex items-center gap-x-2 mb-1">
                    <span>{{ tabName(i) }}</span>
                    <Info
                      v-if="result.error"
                      class="text-yellow-600 w-4 h-auto"
                    />
                  </div>
                </template>
                {{ result.statement }}
              </NTooltip>
            </template>
            <ErrorView
              v-if="result.error"
              :error="result.error"
              :execute-params="executeParams"
              :result-set="resultSet"
              @execute="$emit('execute', $event)"
            >
              <template #suffix>
                <RequestQueryButton
                  v-if="showRequestQueryButton"
                  :database-resources="missingResources"
                />
              </template>
            </ErrorView>
            <SingleResultViewV1
              v-else
              :params="executeParams"
              :database="database"
              :result="result"
              :set-index="i"
              :show-export="false"
              :maximum-export-count="effectiveQueryDataPolicy.maximumResultRows"
              @export="handleExportBtnClick"
            />
          </NTabPane>
          <template v-if="!effectiveQueryDataPolicy.disableExport" #suffix>
            <div class="mb-1">
              <DataExportButton
                size="small"
                :disabled="false"
                :support-formats="[
                  ExportFormat.CSV,
                  ExportFormat.JSON,
                  ExportFormat.SQL,
                  ExportFormat.XLSX,
                ]"
                :view-mode="'DRAWER'"
                :support-password="true"
                :maximum-export-count="effectiveQueryDataPolicy.maximumResultRows"
                @export="
                  ($event) =>
                    handleExportBtnClick({
                      ...$event,
                      statement: executeParams.statement,
                    })
                "
              >
                <template #form>
                  <NFormItem :label="$t('common.database')">
                    <DatabaseInfo :database="database" />
                  </NFormItem>
                </template>
              </DataExportButton>
            </div>
          </template>
        </NTabs>
      </template>
      <template v-else-if="viewMode === 'EMPTY'">
        <EmptyView />
      </template>
      <template v-else-if="viewMode === 'ERROR'">
        <ErrorView
          :error="resultSet.error"
          :execute-params="executeParams"
          :result-set="resultSet"
          @execute="$emit('execute', $event)"
        >
          <template #suffix>
            <RequestQueryButton
              v-if="showRequestQueryButton"
              :database-resources="missingResources"
            />
            <SyncDatabaseButton
              v-else-if="resultSet.error.includes('resource not found')"
              :type="'primary'"
              :text="true"
              :database="database"
            />
          </template>
        </ErrorView>
      </template>
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
      <template v-else-if="!resultSet">
        {{ $t("sql-editor.table-empty-placeholder") }}
      </template>
    </div>

    <Drawer v-if="detail && resultSet" :show="!!detail" @close="detail = undefined">
      <DetailPanel :result="viewMode === 'SINGLE-RESULT' ? resultSet.results[0] : filteredResults[detail.set]" />
    </Drawer>
  </NConfigProvider>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import { Info } from "lucide-vue-next";
import {
  darkTheme,
  NConfigProvider,
  NFormItem,
  NTabPane,
  NTabs,
  NTooltip,
} from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, ref, toRef } from "vue";
import { useI18n } from "vue-i18n";
import { darkThemeOverrides } from "@/../naive-ui.config";
import { BBSpin } from "@/bbkit";
import SyncDatabaseButton from "@/components/DatabaseDetail/SyncDatabaseButton.vue";
import DatabaseInfo from "@/components/DatabaseInfo.vue";
import type {
  DownloadContent,
  ExportOption,
} from "@/components/DataExportButton.vue";
import DataExportButton from "@/components/DataExportButton.vue";
import { parseStringToResource } from "@/components/GrantRequestPanel/DatabaseResourceForm/common";
import { Drawer } from "@/components/v2";
import {
  usePolicyV1Store,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useSQLStore,
} from "@/store";
import type {
  ComposedDatabase,
  DatabaseResource,
  SQLEditorQueryParams,
  SQLResultSetV1,
} from "@/types";
import { ExportFormat } from "@/types/proto-es/v1/common_pb";
import { ExportRequestSchema } from "@/types/proto-es/v1/sql_service_pb";

import type { SQLResultViewContext } from "./context";
import { provideSQLResultViewContext } from "./context";
import { provideBinaryFormatContext } from "./DataTable/common/binary-format-store";
import DetailPanel from "./DetailPanel";
import EmptyView from "./EmptyView.vue";
import ErrorView from "./ErrorView";
import RequestQueryButton from "./RequestQueryButton.vue";
import SingleResultViewV1 from "./SingleResultViewV1.vue";

type ViewMode = "SINGLE-RESULT" | "MULTI-RESULT" | "EMPTY" | "ERROR";

const props = withDefaults(
  defineProps<{
    executeParams: SQLEditorQueryParams;
    database: ComposedDatabase;
    resultSet?: SQLResultSetV1;
    loading?: boolean;
    dark?: boolean;
    contextId: string;
  }>(),
  {
    executeParams: undefined,
    resultSet: undefined,
    loading: false,
    dark: false,
  }
);

defineEmits<{
  (event: "execute", params: SQLEditorQueryParams): void;
}>();

const { t } = useI18n();
const policyStore = usePolicyV1Store();
const tabStore = useSQLEditorTabStore();
const { project } = storeToRefs(useSQLEditorStore());

const effectiveQueryDataPolicy = computed(() => {
  return policyStore.getEffectiveQueryDataPolicyForProject(project.value);
});

const detail: SQLResultViewContext["detail"] = ref(undefined);

provideBinaryFormatContext(computed(() => props.contextId));

const missingResources = computed((): DatabaseResource[] => {
  const resources: DatabaseResource[] = [];
  for (const result of props.resultSet?.results ?? []) {
    if (
      result.detailedError.case === "permissionDenied" &&
      result.detailedError.value?.resources.length > 0
    ) {
      for (const resourceString of result.detailedError.value.resources) {
        const resource = parseStringToResource(resourceString);
        if (resource) {
          resources.push(resource);
        }
      }
    }
  }
  return resources;
});

const showRequestQueryButton = computed(() => {
  return missingResources.value.length > 0;
});

const viewMode = computed((): ViewMode => {
  const { resultSet } = props;
  if (!resultSet) {
    return "EMPTY";
  }
  const { results = [], error } = resultSet;
  if (error) {
    return "ERROR";
  }
  if (results.length === 0) {
    return "EMPTY";
  }
  if (results.length === 1) {
    return "SINGLE-RESULT";
  }
  return "MULTI-RESULT";
});

const naiveUIConfig = computed(() => {
  if (props.dark) {
    return { theme: darkTheme, themeOverrides: darkThemeOverrides.value };
  }
  return {};
});

const showPlaceholder = computed(() => {
  if (viewMode.value === "ERROR") return false;
  if (!props.resultSet) return true;
  if (props.loading) return true;
  return false;
});

const tabName = (index: number) => {
  return `${t("common.query")} #${index + 1}`;
};

const disallowCopyingData = computed(() => {
  if (
    policyStore.getQueryDataPolicyByParent(props.database.project)
      .disableCopyData
  ) {
    return true;
  }
  // If the database is provided, use its effective environment.
  const environment = props.database.effectiveEnvironment;

  // Check if the environment has a policy that disables copying data.
  if (environment) {
    if (policyStore.getQueryDataPolicyByParent(environment).disableCopyData) {
      return true;
    }
  }
  return false;
});

const filteredResults = computed(() => {
  if (!props.resultSet) {
    return []; // If resultSet is undefined, return an empty array
  }

  // Skip SET commands when displaying results
  return props.resultSet.results.filter((result) => {
    return !result.statement.trim().toUpperCase().startsWith("SET");
  });
});

provideSQLResultViewContext({
  dark: toRef(props, "dark"),
  disallowCopyingData,
  detail,
});

const handleExportBtnClick = async ({
  options,
  resolve,
  reject,
  statement,
}: {
  statement: string;
  options: ExportOption;
  reject: (reason?: unknown) => void;
  resolve: (content: DownloadContent[]) => void;
}) => {
  const admin = tabStore.currentTab?.mode === "ADMIN";
  const limit = options.limit;

  try {
    const content = await useSQLStore().exportData(
      create(ExportRequestSchema, {
        name: props.database.name,
        dataSourceId: props.executeParams.connection.dataSourceId ?? "",
        format: options.format,
        statement,
        limit,
        admin,
        password: options.password,
        schema: props.executeParams.connection.schema,
      })
    );

    resolve([
      {
        content,
        // the download file is always zip file.
        filename: `${props.database.databaseName}.${dayjs(new Date()).format("YYYY-MM-DDTHH-mm-ss")}.zip`,
      },
    ]);
  } catch (e) {
    reject(e);
  }
};
</script>
