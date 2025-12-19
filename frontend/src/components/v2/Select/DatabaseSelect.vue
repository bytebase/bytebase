<template>
  <RemoteResourceSelector
    ref="databaseSelectRef"
    v-bind="$attrs"
    class="bb-database-select"
    :multiple="multiple"
    :value="databaseName"
    :values="databaseNames"
    :custom-label="renderLabel"
    :additional-data="additionalData"
    :search="handleSearch"
    :get-option="getOption"
    :filter="filter"
    @update:value="(val) => $emit('update:database-name', val)"
    @update:values="(val) => $emit('update:database-names', val)"
  />
</template>

<script lang="ts" setup>
import { computedAsync } from "@vueuse/core";
import { h } from "vue";
import { useDatabaseV1Store } from "@/store";
import { workspaceNamePrefix } from "@/store/modules/v1/common";
import type { ComposedDatabase } from "@/types";
import { isValidDatabaseName } from "@/types";
import { type Engine } from "@/types/proto-es/v1/common_pb";
import { instanceV1Name, supportedEngineV1List } from "@/utils";
import { InstanceV1EngineIcon } from "../Model";
import RemoteResourceSelector from "./RemoteResourceSelector.vue";

const props = withDefaults(
  defineProps<{
    databaseName?: string; // UNKNOWN_DATABASE_NAME stands for "ALL"
    databaseNames?: string[];
    environmentName?: string;
    projectName?: string;
    allowedEngineTypeList?: Engine[];
    filter?: (database: ComposedDatabase) => boolean;
    multiple?: boolean;
    clearable?: boolean;
  }>(),
  {
    databaseName: undefined,
    databaseNames: undefined,
    environmentName: undefined,
    projectName: undefined,
    allowedEngineTypeList: () => supportedEngineV1List(),
    filter: undefined,
    multiple: false,
    clearable: false,
  }
);

const emit = defineEmits<{
  (event: "update:database-name", value: string | undefined): void;
  (event: "update:database-names", value: string[]): void;
}>();

const databaseStore = useDatabaseV1Store();

const additionalData = computedAsync(async () => {
  const data = [];

  let databaseNames: string[] = [];
  if (props.databaseName) {
    databaseNames = [props.databaseName];
  } else if (props.databaseNames) {
    databaseNames = props.databaseNames;
  }

  for (const databaseName of databaseNames) {
    if (isValidDatabaseName(databaseName)) {
      const db = await databaseStore.getOrFetchDatabaseByName(databaseName);
      data.push(db);
    }
  }

  return data;
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
  return { nextPageToken, data: databases };
};

const getOption = (database: ComposedDatabase) => ({
  value: database.name,
  label: database.databaseName,
});

const renderLabel = (database: ComposedDatabase) => {
  const children = [h("div", {}, [database.databaseName])];
  if (isValidDatabaseName(database.name)) {
    // prefix engine icon
    children.unshift(
      h(InstanceV1EngineIcon, {
        class: "mr-1",
        instance: database.instanceResource,
      })
    );
    // suffix engine name
    children.push(
      h(
        "div",
        {
          class: "text-xs opacity-60 ml-1",
        },
        [`(${instanceV1Name(database.instanceResource)})`]
      )
    );
  }
  return h(
    "div",
    {
      class: "w-full flex flex-row justify-start items-center truncate",
    },
    children
  );
};
</script>
