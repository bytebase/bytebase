<template>
  <div class="w-160 space-y-2 relative">
    <div>
      <BBTableSearch
        class="w-60"
        :placeholder="$t('database.search-database')"
        @change-text="(text: string) => (state.searchText = text)"
      />
    </div>

    <DatabaseTable
      mode="ALL_TINY"
      class="max-h-[55vh] overflow-y-auto"
      table-class="border"
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
      <i18n-t
        keypath="settings.access-control.no-database-in-protected-environment"
        tag="div"
        class="text-sm leading-6 text-gray-500 max-w-[15rem] whitespace-pre-wrap text-center"
      >
        <template #protected_environment>
          <a
            href="https://www.bytebase.com/docs/administration/database-access-control"
            class="normal-link lowercase"
            target="__BLANK"
          >
            {{ $t("environment.protected-environment") }}
            <heroicons-outline:external-link
              class="inline-block w-4 h-4 -mt-0.5 mr-0.5"
            />
          </a>
        </template>
      </i18n-t>
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
import { filterDatabaseByKeyword } from "@/utils";

type LocalState = {
  isLoading: boolean;
  searchText: string;
  selectedDatabaseIdList: Set<DatabaseId>;
};

const props = defineProps({
  policyList: {
    type: Array as PropType<Policy[]>,
    default: () => [],
  },
  databaseList: {
    type: Array as PropType<Database[]>,
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
  selectedDatabaseIdList: new Set(),
});

const presetDatabaseIdList = computed(() => {
  const databaseIdList = props.policyList.map(
    (policy) => policy.resourceId as DatabaseId
  );
  return new Set(databaseIdList);
});

const databaseList = computed(() => {
  // Don't show the databases already have access control policy.
  let list = props.databaseList.filter(
    (db) => !presetDatabaseIdList.value.has(db.id)
  );

  const keyword = state.searchText.trim();
  list = list.filter((db) =>
    filterDatabaseByKeyword(db, keyword, [
      "name",
      "project",
      "instance",
      "environment",
    ])
  );
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
</script>
