<template>
  <div class="mb-2 flex flex-row items-center">
    <span class="text-lg mr-2">{{ $t("common.databases") }}</span>
    <BBLoader v-show="state.isRequesting" class="opacity-60" />
  </div>
  <div
    class="w-full border rounded min-h-[20rem] max-h-[24rem] overflow-y-auto"
  >
    <div
      class="sticky top-0 z-[1] w-full flex flex-row justify-between items-center px-2 py-1 bg-gray-100 border-b cursor-pointer"
      @click="state.showMatchedDatabaseList = !state.showMatchedDatabaseList"
    >
      <div class="text-sm font-medium">
        <span>{{ $t("database-group.matched-database") }}</span>
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
        <NEllipsis class="text-sm" line-clamp="1">
          {{ database.databaseName }}
        </NEllipsis>
        <div class="flex flex-row justify-end items-center shrink-0">
          <InstanceV1EngineIcon :instance="database.instanceEntity" />
          <NEllipsis
            class="ml-1 text-sm text-gray-400 max-w-[124px]"
            line-clamp="1"
          >
            ({{ database.instanceEntity.environmentEntity.title }})
            {{ database.instanceEntity.title }}
          </NEllipsis>
        </div>
      </div>
    </div>
    <div
      class="sticky top-8 z-[1] w-full flex flex-row justify-between items-center px-2 py-1 bg-gray-100 border-y cursor-pointer"
      @click="
        state.showUnmatchedDatabaseList = !state.showUnmatchedDatabaseList
      "
    >
      <div class="text-sm font-medium">
        <span>{{ $t("database-group.unmatched-database") }}</span>
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
        <NEllipsis class="text-sm" line-clamp="1">
          {{ database.databaseName }}
        </NEllipsis>
        <div class="flex flex-row justify-end items-center shrink-0">
          <InstanceV1EngineIcon :instance="database.instanceEntity" />
          <NEllipsis
            class="ml-1 text-sm text-gray-400 max-w-[124px]"
            line-clamp="1"
          >
            ({{ database.instanceEntity.environmentEntity.title }})
            {{ database.instanceEntity.title }}
          </NEllipsis>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NEllipsis } from "naive-ui";
import { ref, watch, reactive } from "vue";
import { ConditionGroupExpr, buildCELExpr } from "@/plugins/cel";
import {
  useDatabaseV1Store,
  useEnvironmentV1Store,
} from "@/store";
import { ComposedDatabase, ComposedProject } from "@/types";
import { InstanceV1EngineIcon } from "../v2";
import { DatabaseGroup } from "@/types/proto/v1/project_service";
import { projectServiceClient } from "@/grpcweb";
import { buildDatabaseGroupExpr } from "@/utils/databaseGroup/cel";
import { Expr } from "@/types/proto/google/type/expr";
import { useDebounceFn } from "@vueuse/core";
import BBLoader from "@/bbkit/BBLoader.vue";
import { databaseGroupNamePrefix } from "@/store/modules/v1/common";
import { convertParsedExprToCELString } from "@/utils";
import { ParsedExpr } from "@/types/proto/google/api/expr/v1alpha1/syntax";

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
  showUnmatchedDatabaseList: true,
});
const matchedDatabaseList = ref<ComposedDatabase[]>([]);
const unmatchedDatabaseList = ref<ComposedDatabase[]>([]);

const updateMatchingState = useDebounceFn(async () => {
  state.isRequesting = true;

  const environment = environmentStore.getEnvironmentByUID(props.environmentId);
  const celString = await convertParsedExprToCELString(
    ParsedExpr.fromJSON({
      expr: buildCELExpr(
        buildDatabaseGroupExpr({
          environmentId: environment.name,
          conditionGroupExpr: props.expr,
        })
      ),
    })
  );
  const validateOnlyResourceId = `creating-database-group-${Date.now()}`;
  const result = await projectServiceClient.createDatabaseGroup({
    parent: props.project.name,
    databaseGroup: {
      name: `${props.project.name}/${databaseGroupNamePrefix}${validateOnlyResourceId}`,
      databasePlaceholder: validateOnlyResourceId,
      databaseExpr: Expr.fromJSON({
        expression: celString,
      }),
    },
    databaseGroupId: validateOnlyResourceId,
    validateOnly: true,
  });

  matchedDatabaseList.value = [];
  unmatchedDatabaseList.value = [];
  for (const item of result.matchedDatabases) {
    const database = await databaseStore.getOrFetchDatabaseByName(item.name);
    if (database) {
      matchedDatabaseList.value.push(database);
    }
  }
  for (const item of result.unmatchedDatabases) {
    const database = await databaseStore.getOrFetchDatabaseByName(item.name);
    if (
      database &&
      database.instanceEntity.environmentEntity.uid === props.environmentId
    ) {
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
