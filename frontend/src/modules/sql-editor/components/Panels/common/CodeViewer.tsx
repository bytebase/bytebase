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
import { ReadonlyMonaco } from "@/components/monaco/ReadonlyMonaco";
import { formatSQL } from "@/components/monaco/sqlFormatter";
import type { MonacoModule } from "@/components/monaco/types";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { cn } from "@/lib/utils";
import type { SqlEditorAgentAction as ChatAction } from "@/modules/sql-editor/model/agentHandoff";
import { dialectOfEngineV1 } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  getInstanceResource,
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
  titlePrefix?: ReactNode;
  contentPrefix?: ReactNode;
  headerClassName?: string;
}

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

  useAIActions({
    monaco: monacoModule,
    editor,
    actions: AI_ACTIONS,
  });

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
              appearance="secondary"
              size="md"
              className="px-1"
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
          <OpenAIButton size="sm" />
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
