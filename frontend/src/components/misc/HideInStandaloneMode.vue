<template>
  <slot v-if="!inIframe" name="default" />
</template>

<script setup lang="ts">
import { useAppFeature } from "@/store";

const inIframe = useAppFeature("bb.feature.embedded-in-iframe");
</script>
