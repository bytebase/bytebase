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
import type { SelectOption, SelectRenderLabel } from "naive-ui";
import { NButton, NEllipsis, NPopover, NSelect } from "naive-ui";
import { computed, h, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorTabStore,
} from "@/store";
import type { DataSource } from "@/types/proto/v1/instance_service";
import { DataSourceType } from "@/types/proto/v1/instance_service";

interface DataSourceSelectOption extends SelectOption {
  value: string;
  dataSource: DataSource;
}

const { t } = useI18n();
const tabStore = useSQLEditorTabStore();
const { connection, instance: selectedInstance } =
  useConnectionOfCurrentSQLEditorTab();
const selectedDataSourceId = computed(() => {
  return connection.value.dataSourceId;
});

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
      NPopover,
      {
        placement: "top",
        trigger: "hover",
        class: "ml-1",
      },
      {
        trigger: () => {
          return h(
            "span",
            {
              class: "ml-1",
            },
            [dataSource.username]
          );
        },
        default: () => {
          return h("div", {}, [dataSource.host, ":", dataSource.port]);
        },
      }
    ),
  ];
  return h(
    NEllipsis,
    {
      class: "w-full",
    },
    {
      default: () => children,
    }
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
