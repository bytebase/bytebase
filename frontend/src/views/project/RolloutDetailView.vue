<template>
  <RolloutView />
</template>

<script lang="ts" setup>
import { usePollPlan } from "@/components/Plan";
import { RolloutView } from "@/components/Plan/components/RolloutView";

usePollPlan();
</script>
