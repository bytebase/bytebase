<template>
  <div class="max-w-[18rem] whitespace-pre-wrap break-all wrap-break-word">
    {{ content }}
  </div>
</template>

<script setup lang="ts">
defineProps<{
  content: string;
}>();
</script>
