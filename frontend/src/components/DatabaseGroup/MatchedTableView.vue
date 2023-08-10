<template>
  <div class="mb-2 flex flex-row items-center">
    <span class="text-lg mr-2">{{ $t("db.tables") }}</span>
    <BBLoader v-show="loading" class="opacity-60" />
  </div>
  <div
    class="w-full border rounded min-h-[20rem] max-h-[24rem] overflow-y-auto"
  >
    <div
      class="sticky top-0 z-[1] w-full flex flex-row justify-between items-center px-2 py-1 bg-gray-100 border-b cursor-pointer"
      @click="state.showMatchedTableList = !state.showMatchedTableList"
    >
      <div class="text-sm font-medium">
        <span>{{ $t("database-group.matched-table") }}</span>
        <span class="ml-1 text-gray-400">({{ matchedTableList.length }})</span>
      </div>
      <button class="opacity-60">
        <heroicons-outline:chevron-right
          v-if="!state.showMatchedTableList"
          class="w-5 h-auto"
        />
        <heroicons-outline:chevron-down v-else class="w-5 h-auto" />
      </button>
    </div>
    <div v-show="state.showMatchedTableList" class="w-full my-1">
      <div
        v-for="table in matchedTableList"
        :key="table.database"
        class="w-full flex flex-row justify-between items-center px-2 py-1 gap-x-2"
      >
        <span class="text-sm">{{ table.table }}</span>
        <div class="flex flex-row justify-end items-center">
          <DatabaseView :database="table.databaseEntity" />
        </div>
      </div>
    </div>
    <div
      class="sticky top-8 z-[1] w-full flex flex-row justify-between items-center px-2 py-1 bg-gray-100 border-y cursor-pointer"
      @click="state.showUnmatchedTableList = !state.showUnmatchedTableList"
    >
      <div class="text-sm font-medium">
        <span>{{ $t("database-group.unmatched-table") }}</span>
        <span class="ml-1 text-gray-400"
          >({{ unmatchedTableList.length }})</span
        >
      </div>
      <button class="opacity-60">
        <heroicons-outline:chevron-right
          v-if="!state.showUnmatchedTableList"
          class="w-5 h-auto"
        />
        <heroicons-outline:chevron-down v-else class="w-5 h-auto" />
      </button>
    </div>
    <div v-show="state.showUnmatchedTableList" class="w-full py-1">
      <div
        v-for="table in unmatchedTableList"
        :key="table.database"
        class="w-full flex flex-row justify-between items-center px-2 py-1 gap-x-2"
      >
        <span class="text-sm">{{ table.table }}</span>
        <div class="flex flex-row justify-end items-center">
          <DatabaseView :database="table.databaseEntity" />
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { reactive } from "vue";
import BBLoader from "@/bbkit/BBLoader.vue";
import { ComposedSchemaGroupTable } from "@/types";
import { DatabaseView } from "../v2";

interface LocalState {
  showMatchedTableList: boolean;
  showUnmatchedTableList: boolean;
}

defineProps<{
  loading: boolean;
  matchedTableList: ComposedSchemaGroupTable[];
  unmatchedTableList: ComposedSchemaGroupTable[];
}>();

const state = reactive<LocalState>({
  showMatchedTableList: true,
  showUnmatchedTableList: true,
});
</script>
