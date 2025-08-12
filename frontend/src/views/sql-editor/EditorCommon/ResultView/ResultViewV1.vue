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
        />
        <SingleResultViewV1
          v-else
          :params="executeParams"
          :database="database"
          :result="resultSet.results[0]"
          :set-index="0"
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
                  <div class="flex items-center space-x-2">
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
            />
            <SingleResultViewV1
              v-else
              :params="executeParams"
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
          @execute="$emit('execute', $event)"
        >
          <template #suffix>
            <RequestQueryButton
              v-if="showRequestQueryButton"
              :database-resource="missingResource!"
            />
            <SyncDatabaseButton
              v-else-if="resultSet.error.includes('resource not found')"
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

    <Drawer :show="!!detail" @close="detail = undefined">
      <DetailPanel />
    </Drawer>
  </NConfigProvider>
</template>

<script lang="ts" setup>
import { Code } from "@connectrpc/connect";
import { Info } from "lucide-vue-next";
import {
  darkTheme,
  NConfigProvider,
  NTabPane,
  NTabs,
  NTooltip,
} from "naive-ui";
import { computed, ref, toRef } from "vue";
import { useI18n } from "vue-i18n";
import { darkThemeOverrides } from "@/../naive-ui.config";
import { BBSpin } from "@/bbkit";
import SyncDatabaseButton from "@/components/DatabaseDetail/SyncDatabaseButton.vue";
import { parseStringToResource } from "@/components/GrantRequestPanel/DatabaseResourceForm/common";
import { Drawer } from "@/components/v2";
import {
  useAppFeature,
  useConnectionOfCurrentSQLEditorTab,
  usePolicyV1Store,
} from "@/store";
import type {
  ComposedDatabase,
  DatabaseResource,
  SQLEditorQueryParams,
  SQLResultSetV1,
} from "@/types";
import { PolicyType } from "@/types/proto-es/v1/org_policy_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import { provideBinaryFormatContext } from "./DataTable/binary-format-store";
import DetailPanel from "./DetailPanel";
import EmptyView from "./EmptyView.vue";
import ErrorView from "./ErrorView";
import RequestQueryButton from "./RequestQueryButton.vue";
import SingleResultViewV1 from "./SingleResultViewV1.vue";
import type { SQLResultViewContext } from "./context";
import { provideSQLResultViewContext } from "./context";

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
const { instance, database: connectedDatabase } =
  useConnectionOfCurrentSQLEditorTab();
const disallowRequestQuery = useAppFeature(
  "bb.feature.sql-editor.disallow-request-query"
);
const keyword = ref("");
const detail: SQLResultViewContext["detail"] = ref(undefined);

provideBinaryFormatContext(computed(() => props.contextId));

const missingResource = computed((): DatabaseResource | undefined => {
  if (props.resultSet?.status !== Code.PermissionDenied) {
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
  return !disallowRequestQuery.value && missingResource.value;
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
  if (hasWorkspacePermissionV2("bb.sql.admin")) {
    // `disableCopyDataPolicy` is only applicable to workspace developers.
    return false;
  }

  let environment = instance.value.environment;
  if (props.database) {
    const projectLevelPolicy = policyStore.getPolicyByParentAndType({
      parentPath: props.database?.project,
      policyType: PolicyType.DISABLE_COPY_DATA,
    });
    if (
      projectLevelPolicy?.policy?.case === "disableCopyDataPolicy" &&
      projectLevelPolicy.policy.value.active
    ) {
      return true;
    }
    // If the database is provided, use its effective environment.
    environment = props.database.effectiveEnvironment;
  }

  // Check if the environment has a policy that disables copying data.
  if (environment) {
    const policy = policyStore.getPolicyByParentAndType({
      parentPath: environment,
      policyType: PolicyType.DISABLE_COPY_DATA,
    });
    if (
      policy?.policy?.case === "disableCopyDataPolicy" &&
      policy.policy.value.active
    ) {
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
  keyword,
  detail,
});
</script>
