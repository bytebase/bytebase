import type * as monaco from "monaco-editor";
import { useCallback, useEffect, useMemo, useState } from "react";
import { aiContextEvents } from "@/plugins/ai/logic";
import * as promptUtils from "@/plugins/ai/logic/prompt";
import type { ChatAction } from "@/plugins/ai/types";
import { ReadonlyMonaco } from "@/react/components/monaco/ReadonlyMonaco";
import { formatSQL } from "@/react/components/monaco/sqlFormatter";
import type { MonacoModule } from "@/react/components/monaco/types";
import { useSQLEditorUIStore } from "@/store";
import { dialectOfEngineV1 } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { getInstanceResource, nextAnimationFrame } from "@/utils";
import { useAIActions } from "./useAIActions";

const AI_ACTIONS: readonly ChatAction[] = ["explain-code"];

interface DefinitionViewerProps {
  db: Database;
  code: string;
  format?: boolean;
  onSelectContent?: (content: string) => void;
}

/**
 * React port of `frontend/src/views/sql-editor/EditorPanel/Panels/ViewsPanel/DefinitionViewer.vue`.
 * Bare Monaco read-only viewer; `format` is owned by the parent (the
 * ViewsPanel detail header has the toggle). Sets `uiStore.isShowingCode`
 * on mount so `Panels.vue` renders the AIChatToSQL pane next to it.
 */
export function DefinitionViewer({
  db,
  code,
  format,
  onSelectContent,
}: DefinitionViewerProps) {
  const uiStore = useSQLEditorUIStore();
  const engine = useMemo(() => getInstanceResource(db).engine, [db]);

  const [formatted, setFormatted] = useState<{
    data: string;
    error: Error | null;
  }>({ data: code, error: null });

  useEffect(() => {
    let cancelled = false;
    formatSQL(code, dialectOfEngineV1(engine)).then((result) => {
      if (cancelled) return;
      setFormatted(result);
    });
    return () => {
      cancelled = true;
    };
  }, [code, engine]);

  const content = format && !formatted.error ? formatted.data : code;

  const [selectedStatement, setSelectedStatement] = useState("");
  const [monacoModule, setMonacoModule] = useState<MonacoModule | null>(null);
  const [editor, setEditor] =
    useState<monaco.editor.IStandaloneCodeEditor | null>(null);

  const handleReady = useCallback(
    (m: MonacoModule, e: monaco.editor.IStandaloneCodeEditor) => {
      setMonacoModule(m);
      setEditor(e);
    },
    []
  );

  useEffect(() => {
    uiStore.isShowingCode = true;
    return () => {
      uiStore.isShowingCode = false;
    };
  }, [uiStore]);

  const handleAIAction = useCallback(
    (action: ChatAction) => {
      const newChat = !uiStore.showAIPanel;
      uiStore.showAIPanel = true;
      if (action !== "explain-code") return;
      const statement = selectedStatement || content;
      void nextAnimationFrame().then(() => {
        void aiContextEvents.emit("send-chat", {
          content: promptUtils.explainCode(statement, engine),
          newChat,
        });
      });
    },
    [uiStore, selectedStatement, content, engine]
  );

  useAIActions({
    monaco: monacoModule,
    editor,
    actions: AI_ACTIONS,
    callback: handleAIAction,
  });

  useEffect(() => {
    if (!editor) return;
    const sub = editor.onDidChangeCursorSelection(() => {
      const selection = editor.getSelection();
      const model = editor.getModel();
      if (!selection || !model) {
        setSelectedStatement("");
        onSelectContent?.("");
        return;
      }
      const selected = model.getValueInRange(selection);
      setSelectedStatement(selected);
      onSelectContent?.(selected);
    });
    return () => sub.dispose();
  }, [editor, onSelectContent]);

  return (
    <ReadonlyMonaco
      autoHeight={false}
      content={content}
      className="w-full h-full relative"
      options={{ automaticLayout: true }}
      onReady={handleReady}
    />
  );
}
