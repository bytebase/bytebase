<template>
  <RemoteResourceSelector
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

<script lang="tsx" setup>
import { computedAsync } from "@vueuse/core";
import { RichDatabaseName } from "@/components/v2";
import { useDatabaseV1Store } from "@/store";
import { workspaceNamePrefix } from "@/store/modules/v1/common";
import type { ComposedDatabase } from "@/types";
import { isValidDatabaseName } from "@/types";
import { type Engine } from "@/types/proto-es/v1/common_pb";
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
    showInstance?: boolean;
  }>(),
  {
    databaseName: undefined,
    databaseNames: undefined,
    environmentName: undefined,
    projectName: undefined,
    // empty equals no limit.
    allowedEngineTypeList: () => [],
    filter: undefined,
    multiple: false,
    clearable: false,
    showInstance: true,
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

const renderLabel = (database: ComposedDatabase, keyword: string) => {
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
</script>
