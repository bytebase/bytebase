<template>
  <section v-if="active" :aria-hidden="!active">
    <slot />
  </section>
</template>

<script lang="ts" setup>
defineProps<{ active: boolean }>();
</script>
