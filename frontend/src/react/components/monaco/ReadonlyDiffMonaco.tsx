import { useEffect, useMemo, useRef, useState } from "react";
import {
  createMonacoDiffEditor,
  setMonacoModelLanguage,
} from "@/components/MonacoEditor/editor";
import type { IStandaloneDiffEditor } from "@/components/MonacoEditor/types";
import { cn } from "@/react/lib/utils";
import type { Language } from "@/types";

export interface ReadonlyDiffMonacoProps {
  original: string;
  modified: string;
  language?: Language;
  className?: string;
  min?: number;
  max?: number;
}

export function ReadonlyDiffMonaco({
  original,
  modified,
  language = "sql",
  className = "",
  min = 120,
  max = 600,
}: ReadonlyDiffMonacoProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const editorRef = useRef<IStandaloneDiffEditor | null>(null);
  const originalRef = useRef(original);
  const modifiedRef = useRef(modified);
  const languageRef = useRef(language);
  const [contentHeight, setContentHeight] = useState(min);
  const clampMeasuredHeight = (height: number): number => {
    return Math.min(max, Math.max(min, height));
  };

  originalRef.current = original;
  modifiedRef.current = modified;
  languageRef.current = language;

  const normalizedOriginal = useMemo(
    () => original.replace(/\r\n?/g, "\n"),
    [original]
  );
  const normalizedModified = useMemo(
    () => modified.replace(/\r\n?/g, "\n"),
    [modified]
  );

  useEffect(() => {
    let disposed = false;
    let contentSizeSubscription: { dispose(): void } | null = null;

    (async () => {
      if (!containerRef.current) return;

      const editor = await createMonacoDiffEditor({
        container: containerRef.current,
        options: {
          readOnly: true,
          domReadOnly: true,
          renderSideBySide: true,
          ignoreTrimWhitespace: true,
        },
      });

      if (disposed) {
        editor.dispose();
        return;
      }

      editorRef.current = editor;

      // Handle height auto-grow by listening to the modified editor's size change.
      const modifiedEditor = editor.getModifiedEditor();
      contentSizeSubscription = modifiedEditor.onDidContentSizeChange(
        (event) => {
          if (!event.contentHeightChanged) return;
          setContentHeight(event.contentHeight);
        }
      );

      const { editor: monacoEditor } = await import("monaco-editor");
      const originalModel = monacoEditor.createModel(
        normalizedOriginal,
        languageRef.current
      );
      const modifiedModel = monacoEditor.createModel(
        normalizedModified,
        languageRef.current
      );

      editor.setModel({
        original: originalModel,
        modified: modifiedModel,
      });

      setContentHeight(modifiedEditor.getContentHeight());

      if (disposed) {
        contentSizeSubscription?.dispose();
        editor.dispose();
        return;
      }
    })();

    return () => {
      disposed = true;
      contentSizeSubscription?.dispose();
      const model = editorRef.current?.getModel();
      model?.original?.dispose();
      model?.modified?.dispose();
      editorRef.current?.dispose();
      editorRef.current = null;
    };
  }, []);

  useEffect(() => {
    const editor = editorRef.current;
    if (!editor) return;
    const model = editor.getModel();
    if (model) {
      if (model.original.getValue() !== normalizedOriginal) {
        model.original.setValue(normalizedOriginal);
      }
      if (model.modified.getValue() !== normalizedModified) {
        model.modified.setValue(normalizedModified);
      }
    }
    const modifiedEditor = editor.getModifiedEditor();
    setContentHeight(modifiedEditor.getContentHeight());
  }, [normalizedOriginal, normalizedModified]);

  useEffect(() => {
    const editor = editorRef.current;
    const model = editor?.getModel();
    if (!model) return;

    (async () => {
      await setMonacoModelLanguage(model.original, languageRef.current);
      await setMonacoModelLanguage(model.modified, languageRef.current);
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

export default ReadonlyDiffMonaco;
