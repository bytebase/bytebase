<template>
  <section v-if="active" :aria-hidden="!active">
    <slot />
  </section>
</template>

<script lang="ts" setup>
import { defineProps } from "vue";

defineProps<{ active: boolean }>();
</script>
