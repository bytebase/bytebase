<template>
  <div class="tooltip-wrapper">
    <span
      v-if="content !== ''"
      class="tooltip whitespace-nowrap"
      :class="`tooltip-${position}`"
    >
      {{ content }}
    </span>
    <heroicons-outline:question-mark-circle class="w-5 h-5" />
  </div>
</template>

<script lang="ts" setup>
withDefaults(
  defineProps<{
    content: string;
    position?: string;
  }>(),
  {
    content: "",
    position: "top",
  }
);
</script>
