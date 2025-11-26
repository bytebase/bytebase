<template>
  <!-- eslint-disable-next-line vue/no-v-text-v-html-on-component -->
  <component :is="tag" :class="[language, $attrs.class]" v-html="highlightedCode" />
</template>

<script lang="ts" setup>
import hljs from "highlight.js/lib/core";
import { computed, onMounted, ref, watch } from "vue";

const props = withDefaults(
  defineProps<{
    code: string;
    language?: string;
    tag?: string;
    lazy?: boolean;
  }>(),
  {
    language: "sql",
    tag: "pre",
    lazy: false,
  }
);

defineOptions({
  inheritAttrs: false,
});

const isReady = ref(!props.lazy);
const cachedHtml = ref("");

const highlightedCode = computed(() => {
  if (!isReady.value) {
    return escapeHtml(props.code);
  }
  return cachedHtml.value || escapeHtml(props.code);
});

const escapeHtml = (text: string): string => {
  if (!text) return "";
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
};

const doHighlight = () => {
  if (!props.code) {
    cachedHtml.value = "";
    isReady.value = true;
    return;
  }

  try {
    const result = hljs.highlight(props.code, {
      language: props.language,
      ignoreIllegals: true,
    });
    cachedHtml.value = result.value;
  } catch {
    cachedHtml.value = escapeHtml(props.code);
  }
  isReady.value = true;
};

const scheduleHighlight = () => {
  if (props.lazy) {
    if (typeof requestIdleCallback !== "undefined") {
      requestIdleCallback(() => doHighlight(), { timeout: 100 });
    } else {
      setTimeout(doHighlight, 0);
    }
  } else {
    doHighlight();
  }
};

// Initialize
if (!props.lazy) {
  doHighlight();
}

onMounted(() => {
  if (props.lazy) {
    scheduleHighlight();
  }
});

// Watch for changes
watch(
  () => [props.code, props.language] as const,
  () => {
    isReady.value = false;
    scheduleHighlight();
  }
);
</script>
