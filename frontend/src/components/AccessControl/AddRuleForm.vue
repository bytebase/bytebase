<template>
  <div class="w-160 space-y-2 relative">
    <div>
      <BBTableSearch
        class="w-60"
        :placeholder="$t('database.search-database-name')"
        @change-text="(text: string) => (state.searchText = text)"
      />
    </div>

    <DatabaseTable
      mode="ALL_TINY"
      :bordered="true"
      :custom-click="true"
      :database-list="databaseList"
      :show-selection-column="true"
      @select-database="
          (db: Database) => toggleDatabaseSelection(db, !isDatabaseSelected(db))
        "
    >
      <template #selection-all="{ databaseList: renderedDatabaseList }">
        <input
          v-if="renderedDatabaseList.length > 0"
          type="checkbox"
          class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
          v-bind="getAllSelectionState(renderedDatabaseList)"
          @input="
            toggleAllDatabasesSelection(
              renderedDatabaseList,
              ($event.target as HTMLInputElement).checked
            )
          "
        />
      </template>
      <template #selection="{ database }">
        <input
          type="checkbox"
          class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
          :checked="isDatabaseSelected(database)"
          @input="(e: any) => toggleDatabaseSelection(database, e.target.checked)"
        />
      </template>
    </DatabaseTable>

    <div
      v-if="!state.isLoading && databaseList.length === 0"
      class="w-full flex flex-col py-6 justify-start items-center"
    >
      <heroicons-outline:inbox class="w-12 h-auto text-gray-500" />
      <span class="text-sm leading-6 text-gray-500">{{
        $t("common.no-data")
      }}</span>
    </div>

    <div class="flex items-center justify-between">
      <div class="textinfolabel">
        <template v-if="state.selectedDatabaseIdList.size > 0">
          {{
            $t("database.selected-n-databases", {
              n: state.selectedDatabaseIdList.size,
            })
          }}
        </template>
      </div>
      <div class="flex items-center gap-x-2">
        <button class="btn-normal" @click="$emit('cancel')">
          {{ $t("common.cancel") }}
        </button>
        <button class="btn-primary" :disabled="!allowAdd" @click="tryAdd">
          {{ $t("common.add") }}
        </button>
      </div>
    </div>

    <div
      v-if="state.isLoading"
      class="absolute w-full h-full inset-0 bg-white/50 flex flex-col items-center justify-center"
    >
      <BBSpin />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, PropType, reactive } from "vue";

import type { Database, DatabaseId, Policy } from "@/types";
import { DEFAULT_PROJECT_ID } from "@/types";
import { useDatabaseStore } from "@/store";

type LocalState = {
  isLoading: boolean;
  searchText: string;
  databaseList: Database[];
  selectedDatabaseIdList: Set<DatabaseId>;
};

const props = defineProps({
  policyList: {
    type: Array as PropType<Policy[]>,
    default: () => [],
  },
});

const emit = defineEmits<{
  (e: "cancel"): void;
  (e: "add", databaseList: Database[]): void;
}>();

const state = reactive<LocalState>({
  isLoading: false,
  searchText: "",
  databaseList: [],
  selectedDatabaseIdList: new Set(),
});

const databaseStore = useDatabaseStore();

const prepareList = async () => {
  state.isLoading = true;

  const allDatabaseList = await databaseStore.fetchDatabaseList();
  state.databaseList = allDatabaseList
    .filter((db) => db.instance.environment.tier === "PROTECTED")
    .filter((db) => db.project.id !== DEFAULT_PROJECT_ID);
  state.isLoading = false;
};

const presetDatabaseIdList = computed(() => {
  const databaseIdList = props.policyList.map(
    (policy) => policy.resourceId as DatabaseId
  );
  return new Set(databaseIdList);
});

const databaseList = computed(() => {
  // Don't show the databases already have access control policy.
  let list = state.databaseList.filter(
    (db) => !presetDatabaseIdList.value.has(db.id)
  );

  const keyword = state.searchText.trim();
  if (keyword) {
    list = list.filter((db) => db.name.toLowerCase().includes(keyword));
  }
  return list;
});

const allowAdd = computed(() => {
  return state.selectedDatabaseIdList.size > 0;
});

const tryAdd = () => {
  const selectedDatabaseList = databaseList.value.filter((db) =>
    isDatabaseSelected(db)
  );
  emit("add", selectedDatabaseList);
};

const toggleDatabaseSelection = (database: Database, on: boolean) => {
  if (on) {
    state.selectedDatabaseIdList.add(database.id);
  } else {
    state.selectedDatabaseIdList.delete(database.id);
  }
};

const isDatabaseSelected = (database: Database) => {
  return state.selectedDatabaseIdList.has(database.id);
};

const getAllSelectionState = (
  databaseList: Database[]
): { checked: boolean; indeterminate: boolean } => {
  const checked = databaseList.every((db) => isDatabaseSelected(db));
  const indeterminate =
    !checked && databaseList.some((db) => isDatabaseSelected(db));

  return {
    checked,
    indeterminate,
  };
};

const toggleAllDatabasesSelection = (
  databaseList: Database[],
  on: boolean
): void => {
  const set = state.selectedDatabaseIdList;
  if (on) {
    databaseList.forEach((db) => {
      set.add(db.id);
    });
  } else {
    databaseList.forEach((db) => {
      set.delete(db.id);
    });
  }
};

prepareList();
</script>
