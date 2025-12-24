<template>
  <div class="relative bg-white pt-3">
    <div class="absolute w-full bottom-0 leading-0 border-b border-b-gray-200" />
    <div class="flex items-center justify-between gap-x-4">
      <!-- Left: Stage tabs -->
      <div class="flex items-center overflow-x-auto px-4">
        <template v-for="(stage, index) in stages" :key="stage.name">
          <div
            class="relative px-4 py-2 transition-all border rounded-t-lg cursor-pointer"
            :class="[
              selectedStageId === stage.name
                ? 'bg-gray-50 border-gray-200 border-b-transparent'
                : 'bg-white hover:bg-gray-50 border-transparent border-b-gray-200',
              !isStageCreated(stage) && 'opacity-80',
            ]"
            @click="handleStageClick(stage)"
          >
            <StageProgressCard
              :stage="stage"
              :is-created="isStageCreated(stage)"
            />
          </div>
          <ArrowRightIcon
            v-if="index < stages.length - 1"
            :size="16"
            class="text-gray-400 shrink-0 mx-2"
          />
        </template>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ArrowRightIcon } from "lucide-vue-next";
import type { Stage } from "@/types/proto-es/v1/rollout_service_pb";
import StageProgressCard from "./StageProgressCard.vue";

defineProps<{
  stages: Stage[];
  selectedStageId?: string;
  isStageCreated: (stage: Stage) => boolean;
}>();

const emit = defineEmits<{
  (event: "select-stage", stage: Stage): void;
}>();

const handleStageClick = (stage: Stage) => {
  emit("select-stage", stage);
};
</script>
