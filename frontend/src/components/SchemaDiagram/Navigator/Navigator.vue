<template>
  <div class="relative h-full">
    <div
      class="bb-schema-diagram--navigator--main h-full bg-white overflow-hidden border-y border-gray-200 flex flex-col transition-all"
      :class="[state.expand ? 'w-72 shadow-sm border-l' : 'w-0']"
    >
      <div class="p-1 flex flex-col gap-y-2">
        <SchemaSelector
          v-if="hasSchemaProperty(database.instanceResource.engine)"
          :schemas="databaseMetadata.schemas"
          v-model:value="selectedSchemaNames"
        />
        <NInput
          :size="'small'"
          v-model:value="state.keyword"
          :placeholder="$t('common.search')"
        >
          <template #prefix>
            <heroicons-outline:search class="h-5 w-5 text-gray-300" />
          </template>
        </NInput>
      </div>
      <div class="w-full flex-1 overflow-x-hidden overflow-y-auto p-1 pr-2">
        <Tree :keyword="state.keyword" />
      </div>
    </div>

    <div
      class="absolute rounded-full shadow-lg w-6 h-6 top-16 flex items-center justify-center bg-white hover:bg-control-bg cursor-pointer z-1 transition-all"
      :class="[state.expand ? 'left-full -translate-x-3' : '-left-3']"
      @click="state.expand = !state.expand"
    >
      <heroicons-outline:chevron-left
        class="w-4 h-4 transition-transform"
        :class="[state.expand ? '' : '-scale-100 translate-x-[3px]']"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NInput } from "naive-ui";
import { reactive } from "vue";
import { hasSchemaProperty } from "@/utils";
import { useSchemaDiagramContext } from "../common";
import SchemaSelector from "./SchemaSelector.vue";
import Tree from "./Tree.vue";

type LocalState = {
  expand: boolean;
  keyword: string;
};

const state = reactive<LocalState>({
  expand: true,
  keyword: "",
});

const { databaseMetadata, selectedSchemaNames, database } =
  useSchemaDiagramContext();
</script>
