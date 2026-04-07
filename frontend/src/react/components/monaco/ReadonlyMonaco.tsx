import { useEffect, useRef, useState } from "react";
import { createMonacoEditor } from "@/components/MonacoEditor/editor";
import type { IStandaloneCodeEditor } from "@/components/MonacoEditor/types";
import { cn } from "@/react/lib/utils";
import type { Language } from "@/types";

export interface ReadonlyMonacoProps {
  content: string;
  language?: Language;
  className?: string;
  min?: number;
  max?: number;
}

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
  const [contentHeight, setContentHeight] = useState(min);
  const clampMeasuredHeight = (height: number): number => {
    return Math.min(max, Math.max(min, height));
  };

  contentRef.current = content;
  languageRef.current = language;

  useEffect(() => {
    let disposed = false;
    let contentSizeSubscription: { dispose(): void } | null = null;

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

      contentSizeSubscription = editor.onDidContentSizeChange((event) => {
        if (!event.contentHeightChanged) return;
        setContentHeight(event.contentHeight);
      });

      if (editor.getValue() !== contentRef.current) {
        editor.setValue(contentRef.current);
      }

      setContentHeight(editor.getContentHeight());

      const model = editor.getModel();
      if (model) {
        const { editor: monacoEditor } = await import("monaco-editor");
        if (disposed) {
          contentSizeSubscription?.dispose();
          editor.dispose();
          return;
        }
        monacoEditor.setModelLanguage(model, languageRef.current);
      }

      if (disposed) {
        contentSizeSubscription?.dispose();
        editor.dispose();
        return;
      }
    })();

    return () => {
      disposed = true;
      contentSizeSubscription?.dispose();
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
    setContentHeight(editor.getContentHeight());
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

  const height = clampMeasuredHeight(contentHeight);

  return (
    <div
      ref={containerRef}
      className={cn(
        "w-full overflow-clip rounded-md border text-sm",
        className
      )}
      style={{ height }}
    />
  );
}

export default ReadonlyMonaco;
