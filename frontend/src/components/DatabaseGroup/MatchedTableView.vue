<template>
  <p class="text-lg mb-2">Tables</p>
  <div class="w-full border min-h-[20rem] max-h-[24rem] overflow-y-auto">
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
import { computed, ref, watch, reactive } from "vue";
import { ConditionGroupExpr, convertToCELString } from "@/plugins/cel";
import { ComposedProject } from "@/types";
import { DatabaseView } from "../v2";
import {
  SchemaGroup,
  SchemaGroupView,
  SchemaGroup_Table,
} from "@/types/proto/v1/project_service";
import { projectServiceClient } from "@/grpcweb";
import { Expr } from "@/types/proto/google/type/expr";
import { useDebounceFn } from "@vueuse/core";
import { schemaGroupNamePrefix } from "@/store/modules/v1/common";

interface LocalState {
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
  showMatchedTableList: true,
  showUnmatchedTableList: false,
});
const matchedTableList = ref<SchemaGroup_Table[]>([]);
const unmatchedTableList = ref<SchemaGroup_Table[]>([]);

const isCreating = computed(() => props.schemaGroup === undefined);

const updateMatchingState = useDebounceFn(async () => {
  const tempMatchedTableList: SchemaGroup_Table[] = [];
  const tempUnmatchedTableList: SchemaGroup_Table[] = [];

  if (isCreating.value) {
    const celString = convertToCELString(props.expr);
    const validateOnlyResourceId = "creating-schema-group";
    const result = await projectServiceClient.createSchemaGroup({
      parent: props.databaseGroupName,
      schemaGroup: {
        name: `${props.databaseGroupName}/${schemaGroupNamePrefix}/${validateOnlyResourceId}`,
        tablePlaceholder: validateOnlyResourceId,
        tableExpr: Expr.fromJSON({
          expression: celString,
        }),
      },
      schemaGroupId: validateOnlyResourceId,
      validateOnly: true,
    });
    tempMatchedTableList.push(...result.matchedTables);
    tempUnmatchedTableList.push(...result.unmatchedTables);
  } else {
    const result = await projectServiceClient.getSchemaGroup({
      name: props.schemaGroup!.name,
      view: SchemaGroupView.SCHEMA_GROUP_VIEW_FULL,
    });
    tempMatchedTableList.push(...result.matchedTables);
    tempUnmatchedTableList.push(...result.unmatchedTables);
  }
  matchedTableList.value = tempMatchedTableList;
  unmatchedTableList.value = tempUnmatchedTableList;
}, 500);

watch(() => props, updateMatchingState, {
  immediate: true,
  deep: true,
});
</script>
