<template>
  <div
    class="absolute inset-0 flex flex-col justify-center items-center z-10"
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
      <div class="w-full flex flex-row justify-start items-center p-2 gap-2">
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
        class="w-full h-full"
        :execute-params="executeParams"
        :result-set="selectedResultSet"
      />
    </template>
  </div>
</template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { Info } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useDatabaseV1Store, useTabStore } from "@/store";
import { ComposedDatabase } from "@/types";
import { ResultViewV1 } from "../EditorCommon/";

const tabStore = useTabStore();
const databaseStore = useDatabaseV1Store();
const selectedDatabase = ref<ComposedDatabase>();

const queriedDatabaseNames = computed(() =>
  Array.from(tabStore.currentTab.databaseQueryResultMap?.keys() || [])
);
const databases = computed(() => {
  return queriedDatabaseNames.value.map((databaseName) => {
    return databaseStore.getDatabaseByName(databaseName);
  });
});
const selectedResultSet = computed(() => {
  return tabStore.currentTab.databaseQueryResultMap?.get(
    selectedDatabase.value?.name || ""
  );
});
const executeParams = computed(() => tabStore.currentTab.executeParams);
const loading = computed(() => tabStore.currentTab.isExecutingSQL);

const isDatabaseQueryFailed = (database: ComposedDatabase) => {
  return tabStore.currentTab.databaseQueryResultMap?.get(database.name || "")
    ?.error;
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
