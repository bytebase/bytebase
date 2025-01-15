<template>
  <div class="w-full flex grow overflow-auto bg-zinc-100 p-6 rounded-md">
    <div class="relative w-auto flex flex-row items-start justify-start gap-6">
      <div class="absolute top-5 border-2 w-full"></div>
      <StageCard
        v-for="stage in mergedStages"
        :key="stage.name"
        :stage="stage"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRolloutDetailContext } from "../../context";
import StageCard from "./StageCard.vue";

const { rollout, rolloutPreview } = useRolloutDetailContext();

const mergedStages = computed(() => {
  // Merge preview stages with created rollout stages.
  return rolloutPreview.value.stages.map((sp) => {
    const createdStage = rollout.value.stages.find((s) => s.id === sp.id);
    return createdStage || sp;
  });
});
</script>
