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
        <NTooltip
          v-if="showRedisConfig"
          :disabled="
            selectedDataSource?.redisType === DataSource_RedisType.CLUSTER
          "
        >
          <template #trigger>
            <div class="border-t pt-1" style="">
              <p class="mb-1 textinfolabel">
                {{ $t("sql-editor.redis-command.self") }}
              </p>
              <NRadioGroup
                :disabled="
                  selectedDataSource?.redisType !== DataSource_RedisType.CLUSTER
                "
                v-model:value="redisCommandOption"
                class="max-w-44"
              >
                <NRadio :value="QueryOption_RedisRunCommandsOn.SINGLE_NODE">
                  {{ $t("sql-editor.redis-command.single-node") }}
                </NRadio>
                <NRadio :value="QueryOption_RedisRunCommandsOn.ALL_NODES">
                  {{ $t("sql-editor.redis-command.all-nodes") }}
                </NRadio>
              </NRadioGroup>
            </div>
          </template>
          {{ $t("sql-editor.redis-command.only-for-cluster") }}
        </NTooltip>
        <div class="border-t pt-1">
          <MaxRowCountSelect
            v-model:value="resultRowsLimit"
            :quaternary="true"
            :maximum-export-count="policy.maximumResultRows"
          />
        </div>
      </div>
    </template>
  </NPopover>
</template>

<script lang="ts" setup>
import { orderBy } from "lodash-es";
import { ChevronDown } from "lucide-vue-next";
import { NButton, NPopover, NRadio, NRadioGroup, NTooltip } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, watch, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import MaxRowCountSelect from "@/components/GrantRequestPanel/MaxRowCountSelect.vue";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorStore,
  useSQLEditorTabStore,
} from "@/store";
import { useQueryDataPolicy } from "@/store/modules/v1/policy";
import { isValidDatabaseName } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { DataSource } from "@/types/proto-es/v1/instance_service_pb";
import {
  DataSource_RedisType,
  DataSourceType,
} from "@/types/proto-es/v1/instance_service_pb";
import { QueryOption_RedisRunCommandsOn } from "@/types/proto-es/v1/sql_service_pb";
import {
  getInstanceResource,
  getValidDataSourceByPolicy,
  readableDataSourceType,
} from "@/utils";

defineProps<{
  disabled?: boolean;
}>();

const { t } = useI18n();
const tabStore = useSQLEditorTabStore();
const { connection, database } = useConnectionOfCurrentSQLEditorTab();

const { redisCommandOption, resultRowsLimit } = storeToRefs(
  useSQLEditorStore()
);

const { policy } = useQueryDataPolicy();

const show = computed(() => {
  return tabStore.currentTab?.mode !== "ADMIN";
});

const showRedisConfig = computed(() => {
  if (!database.value) {
    return false;
  }
  const instance = getInstanceResource(database.value);
  return instance.engine === Engine.REDIS;
});

const selectedDataSourceId = computed(() => {
  return connection.value.dataSourceId;
});

const selectedDataSource = computed(() => {
  const instance = getInstanceResource(database.value);
  return instance.dataSources.find(
    (ds) => ds.id === selectedDataSourceId.value
  );
});

const dataSources = computed(() => {
  return orderBy(getInstanceResource(database.value).dataSources, "type");
});

const dataSourceUnaccessibleReason = (
  dataSource: DataSource
): string | undefined => {
  if (policy.value.allowAdminDataSource) {
    return undefined;
  }
  if (dataSource.type !== DataSourceType.ADMIN) {
    return undefined;
  }

  const readOnlyDataSources = dataSources.value.filter(
    (ds) => ds.type === DataSourceType.READ_ONLY
  );
  if (readOnlyDataSources.length === 0) {
    return undefined;
  }

  return t("sql-editor.query-context.admin-data-source-is-disallowed-to-query");
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
  [() => selectedDataSourceId.value, () => database.value],
  async ([current, database]) => {
    if (!isValidDatabaseName(database.name)) return;
    if (!current) {
      const fixed = await getValidDataSourceByPolicy(database);
      onDataSourceSelected(fixed);
    }
  },
  {
    immediate: true,
  }
);

watchEffect(() => {
  if (resultRowsLimit.value > policy.value.maximumResultRows) {
    resultRowsLimit.value = policy.value.maximumResultRows;
  }
});
</script>
