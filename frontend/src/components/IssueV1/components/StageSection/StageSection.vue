<template>
  <div class="w-full flex flex-col divide-y">
    <template v-if="stageList.length > 0">
      <div class="px-0">
        <NScrollbar :x-scrollable="true" class="max-h-[180px] sm:max-h-max">
          <div
            class="flex flex-col sm:flex-row justify-between divide-y sm:divide-y-0"
          >
            <template v-for="(stage, index) in stageList" :key="stage.name">
              <StageCard
                :stage="stage"
                :index="index"
                class="h-[54px] px-2 sm:px-0"
                :class="[
                  index === 0 && 'sm:pl-4',
                  index === stageList.length - 1 && 'sm:pr-4',
                ]"
              />
              <div
                v-if="index < stageList.length - 1"
                class="hidden sm:block w-3.5 h-[54px] mr-2 pointer-events-none shrink-0"
                aria-hidden="true"
              >
                <svg
                  class="h-full w-full text-gray-300"
                  viewBox="0 0 22 80"
                  fill="none"
                  preserveAspectRatio="none"
                >
                  <path
                    d="M0 -2L20 40L0 82"
                    vector-effect="non-scaling-stroke"
                    stroke="currentcolor"
                    stroke-linejoin="round"
                  />
                </svg>
              </div>
            </template>
          </div>
        </NScrollbar>
      </div>
    </template>
    <template v-else>
      <NEmpty class="py-6" />
    </template>
  </div>
</template>

<script lang="ts" setup>
import { NEmpty, NScrollbar } from "naive-ui";
import { computed } from "vue";
import { useIssueContext } from "@/components/IssueV1/logic";
import StageCard from "./StageCard.vue";

const { issue } = useIssueContext();

const stageList = computed(() => {
  return issue.value.rolloutEntity?.stages || [];
});
</script>
