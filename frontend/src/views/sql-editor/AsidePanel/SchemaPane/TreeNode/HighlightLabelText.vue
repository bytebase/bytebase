<template>
  <span v-if="!keyword.trim()" v-bind="$attrs">{{ text }}</span>
  <!-- eslint-disable-next-line vue/no-v-html -->
  <span v-else v-bind="$attrs" v-html="html" />
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
