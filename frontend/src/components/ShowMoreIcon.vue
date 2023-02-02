<template>
  <n-tooltip trigger="hover">
    <template #trigger>
      <heroicons-outline:question-mark-circle class="w-5 h-5" v-bind="$attrs" />
    </template>
    {{ content }}
  </n-tooltip>
</template>

<script lang="ts" setup>
withDefaults(
  defineProps<{
    content: string;
  }>(),
  {
    content: "",
  }
);
</script>
