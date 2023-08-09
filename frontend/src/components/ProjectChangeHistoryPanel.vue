<template>
  <div class="flex flex-col space-y-4">
    <template v-if="state.changeHistorySectionList.length > 0">
      <div class="text-center textinfolabel">
        {{ $t("change-history.list-limit") }}
      </div>
      <ChangeHistoryTable
        :mode="'PROJECT'"
        :database-section-list="state.databaseSectionList"
        :history-section-list="state.changeHistorySectionList"
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
import { BBTableSectionDataSource } from "@/bbkit/types";
import { ChangeHistoryTable } from "@/components/ChangeHistory";
import { useChangeHistoryStore } from "@/store";
import { ComposedDatabase } from "@/types";
import { ChangeHistory } from "@/types/proto/v1/database_service";
import { databaseV1Slug } from "@/utils";

// Show at most 5 recent migration history for each database
const MAX_MIGRATION_HISTORY_COUNT = 5;

interface LocalState {
  databaseSectionList: ComposedDatabase[];
  changeHistorySectionList: BBTableSectionDataSource<ChangeHistory>[];
}

const props = defineProps({
  databaseList: {
    required: true,
    type: Object as PropType<ComposedDatabase[]>,
  },
});

const changeHistoryStore = useChangeHistoryStore();

const state = reactive<LocalState>({
  databaseSectionList: [],
  changeHistorySectionList: [],
});

const fetchChangeHistory = async (databaseList: ComposedDatabase[]) => {
  state.databaseSectionList = [];
  state.changeHistorySectionList = [];
  for (const database of databaseList) {
    const changeHistoryList = await changeHistoryStore.fetchChangeHistoryList({
      parent: database.name,
      pageSize: MAX_MIGRATION_HISTORY_COUNT,
    });
    if (changeHistoryList.length > 0) {
      state.databaseSectionList.push(database);

      const title = `${database.databaseName} (${database.instanceEntity.environmentEntity.title})`;
      const index = state.changeHistorySectionList.findIndex(
        (item: BBTableSectionDataSource<ChangeHistory>) => {
          return item.title == title;
        }
      );
      const newItem = {
        title: title,
        link: `/db/${databaseV1Slug(database)}#change-history`,
        list: changeHistoryList,
      };
      if (index >= 0) {
        state.changeHistorySectionList[index] = newItem;
      } else {
        state.changeHistorySectionList.push(newItem);
      }
    }
  }
};

const prepareChangeHistoryList = () => {
  fetchChangeHistory(props.databaseList);
};
watchEffect(prepareChangeHistoryList);
</script>
