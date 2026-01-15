<template>
  <div class="relative bg-white pt-3">
    <div class="absolute w-full bottom-0 leading-0 border-b border-b-gray-200" />
    <div class="flex items-center justify-between gap-x-4">
      <div class="flex items-center overflow-x-auto px-4">
        <template v-for="(stage, index) in rollout.stages" :key="stage.name">
          <div
            class="relative px-4 py-2 transition-all border rounded-t-lg cursor-pointer"
            :class="[
              selectedStageId === stage.name
                ? 'bg-gray-50 border-gray-200 border-b-transparent'
                : 'bg-white hover:bg-gray-50 border-transparent border-b-gray-200',
            ]"
            @click="$emit('select-stage', stage)"
          >
            <StageProgressCard :stage="stage" />
          </div>
          <ArrowRightIcon
            v-if="index < rollout.stages.length - 1"
            :size="16"
            class="text-gray-400 shrink-0 mx-2"
          />
        </template>
      </div>

      <div v-if="hasPendingTasks" class="flex items-center px-4 shrink-0">
        <NButton size="small" @click="$emit('open-preview')">
          <template #icon>
            <EyeIcon class="w-4 h-4" />
          </template>
          {{ $t("rollout.pending-tasks-preview.action") }}
        </NButton>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ArrowRightIcon, EyeIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import type { Rollout, Stage } from "@/types/proto-es/v1/rollout_service_pb";
import StageProgressCard from "./StageProgressCard.vue";

defineProps<{
  selectedStageId?: string;
  rollout: Rollout;
  hasPendingTasks: boolean;
}>();

defineEmits<{
  (event: "select-stage", stage: Stage): void;
  (event: "open-preview"): void;
}>();
</script>
