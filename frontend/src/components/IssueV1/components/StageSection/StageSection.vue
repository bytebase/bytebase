<template>
  <div class="w-full flex flex-col px-4">
    <div class="flex flex-col lg:flex-row justify-between">
      <template v-for="(stage, index) in stageList" :key="stage.uid">
        <StageCard :stage="stage" :index="index" class="h-[60px]" />
        <div
          v-if="index < stageList.length - 1"
          class="hidden lg:block w-5 h-[60px] mr-2 pointer-events-none shrink-0"
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

    <div class="w-full border-t mb-4" />

    <div class="lg:flex items-start justify-between">
      <StageInfo />

      <Actions />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useIssueContext } from "@/components/IssueV1/logic";
import Actions from "./Actions";
import StageCard from "./StageCard.vue";
import StageInfo from "./StageInfo";

const { issue } = useIssueContext();

const stageList = computed(() => {
  return issue.value.rolloutEntity.stages;
});
</script>
