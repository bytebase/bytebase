<template>
  <NPopover
    v-if="show"
    placement="bottom-end"
    trigger="click"
    :disabled="disabled"
  >
    <template #trigger>
      <NButton
        :disabled="disabled"
        type="primary"
        size="small"
        style="--n-padding: 0 0.25rem"
      >
        <template #icon>
          <ChevronDown />
        </template>
      </NButton>
    </template>
    <template #default>
      <div class="flex flex-col gap-1">
        <div>
          <p class="mb-1 textinfolabel">
            {{ $t("data-source.select-query-data-source") }}
          </p>
          <NRadioGroup
            class="max-w-44"
            :value="selectedDataSourceId"
            @update:value="onDataSourceSelected"
          >
            <NTooltip
              v-for="ds in dataSources"
              :key="ds.id"
              :disabled="!Boolean(dataSourceUnaccessibleReason(ds))"
            >
              <template #trigger>
                <NRadio
                  class="w-full"
                  :value="ds.id"
                  :disabled="Boolean(dataSourceUnaccessibleReason(ds))"
                >
                  <div
                    class="max-w-36 flex flex-row justify-start items-center"
                  >
                    <span class="text-xs opacity-60 shrink-0">{{
                      readableDataSourceType(ds.type)
                    }}</span>
                    <span class="ml-1 truncate">{{ ds.username }}</span>
                  </div>
                </NRadio>
              </template>
              <p class="text-nowrap">
                {{ dataSourceUnaccessibleReason(ds) }}
              </p>
            </NTooltip>
          </NRadioGroup>
        </div>
        <div class="border-t pt-1 -mx-2" style="width: calc(100% + 1rem)">
          <ResultLimitSelect placement="right-start" trigger="hover">
            <template
              #default="{ resultRowsLimit }: { resultRowsLimit: number }"
            >
              <NButton
                quaternary
                style="justify-content: start; --n-padding: 0 8px; width: 100%"
              >
                {{
                  $t("sql-editor.result-limit.n-rows", { n: resultRowsLimit })
                }}
              </NButton>
            </template>
          </ResultLimitSelect>
        </div>
      </div>
    </template>
  </NPopover>
</template>

<script lang="ts" setup>
import { head, orderBy } from "lodash-es";
import { ChevronDown } from "lucide-vue-next";
import { NButton, NPopover, NRadioGroup, NRadio, NTooltip } from "naive-ui";
import { computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  useConnectionOfCurrentSQLEditorTab,
  usePolicyV1Store,
  useSQLEditorTabStore,
} from "@/store";
import { isValidDatabaseName } from "@/types";
import { DataSource, DataSourceType } from "@/types/proto/v1/instance_service";
import {
  DataSourceQueryPolicy_Restriction,
  PolicyType,
} from "@/types/proto/v1/org_policy_service";
import { readableDataSourceType } from "@/utils";
import { getAdminDataSourceRestrictionOfDatabase } from "@/utils";
import ResultLimitSelect from "./ResultLimitSelect.vue";

defineProps<{
  disabled?: boolean;
}>();

const { t } = useI18n();
const tabStore = useSQLEditorTabStore();
const { connection, database } = useConnectionOfCurrentSQLEditorTab();
const policyStore = usePolicyV1Store();

const show = computed(() => {
  return tabStore.currentTab?.mode !== "ADMIN";
});

const adminDataSourceRestriction = computed(() => {
  if (!database.value) {
    return {
      environmentPolicy:
        DataSourceQueryPolicy_Restriction.RESTRICTION_UNSPECIFIED,
      projectPolicy: DataSourceQueryPolicy_Restriction.RESTRICTION_UNSPECIFIED,
    };
  }
  return getAdminDataSourceRestrictionOfDatabase(database.value);
});

const selectedDataSourceId = computed(() => {
  return connection.value.dataSourceId;
});

const dataSources = computed(() => {
  return orderBy(database.value.instanceResource.dataSources, "type");
});

const dataSourceUnaccessibleReason = (
  dataSource: DataSource
): string | undefined => {
  if (dataSource.type === DataSourceType.ADMIN) {
    if (
      adminDataSourceRestriction.value.environmentPolicy ===
        DataSourceQueryPolicy_Restriction.DISALLOW ||
      adminDataSourceRestriction.value.projectPolicy ===
        DataSourceQueryPolicy_Restriction.DISALLOW
    ) {
      return t(
        "sql-editor.query-context.admin-data-source-is-disallowed-to-query"
      );
    }
    const readOnlyDataSources = dataSources.value.filter(
      (ds) => ds.type === DataSourceType.READ_ONLY
    );
    if (
      readOnlyDataSources.length > 0 &&
      (adminDataSourceRestriction.value.environmentPolicy ===
        DataSourceQueryPolicy_Restriction.FALLBACK ||
        adminDataSourceRestriction.value.projectPolicy ===
          DataSourceQueryPolicy_Restriction.FALLBACK)
    ) {
      return t(
        "sql-editor.query-context.admin-data-source-is-disallowed-to-query-when-read-only-data-source-is-available"
      );
    }
  }

  return undefined;
};

const onDataSourceSelected = (dataSourceId?: string) => {
  tabStore.updateCurrentTab({
    connection: {
      ...connection.value,
      dataSourceId: dataSourceId,
    },
  });
};

watch(
  () => selectedDataSourceId.value,
  () => {
    // If current connection has data source, skip initial selection.
    if (selectedDataSourceId.value) {
      return;
    }

    const readOnlyDataSources = dataSources.value.filter(
      (dataSource) => dataSource.type === DataSourceType.READ_ONLY
    );
    // Default set the first read only data source as selected.
    if (readOnlyDataSources.length > 0) {
      onDataSourceSelected(readOnlyDataSources[0].id);
    } else {
      onDataSourceSelected(head(dataSources.value)?.id);
    }
  },
  {
    immediate: true,
  }
);

watch(
  () => database.value.name,
  async () => {
    if (!isValidDatabaseName(database.value.name)) {
      return;
    }
    await policyStore.getOrFetchPolicyByParentAndType({
      parentPath: database.value.effectiveEnvironment,
      policyType: PolicyType.DATA_SOURCE_QUERY,
    });
    await policyStore.getOrFetchPolicyByParentAndType({
      parentPath: database.value.project,
      policyType: PolicyType.DATA_SOURCE_QUERY,
    });
  },
  {
    immediate: true,
  }
);
</script>
