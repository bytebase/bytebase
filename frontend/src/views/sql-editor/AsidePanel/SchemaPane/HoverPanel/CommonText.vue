<template>
  <div class="max-w-[18rem] whitespace-pre-wrap break-all break-words">
    {{ content }}
  </div>
</template>

<script setup lang="ts">
defineProps<{
  content: string;
}>();
</script>
