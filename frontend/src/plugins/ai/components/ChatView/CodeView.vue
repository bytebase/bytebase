<template>
  <div class="flex items-start w-full">
    <div class="flex-1 overflow-x-hidden">
      <MonacoEditor
        ref="editorRef"
        class="border h-auto"
        language="sql"
        :value="state.code"
        :readonly="!state.editing"
        :auto-focus="false"
        :options="{
          automaticLayout: true,
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
            alwaysConsumeMouseWheel: false,
          },
        }"
        @ready="handleMonacoEditorReady"
        @change="state.code = $event"
      />
    </div>
    <div class="flex flex-col gap-y-2 ml-0.5 mt-1">
      <NTooltip placement="right">
        <template #trigger>
          <button
            class="inline-flex items-center justify-center hover:text-accent cursor-pointer"
            @click="handleExecute"
          >
            <heroicons:play-circle class="w-4 h-4" />
          </button>
        </template>
        <div class="whitespace-nowrap">
          {{ $t("common.run") }}
        </div>
      </NTooltip>
      <NTooltip placement="right">
        <template #trigger>
          <button
            class="inline-flex items-center justify-center hover:text-accent cursor-pointer"
            @click="handleCopy"
          >
            <heroicons:clipboard class="w-4 h-4" />
          </button>
        </template>
        <div class="whitespace-nowrap">
          {{ $t("common.copy") }}
        </div>
      </NTooltip>
      <template v-if="!state.editing">
        <NTooltip placement="right">
          <template #trigger>
            <button
              class="inline-flex items-center justify-center hover:text-accent cursor-pointer"
              @click="state.editing = true"
            >
              <heroicons:pencil class="w-4 h-4" />
            </button>
          </template>
          <div class="whitespace-nowrap">
            {{ $t("common.edit") }}
          </div>
        </NTooltip>
      </template>
      <template v-if="state.editing">
        <NTooltip placement="right">
          <template #trigger>
            <button
              class="inline-flex items-center justify-center hover:text-accent cursor-pointer"
              @click="finishEditing(false)"
            >
              <heroicons:arrow-uturn-left class="w-4 h-4" />
            </button>
          </template>
          <div class="whitespace-nowrap">
            {{ $t("common.cancel") }}
          </div>
        </NTooltip>
        <NTooltip placement="right">
          <template #trigger>
            <button
              class="inline-flex items-center justify-center hover:text-accent cursor-pointer"
              @click="finishEditing(true)"
            >
              <heroicons:check class="w-4 h-4" />
            </button>
          </template>
          <div class="whitespace-nowrap">
            {{ $t("common.save") }}
          </div>
        </NTooltip>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import { NTooltip } from "naive-ui";
import { reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import MonacoEditor from "@/components/MonacoEditor";
import { pushNotification } from "@/store";
import { minmax } from "@/utils";
import { useAIContext } from "../../logic";
import { useConversationStore } from "../../store";
import type { Message } from "../../types";

type LocalState = {
  code: string;
  editing: boolean;
};

const props = defineProps<{
  message: Message;
}>();

const EDITOR_HEIGHT = {
  min: 48,
  max: 120,
};

const state = reactive<LocalState>({
  code: props.message.content,
  editing: false,
});
const { t } = useI18n();
const { events, showHistoryDialog } = useAIContext();
const editorRef = ref<InstanceType<typeof MonacoEditor>>();

const updateEditorSize = () => {
  const contentHeight =
    editorRef.value?.editorInstance?.getContentHeight() as number;
  const paddings = 2;
  editorRef.value?.setEditorContentHeight(
    minmax(contentHeight + paddings, EDITOR_HEIGHT.min, EDITOR_HEIGHT.max)
  );
};

const handleMonacoEditorReady = () => {
  updateEditorSize();
};

const handleExecute = () => {
  events.emit("apply-statement", {
    statement: props.message.content,
    run: true,
  });
  showHistoryDialog.value = false;
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

const finishEditing = async (update: boolean) => {
  state.editing = false;
  if (!update) {
    state.code = props.message.content;
  } else {
    // eslint-disable-next-line vue/no-mutating-props
    props.message.content = state.code;
    await useConversationStore().updateMessage(props.message);
  }
};

watch(
  () => props.message.content,
  (content) => {
    state.code = content;
    state.editing = false;
  }
);
</script>
