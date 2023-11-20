<template>
  <NPopover placement="bottom-center" trigger="click">
    <template #trigger>
      <NButton type="primary" class="!px-1" size="small">
        <template #icon>
          <ChevronDown />
        </template>
      </NButton>
    </template>
    <div>
      <p class="mb-1">{{ $t("data-source.select-data-source") }}</p>
      <NSelect
        class="!w-40"
        :options="dataSourceOptions"
        :value="selectedDataSourceId"
        :render-label="renderLabel"
        @update:value="handleDataSourceSelect"
      />
    </div>
  </NPopover>
</template>

<script lang="ts" setup>
import { head, orderBy } from "lodash-es";
import { ChevronDown } from "lucide-vue-next";
import {
  NButton,
  NEllipsis,
  NPopover,
  NSelect,
  SelectOption,
  SelectRenderLabel,
} from "naive-ui";
import { computed, h, watch } from "vue";
import { useInstanceV1ByUID, useTabStore } from "@/store";
import { DataSource, DataSourceType } from "@/types/proto/v1/instance_service";

interface DataSourceSelectOption extends SelectOption {
  value: string;
  dataSource: DataSource;
}

const tabStore = useTabStore();
const connection = computed(() => tabStore.currentTab.connection);
const selectedDataSourceId = computed(() => {
  return connection.value.dataSourceId;
});

const { instance: selectedInstance } = useInstanceV1ByUID(
  computed(() => connection.value.instanceId)
);

const dataSources = computed(() => {
  return orderBy(selectedInstance.value?.dataSources ?? [], "type");
});

const dataSourceOptions = computed(() => {
  return dataSources.value.map((dataSource) => {
    return {
      value: dataSource.id,
      label: dataSource.username,
      dataSource: dataSource,
    };
  });
});

const renderLabel: SelectRenderLabel = (option) => {
  const { dataSource } = option as DataSourceSelectOption;
  if (!dataSource) {
    return;
  }

  const children = [
    h(
      "span",
      {
        class: "text-gray-400 text-sm",
      },
      [readableDataSourceType(dataSource.type)]
    ),
    h(
      "span",
      {
        class: "ml-1",
      },
      [dataSource.username]
    ),
  ];
  return h(
    NEllipsis,
    {
      class: "w-full",
    },
    children
  );
};

const handleDataSourceSelect = (dataSourceId?: string) => {
  tabStore.updateCurrentTab({
    connection: {
      ...connection.value,
      dataSourceId: dataSourceId,
    },
  });
};

const readableDataSourceType = (type: DataSourceType): string => {
  if (type === DataSourceType.ADMIN) {
    return "Admin";
  } else if (type === DataSourceType.READ_ONLY) {
    return "RO";
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
      handleDataSourceSelect(readOnlyDataSources[0].id);
    } else {
      handleDataSourceSelect(head(dataSources.value)?.id);
    }
  },
  {
    immediate: true,
  }
);
</script>
