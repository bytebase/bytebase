<template>
  <div class="mb-2 flex flex-row items-center">
    <span class="text-lg mr-2">{{ $t("db.tables") }}</span>
    <BBLoader v-show="state.isRequesting" class="opacity-60" />
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
          <DatabaseView :database-name="table.database" />
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
          <DatabaseView :database-name="table.database" />
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, watch, reactive } from "vue";
import { ConditionGroupExpr, convertToCELString } from "@/plugins/cel";
import { ComposedProject } from "@/types";
import { DatabaseView } from "../v2";
import {
  SchemaGroup,
  SchemaGroup_Table,
} from "@/types/proto/v1/project_service";
import { projectServiceClient } from "@/grpcweb";
import { Expr } from "@/types/proto/google/type/expr";
import { useDebounceFn } from "@vueuse/core";
import {
  databaseGroupNamePrefix,
  schemaGroupNamePrefix,
} from "@/store/modules/v1/common";
import BBLoader from "@/bbkit/BBLoader.vue";

interface LocalState {
  isRequesting: boolean;
  showMatchedTableList: boolean;
  showUnmatchedTableList: boolean;
}

const props = defineProps<{
  project: ComposedProject;
  databaseGroupName: string;
  expr: ConditionGroupExpr;
  schemaGroup?: SchemaGroup;
}>();

const state = reactive<LocalState>({
  isRequesting: false,
  showMatchedTableList: true,
  showUnmatchedTableList: true,
});
const matchedTableList = ref<SchemaGroup_Table[]>([]);
const unmatchedTableList = ref<SchemaGroup_Table[]>([]);

const updateMatchingState = useDebounceFn(async () => {
  state.isRequesting = true;
  try {
    const celString = convertToCELString(props.expr);
    const validateOnlyResourceId = `creating-schema-group-${Date.now()}`;
    const databaseGroupName = `${props.project.name}/${databaseGroupNamePrefix}${props.databaseGroupName}`;
    const result = await projectServiceClient.createSchemaGroup({
      parent: databaseGroupName,
      schemaGroup: {
        name: `${databaseGroupName}/${schemaGroupNamePrefix}${validateOnlyResourceId}`,
        tablePlaceholder: validateOnlyResourceId,
        tableExpr: Expr.fromJSON({
          expression: celString || "true",
        }),
      },
      schemaGroupId: validateOnlyResourceId,
      validateOnly: true,
    });
    matchedTableList.value = result.matchedTables;
    unmatchedTableList.value = result.unmatchedTables;
  } catch (error) {
    console.error(error);
    matchedTableList.value = [];
    unmatchedTableList.value = [];
  }
  state.isRequesting = false;
}, 500);

watch(() => props, updateMatchingState, {
  immediate: true,
  deep: true,
});
</script>
