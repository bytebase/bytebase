<template>
  <div class="space-y-2">
    <div
      class="text-lg font-medium leading-7 text-main flex items-center justify-between"
    >
      <div class="flex items-center">
        <EnvironmentTabFilter
          :environment="state.environment"
          :include-all="true"
          @update:environment="state.environment = $event ?? UNKNOWN_ID"
        />
      </div>
      <NInputGroup style="width: auto">
        <InstanceSelect
          :instance="state.instance"
          :include-all="true"
          :filter="filterInstance"
          :environment="state.environment"
          @update:instance="
            state.instance = $event ? String($event) : String(UNKNOWN_ID)
          "
        />
        <SearchBox
          :value="state.keyword"
          :placeholder="$t('database.search-database')"
          @update:value="state.keyword = $event"
        />
      </NInputGroup>
    </div>

    <template v-if="databaseList.length > 0">
      <DatabaseV1Table
        mode="PROJECT"
        table-class="border"
        :database-list="filteredDatabaseList"
      />
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
import { NInputGroup } from "naive-ui";
import { uniqBy } from "lodash-es";

import { ComposedDatabase, Instance, UNKNOWN_ID } from "../types";
import { filterDatabaseV1ByKeyword } from "@/utils";
import { DatabaseV1Table } from "./v2";
import { EnvironmentTabFilter, InstanceSelect, SearchBox } from "./v2";

interface LocalState {
  environment: string;
  instance: string;
  keyword: string;
}

const props = defineProps({
  databaseList: {
    required: true,
    type: Object as PropType<ComposedDatabase[]>,
  },
});

const state = reactive<LocalState>({
  environment: String(UNKNOWN_ID),
  instance: String(UNKNOWN_ID),
  keyword: "",
});

const filteredDatabaseList = computed(() => {
  let list = [...props.databaseList];
  if (state.environment !== String(UNKNOWN_ID)) {
    list = list.filter(
      (db) => db.instanceEntity.environmentEntity.uid === state.environment
    );
  }
  if (state.instance !== String(UNKNOWN_ID)) {
    list = list.filter((db) => db.instanceEntity.uid === state.instance);
  }
  const keyword = state.keyword.trim().toLowerCase();
  if (keyword) {
    list = list.filter((db) =>
      filterDatabaseV1ByKeyword(db, keyword, [
        "name",
        "environment",
        "instance",
      ])
    );
  }
  return list;
});

const instanceList = computed(() => {
  return uniqBy(
    props.databaseList.map((db) => db.instanceEntity),
    (instance) => instance.uid
  );
});

const filterInstance = (instance: Instance) => {
  return (
    instanceList.value.findIndex((inst) => inst.uid === String(instance.id)) >=
    0
  );
};
</script>
