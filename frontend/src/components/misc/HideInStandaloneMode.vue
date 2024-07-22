<template>
  <slot v-if="!embedded" name="default" />
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useActuatorV1Store } from "@/store";

const embedded = computed(() => useActuatorV1Store().appProfile.embedded);
</script>
