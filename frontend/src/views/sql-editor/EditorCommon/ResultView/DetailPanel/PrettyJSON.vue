<template>
  <!-- eslint-disable-next-line vue/no-v-html -->
  <div v-html="state.html" />
</template>

<script setup lang="ts">
import { escape } from "lodash-es";
import { shallowReactive, watch } from "vue";

const props = defineProps<{
  content: string;
}>();

const state = shallowReactive({
  html: "",
  controller: new AbortController(),
});

watch(
  () => props.content,
  async (content) => {
    if (state.controller) {
      state.controller.abort();
      state.controller = new AbortController();
    }
    const controller = state.controller;
    const finish = (html: string) => {
      if (controller.signal.aborted) {
        return;
      }
      state.html = html;
    };

    try {
      const { prettyPrintJson } = await import("pretty-print-json");
      const { parse } = await import("lossless-json");
      const { losslessReviver } = await import("@/composables/utils");
      await import("./pretty-print-json.css");
      // Use lossless-json to preserve precision for large integers (> 2^53-1)
      const obj = parse(content, null, losslessReviver);
      const html = prettyPrintJson.toHtml(obj, {
        quoteKeys: true,
        trailingCommas: false,
      });
      finish(html);
    } catch (err) {
      console.warn("[PrettyJSON]", err);

      finish(escape(props.content));
    }
  },
  {
    immediate: true,
  }
);
</script>
