<template>
  <ul
    class="flex flex-col whitespace-nowrap"
    :class="[errors.length > 1 && 'list-disc pl-4']"
  >
    <li v-for="(error, i) in errors" :key="i">
      {{ error }}
    </li>
  </ul>
</template>

<script setup lang="ts">
defineProps<{
  errors: string[];
}>();
</script>
