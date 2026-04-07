import { useEffect, useRef } from "react";
import { createMonacoEditor } from "@/components/MonacoEditor/editor";
import type { IStandaloneCodeEditor } from "@/components/MonacoEditor/types";
import { cn } from "@/react/lib/utils";
import type { Language } from "@/types";
import { clampEditorHeight } from "./height";

export interface ReadonlyMonacoProps {
  content: string;
  language?: Language;
  className?: string;
  min?: number;
  max?: number;
}

const MONACO_LINE_HEIGHT = 24;

export function ReadonlyMonaco({
  content,
  language = "sql",
  className = "",
  min = 120,
  max = 600,
}: ReadonlyMonacoProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const editorRef = useRef<IStandaloneCodeEditor | null>(null);
  const contentRef = useRef(content);
  const languageRef = useRef(language);

  contentRef.current = content;
  languageRef.current = language;

  useEffect(() => {
    let disposed = false;

    (async () => {
      if (!containerRef.current) return;

      const editor = await createMonacoEditor({
        container: containerRef.current,
        options: {
          language: languageRef.current,
          value: contentRef.current,
          readOnly: true,
          domReadOnly: true,
        },
      });

      if (disposed) {
        editor.dispose();
        return;
      }

      editorRef.current = editor;
    })();

    return () => {
      disposed = true;
      editorRef.current?.dispose();
      editorRef.current = null;
    };
  }, []);

  useEffect(() => {
    const editor = editorRef.current;
    if (!editor) return;
    if (editor.getValue() !== contentRef.current) {
      editor.setValue(contentRef.current);
    }
  }, [content]);

  useEffect(() => {
    const editor = editorRef.current;
    const model = editor?.getModel();
    if (!model) return;

    (async () => {
      const { editor: monacoEditor } = await import("monaco-editor");
      monacoEditor.setModelLanguage(model, languageRef.current);
    })();
  }, [language]);

  const lineCount = Math.max(1, content.split(/\r?\n/).length);
  const height = clampEditorHeight({
    lineCount,
    lineHeight: MONACO_LINE_HEIGHT,
    min,
    max,
  });

  return (
    <div
      ref={containerRef}
      className={cn("overflow-clip rounded-md border text-sm", className)}
      style={{ height }}
    />
  );
}

export default ReadonlyMonaco;
