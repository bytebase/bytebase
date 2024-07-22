<template>
  <slot v-if="!inIframe" name="default" />
</template>

<script setup lang="ts">
import { useCustomFeature } from "@/store";

const inIframe = useCustomFeature("bb.feature.embedded-in-iframe");
</script>
