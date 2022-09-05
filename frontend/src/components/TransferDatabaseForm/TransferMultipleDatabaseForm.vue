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
    <!-- Update button group -->
    <div class="pt-4 border-t border-block-border flex justify-between">
      <div>
        <div v-if="state.selectedDatabaseIdList.size > 0" class="textinfolabel">
          {{
            $t("database.selected-n-databases", {
              n: state.selectedDatabaseIdList.size,
            })
          }}
        </div>
      </div>
      <div class="flex items-center">
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
  </div>
</template>

<script lang="ts" setup>
import { computed, PropType, reactive, watch } from "vue";
import { Database, DatabaseId } from "@/types";
import { TransferSource } from "./utils";
import { useDatabaseStore } from "@/store";

type LocalState = {
  selectedDatabaseIdList: Set<DatabaseId>;
};

const props = defineProps({
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

const databaseStore = useDatabaseStore();

const state = reactive<LocalState>({
  selectedDatabaseIdList: new Set(),
});

watch(
  () => props.transferSource,
  () => {
    // Clear selected database ID list when transferSource changed.
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
  const set = state.selectedDatabaseIdList;

  const checked = databaseList.every((db) => set.has(db.id));
  const indeterminate = !checked && databaseList.some((db) => set.has(db.id));

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

const allowTransfer = computed(() => state.selectedDatabaseIdList.size > 0);

const transferDatabase = () => {
  if (state.selectedDatabaseIdList.size === 0) return;

  // If a database can be selected, it must be fetched already.
  // So it's safe that we won't get <<Unknown database>> here.
  const databaseList = [...state.selectedDatabaseIdList.values()].map((id) =>
    databaseStore.getDatabaseById(id)
  );
  emit("submit", databaseList);
};
</script>
