import { ChevronLeft } from "lucide-react";
import type * as monaco from "monaco-editor";
import {
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import type { MonacoModule } from "@/components/MonacoEditor";
import formatSQL from "@/components/MonacoEditor/sqlFormatter";
import { aiContextEvents } from "@/plugins/ai/logic";
import * as promptUtils from "@/plugins/ai/logic/prompt";
import type { ChatAction } from "@/plugins/ai/types";
import { ReadonlyMonaco } from "@/react/components/monaco/ReadonlyMonaco";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import { cn } from "@/react/lib/utils";
import { useSQLEditorUIStore } from "@/store";
import { dialectOfEngineV1 } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  getInstanceResource,
  nextAnimationFrame,
  STORAGE_KEY_SQL_EDITOR_CODE_VIEWER_FORMAT,
} from "@/utils";
import { OpenAIButton } from "../../OpenAIButton";
import { useAIActions } from "./useAIActions";

const AI_ACTIONS: readonly ChatAction[] = ["explain-code"];

interface CodeViewerProps {
  db: Database;
  title: string;
  code: string;
  onBack?: () => void;
  /** Replaces the default back-button + title row in the header. */
  titlePrefix?: ReactNode;
  /** Slot above the Monaco editor; mirrors Vue's `content-prefix`. */
  contentPrefix?: ReactNode;
  headerClassName?: string;
}

/**
 * React port of the Vue `CodeViewer`. Renders header (back button +
 * format toggle + OpenAIButton) and a read-only Monaco editor.
 *
 * Note on AI integration: the AIChatToSQL side pane lives in
 * `Panels.vue` and is gated by `uiStore.showAIPanel && uiStore.isShowingCode`.
 * This component flips `isShowingCode` on mount so the host knows when
 * to render the AI pane next to it. Same parity as the Vue CodeViewer
 * NSplit, just hoisted one level up.
 */
export function CodeViewer({
  db,
  title,
  code,
  onBack,
  titlePrefix,
  contentPrefix,
  headerClassName,
}: CodeViewerProps) {
  const { t } = useTranslation();
  const uiStore = useSQLEditorUIStore();

  const [format, setFormat] = useState<boolean>(() => {
    if (typeof window === "undefined") return false;
    const stored = localStorage.getItem(
      STORAGE_KEY_SQL_EDITOR_CODE_VIEWER_FORMAT
    );
    return stored === "true";
  });

  useEffect(() => {
    localStorage.setItem(
      STORAGE_KEY_SQL_EDITOR_CODE_VIEWER_FORMAT,
      String(format)
    );
  }, [format]);

  const [formatted, setFormatted] = useState<{
    data: string;
    error: Error | null;
  }>({ data: code, error: null });

  const engine = useMemo(() => getInstanceResource(db).engine, [db]);

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

  // Tell Panels.vue to host the AIChatToSQL pane while a code surface is
  // mounted. Cleared on unmount so navigating back to the list view drops
  // the pane (matches Vue CodeViewer's auto-unmount on detail-clear).
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
        return;
      }
      setSelectedStatement(model.getValueInRange(selection));
    });
    return () => sub.dispose();
  }, [editor]);

  return (
    <div className="h-full flex flex-col overflow-hidden">
      <div
        className={cn(
          "w-full h-11 px-2 py-2 border-b border-block-border flex flex-row justify-between items-center shrink-0",
          headerClassName
        )}
      >
        <div className="flex justify-start items-center gap-2 min-w-0">
          {titlePrefix ?? (
            <Button
              variant="ghost"
              className="h-8 px-1 text-sm"
              onClick={onBack}
            >
              <ChevronLeft className="size-5" />
              <span className="truncate">{title}</span>
            </Button>
          )}
        </div>
        <div className="flex justify-end items-center gap-x-3">
          <label className="flex items-center gap-x-1 text-sm text-control cursor-pointer select-none">
            <Checkbox
              checked={format}
              onCheckedChange={(checked) => setFormat(checked)}
            />
            {t("sql-editor.format")}
          </label>
          <OpenAIButton
            size="sm"
            actions={["explain-code"]}
            statement={selectedStatement || content}
          />
        </div>
      </div>
      <div className="flex-1 min-h-0 flex flex-col overflow-hidden">
        {contentPrefix}
        <ReadonlyMonaco
          autoHeight={false}
          content={content}
          className="flex-1 w-full h-full relative"
          options={{ automaticLayout: true }}
          onReady={handleReady}
        />
      </div>
    </div>
  );
}
