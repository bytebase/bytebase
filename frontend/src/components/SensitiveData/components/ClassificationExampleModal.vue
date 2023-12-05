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
  title: "数据分级分类样例",
  levels: [
    {
      id: "1",
      title: "1级",
      description: "",
    },
    {
      id: "2",
      title: "2级",
      description: "",
    },
  ],
  classification: {
    "1": {
      id: "1",
      title: "个人",
      description: "",
    },
    "1-1": {
      id: "1-1",
      title: "个人自然信息",
      description: "",
    },
    "1-1-1": {
      id: "1-1-1",
      title: "个人基本信息",
      description: "指个人基本情况数据，如个人姓名、性别、国籍、家庭住址等。",
      levelId: "1",
    },
    "1-1-2": {
      id: "1-1-2",
      title: "个人财产信息",
      description: "指个人的财产数据，如个人收入状况、拥有的不动产状况等。",
      levelId: "2",
    },
    "1-2": {
      id: "1-2",
      title: "个人身份信息",
      description: "",
    },
    "1-2-1": {
      id: "1-2-1",
      title: "传统鉴别信息",
      description: "指各类常规个人身份鉴别技术手段所依赖的数据。",
      levelId: "2",
    },
    "2": {
      id: "2",
      title: "单位",
      description: "",
    },
    "2-1": {
      id: "2-1",
      title: "单位基本信息",
      description: "",
    },
    "2-1-1": {
      id: "2-1-1",
      title: "单位基本概况",
      description: "指单位基础概况数据。",
      levelId: "1",
    },
  },
};

const { copy: copyTextToClipboard } = useClipboard();
const { t } = useI18n();

const handleCopy = () => {
  console.log("handleCopy");
  copyTextToClipboard(JSON.stringify(example, null, 2));
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.sensitive-data.classification.copy-succeed"),
  });
};
</script>
