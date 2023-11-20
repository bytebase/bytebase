<template>
  <div
    class="relative w-full h-full flex flex-col justify-start items-start z-10 overflow-x-hidden"
  >
    <template v-if="loading">
      <div
        class="w-full h-full flex flex-col justify-center items-center text-sm gap-y-1 bg-white/80 dark:bg-black/80"
      >
        <div class="flex flex-row gap-x-1">
          <BBSpin />
          <span>{{ $t("sql-editor.executing-query") }}</span>
          <span>-</span>
          <!-- use mono font to prevent the UI jitters frequently -->
          <span class="font-mono">{{ queryElapsedTime }}</span>
        </div>
        <div>
          <NButton size="small" @click="cancelQuery">
            {{ $t("common.cancel") }}
          </NButton>
        </div>
      </div>
    </template>
    <template v-else>
      <div
        v-if="batchQueryDatabases.length > 0"
        class="w-full flex flex-row justify-start items-center p-2 pb-0 gap-2 shrink-0 overflow-x-auto hide-scrollbar"
      >
        <NTooltip
          v-for="database in databases"
          :key="database.name"
          trigger="hover"
        >
          <template #trigger>
            <NButton
              secondary
              strong
              size="small"
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
                class="ml-1 text-yellow-600 w-4 h-auto"
              />
              <X
                class="ml-1 text-gray-400 w-4 h-auto hover:text-gray-600"
                @click.stop="handleCloseSingleResultView(database)"
              />
            </NButton>
          </template>
          {{ database.instanceEntity.title }}
        </NTooltip>
      </div>
      <template v-if="!selectedResultSet">
        <div
          class="w-full h-full flex flex-col justify-center items-center text-sm"
        >
          <span>{{ $t("sql-editor.table-empty-placeholder") }}</span>
        </div>
      </template>
      <ResultViewV1
        v-else
        class="w-full h-auto grow"
        :execute-params="executeParams"
        :result-set="selectedResultSet"
      />
    </template>
  </div>
</template>

<script lang="ts" setup>
import { useTimestamp } from "@vueuse/core";
import { head } from "lodash-es";
import { Info, X } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useDatabaseV1Store, useTabStore } from "@/store";
import { ComposedDatabase } from "@/types";
import { ResultViewV1 } from "../EditorCommon/";

const tabStore = useTabStore();
const databaseStore = useDatabaseV1Store();
const selectedDatabase = ref<ComposedDatabase>();

const batchQueryDatabases = computed(() => {
  return tabStore.currentTab.batchQueryContext?.selectedDatabaseNames || [];
});
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
const currentTimestampMS = useTimestamp();
const queryElapsedTime = computed(() => {
  if (!loading.value) return "";
  const tab = tabStore.currentTab;
  const { isExecutingSQL, queryContext } = tab;
  if (!isExecutingSQL) return "";
  if (!queryContext) return;
  const beginMS = queryContext.beginTimestampMS;
  const elapsedMS = currentTimestampMS.value - beginMS;
  return `${(elapsedMS / 1000).toFixed(1)}s`;
});

const isDatabaseQueryFailed = (database: ComposedDatabase) => {
  const resultSet = tabStore.currentTab.databaseQueryResultMap?.get(
    database.name || ""
  );
  // If there is any error in the result set, we consider the query failed.
  return resultSet?.error || resultSet?.results.find((result) => result.error);
};

const cancelQuery = () => {
  const { queryContext } = tabStore.currentTab;
  if (!queryContext) return;
  const { abortController } = queryContext;
  abortController?.abort();
};

const handleCloseSingleResultView = (database: ComposedDatabase) => {
  tabStore.currentTab.databaseQueryResultMap?.delete(database.name || "");
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
