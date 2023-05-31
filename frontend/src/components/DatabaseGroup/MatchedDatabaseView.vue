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
        v-for="database in unmatchedDatabaseList"
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
  </div>
</template>

<script lang="ts" setup>
import { ref, watch, reactive, computed } from "vue";
import { ConditionGroupExpr } from "@/plugins/cel";
import { useDatabaseV1Store, useEnvironmentV1Store } from "@/store";
import { ComposedDatabase, ComposedProject } from "@/types";
import { InstanceV1EngineIcon } from "../v2";
import {
  DatabaseGroup,
  DatabaseGroupView,
} from "@/types/proto/v1/project_service";
import { projectServiceClient } from "@/grpcweb";
import { stringifyDatabaseGroupExpr } from "@/utils/databaseGroup/cel";
import { Expr } from "@/types/proto/google/type/expr";
import { useDebounceFn } from "@vueuse/core";

interface LocalState {
  showMatchedDatabaseList: boolean;
  showUnmatchedDatabaseList: boolean;
}

const props = defineProps<{
  project: ComposedProject;
  environmentId: string;
  expr: ConditionGroupExpr;
  databaseGroup?: DatabaseGroup;
}>();

const environmentStore = useEnvironmentV1Store();
const databaseStore = useDatabaseV1Store();
const state = reactive<LocalState>({
  showMatchedDatabaseList: false,
  showUnmatchedDatabaseList: false,
});
const matchedDatabaseList = ref<ComposedDatabase[]>([]);
const unmatchedDatabaseList = ref<ComposedDatabase[]>([]);

const isCreating = computed(() => props.databaseGroup === undefined);

const updateMatchingState = useDebounceFn(async () => {
  const matchedDatabaseNameList: string[] = [];
  const unmatchedDatabaseNameList: string[] = [];

  if (isCreating.value) {
    const environment = environmentStore.getEnvironmentByUID(
      props.environmentId
    );
    const celString = stringifyDatabaseGroupExpr({
      environmentId: environment.name,
      conditionGroupExpr: props.expr,
    });
    const validateOnlyResourceId = "creating-database-group";
    const result = await projectServiceClient.createDatabaseGroup({
      parent: props.project.name,
      databaseGroup: {
        name: `${props.project.name}/databaseGroups/${validateOnlyResourceId}`,
        databasePlaceholder: validateOnlyResourceId,
        databaseExpr: Expr.fromJSON({
          expression: celString,
        }),
      },
      databaseGroupId: validateOnlyResourceId,
      validateOnly: true,
    });
    matchedDatabaseNameList.push(
      ...result.matchedDatabases.map((item) => item.name)
    );
    unmatchedDatabaseNameList.push(
      ...result.unmatchedDatabases.map((item) => item.name)
    );
  } else {
    const result = await projectServiceClient.getDatabaseGroup({
      name: props.databaseGroup!.name,
      view: DatabaseGroupView.DATABASE_GROUP_VIEW_FULL,
    });
    matchedDatabaseNameList.push(
      ...result.matchedDatabases.map((item) => item.name)
    );
    unmatchedDatabaseNameList.push(
      ...result.unmatchedDatabases.map((item) => item.name)
    );
  }

  for (const name of matchedDatabaseNameList) {
    const database = await databaseStore.getOrFetchDatabaseByName(name);
    if (database) {
      matchedDatabaseList.value.push(database);
    }
  }
  for (const name of unmatchedDatabaseNameList) {
    const database = await databaseStore.getOrFetchDatabaseByName(name);
    if (database) {
      unmatchedDatabaseList.value.push(database);
    }
  }
}, 500);

watch(() => props, updateMatchingState, {
  immediate: true,
  deep: true,
});
</script>
