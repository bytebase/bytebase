<template>
  <NPerformantEllipsis v-if="!downGrade">
    <template #default>{{ content }}</template>
    <template #tooltip>
      <div
        class="text-wrap whitespace-pre break-words break-all"
        style="max-width: calc(min(33vw, 320px))"
      >
        {{ content }}
      </div>
    </template>
  </NPerformantEllipsis>
  <div v-else :title="content" class="truncate">
    {{ content }}
  </div>
</template>

<script setup lang="ts">
import { NPerformantEllipsis } from "naive-ui";

defineProps<{
  content: string;
  downGrade?: boolean;
}>();
</script>
