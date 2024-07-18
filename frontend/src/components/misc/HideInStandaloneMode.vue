<template>
  <slot v-if="!inIframe" name="default" />
</template>

<script setup lang="ts">
import { useCustomFeature } from "@/store";

const inIframe = useCustomFeature("bb.custom-feature.embedded-in-iframe");
</script>
