<template>
  <NConfigProvider
    v-bind="naiveUIConfig"
    class="relative flex flex-col justify-start items-start p-2 pb-1 overflow-y-auto"
    :class="dark && 'dark bg-dark-bg'"
  >
    <template v-if="executeParams && resultSet && !showPlaceholder">
      <template v-if="viewMode === 'SINGLE-RESULT'">
        <SingleResultViewV1
          :params="executeParams"
          :database="database"
          :sql-result-set="resultSet"
          :result="resultSet.results[0]"
          :set-index="0"
          :max-data-table-height="maxDataTableHeight"
        />
      </template>
      <template v-else-if="viewMode === 'MULTI-RESULT'">
        <NTabs
          type="card"
          size="small"
          class="flex-1 flex flex-col overflow-hidden"
          style="--n-tab-padding: 4px 12px"
        >
          <NTabPane
            v-for="(result, i) in resultSet.results"
            :key="i"
            :name="tabName(result, i)"
            class="flex-1 flex flex-col overflow-hidden"
          >
            <template #tab>
              <span>{{ tabName(result, i) }}</span>
              <Info
                v-if="result.error"
                class="ml-2 text-yellow-600 w-4 h-auto"
              />
            </template>
            <SingleResultViewV1
              :params="executeParams"
              :sql-result-set="resultSet"
              :database="database"
              :result="result"
              :set-index="i"
            />
          </NTabPane>
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
        >
          <template #suffix>
            <RequestQueryButton
              v-if="showRequestQueryButton"
              :database-resource="missingResource!"
            />
            <SyncDatabaseButton
              v-else-if="
                !disallowSyncSchema &&
                resultSet.error.includes('resource not found')
              "
              :type="'primary'"
              :text="true"
              :database="database ?? connectedDatabase"
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

    <Drawer v-model:show="detail.show" @close="detail.show = false">
      <DetailPanel v-if="detail.show" />
    </Drawer>
  </NConfigProvider>
</template>

<script lang="ts" setup>
import { Info } from "lucide-vue-next";
import { darkTheme, NConfigProvider, NTabs, NTabPane } from "naive-ui";
import { Status } from "nice-grpc-common";
import type { PropType } from "vue";
import { computed, ref, toRef } from "vue";
import { useI18n } from "vue-i18n";
import { darkThemeOverrides } from "@/../naive-ui.config";
import { BBSpin } from "@/bbkit";
import SyncDatabaseButton from "@/components/DatabaseDetail/SyncDatabaseButton.vue";
import { parseStringToResource } from "@/components/GrantRequestPanel/DatabaseResourceForm/common";
import { Drawer } from "@/components/v2";
import {
  hasFeature,
  useAppFeature,
  useConnectionOfCurrentSQLEditorTab,
  usePolicyV1Store,
} from "@/store";
import type {
  ComposedDatabase,
  SQLEditorQueryParams,
  SQLResultSetV1,
  DatabaseResource,
} from "@/types";
import { PolicyType } from "@/types/proto/v1/org_policy_service";
import type { QueryResult } from "@/types/proto/v1/sql_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import DetailPanel from "./DetailPanel";
import EmptyView from "./EmptyView.vue";
import ErrorView from "./ErrorView";
import RequestQueryButton from "./RequestQueryButton.vue";
import SingleResultViewV1 from "./SingleResultViewV1.vue";
import type { SQLResultViewContext } from "./context";
import { provideSQLResultViewContext } from "./context";

type ViewMode = "SINGLE-RESULT" | "MULTI-RESULT" | "EMPTY" | "ERROR";

const props = defineProps({
  executeParams: {
    type: Object as PropType<SQLEditorQueryParams>,
    default: undefined,
  },
  database: {
    type: Object as PropType<ComposedDatabase>,
    default: undefined,
  },
  resultSet: {
    type: Object as PropType<SQLResultSetV1>,
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
  maxDataTableHeight: {
    type: Number,
    default: undefined,
  },
});

const { t } = useI18n();
const policyStore = usePolicyV1Store();
const { instance, database: connectedDatabase } =
  useConnectionOfCurrentSQLEditorTab();
const disallowRequestQuery = useAppFeature(
  "bb.feature.sql-editor.disallow-request-query"
);
const disallowSyncSchema = useAppFeature(
  "bb.feature.sql-editor.disallow-sync-schema"
);
const keyword = ref("");
const detail: SQLResultViewContext["detail"] = ref({
  show: false,
  set: 0,
  row: 0,
  col: 0,
  table: undefined,
});

const missingResource = computed((): DatabaseResource | undefined => {
  if (props.resultSet?.status !== Status.PERMISSION_DENIED) {
    return;
  }
  const prefix = "permission denied to access resource: ";
  if (!props.resultSet.error.includes(prefix)) {
    return;
  }
  const resource = props.resultSet.error.split(prefix).pop();
  if (!resource) {
    return;
  }
  return parseStringToResource(resource);
});

const showRequestQueryButton = computed(() => {
  // Developer self-helped request query is guarded by "Access Control" feature
  return (
    hasFeature("bb.feature.access-control") &&
    !disallowRequestQuery.value &&
    missingResource.value
  );
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

const tabName = (result: QueryResult, index: number) => {
  return `${t("common.query")} #${index + 1}`;
};

const disallowCopyingData = computed(() => {
  if (hasWorkspacePermissionV2("bb.sql.admin")) {
    // `disableCopyDataPolicy` is only applicable to workspace developers.
    return false;
  }

  if (props.database) {
    const projectLevelPolicy = policyStore.getPolicyByParentAndType({
      parentPath: props.database?.project,
      policyType: PolicyType.DISABLE_COPY_DATA,
    });
    if (projectLevelPolicy?.disableCopyDataPolicy?.active) {
      return true;
    }
  }

  const policy = policyStore.getPolicyByParentAndType({
    parentPath: instance.value.environment,
    policyType: PolicyType.DISABLE_COPY_DATA,
  });
  if (policy?.disableCopyDataPolicy?.active) {
    return true;
  }
  return false;
});

provideSQLResultViewContext({
  dark: toRef(props, "dark"),
  disallowCopyingData,
  keyword,
  detail,
});
</script>
