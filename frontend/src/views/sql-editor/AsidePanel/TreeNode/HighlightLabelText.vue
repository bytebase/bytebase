<template>
  <template v-if="!keyword.trim()">{{ text }}</template>
  <!-- eslint-disable-next-line vue/no-v-html -->
  <span v-else v-html="html" />
</template>

<script lang="ts" setup>
import { escape } from "lodash-es";
import { computed } from "vue";
import { getHighlightHTMLByRegExp } from "@/utils";

const props = defineProps<{
  text: string;
  keyword: string;
}>();

const html = computed(() => {
  return getHighlightHTMLByRegExp(
    escape(props.text),
    escape(props.keyword.trim()),
    false /* !caseSensitive */
  );
});
</script>
