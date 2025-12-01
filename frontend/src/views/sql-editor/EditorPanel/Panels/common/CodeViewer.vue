<template>
  <div
    class="w-full h-11 px-2 py-2 border-b flex flex-row justify-between items-center"
    :class="headerClass"
  >
    <div class="flex justify-start items-center gap-2">
      <slot name="title-prefix">
        <NButton text @click="$emit('back')">
          <ChevronLeftIcon class="w-5 h-5" />
          <div class="flex items-center gap-1">
            <slot name="title-icon" />
            <span>{{ title }}</span>
          </div>
        </NButton>
      </slot>
    </div>
    <div class="flex justify-end items-center gap-2">
      <NCheckbox v-model:checked="format">
        {{ $t("sql-editor.format") }}
      </NCheckbox>
      <OpenAIButton
        size="small"
        :statement="selectedStatement || content"
        :actions="['explain-code']"
      />
    </div>
  </div>

  <NSplit
    :disabled="!showAIPanel"
    :size="editorPanelSize.size"
    :min="editorPanelSize.min"
    :max="editorPanelSize.max"
    :resize-trigger-size="1"
    @update:size="handleEditorPanelResize"
  >
    <template #1>
      <div class="flex flex-col h-full overflow-hidden">
        <slot name="content-prefix" />
        <MonacoEditor
          :content="content"
          :readonly="true"
          :format-content-options="{
            disabled: format,
            callback: handleFormatContent,
          }"
          class="flex-1 w-full h-full relative"
          @select-content="selectedStatement = $event"
          @ready="handleEditorReady"
        />
      </div>
    </template>
    <template #2>
      <div
        class="h-full overflow-hidden flex flex-col"
      >
        <Suspense>
          <AIChatToSQL key="ai-chat-to-sql" />
          <template #fallback>
            <div
              class="w-full h-full grow flex flex-col items-center justify-center"
            >
              <BBSpin />
            </div>
          </template>
        </Suspense>
      </div>
    </template>
  </NSplit>
</template>

<script setup lang="ts">
import { computedAsync, useLocalStorage } from "@vueuse/core";
import { ChevronLeftIcon } from "lucide-vue-next";
import { NButton, NCheckbox, NSplit } from "naive-ui";
import { computed, ref } from "vue";
import { BBSpin } from "@/bbkit";
import {
  type IStandaloneCodeEditor,
  MonacoEditor,
  type MonacoModule,
} from "@/components/MonacoEditor";
import formatSQL from "@/components/MonacoEditor/sqlFormatter";
import { AIChatToSQL, useAIActions } from "@/plugins/ai";
import { useAIContext } from "@/plugins/ai/logic";
import * as promptUtils from "@/plugins/ai/logic/prompt";
import type { ComposedDatabase } from "@/types";
import { dialectOfEngineV1 } from "@/types";
import { nextAnimationFrame, type VueClass } from "@/utils";
import { useSQLEditorContext } from "@/views/sql-editor/context";
import { OpenAIButton } from "@/views/sql-editor/EditorCommon";

const props = defineProps<{
  db: ComposedDatabase;
  title: string;
  code: string;
  headerClass?: VueClass;
}>();

defineEmits<{
  (event: "back"): void;
}>();

const { showAIPanel, editorPanelSize, handleEditorPanelResize } =
  useSQLEditorContext();
const AIContext = useAIContext();
const format = useLocalStorage<boolean>(
  "bb.sql-editor.editor-panel.code-viewer.format",
  false
);
const instanceEngine = computed(() => props.db.instanceResource.engine);
const selectedStatement = ref("");

const formatted = computedAsync(
  async () => {
    const sql = props.code;
    try {
      const result = await formatSQL(
        sql,
        dialectOfEngineV1(instanceEngine.value)
      );
      return result;
    } catch (err) {
      return {
        error: err,
        data: sql,
      };
    }
  },
  {
    error: null,
    data: props.code,
  }
);

const content = computed(() => {
  return format.value && !formatted.value.error
    ? formatted.value.data
    : props.code;
});

const handleFormatContent = () => {
  format.value = true;
};

const handleEditorReady = (
  monaco: MonacoModule,
  editor: IStandaloneCodeEditor
) => {
  useAIActions(monaco, editor, AIContext, {
    actions: ["explain-code"],
    callback: async (action) => {
      // start new chat if AI panel is not open
      // continue current chat otherwise
      const newChat = !showAIPanel.value;

      showAIPanel.value = true;
      if (action !== "explain-code") return;

      const statement = selectedStatement.value || content.value;

      await nextAnimationFrame();
      AIContext.events.emit("send-chat", {
        content: promptUtils.explainCode(
          statement,
          props.db.instanceResource.engine
        ),
        newChat,
      });
    },
  });
};
</script>
