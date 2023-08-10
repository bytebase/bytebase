<template>
  <div class="mb-2 flex flex-row items-center">
    <span class="text-lg mr-2">{{ $t("common.databases") }}</span>
    <BBLoader v-show="loading" class="opacity-60" />
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
        class="w-full flex flex-row justify-start items-center px-2 py-1 gap-x-2"
      >
        <NEllipsis class="text-sm" line-clamp="1">
          {{ database.databaseName }}
        </NEllipsis>
        <div class="flex-1 flex flex-row justify-end items-center shrink-0">
          <FeatureBadge
            feature="bb.feature.database-grouping"
            custom-class="mr-2"
            :instance="database.instanceEntity"
          />
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
          ({{ database.instanceEntity.environmentEntity.title }})
        </NEllipsis>
        <div class="flex flex-row justify-end items-center shrink-0">
          <NEllipsis class="ml-1 text-sm text-gray-400" line-clamp="1">
            <InstanceV1Name :instance="database.instanceEntity" :link="false" />
          </NEllipsis>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NEllipsis } from "naive-ui";
import { reactive } from "vue";
import BBLoader from "@/bbkit/BBLoader.vue";
import { ComposedDatabase } from "@/types";
import { InstanceV1EngineIcon } from "../v2";

interface LocalState {
  showMatchedDatabaseList: boolean;
  showUnmatchedDatabaseList: boolean;
}

defineProps<{
  loading: boolean;
  matchedDatabaseList: ComposedDatabase[];
  unmatchedDatabaseList: ComposedDatabase[];
}>();

const state = reactive<LocalState>({
  showMatchedDatabaseList: true,
  showUnmatchedDatabaseList: true,
});
</script>
