import type monaco from "monaco-editor";
import { Ref, watchEffect } from "vue";
import { SQLDialect } from "@/types";
import sqlFormatter from "../sqlFormatter";
import type { MonacoModule } from "../types";
import { trySetContentWithUndo } from "../utils";
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
        run: () => {
          const readonly = editor.getOption(
            monaco.editor.EditorOption.readOnly
          );
          if (readonly) return;
          formatEditorContent(editor, dialect.value);
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

export const formatEditorContent = (
  editor: monaco.editor.IStandaloneCodeEditor,
  dialect: SQLDialect | undefined
) => {
  const model = editor.getModel();
  if (!model) return;
  const sql = model.getValue();
  const { data, error } = sqlFormatter(sql, dialect);
  if (error) {
    return;
  }
  const pos = editor.getPosition();

  trySetContentWithUndo(editor, data, "Format content");

  if (pos) {
    // Not that smart but best efforts to keep the cursor position
    editor.setPosition(pos);
  }
};
