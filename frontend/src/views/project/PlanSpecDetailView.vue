<template>
  <SpecsView />
</template>

<script lang="ts" setup>
import { usePollPlan } from "@/components/Plan";
import SpecsView from "@/components/Plan/components/SpecsView.vue";

usePollPlan();
</script>
