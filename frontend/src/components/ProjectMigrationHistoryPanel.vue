<template>
  <div class="flex flex-col space-y-4">
    <template v-if="state.migrationHistorySectionList.length > 0">
      <div class="text-center textinfolabel">
        {{ $t("change-history.list-limit") }}
      </div>
      <MigrationHistoryTable
        :mode="'PROJECT'"
        :database-section-list="state.databaseSectionList"
        :history-section-list="state.migrationHistorySectionList"
      />
    </template>
    <template v-else>
      <!-- This example requires Tailwind CSS v2.0+ -->
      <div class="text-center">
        <heroicons-outline:inbox class="mx-auto w-16 h-16 text-control-light" />
        <h3 class="mt-2 text-sm font-medium text-main">
          {{ $t("change-history.no-history-in-project") }}
        </h3>
        <p class="mt-1 text-sm text-control-light">
          {{ $t("change-history.recording-info") }}
        </p>
      </div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { PropType, reactive, watchEffect } from "vue";
import MigrationHistoryTable from "../components/MigrationHistoryTable.vue";
import { ComposedDatabase, MigrationHistory } from "../types";
import { BBTableSectionDataSource } from "../bbkit/types";
import { databaseV1Slug } from "../utils";
import { useLegacyInstanceStore } from "@/store";

// Show at most 5 recent migration history for each database
const MAX_MIGRATION_HISTORY_COUNT = 5;

interface LocalState {
  databaseSectionList: ComposedDatabase[];
  migrationHistorySectionList: BBTableSectionDataSource<MigrationHistory>[];
}

const props = defineProps({
  databaseList: {
    required: true,
    type: Object as PropType<ComposedDatabase[]>,
  },
});

const instanceStore = useLegacyInstanceStore();

const state = reactive<LocalState>({
  databaseSectionList: [],
  migrationHistorySectionList: [],
});

const fetchMigrationHistory = (databaseList: ComposedDatabase[]) => {
  state.databaseSectionList = [];
  state.migrationHistorySectionList = [];
  for (const database of databaseList) {
    instanceStore
      .fetchMigrationHistory({
        instanceId: Number(database.instanceEntity.uid),
        databaseName: database.databaseName,
        limit: MAX_MIGRATION_HISTORY_COUNT,
      })
      .then((migrationHistoryList: MigrationHistory[]) => {
        if (migrationHistoryList.length > 0) {
          state.databaseSectionList.push(database);

          const title = `${database.databaseName} (${database.instanceEntity.environmentEntity.title})`;
          const index = state.migrationHistorySectionList.findIndex(
            (item: BBTableSectionDataSource<MigrationHistory>) => {
              return item.title == title;
            }
          );
          const newItem = {
            title: title,
            link: `/db/${databaseV1Slug(database)}#change-history`,
            list: migrationHistoryList,
          };
          if (index >= 0) {
            state.migrationHistorySectionList[index] = newItem;
          } else {
            state.migrationHistorySectionList.push(newItem);
          }
        }
      });
  }
};

const prepareMigrationHistoryList = () => {
  fetchMigrationHistory(props.databaseList);
};
watchEffect(prepareMigrationHistoryList);
</script>
