<template>
  <div v-if="snippet" class="flex flex-col gap-y-3">
    <p class="text-sm text-main leading-relaxed">
      {{ snippet.content }}
    </p>
    <div v-if="snippet.codeBlock" class="flex flex-col gap-y-1">
      <div class="flex flex-row">
        <NConfigProvider
          class="flex-1 min-w-0 w-full inline-flex items-center px-3 py-2 border border-control-border bg-gray-50 text-xs whitespace-pre-line rounded-l-[3px] overflow-x-auto"
          :hljs="hljs"
        >
          <NCode
            word-wrap
            :language="snippet.codeBlock.language"
            :code="snippet.codeBlock.code"
          />
        </NConfigProvider>
        <div
          class="flex items-center -ml-px px-2 py-2 border border-gray-300 text-sm font-medium text-control-light bg-gray-50 hover:bg-gray-100 rounded-r-[3px]"
        >
          <CopyButton :content="snippet.codeBlock.code" />
        </div>
      </div>
    </div>
    <div
      v-if="snippet.learnMoreLinks && snippet.learnMoreLinks.length > 0"
      class="flex flex-col gap-y-1"
    >
      <a
        v-for="link in snippet.learnMoreLinks"
        :key="link.url"
        :href="link.url"
        target="_blank"
        rel="noopener noreferrer"
        class="text-xs accent-link"
      >
        {{ link.title }}
      </a>
    </div>
  </div>
  <div v-else class="text-sm text-control-light italic">
    {{ $t("instance.info-panel.no-info") }}
  </div>
</template>

<script lang="ts" setup>
import hljs from "highlight.js/lib/core";
import { NCode, NConfigProvider } from "naive-ui";
import { computed } from "vue";
import { CopyButton } from "@/components/v2";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import { getInfoContent, type InfoSection } from "./info-content";

const props = defineProps<{
  engine: Engine;
  section: InfoSection;
}>();

const snippet = computed(() => {
  return getInfoContent(props.engine, props.section);
});
</script>
