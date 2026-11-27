import type * as monaco from "monaco-editor";
import { useCallback, useEffect, useMemo, useState } from "react";
import { ReadonlyMonaco } from "@/components/monaco/ReadonlyMonaco";
import { formatSQL } from "@/components/monaco/sqlFormatter";
import type { MonacoModule } from "@/components/monaco/types";
import type { SqlEditorAgentAction as ChatAction } from "@/modules/sql-editor/model/agentHandoff";
import { dialectOfEngineV1 } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { getInstanceResource } from "@/utils";
import { useAIActions } from "./useAIActions";

const AI_ACTIONS: readonly ChatAction[] = ["explain-code"];

interface DefinitionViewerProps {
  db: Database;
  code: string;
  format?: boolean;
}

export function DefinitionViewer({ db, code, format }: DefinitionViewerProps) {
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
    <ReadonlyMonaco
      autoHeight={false}
      content={content}
      className="w-full h-full relative"
      options={{ automaticLayout: true }}
      onReady={handleReady}
    />
  );
}
