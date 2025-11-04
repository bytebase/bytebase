<template>
  <NPerformantEllipsis v-if="!downGrade">
    <template #default>
      <!-- eslint-disable-next-line vue/no-v-html -->
      <span v-bind="$attrs" v-html="html" />
    </template>
    <template #tooltip>
      <!-- eslint-disable vue/no-v-html -->
      <div
        class="text-wrap whitespace-pre wrap-break-word break-all"
        style="max-width: calc(min(33vw, 320px))"
        v-html="tooltipHTML"
      />
      <!-- eslint-enable -->
    </template>
  </NPerformantEllipsis>
  <!-- eslint-disable-next-line vue/no-v-html -->
  <div v-else :title="content" class="truncate" v-html="html" />
</template>

<script setup lang="ts">
import { escape } from "lodash-es";
import { NPerformantEllipsis } from "naive-ui";
import { computed } from "vue";
import { getHighlightHTMLByRegExp, type VueClass } from "@/utils";

defineOptions({
  inheritAttrs: false,
});

const props = defineProps<{
  content: string;
  downGrade?: boolean;
  keyword?: string;
  tooltip?: string;
  tooltipClass?: VueClass;
}>();

const html = computed(() => {
  const { content, keyword } = props;
  const kw = (keyword ?? "").trim();
  if (!kw) {
    return escape(content);
  }
  return getHighlightHTMLByRegExp(
    escape(content),
    escape(kw),
    false /* !caseSensitive */
  );
});

const tooltipHTML = computed(() => {
  const { keyword, tooltip } = props;
  if (typeof tooltip === "string") {
    const kw = (keyword ?? "").trim();
    if (!kw) {
      return escape(tooltip);
    }
    return getHighlightHTMLByRegExp(
      escape(tooltip),
      escape(kw),
      false /* !caseSensitive */
    );
  }
  return html.value;
});
</script>
