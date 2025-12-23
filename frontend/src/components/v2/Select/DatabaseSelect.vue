<template>
  <RemoteResourceSelector
    v-bind="$attrs"
    class="bb-database-select"
    :multiple="multiple"
    :disabled="disabled"
    :size="size"
    :value="value"
    :render-label="renderLabel"
    :render-tag="renderTag"
    :additional-options="additionalOptions"
    :search="handleSearch"
    :filter="filter"
    @update:value="(val) => $emit('update:value', val)"
  />
</template>

<script lang="tsx" setup>
import { computedAsync } from "@vueuse/core";
import { computed } from "vue";
import { RichDatabaseName } from "@/components/v2";
import { useDatabaseV1Store } from "@/store";
import { workspaceNamePrefix } from "@/store/modules/v1/common";
import type { ComposedDatabase } from "@/types";
import { type Engine } from "@/types/proto-es/v1/common_pb";
import RemoteResourceSelector from "./RemoteResourceSelector/index.vue";
import type {
  ResourceSelectOption,
  SelectSize,
} from "./RemoteResourceSelector/types";
import {
  getRenderLabelFunc,
  getRenderTagFunc,
} from "./RemoteResourceSelector/utils";

const props = withDefaults(
  defineProps<{
    value?: string | string[] | undefined;
    environmentName?: string;
    projectName?: string;
    allowedEngineTypeList?: Engine[];
    filter?: (database: ComposedDatabase) => boolean;
    multiple?: boolean;
    disabled?: boolean;
    size?: SelectSize;
    showInstance?: boolean;
  }>(),
  {
    // empty equals no limit.
    allowedEngineTypeList: () => [],
    showInstance: true,
  }
);

defineEmits<{
  (event: "update:value", value: string[] | string | undefined): void;
}>();

const databaseStore = useDatabaseV1Store();

const getOption = (db: ComposedDatabase) => {
  return {
    resource: db,
    value: db.name,
    label: db.databaseName,
  };
};

const additionalOptions = computedAsync(async () => {
  const options: ResourceSelectOption<ComposedDatabase>[] = [];

  let databaseNames: string[] = [];
  if (Array.isArray(props.value)) {
    databaseNames = props.value;
  } else if (props.value) {
    databaseNames = [props.value];
  }

  const databases = await databaseStore.batchGetOrFetchDatabases(databaseNames);
  options.push(...databases.map(getOption));

  return options;
}, []);

const handleSearch = async (params: {
  search: string;
  pageToken: string;
  pageSize: number;
}) => {
  const { databases, nextPageToken } = await databaseStore.fetchDatabases({
    parent: props.projectName ?? `${workspaceNamePrefix}-`,
    filter: {
      environment: props.environmentName,
      engines: props.allowedEngineTypeList,
      query: params.search,
    },
    pageToken: params.pageToken,
    pageSize: params.pageSize,
  });
  return {
    nextPageToken,
    options: databases.map(getOption),
  };
};

const customLabel = (database: ComposedDatabase, keyword: string) => {
  return (
    <RichDatabaseName
      database={database}
      keyword={keyword}
      showProject={false}
      showInstance={props.showInstance}
      showArrow={props.showInstance}
    />
  );
};

const renderLabel = computed(() => {
  return getRenderLabelFunc({
    multiple: props.multiple,
    customLabel,
    showResourceName: true,
  });
});

const renderTag = computed(() => {
  return getRenderTagFunc({
    multiple: props.multiple,
    disabled: props.disabled,
    size: props.size,
    customLabel,
  });
});
</script>
