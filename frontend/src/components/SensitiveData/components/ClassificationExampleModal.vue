<template>
  <BBModal
    :close-on-esc="true"
    :mask-closable="true"
    :trap-focus="false"
    :title="$t('settings.sensitive-data.classification.json-format-example')"
    class="w-[48rem] max-w-full"
    @close="$emit('dismiss')"
  >
    <div class="my-4 rounded-sm p-4 bg-gray-100 relative">
      <div
        class="absolute top-2 right-2 p-2 rounded bg-gray-200 text-gray-500 hover:text-gray-700 hover:bg-gray-300 cursor-pointer"
        @click="handleCopy"
      >
        <heroicons-outline:clipboard-document class="w-4 h-4" />
      </div>
      <NConfigProvider :hljs="hljs">
        <NCode language="json" :code="JSON.stringify(example, null, 2)" />
      </NConfigProvider>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { useClipboard } from "@vueuse/core";
import hljs from "highlight.js/lib/core";
import json from "highlight.js/lib/languages/json";
import { NCode, NConfigProvider } from "naive-ui";
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";

hljs.registerLanguage("json", json);

defineEmits<{
  (event: "dismiss"): void;
}>();

// TODO: localization.
const example = {
  title: "Classification Example",
  levels: [
    {
      id: "1",
      title: "Level 1",
      description: "",
    },
    {
      id: "2",
      title: "Level 2",
      description: "",
    },
  ],
  classification: {
    "1": {
      id: "1",
      title: "Basic",
      description: "",
    },
    "1-1": {
      id: "1-1",
      title: "Basic",
      description: "",
    },
    "1-2": {
      id: "1-2",
      title: "Assert",
      description: "",
    },
    "1-3": {
      id: "1-3",
      title: "Contact",
      description: "",
    },
    "1-4": {
      id: "1-4",
      title: "Health",
      description: "",
    },
    "2": {
      id: "2",
      title: "Relationship",
      description: "",
    },
    "2-1": {
      id: "2-1",
      title: "Social",
      description: "",
    },
    "2-2": {
      id: "2-1",
      title: "Business",
      description: "",
    },
  },
};

const { copy: copyTextToClipboard } = useClipboard();
const { t } = useI18n();

const handleCopy = () => {
  copyTextToClipboard(JSON.stringify(example, null, 2));
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.sensitive-data.classification.copy-succeed"),
  });
};
</script>
