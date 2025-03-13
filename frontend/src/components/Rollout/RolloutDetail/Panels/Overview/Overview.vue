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
import { useRolloutDetailContext } from "../../context";
import StageCard from "./StageCard.vue";

const { mergedStages } = useRolloutDetailContext();
</script>
