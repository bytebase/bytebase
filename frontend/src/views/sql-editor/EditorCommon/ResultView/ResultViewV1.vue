<template>
  <NConfigProvider
    v-bind="naiveUIConfig"
    class="relative flex flex-col justify-start items-start p-2 pb-1"
    :class="dark && 'dark bg-dark-bg'"
  >
    <template v-if="executeParams && resultSet && !showPlaceholder">
      <template v-if="viewMode === 'SINGLE-RESULT'">
        <SingleResultViewV1
          :params="executeParams"
          :result="resultSet.results[0]"
          :set-index="0"
        />
      </template>
      <template v-else-if="viewMode === 'MULTI-RESULT'">
        <NTabs
          type="card"
          size="small"
          class="flex-1 flex flex-col overflow-hidden"
        >
          <NTabPane
            v-for="(result, i) in resultSet.results"
            :key="i"
            :name="tabName(result, i)"
            class="flex-1 flex flex-col overflow-hidden"
          >
            <SingleResultViewV1
              :params="executeParams"
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
        <ErrorView :error="resultSet.error">
          <template
            v-if="resultSet.status === Status.PERMISSION_DENIED"
            #suffix
          >
            <RequestQueryButton />
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

    <Drawer
      v-model:show="detail.show"
      :close-on-esc="true"
      :trap-focus="true"
      @close="detail.show = false"
    >
      <DetailPanel v-if="detail.show" :result-set="resultSet" />
    </Drawer>
  </NConfigProvider>
</template>

<script lang="ts" setup>
import { darkTheme, NConfigProvider, NTabs, NTabPane } from "naive-ui";
import { Status } from "nice-grpc-common";
import { computed, PropType, ref, toRef } from "vue";
import { useI18n } from "vue-i18n";
import { darkThemeOverrides } from "@/../naive-ui.config";
import { Drawer } from "@/components/v2";
import {
  useCurrentUserV1,
  useInstanceV1Store,
  usePolicyV1Store,
  useTabStore,
} from "@/store";
import { ExecuteConfig, ExecuteOption, SQLResultSetV1 } from "@/types";
import { PolicyType } from "@/types/proto/v1/org_policy_service";
import { QueryResult } from "@/types/proto/v1/sql_service";
import { hasWorkspacePermissionV1 } from "@/utils";
import DetailPanel from "./DetailPanel.vue";
import EmptyView from "./EmptyView.vue";
import ErrorView from "./ErrorView.vue";
import RequestQueryButton from "./RequestQueryButton.vue";
import SingleResultViewV1 from "./SingleResultViewV1.vue";
import { provideSQLResultViewContext, SQLResultViewContext } from "./context";

type ViewMode = "SINGLE-RESULT" | "MULTI-RESULT" | "EMPTY" | "ERROR";

const props = defineProps({
  executeParams: {
    type: Object as PropType<{
      query: string;
      config: ExecuteConfig;
      option?: Partial<ExecuteOption> | undefined;
    }>,
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
});

const { t } = useI18n();
const currentUser = useCurrentUserV1();
const connection = computed(() => useTabStore().currentTab.connection);
const keyword = ref("");
const detail: SQLResultViewContext["detail"] = ref({
  show: false,
  set: 0,
  row: 0,
  col: 0,
});

const viewMode = computed((): ViewMode => {
  const { resultSet } = props;
  if (!resultSet) {
    return "EMPTY";
  }
  const { results, error } = resultSet;
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
  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.admin-sql-editor",
      currentUser.value.userRole
    )
  ) {
    // `disableCopyDataPolicy` is only applicable to workspace developers.
    return false;
  }
  const instance = useInstanceV1Store().getInstanceByUID(
    connection.value.instanceId
  );
  const environment = instance.environment;
  const policy = usePolicyV1Store().getPolicyByParentAndType({
    parentPath: environment,
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
