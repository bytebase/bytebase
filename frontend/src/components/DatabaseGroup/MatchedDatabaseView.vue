<template>
  <div class="w-full border min-h-[20rem]">
    <div
      class="w-full flex flex-row justify-between items-center px-2 py-1 bg-gray-100 border-b"
    >
      <div>
        <span>Matched database</span>
        <span class="ml-1 text-gray-400"
          >({{ matchedDatabaseList.length }})</span
        >
      </div>
      <button
        @click="state.showMatchedDatabaseList = !state.showMatchedDatabaseList"
      >
        <heroicons-outline:chevron-right
          v-if="!state.showMatchedDatabaseList"
          class="w-5 h-auto"
        />
        <heroicons-outline:chevron-down v-else class="w-5 h-auto" />
      </button>
    </div>
    <div v-show="state.showMatchedDatabaseList" class="w-full">
      <div
        v-for="database in matchedDatabaseList"
        :key="database.name"
        class="w-full flex flex-row justify-between items-center px-2 py-1 gap-x-2"
      >
        <span>{{ database.name }}</span>
        <div class="flex flex-row justify-end items-center">
          <InstanceV1EngineIcon :instance="database.instanceEntity" />
          <span class="ml-1 text-sm text-gray-400">{{
            database.instanceEntity.title
          }}</span>
        </div>
      </div>
    </div>
    <div
      class="w-full flex flex-row justify-between items-center px-2 py-1 bg-gray-100 border-b"
      :class="[state.showMatchedDatabaseList && 'border-t']"
    >
      <div>
        <span>Unmatched database</span>
        <span class="ml-1 text-gray-400"
          >({{ unmatchedDatabaseList.length }})</span
        >
      </div>
      <button
        @click="
          state.showUnmatchedDatabaseList = !state.showUnmatchedDatabaseList
        "
      >
        <heroicons-outline:chevron-right
          v-if="!state.showUnmatchedDatabaseList"
          class="w-5 h-auto"
        />
        <heroicons-outline:chevron-down v-else class="w-5 h-auto" />
      </button>
    </div>
    <div v-show="state.showUnmatchedDatabaseList">
      <div
        v-for="database in matchedDatabaseList"
        :key="database.name"
        class="w-full flex flex-row justify-between items-center"
      >
        <span>{{ database.name }}</span>
        <div class="flex flex-row justify-end items-center">
          <InstanceV1EngineIcon :instance="database.instanceEntity" />
          <span class="ml-1 text-sm text-gray-400">{{
            database.instanceEntity.title
          }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref, watch, reactive } from "vue";
import { ConditionGroupExpr } from "@/plugins/cel";
import { useDatabaseV1Store } from "@/store";
import { ComposedDatabase, ComposedProject } from "@/types";
import { sortDatabaseV1List } from "@/utils";
import { InstanceV1EngineIcon } from "../v2";

interface LocalState {
  showMatchedDatabaseList: boolean;
  showUnmatchedDatabaseList: boolean;
}

const props = defineProps<{
  project: ComposedProject;
  environmentId: string;
  expr: ConditionGroupExpr;
}>();

const state = reactive<LocalState>({
  showMatchedDatabaseList: false,
  showUnmatchedDatabaseList: false,
});
const matchedDatabaseList = ref<ComposedDatabase[]>([]);
const unmatchedDatabaseList = ref<ComposedDatabase[]>([]);
const databaseList = computed(() => {
  const list = useDatabaseV1Store().databaseListByProject(props.project.name);
  return sortDatabaseV1List(list);
});

watch(
  () => [databaseList.value, props.expr],
  async () => {
    // TODO: fetch matched and unmatched database list with expr.
    matchedDatabaseList.value = databaseList.value;
    unmatchedDatabaseList.value = [];
  },
  {
    immediate: true,
  }
);
</script>
