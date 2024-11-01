<template>
  <div v-if="!hideTitle" class="mb-2 flex flex-row items-center">
    <span class="font-medium text-main mr-2">{{ $t("common.databases") }}</span>
    <BBSpin v-if="loading" class="opacity-60" />
  </div>

  <NCollapse
    class="border p-2 rounded-lg"
    :expanded-names="collapseExpandedNames"
    @update:expanded-names="onExpandedNamesChange"
  >
    <NCollapseItem
      :title="$t('database-group.matched-database')"
      :disabled="matchedDatabaseList.length === 0"
      name="matched"
    >
      <template #header-extra>{{ matchedDatabaseList.length }}</template>
      <NVirtualList
        class="w-full py-1 max-h-[12rem]"
        :item-size="28"
        :items="matchedDatabaseList"
      >
        <template #default="{ item: database }">
          <div
            :key="database.name"
            class="w-full flex flex-row justify-between items-center px-2 py-1 gap-x-2"
          >
            <NEllipsis class="text-sm" line-clamp="1">
              {{ database.databaseName }}
            </NEllipsis>
            <div class="flex-1 flex flex-row justify-end items-center shrink-0">
              <FeatureBadge
                feature="bb.feature.database-grouping"
                custom-class="mr-2"
                :instance="database.instanceResource"
              />
              <InstanceV1EngineIcon :instance="database.instanceResource" />
              <NEllipsis
                class="ml-1 text-sm text-gray-400 max-w-[124px]"
                line-clamp="1"
              >
                ({{ database.effectiveEnvironmentEntity.title }})
                {{ database.instanceResource.title }}
              </NEllipsis>
            </div>
          </div>
        </template>
      </NVirtualList>
    </NCollapseItem>
    <NCollapseItem
      :title="$t('database-group.unmatched-database')"
      :disabled="unmatchedDatabaseList.length === 0"
      name="unmatched"
    >
      <template #header-extra>{{ unmatchedDatabaseList.length }}</template>
      <NVirtualList
        class="w-full py-1 max-h-[12rem]"
        :item-size="28"
        :items="unmatchedDatabaseList"
      >
        <template #default="{ item: database }">
          <div
            :key="database.name"
            class="w-full flex flex-row justify-between items-center px-2 py-1 gap-x-2"
          >
            <NEllipsis class="text-sm" line-clamp="1">
              {{ database.databaseName }}
            </NEllipsis>
            <div class="flex-1 flex flex-row justify-end items-center shrink-0">
              <FeatureBadge
                feature="bb.feature.database-grouping"
                custom-class="mr-2"
                :instance="database.instanceResource"
              />
              <InstanceV1EngineIcon :instance="database.instanceResource" />
              <NEllipsis
                class="ml-1 text-sm text-gray-400 max-w-[124px]"
                line-clamp="1"
              >
                ({{ database.effectiveEnvironmentEntity.title }})
                {{ database.instanceResource.title }}
              </NEllipsis>
            </div>
          </div>
        </template>
      </NVirtualList>
    </NCollapseItem>
  </NCollapse>
</template>

<script lang="ts" setup>
import { NEllipsis, NVirtualList, NCollapse, NCollapseItem } from "naive-ui";
import { ref, watch } from "vue";
import { BBSpin } from "@/bbkit";
import type { ComposedDatabase } from "@/types";
import { FeatureBadge } from "../FeatureGuard";
import { InstanceV1EngineIcon } from "../v2";

const props = defineProps<{
  matchedDatabaseList: ComposedDatabase[];
  unmatchedDatabaseList: ComposedDatabase[];
  loading?: boolean;
  hideTitle?: boolean;
}>();

const collapseExpandedNames = ref<string[]>([]);

const onExpandedNamesChange = (expandedNames: string[]) => {
  collapseExpandedNames.value = expandedNames;
};

watch(
  () => props.matchedDatabaseList.length,
  () => {
    collapseExpandedNames.value = [];
    if (props.matchedDatabaseList.length > 0) {
      collapseExpandedNames.value.push("matched");
    }
    if (props.unmatchedDatabaseList.length > 0) {
      collapseExpandedNames.value.push("unmatched");
    }
  },
  {
    immediate: true,
  }
);
</script>
