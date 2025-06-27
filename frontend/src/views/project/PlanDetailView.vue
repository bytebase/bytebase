<template>
  <Overview />
</template>

<script lang="ts" setup>
import { usePollPlan } from "@/components/Plan";
import { Overview } from "@/components/Plan/components";

usePollPlan();
</script>
