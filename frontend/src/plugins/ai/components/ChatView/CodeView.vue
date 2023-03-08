<template>
  <div class="flex items-start w-full">
    <MonacoEditor
      ref="editorRef"
      class="flex-1 border h-auto max-h-[240px]"
      language="sql"
      :value="message.content"
      :readonly="true"
      :auto-focus="false"
      :options="{
        fontSize: 12,
        lineHeight: 14,
        lineNumbers: 'off',
        wordWrap: 'off',
        scrollbar: {
          vertical: 'hidden',
          horizontal: 'hidden',
          useShadows: false,
          verticalScrollbarSize: 0,
          horizontalScrollbarSize: 0,
        },
      }"
      @ready="handleMonacoEditorReady"
    />
    <div class="flex flex-col gap-y-2 ml-0.5 mt-1">
      <button
        class="inline-flex items-center justify-center hover:text-accent cursor-pointer"
        @click="handleExecute"
      >
        <heroicons-outline:play class="w-4 h-4" />
      </button>
      <button
        class="inline-flex items-center justify-center hover:text-accent cursor-pointer"
        @click="handleCopy"
      >
        <heroicons-outline:clipboard-copy class="w-4 h-4" />
      </button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { toClipboard } from "@soerenmartius/vue3-clipboard";

import type { Message } from "../../types";
import MonacoEditor from "@/components/MonacoEditor";
import { useAIContext } from "../../logic";
import { pushNotification } from "@/store";
import { useI18n } from "vue-i18n";

const props = defineProps<{
  message: Message;
}>();

const { t } = useI18n();
const context = useAIContext();
const editorRef = ref<InstanceType<typeof MonacoEditor>>();

const updateEditorHeight = () => {
  const contentHeight =
    editorRef.value?.editorInstance?.getContentHeight() as number;
  editorRef.value?.setEditorContentHeight(contentHeight);
};

const handleMonacoEditorReady = () => {
  updateEditorHeight();
};

const handleExecute = () => {
  context.events.emit("apply-statement", {
    statement: props.message.content,
    run: true,
  });
};

const handleCopy = () => {
  toClipboard(props.message.content).then(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("plugin.ai.statement-copied"),
    });
  });
};
</script>
