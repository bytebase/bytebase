import type { MonacoModule } from "@/components/MonacoEditor";
import {
  useSelectedContent,
  useSelection,
} from "@/components/MonacoEditor/composables";
import { useTextModelLanguage } from "@/components/MonacoEditor/composables/common";
import { useEditorContextKey } from "@/components/MonacoEditor/utils";
import type monaco from "monaco-editor";
import type { MaybeRef } from "vue";
import { computed, shallowRef, unref, watchEffect } from "vue";
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

  const checkContentEmpty = () => {
    const content = editor.getModel()?.getValue() ?? "";
    return content.length === 0;
  };
  const contentEmpty = useEditorContextKey(
    editor,
    "bb.ai.contentEmpty",
    checkContentEmpty()
  );
  editor.onDidChangeModelContent(() => {
    contentEmpty.set(checkContentEmpty());
  });

  const selection = useSelection(editor);
  const selectedContent = useSelectedContent(editor, selection);
  useEditorContextKey(
    editor,
    "bb.ai.selectedContentEmpty",
    computed(() => selectedContent.value.length === 0)
  );

  watchEffect(() => {
    const opts = unref(options);
    actions.value.forEach((action) => {
      action.dispose();
    });
    actions.value = [];

    if (!context.aiSetting.value.enabled) {
      return;
    }

    if (language.value === "sql") {
      if (opts.actions.includes("explain-code")) {
        const action = editor.addAction({
          id: "explain-code",
          label: "Explain code",
          precondition: "!bb.ai.contentEmpty",
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
          precondition: "!bb.ai.contentEmpty",
          contextMenuGroupId: "2_ai_assistant",
          contextMenuOrder: 2,
          run: async () => {
            opts.callback("find-problems", monaco, editor);
          },
        });
        actions.value.push(action);
      }
      if (opts.actions.includes("new-chat")) {
        const action = editor.addAction({
          id: "new-chat-using-selection",
          label: "New chat using selection",
          precondition: "!bb.ai.selectedContentEmpty",
          contextMenuGroupId: "2_ai_assistant",
          contextMenuOrder: 2,
          run: async () => {
            opts.callback("new-chat", monaco, editor);
          },
        });
        actions.value.push(action);
      }
    } else {
      // When the language is "javascript" we do not have AI Assistant actions
    }
  });
};
