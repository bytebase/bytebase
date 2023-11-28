import type monaco from "monaco-editor";
import { Ref, watchEffect } from "vue";
import type { SQLDialect } from "@/types";
import type { MonacoModule } from "../types";
import { formatEditorContent } from "../utils";
import { useTextModelLanguage } from "./common";

export const useFormatContent = async (
  monaco: MonacoModule,
  editor: monaco.editor.IStandaloneCodeEditor,
  dialect: Ref<SQLDialect | undefined>
) => {
  const language = useTextModelLanguage(editor);

  watchEffect((onCleanup) => {
    if (language.value === "sql") {
      // add `Format SQL` action into context menu
      const action = editor.addAction({
        id: "format-sql",
        label: "Format SQL",
        keybindings: [
          monaco.KeyMod.Alt | monaco.KeyMod.Shift | monaco.KeyCode.KeyF,
        ],
        contextMenuGroupId: "operation",
        contextMenuOrder: 1,
        run: async () => {
          const readonly = editor.getOption(
            monaco.editor.EditorOption.readOnly
          );
          if (readonly) return;
          await formatEditorContent(editor, dialect.value);
        },
      });
      onCleanup(() => {
        action.dispose();
      });
    } else {
      // When the language is "javascript", we can still use Alt+Shift+F to
      // format the document (the native feature of monaco-editor).
    }
  });
};
