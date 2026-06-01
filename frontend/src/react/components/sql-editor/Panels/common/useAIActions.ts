import type * as monaco from "monaco-editor";
import { useEffect } from "react";
import type { ChatAction } from "@/plugins/ai/types";
import type { MonacoModule } from "@/react/components/monaco/types";
import { useAppStore } from "@/react/stores/app";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";

interface UseAIActionsOptions {
  monaco: MonacoModule | null;
  editor: monaco.editor.IStandaloneCodeEditor | null;
  actions: readonly ChatAction[];
  callback: (
    action: ChatAction,
    monaco: MonacoModule,
    editor: monaco.editor.IStandaloneCodeEditor
  ) => void;
}

/**
 * React port of `frontend/src/plugins/ai/components/editor-actions.ts`.
 * Registers AI context-menu actions on the underlying Monaco editor.
 * Reads `aiSetting.enabled` from the same Pinia setting store the Vue
 * version uses, gating registration the same way.
 */
export function useAIActions({
  monaco,
  editor,
  actions,
  callback,
}: UseAIActionsOptions) {
  const getOrFetchSettingByName = useAppStore((s) => s.getOrFetchSettingByName);
  // Subscribe to the AI setting so the registered Monaco actions re-register
  // or unregister live when an admin toggles AI access while the editor is
  // mounted (matches the Vue `watchEffect` behavior in `editor-actions.ts`).
  const aiEnabled = useAppStore((s) => {
    const setting = s.getSettingByName(Setting_SettingName.AI);
    return setting?.value?.value?.case === "ai"
      ? (setting.value.value.value.enabled ?? false)
      : false;
  });
  // Defensive fetch — the SQL editor route doesn't load the AI setting at
  // bootstrap, and a deep-link to a code panel can land here first.
  useEffect(() => {
    void getOrFetchSettingByName(Setting_SettingName.AI, true);
  }, [getOrFetchSettingByName]);

  useEffect(() => {
    if (!monaco || !editor || !aiEnabled) return;
    if (editor.getModel()?.getLanguageId() !== "sql") return;

    const updateContentEmpty = () => {
      const content = editor.getModel()?.getValue() ?? "";
      contentEmpty.set(content.length === 0);
    };
    const contentEmpty = editor.createContextKey<boolean>(
      "bb.ai.contentEmpty",
      (editor.getModel()?.getValue() ?? "").length === 0
    );
    const selectedContentEmpty = editor.createContextKey<boolean>(
      "bb.ai.selectedContentEmpty",
      true
    );
    const updateSelectedContentEmpty = () => {
      const selection = editor.getSelection();
      const model = editor.getModel();
      if (!selection || !model) {
        selectedContentEmpty.set(true);
        return;
      }
      const selected = model.getValueInRange(selection);
      selectedContentEmpty.set(selected.length === 0);
    };
    updateSelectedContentEmpty();

    const subscriptions: monaco.IDisposable[] = [];
    subscriptions.push(
      editor.onDidChangeModelContent(() => {
        updateContentEmpty();
        updateSelectedContentEmpty();
      })
    );
    subscriptions.push(
      editor.onDidChangeCursorSelection(() => updateSelectedContentEmpty())
    );

    if (actions.includes("explain-code")) {
      subscriptions.push(
        editor.addAction({
          id: "explain-code",
          label: "Explain code",
          precondition: "!bb.ai.contentEmpty",
          contextMenuGroupId: "2_ai_assistant",
          contextMenuOrder: 1,
          run: () => callback("explain-code", monaco, editor),
        })
      );
    }
    if (actions.includes("find-problems")) {
      subscriptions.push(
        editor.addAction({
          id: "find-problems",
          label: "Find problems",
          precondition: "!bb.ai.contentEmpty",
          contextMenuGroupId: "2_ai_assistant",
          contextMenuOrder: 2,
          run: () => callback("find-problems", monaco, editor),
        })
      );
    }
    if (actions.includes("new-chat")) {
      subscriptions.push(
        editor.addAction({
          id: "new-chat-using-selection",
          label: "New chat using selection",
          precondition: "!bb.ai.selectedContentEmpty",
          contextMenuGroupId: "2_ai_assistant",
          contextMenuOrder: 2,
          run: () => callback("new-chat", monaco, editor),
        })
      );
    }
    return () => {
      subscriptions.forEach((sub) => sub.dispose());
    };
  }, [monaco, editor, aiEnabled, actions, callback]);
}
