<template>
  <NPerformantEllipsis v-if="!downGrade">
    <template #default>
      <!-- eslint-disable-next-line vue/no-v-html -->
      <span v-html="html" />
    </template>
    <template #tooltip>
      <!-- eslint-disable vue/no-v-html -->
      <div
        class="text-wrap whitespace-pre break-words break-all"
        style="max-width: calc(min(33vw, 320px))"
        v-html="html"
      />
      <!-- eslint-enable -->
    </template>
  </NPerformantEllipsis>
  <!-- eslint-disable-next-line vue/no-v-html -->
  <div v-else :title="content" class="truncate" v-html="html" />
</template>

<script setup lang="ts">
import { NPerformantEllipsis } from "naive-ui";
import { computed } from "vue";
import { getHighlightHTMLByRegExp } from "@/utils";

const props = defineProps<{
  content: string;
  downGrade?: boolean;
  keyword?: string;
}>();

const html = computed(() => {
  const { content, keyword } = props;
  if (!keyword) return content;
  return getHighlightHTMLByRegExp(content, keyword);
});
</script>
