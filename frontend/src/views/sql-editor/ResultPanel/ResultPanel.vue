<template>
  <NConfigProvider v-bind="naiveUIConfig" :class="dark && 'dark bg-dark-bg'">
    <div
      class="w-full flex flex-col justify-center items-center z-10"
      :class="loading && 'bg-white/80 dark:bg-black/80'"
    >
      <template v-if="loading">
        <BBSpin />
        {{ $t("sql-editor.loading-data") }}
      </template>
      <template v-else-if="!selectedResultSet">
        {{ $t("sql-editor.table-empty-placeholder") }}
      </template>
      <template v-else>
        <div
          v-if="databases.length > 1"
          class="w-full flex flex-row justify-start items-center p-2 gap-2"
        >
          <NTooltip
            v-for="database in databases"
            :key="database.name"
            trigger="hover"
          >
            <template #trigger>
              <NButton
                :type="selectedDatabase === database ? 'primary' : 'default'"
                @click="selectedDatabase = database"
              >
                <InstanceV1EngineIcon
                  :instance="database.instanceEntity"
                  :tooltip="false"
                />
                <span class="mx-2 opacity-60">{{
                  database.effectiveEnvironmentEntity.title
                }}</span>
                <span>{{ database.databaseName }}</span>
                <Info
                  v-if="isDatabaseQueryFailed(database)"
                  class="ml-2 text-yellow-600 w-4 h-auto"
                />
              </NButton>
            </template>
            {{ database.instanceEntity.title }}
          </NTooltip>
        </div>
        <ResultViewV1
          :key="selectedDatabase?.name"
          class="w-full h-full"
          :execute-params="executeParams"
          :result-set="selectedResultSet"
        />
      </template>
    </div>
  </NConfigProvider>
</template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { Info } from "lucide-vue-next";
import { NButton, NTooltip, NConfigProvider, darkTheme } from "naive-ui";
import { computed, ref, watch } from "vue";
import { darkThemeOverrides } from "@/../naive-ui.config";
import { useDatabaseV1Store } from "@/store";
import { ComposedDatabase, ExecuteParams, SQLResultSetV1 } from "@/types";
import { ResultViewV1 } from "../EditorCommon/";

const props = defineProps<{
  loading?: boolean;
  databaseQueryResultMap?: Map<string, SQLResultSetV1>;
  executeParams?: ExecuteParams;
  dark?: boolean;
}>();

const databaseStore = useDatabaseV1Store();
const selectedDatabase = ref<ComposedDatabase>();

const naiveUIConfig = computed(() => {
  if (props.dark) {
    return { theme: darkTheme, themeOverrides: darkThemeOverrides.value };
  }
  return {};
});

const queriedDatabaseNames = computed(() =>
  Array.from(props.databaseQueryResultMap?.keys() || [])
);
const databases = computed(() => {
  return queriedDatabaseNames.value.map((databaseName) => {
    return databaseStore.getDatabaseByName(databaseName);
  });
});
const selectedResultSet = computed(() => {
  return props.databaseQueryResultMap?.get(selectedDatabase.value?.name || "");
});

const isDatabaseQueryFailed = (database: ComposedDatabase) => {
  return props.databaseQueryResultMap?.get(database.name || "")?.error;
};

// Auto select the first database when the databases are ready.
watch(
  () => databases.value,
  () => {
    selectedDatabase.value = head(databases.value);
  },
  {
    immediate: true,
  }
);
</script>
