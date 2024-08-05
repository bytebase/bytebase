<template>
  <NPopover placement="bottom" trigger="click">
    <template #trigger>
      <NButton type="primary" class="!px-1" size="small">
        <template #icon>
          <ChevronDown />
        </template>
      </NButton>
    </template>
    <div>
      <p class="mb-1 textinfolabel">
        {{ $t("data-source.select-data-source") }}
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
              <div class="max-w-36 flex flex-row justify-start items-center">
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

const { t } = useI18n();
const tabStore = useSQLEditorTabStore();
const { connection, database } = useConnectionOfCurrentSQLEditorTab();
const policyStore = usePolicyV1Store();

const adminDataSourceRestriction = computed(() => {
  if (!database.value) {
    return {
      environmentPolicy:
        DataSourceQueryPolicy_Restriction.RESTRICTION_UNSPECIFIED,
      projectPolicy: DataSourceQueryPolicy_Restriction.RESTRICTION_UNSPECIFIED,
    };
  }

  const projectLevelPolicy = policyStore.getPolicyByParentAndType({
    parentPath: database.value.project,
    policyType: PolicyType.DATA_SOURCE_QUERY,
  });
  const projectLevelAdminDSRestriction =
    projectLevelPolicy?.dataSourceQueryPolicy?.adminDataSourceRestriction;
  const envLevelPolicy = policyStore.getPolicyByParentAndType({
    parentPath: database.value.effectiveEnvironment,
    policyType: PolicyType.DATA_SOURCE_QUERY,
  });
  const envLevelAdminDSRestriction =
    envLevelPolicy?.dataSourceQueryPolicy?.adminDataSourceRestriction;
  return {
    environmentPolicy:
      envLevelAdminDSRestriction ??
      DataSourceQueryPolicy_Restriction.RESTRICTION_UNSPECIFIED,
    projectPolicy:
      projectLevelAdminDSRestriction ??
      DataSourceQueryPolicy_Restriction.RESTRICTION_UNSPECIFIED,
  };
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

const readableDataSourceType = (type: DataSourceType): string => {
  if (type === DataSourceType.ADMIN) {
    return t("data-source.admin");
  } else if (type === DataSourceType.READ_ONLY) {
    return t("data-source.read-only");
  } else {
    return "Unknown";
  }
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
