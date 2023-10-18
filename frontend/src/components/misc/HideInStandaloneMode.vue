<template>
  <slot v-if="pageMode !== 'STANDALONE'" name="default" />
</template>

<script setup lang="ts">
import { storeToRefs } from "pinia";
import { useActuatorV1Store } from "@/store";

const { pageMode } = storeToRefs(useActuatorV1Store());
</script>
