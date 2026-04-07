import { useEffect, useRef } from "react";
import { createMonacoEditor } from "@/components/MonacoEditor/editor";
import type { IStandaloneCodeEditor } from "@/components/MonacoEditor/types";
import { cn } from "@/react/lib/utils";
import type { Language } from "@/types";
import { getReadonlyMonacoHeight } from "./height";

export interface ReadonlyMonacoProps {
  content: string;
  language?: Language;
  className?: string;
  minHeight?: number;
  maxHeight?: number;
}

const MONACO_LINE_HEIGHT = 24;
const MONACO_PADDING = 16;

export function ReadonlyMonaco({
  content,
  language = "sql",
  className,
  minHeight = 160,
  maxHeight = 720,
}: ReadonlyMonacoProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const editorRef = useRef<IStandaloneCodeEditor | null>(null);

  useEffect(() => {
    let disposed = false;

    (async () => {
      if (!containerRef.current) return;

      const editor = await createMonacoEditor({
        container: containerRef.current,
        options: {
          language,
          value: content,
          readOnly: true,
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
    if (editor.getValue() !== content) {
      editor.setValue(content);
    }
  }, [content]);

  useEffect(() => {
    const editor = editorRef.current;
    const model = editor?.getModel();
    if (!model) return;

    (async () => {
      const { editor: monacoEditor } = await import("monaco-editor");
      monacoEditor.setModelLanguage(model, language);
    })();
  }, [language]);

  const height = getReadonlyMonacoHeight(content, {
    minHeight,
    maxHeight,
    lineHeight: MONACO_LINE_HEIGHT,
    padding: MONACO_PADDING,
  });

  return (
    <div
      ref={containerRef}
      className={cn("w-full overflow-hidden", className)}
      style={{ height }}
    />
  );
}

export default ReadonlyMonaco;
