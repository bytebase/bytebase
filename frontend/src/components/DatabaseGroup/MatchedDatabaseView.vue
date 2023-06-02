<template>
  <div class="mb-2 flex flex-row items-center">
    <span class="text-lg mr-2">Databases</span>
    <BBLoader v-show="state.isRequesting" class="opacity-60" />
  </div>
  <div
    class="w-full border rounded min-h-[20rem] max-h-[24rem] overflow-y-auto"
  >
    <div
      class="sticky top-0 z-[1] w-full flex flex-row justify-between items-center px-2 py-1 bg-gray-100 border-b cursor-pointer"
      @click="state.showMatchedDatabaseList = !state.showMatchedDatabaseList"
    >
      <div>
        <span>Matched database</span>
        <span class="ml-1 text-gray-400"
          >({{ matchedDatabaseList.length }})</span
        >
      </div>
      <button class="opacity-60">
        <heroicons-outline:chevron-right
          v-if="!state.showMatchedDatabaseList"
          class="w-5 h-auto"
        />
        <heroicons-outline:chevron-down v-else class="w-5 h-auto" />
      </button>
    </div>
    <div v-show="state.showMatchedDatabaseList" class="w-full py-1">
      <div
        v-for="database in matchedDatabaseList"
        :key="database.name"
        class="w-full flex flex-row justify-between items-center px-2 py-1 gap-x-2"
      >
        <span class="text-sm">{{ database.databaseName }}</span>
        <div class="flex flex-row justify-end items-center">
          <InstanceV1EngineIcon :instance="database.instanceEntity" />
          <span class="ml-1 text-sm text-gray-400"
            >{{ database.instanceEntity.title }} ({{
              database.instanceEntity.environmentEntity.title
            }})</span
          >
        </div>
      </div>
    </div>
    <div
      class="sticky top-8 z-[1] w-full flex flex-row justify-between items-center px-2 py-1 bg-gray-100 border-y cursor-pointer"
      @click="
        state.showUnmatchedDatabaseList = !state.showUnmatchedDatabaseList
      "
    >
      <div>
        <span>Unmatched database</span>
        <span class="ml-1 text-gray-400"
          >({{ unmatchedDatabaseList.length }})</span
        >
      </div>
      <button class="opacity-60">
        <heroicons-outline:chevron-right
          v-if="!state.showUnmatchedDatabaseList"
          class="w-5 h-auto"
        />
        <heroicons-outline:chevron-down v-else class="w-5 h-auto" />
      </button>
    </div>
    <div v-show="state.showUnmatchedDatabaseList" class="w-full py-1">
      <div
        v-for="database in unmatchedDatabaseList"
        :key="database.name"
        class="w-full flex flex-row justify-between items-center px-2 py-1 gap-x-2"
      >
        <span class="text-sm">{{ database.databaseName }}</span>
        <div class="flex flex-row justify-end items-center">
          <InstanceV1EngineIcon :instance="database.instanceEntity" />
          <span class="ml-1 text-sm text-gray-400"
            >{{ database.instanceEntity.title }} ({{
              database.instanceEntity.environmentEntity.title
            }})</span
          >
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
import BBLoader from "@/bbkit/BBLoader.vue";

interface LocalState {
  isRequesting: boolean;
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
  isRequesting: false,
  showMatchedDatabaseList: true,
  showUnmatchedDatabaseList: false,
});
const matchedDatabaseList = ref<ComposedDatabase[]>([]);
const unmatchedDatabaseList = ref<ComposedDatabase[]>([]);

const isCreating = computed(() => props.databaseGroup === undefined);

const updateMatchingState = useDebounceFn(async () => {
  state.isRequesting = true;
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

  matchedDatabaseList.value = [];
  unmatchedDatabaseList.value = [];
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
  state.isRequesting = false;
}, 500);

watch(() => props, updateMatchingState, {
  immediate: true,
  deep: true,
});
</script>
