<template>
  <div
    ref="containerRef"
    class="flex flex-col overflow-x-hidden border border-gray-500 rounded-sm bg-white"
    :data-message-wrapper-width="messageWrapperWidth"
    :style="`width: ${combinedWidth}px`"
  >
    <div class="flex items-center justify-between px-1 pt-1">
      <div class="text-xs">SQL</div>
      <div class="flex items-center justify-end gap-2">
        <NPopover placement="bottom">
          <template #trigger>
            <button
              class="inline-flex items-center justify-center hover:text-accent cursor-pointer"
              @click="handleExecute"
            >
              <PlayIcon class="w-3.5 h-3.5" />
            </button>
          </template>
          <div class="whitespace-nowrap">
            {{ $t("common.run") }}
          </div>
        </NPopover>
        <NPopover placement="bottom">
          <template #trigger>
            <button
              class="inline-flex items-center justify-center hover:text-accent cursor-pointer"
              @click.capture="handleInsertAtCaret"
            >
              <InsertAtCaretIcon :size="14" />
            </button>
          </template>
          <div class="whitespace-nowrap">
            {{ $t("plugin.ai.actions.insert-at-caret") }}
          </div>
        </NPopover>
        <NPopover placement="bottom">
          <template #trigger>
            <CopyButton :content="code" />
          </template>
          <div class="whitespace-nowrap">
            {{ $t("common.copy") }}
          </div>
        </NPopover>
      </div>
    </div>

    <MonacoEditor
      :content="code"
      :readonly="true"
      :auto-focus="false"
      :auto-height="{
        min: 20,
        max: 120,
        padding: 2,
      }"
      :options="{
        fontSize: 12,
        lineHeight: 14,
        lineNumbers: 'off',
        wordWrap: 'on',
        scrollbar: {
          vertical: 'hidden',
          horizontal: 'hidden',
          useShadows: false,
          verticalScrollbarSize: 0,
          horizontalScrollbarSize: 0,
          alwaysConsumeMouseWheel: false,
        },
      }"
      class="h-auto"
    />
  </div>
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { PlayIcon } from "lucide-vue-next";
import { NPopover } from "naive-ui";
import { computed, ref } from "vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import { CopyButton } from "@/components/v2";
import { useAIContext } from "@/plugins/ai/logic";
import { findAncestor } from "@/utils";
import { useSQLEditorContext } from "@/views/sql-editor/context";
import InsertAtCaretIcon from "./InsertAtCaretIcon.vue";

export type CodeBlockProps = {
  width: number;
};

const props = defineProps<
  CodeBlockProps & {
    code: string;
  }
>();

const { events, showHistoryDialog } = useAIContext();
const { events: editorEvents } = useSQLEditorContext();
const containerRef = ref<HTMLElement>();
const messageWrapperRef = computed(() =>
  findAncestor(containerRef.value, ".message")
);
const { width: messageWrapperWidth } = useElementSize(messageWrapperRef);
const combinedWidth = computed(() => {
  const PADDING = 8;
  const min = 8 * 16; /* 8rem */
  const auto = messageWrapperWidth.value * props.width - PADDING * 2;
  return Math.max(min, auto);
});

const handleExecute = () => {
  events.emit("run-statement", {
    statement: props.code,
  });
  showHistoryDialog.value = false;
};

const handleInsertAtCaret = () => {
  const { code } = props;
  editorEvents.emit("insert-at-caret", {
    content: code,
  });
  showHistoryDialog.value = false;
};
</script>
