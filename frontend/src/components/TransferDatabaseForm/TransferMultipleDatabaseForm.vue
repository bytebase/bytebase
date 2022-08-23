<template>
  <div class="px-4 space-y-6 w-208">
    <slot name="transfer-source-selector" />

    <DatabaseTable
      :mode="'ALL_SHORT'"
      :bordered="true"
      :custom-click="true"
      :database-list="databaseList"
      :show-selection-column="true"
      @select-database="
        (db) => toggleDatabaseSelection(db, !isDatabaseSelected(db))
      "
    >
      <template #selection-all="{ databaseList: renderedDatabaseList }">
        <input
          v-if="renderedDatabaseList.length > 0"
          type="checkbox"
          class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
          v-bind="getAllSelectionState(renderedDatabaseList)"
          @input="toggleAllDatabasesSelection(renderedDatabaseList, ($event.target as HTMLInputElement).checked)"
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
    <!-- Update button group -->
    <div class="pt-4 border-t border-block-border flex justify-end">
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="$emit('dismiss')"
      >
        {{ $t("common.cancel") }}
      </button>
      <button
        type="button"
        class="btn-primary py-2 px-4 ml-3"
        :disabled="!allowTransfer"
        @click.prevent="transferDatabase"
      >
        {{ $t("common.transfer") }}
      </button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, PropType, reactive, watch } from "vue";
import { Database, DatabaseId, Project } from "@/types";
import { TransferSource } from "./utils";

type LocalState = {
  selectedDatabaseIdList: Set<DatabaseId>;
};

const props = defineProps({
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
  transferSource: {
    type: String as PropType<TransferSource>,
    required: true,
  },
  databaseList: {
    type: Array as PropType<Database[]>,
    default: () => [],
  },
});

const emit = defineEmits<{
  (e: "dismiss"): void;
  (e: "submit", databaseList: Database[]): void;
}>();

const state = reactive<LocalState>({
  selectedDatabaseIdList: new Set(),
});

watch(
  [() => props.project, () => props.transferSource, () => props.databaseList],
  () => {
    state.selectedDatabaseIdList.clear();
  }
);

const isDatabaseSelected = (database: Database): boolean => {
  return state.selectedDatabaseIdList.has(database.id);
};

const toggleDatabaseSelection = (database: Database, on: boolean) => {
  if (on) {
    state.selectedDatabaseIdList.add(database.id);
  } else {
    state.selectedDatabaseIdList.delete(database.id);
  }
};

const getAllSelectionState = (
  databaseList: Database[]
): { checked: boolean; indeterminate: boolean } => {
  const allCount = databaseList.length;
  const selectedCount = state.selectedDatabaseIdList.size;
  return {
    checked: selectedCount === allCount,
    indeterminate: selectedCount > 0 && selectedCount !== allCount,
  };
};

const toggleAllDatabasesSelection = (
  databaseList: Database[],
  on: boolean
): void => {
  if (on) {
    state.selectedDatabaseIdList = new Set(databaseList.map((db) => db.id));
  } else {
    state.selectedDatabaseIdList.clear();
  }
};

const allowTransfer = computed(() => state.selectedDatabaseIdList.size > 0);

const transferDatabase = () => {
  if (state.selectedDatabaseIdList.size === 0) return;

  const databaseList = [...state.selectedDatabaseIdList.values()].map(
    (id) => props.databaseList.find((db) => db.id === id)!
  );
  emit("submit", databaseList);
};
</script>
