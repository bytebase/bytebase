<template>
  <div class="w-full border min-h-[20rem]">
    <div
      class="w-full flex flex-row justify-between items-center px-2 py-1 bg-gray-100 border-b"
    >
      <div>
        <span>Matched table</span>
        <span class="ml-1 text-gray-400">({{ matchedTableList.length }})</span>
      </div>
      <button @click="state.showMatchedTableList = !state.showMatchedTableList">
        <heroicons-outline:chevron-right
          v-if="!state.showMatchedTableList"
          class="w-5 h-auto"
        />
        <heroicons-outline:chevron-down v-else class="w-5 h-auto" />
      </button>
    </div>
    <div v-show="state.showMatchedTableList" class="w-full">
      <div
        v-for="database in matchedTableList"
        :key="database.name"
        class="w-full flex flex-row justify-between items-center px-2 py-1 gap-x-2"
      >
        <span>{{ database.databaseName }}</span>
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
      :class="[state.showMatchedTableList && 'border-t']"
    >
      <div>
        <span>Unmatched table</span>
        <span class="ml-1 text-gray-400"
          >({{ unmatchedTableList.length }})</span
        >
      </div>
      <button
        @click="state.showUnmatchedTableList = !state.showUnmatchedTableList"
      >
        <heroicons-outline:chevron-right
          v-if="!state.showUnmatchedTableList"
          class="w-5 h-auto"
        />
        <heroicons-outline:chevron-down v-else class="w-5 h-auto" />
      </button>
    </div>
    <div v-show="state.showUnmatchedTableList">
      <div
        v-for="database in matchedTableList"
        :key="database.name"
        class="w-full flex flex-row justify-between items-center"
      >
        <span>{{ database.databaseName }}</span>
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
  showMatchedTableList: boolean;
  showUnmatchedTableList: boolean;
}

const props = defineProps<{
  project: ComposedProject;
  environmentId: string;
  expr: ConditionGroupExpr;
}>();

const state = reactive<LocalState>({
  showMatchedTableList: false,
  showUnmatchedTableList: false,
});
const matchedTableList = ref<ComposedDatabase[]>([]);
const unmatchedTableList = ref<ComposedDatabase[]>([]);
const databaseList = computed(() => {
  const list = useDatabaseV1Store().databaseListByProject(props.project.name);
  return sortDatabaseV1List(list);
});

watch(
  () => [databaseList.value, props.expr],
  async () => {
    // TODO: fetch matched and unmatched table list with expr.
    matchedTableList.value = databaseList.value;
    unmatchedTableList.value = [];
  },
  {
    immediate: true,
  }
);
</script>
