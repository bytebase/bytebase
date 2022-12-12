<template>
  <div class="space-y-2">
    <div
      class="text-lg font-medium leading-7 text-main flex items-center justify-between"
    >
      {{ $t("common.database") }}
    </div>

    <template v-if="filteredDatabaseList.length > 0">
      <DatabaseTable mode="PROJECT" :database-list="filteredDatabaseList" />
    </template>
    <div v-else class="text-center textinfolabel">
      <i18n-t keypath="project.overview.no-db-prompt" tag="p">
        <template #newDb>
          <span class="text-main">{{ $t("quick-action.new-db") }}</span>
        </template>
        <template #transferInDb>
          <span class="text-main">{{ $t("quick-action.transfer-in-db") }}</span>
        </template>
      </i18n-t>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { reactive, PropType, computed } from "vue";

import type { Database, Project } from "../types";
import { filterDatabaseByKeyword } from "@/utils";
import DatabaseTable from "../components/DatabaseTable.vue";

interface LocalState {
  keyword: string;
}

const props = defineProps({
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
  databaseList: {
    required: true,
    type: Object as PropType<Database[]>,
  },
});

const state = reactive<LocalState>({
  keyword: "",
});

const filteredDatabaseList = computed(() => {
  return props.databaseList.filter((db) => {
    return filterDatabaseByKeyword(db, state.keyword, [
      "name",
      "environment",
      "instance",
    ]);
  });
});
</script>
