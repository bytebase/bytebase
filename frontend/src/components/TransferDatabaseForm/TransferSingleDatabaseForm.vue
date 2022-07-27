<template>
  <div
    class="px-4 space-y-6"
    :class="!state.selectedDatabase ? 'w-208' : 'w-112'"
  >
    <template v-if="!state.selectedDatabase">
      <slot name="transfer-source-selector" />
      <DatabaseTable
        :mode="'ALL_SHORT'"
        :bordered="true"
        :custom-click="true"
        :database-list="databaseList"
        @select-database="selectDatabase"
      />
      <!-- Update button group -->
      <div class="pt-4 border-t border-block-border flex justify-end">
        <button
          type="button"
          class="btn-normal py-2 px-4"
          @click.prevent="cancel"
        >
          {{ $t("common.cancel") }}
        </button>
      </div>
    </template>

    <template v-else>
      <SelectDatabaseLabel
        :database="state.selectedDatabase"
        :target-project-id="project.id"
        @next="transferDatabase"
      >
        <template #buttons="{ next, valid }">
          <div
            class="w-full pt-4 mt-6 flex justify-end border-t border-block-border"
          >
            <button
              type="button"
              class="btn-normal py-2 px-4"
              @click.prevent="state.selectedDatabase = undefined"
            >
              {{ $t("common.back") }}
            </button>
            <button
              type="button"
              class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
              :disabled="!valid"
              @click.prevent="next"
            >
              {{ $t("common.transfer") }}
            </button>
          </div>
        </template>
      </SelectDatabaseLabel>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { PropType, reactive } from "vue";
import DatabaseTable from "@/components/DatabaseTable.vue";
import { SelectDatabaseLabel, TransferSource } from "./";
import { Database, DatabaseLabel, Project } from "@/types";

interface LocalState {
  selectedDatabase?: Database;
}

defineProps({
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
  (e: "submit", database: Database, labels: DatabaseLabel[]): void;
}>();

const state = reactive<LocalState>({
  selectedDatabase: undefined,
});

const selectDatabase = (database: Database) => {
  state.selectedDatabase = database;
};

const transferDatabase = (labels: DatabaseLabel[]) => {
  if (!state.selectedDatabase) {
    return;
  }
  emit("submit", state.selectedDatabase, labels);
};

const cancel = () => {
  emit("dismiss");
};
</script>
