<template>
  <NSplit
    :disabled="!showAIPanel"
    :size="editorPanelSize.size"
    :min="editorPanelSize.min"
    :max="editorPanelSize.max"
    :resize-trigger-size="1"
    @update:size="handleEditorPanelResize"
  >
    <template #1>
      <MonacoEditor
        :content="content"
        :readonly="true"
        class="w-full h-full relative"
        @select-content="handleSelectContent"
        @ready="handleEditorReady"
      />
    </template>
    <template #2>
      <div class="h-full overflow-hidden flex flex-col">
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
import { computedAsync } from "@vueuse/core";
import { NSplit } from "naive-ui";
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
import { nextAnimationFrame } from "@/utils";
import { useSQLEditorContext } from "@/views/sql-editor/context";

const props = defineProps<{
  db: ComposedDatabase;
  code: string;
  format?: boolean;
}>();

const emit = defineEmits<{
  (event: "select-content", content: string): void;
  (event: "back"): void;
}>();

const { showAIPanel, editorPanelSize, handleEditorPanelResize } =
  useSQLEditorContext();
const instanceEngine = computed(() => props.db.instanceResource.engine);
const AIContext = useAIContext();
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
  return props.format && !formatted.value.error
    ? formatted.value.data
    : props.code;
});

const handleSelectContent = (selected: string) => {
  selectedStatement.value = selected;
  emit("select-content", selected);
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
