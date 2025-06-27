<template>
  <ChecksView />
</template>

<script lang="ts" setup>
import { usePollPlan } from "@/components/Plan";
import { ChecksView } from "@/components/Plan/components";

usePollPlan();
</script>
