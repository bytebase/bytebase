<template>
  <BBModal
    :close-on-esc="true"
    :mask-closable="true"
    :trap-focus="false"
    :title="$t('settings.sensitive-data.json-format-example')"
    class="w-[48rem] max-w-full"
    @close="$emit('dismiss')"
  >
    <div class="my-4 rounded-sm p-4 bg-gray-100 relative">
      <div
        v-if="isSupported"
        class="absolute top-2 right-2 p-2 rounded bg-gray-200 text-gray-500 hover:text-gray-700 hover:bg-gray-300 cursor-pointer"
        @click="handleCopy"
      >
        <heroicons-outline:clipboard-document class="w-4 h-4" />
      </div>
      <NConfigProvider :hljs="hljs">
        <NCode language="json" :code="example" />
      </NConfigProvider>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { useClipboard } from "@vueuse/core";
import hljs from "highlight.js/lib/core";
import { NCode, NConfigProvider } from "naive-ui";
import { useI18n } from "vue-i18n";
import { BBModal } from "@/bbkit";
import { pushNotification } from "@/store";

const props = defineProps<{
  example: string;
}>();

defineEmits<{
  (event: "dismiss"): void;
}>();

const { copy: copyTextToClipboard, isSupported } = useClipboard({
  legacy: true,
});
const { t } = useI18n();

const handleCopy = () => {
  copyTextToClipboard(props.example).then(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("settings.sensitive-data.classification.copy-succeed"),
    });
  });
};
</script>
