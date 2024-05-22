<template>
  <div class="w-full flex flex-col divide-y">
    <template v-if="stepList.length > 0">
      <div class="px-0">
        <NScrollbar :x-scrollable="true" class="max-h-[180px] sm:max-h-max">
          <div
            class="flex flex-col sm:flex-row justify-between divide-y sm:divide-y-0"
          >
            <template
              v-for="(step, index) in stepList"
              :key="`${index}-${step - title}`"
            >
              <StepCard
                :step="step"
                :index="index"
                class="px-2 sm:px-0 h-10"
                :class="[
                  index === 0 && 'sm:pl-4',
                  index === stepList.length - 1 && 'sm:pr-4',
                ]"
              />
              <div
                v-if="index < stepList.length - 1"
                class="hidden h-10 sm:block w-3.5 mr-2 pointer-events-none shrink-0"
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
  </div>
</template>

<script lang="ts" setup>
import { NScrollbar } from "naive-ui";
import { computed } from "vue";
import { usePlanContext } from "@/components/Plan/logic";
import StepCard from "./StepCard.vue";

const { plan } = usePlanContext();

const stepList = computed(() => {
  return plan.value.steps || [];
});
</script>
