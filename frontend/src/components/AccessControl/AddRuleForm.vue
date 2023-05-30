<template>
  <div class="w-[calc(100vw-8rem)] max-w-[60rem] space-y-2 relative">
    <div>
      <SearchBox
        v-model:value="state.searchText"
        style="width: 15rem"
        :placeholder="$t('database.search-database')"
      />
    </div>

    <DatabaseV1Table
      mode="ALL_TINY"
      class="overflow-y-auto"
      style="max-height: calc(100vh - 320px)"
      table-class="border"
      :custom-click="true"
      :database-list="databaseList"
      :show-selection-column="true"
      @select-database="
          (db: ComposedDatabase) => toggleDatabaseSelection(db, !isDatabaseSelected(db))
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
    </DatabaseV1Table>

    <div
      v-if="!state.isLoading && databaseList.length === 0"
      class="w-full flex flex-col py-6 justify-start items-center"
    >
      <i18n-t
        keypath="settings.access-control.no-database-in-production-environment"
        tag="div"
        class="text-sm leading-6 text-gray-500 max-w-[15rem] whitespace-pre-wrap text-center"
      >
        <template #production_environment>
          <a
            href="https://www.bytebase.com/docs/administration/database-access-control"
            class="normal-link lowercase"
            target="_BLANK"
          >
            {{ $t("environment.production-environment") }}
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

import type { ComposedDatabase } from "@/types";
import { filterDatabaseV1ByKeyword } from "@/utils";
import { Policy } from "@/types/proto/v1/org_policy_service";
import { SearchBox } from "@/components/v2";

type LocalState = {
  isLoading: boolean;
  searchText: string;
  selectedDatabaseIdList: Set<string>;
};

const props = defineProps({
  policyList: {
    type: Array as PropType<Policy[]>,
    default: () => [],
  },
  databaseList: {
    type: Array as PropType<ComposedDatabase[]>,
    default: () => [],
  },
});

const emit = defineEmits<{
  (e: "cancel"): void;
  (e: "add", databaseList: ComposedDatabase[]): void;
}>();

const state = reactive<LocalState>({
  isLoading: false,
  searchText: "",
  selectedDatabaseIdList: new Set(),
});

const presetDatabaseIdList = computed(() => {
  const databaseIdList = props.policyList.map(
    (policy) => policy.resourceUid as string
  );
  return new Set(databaseIdList);
});

const databaseList = computed(() => {
  // Don't show the databases already have access control policy.
  let list = props.databaseList.filter(
    (db) => !presetDatabaseIdList.value.has(db.uid)
  );

  const keyword = state.searchText.trim();
  list = list.filter((db) =>
    filterDatabaseV1ByKeyword(db, keyword, [
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

const toggleDatabaseSelection = (database: ComposedDatabase, on: boolean) => {
  if (on) {
    state.selectedDatabaseIdList.add(database.uid);
  } else {
    state.selectedDatabaseIdList.delete(database.uid);
  }
};

const isDatabaseSelected = (database: ComposedDatabase) => {
  return state.selectedDatabaseIdList.has(database.uid);
};

const getAllSelectionState = (
  databaseList: ComposedDatabase[]
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
  databaseList: ComposedDatabase[],
  on: boolean
): void => {
  const set = state.selectedDatabaseIdList;
  if (on) {
    databaseList.forEach((db) => {
      set.add(db.uid);
    });
  } else {
    databaseList.forEach((db) => {
      set.delete(db.uid);
    });
  }
};
</script>
