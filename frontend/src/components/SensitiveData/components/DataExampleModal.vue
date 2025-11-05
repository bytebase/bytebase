<template>
  <BBModal
    :close-on-esc="true"
    :mask-closable="true"
    :trap-focus="false"
    :title="$t('settings.sensitive-data.json-format-example')"
    class="w-3xl max-w-full"
    @close="$emit('dismiss')"
  >
    <div class="my-4 rounded-xs p-4 bg-gray-100 relative">
      <div class="absolute top-2 right-2 p-2">
        <CopyButton
          quaternary
          :text="false"
          :size="'medium'"
          :content="example"
        />
      </div>
      <NConfigProvider :hljs="hljs">
        <NCode language="json" :code="example" />
      </NConfigProvider>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import hljs from "highlight.js/lib/core";
import { NCode, NConfigProvider } from "naive-ui";
import { BBModal } from "@/bbkit";
import { CopyButton } from "@/components/v2";

defineProps<{
  example: string;
}>();

defineEmits<{
  (event: "dismiss"): void;
}>();
</script>
