import type monaco from "monaco-editor";
import type { MaybeRef } from "vue";
import { shallowRef, unref, watchEffect } from "vue";
import type { MonacoModule } from "@/components/MonacoEditor";
import { useTextModelLanguage } from "@/components/MonacoEditor/composables/common";
import type { AIContext, ChatAction } from "../types";

type AIActionsOptions = {
  actions: ChatAction[];
  callback: (
    action: ChatAction,
    monaco: MonacoModule,
    editor: monaco.editor.IStandaloneCodeEditor
  ) => void;
};

export const useAIActions = async (
  monaco: MonacoModule,
  editor: monaco.editor.IStandaloneCodeEditor,
  context: AIContext,
  options: MaybeRef<AIActionsOptions>
) => {
  const language = useTextModelLanguage(editor);
  const actions = shallowRef<monaco.IDisposable[]>([]);

  watchEffect(() => {
    const opts = unref(options);
    actions.value.forEach((action) => {
      action.dispose();
    });
    actions.value = [];

    if (!context.openAIKey.value) {
      return;
    }

    if (language.value === "sql") {
      if (opts.actions.includes("explain-code")) {
        const action = editor.addAction({
          id: "explain-code",
          label: "Explain code",
          contextMenuGroupId: "2_ai_assistant",
          contextMenuOrder: 1,
          run: async () => {
            opts.callback("explain-code", monaco, editor);
          },
        });
        actions.value.push(action);
      }
      if (opts.actions.includes("find-problems")) {
        const action = editor.addAction({
          id: "find-problems",
          label: "Find problems",
          contextMenuGroupId: "2_ai_assistant",
          contextMenuOrder: 2,
          run: async () => {
            opts.callback("find-problems", monaco, editor);
          },
        });
        actions.value.push(action);
      }
    } else {
      // When the language is "javascript" we do not have AI Assistant actions
    }
  });
};
