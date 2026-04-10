import { Loader2 } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { cn } from "@/react/lib/utils";
import type { Language } from "@/types";
import {
  createMonacoDiffEditor,
  loadMonacoEditor,
  setMonacoModelLanguage,
} from "./core";
import type {
  IStandaloneDiffEditor,
  IStandaloneDiffEditorConstructionOptions,
  MonacoModule,
} from "./types";

export interface DiffMonacoProps {
  autoHeightAlignment?: "original" | "modified";
  className?: string;
  language?: Language;
  max?: number;
  min?: number;
  modified?: string;
  onModifiedChange?: (modified: string) => void;
  onReady?: (monaco: MonacoModule, editor: IStandaloneDiffEditor) => void;
  options?: IStandaloneDiffEditorConstructionOptions;
  original?: string;
  readOnly?: boolean;
}

export function DiffMonaco({
  autoHeightAlignment = "modified",
  className = "",
  language = "sql",
  max = 600,
  min = 120,
  modified = "",
  onModifiedChange,
  onReady,
  options,
  original = "",
  readOnly = false,
}: DiffMonacoProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const editorRef = useRef<IStandaloneDiffEditor | null>(null);
  const [contentHeight, setContentHeight] = useState(min);
  const [ready, setReady] = useState(false);

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
          readOnly,
          domReadOnly: readOnly,
          ...options,
        },
      });
      if (disposed) {
        editor.dispose();
        return;
      }
      editorRef.current = editor;
      const monaco = await loadMonacoEditor();
      const originalModel = monaco.editor.createModel(
        normalizedOriginal,
        language
      );
      const modifiedModel = monaco.editor.createModel(
        normalizedModified,
        language
      );
      editor.setModel({
        original: originalModel,
        modified: modifiedModel,
      });
      await setMonacoModelLanguage(originalModel, language);
      await setMonacoModelLanguage(modifiedModel, language);
      const alignedEditor =
        autoHeightAlignment === "original"
          ? editor.getOriginalEditor()
          : editor.getModifiedEditor();
      contentSizeSubscription = alignedEditor.onDidContentSizeChange(
        (event) => {
          if (!event.contentHeightChanged) return;
          setContentHeight(event.contentHeight);
        }
      );
      editor.onDidUpdateDiff(() => {
        onModifiedChange?.(editor.getModel()?.modified.getValue() ?? "");
      });
      setContentHeight(alignedEditor.getContentHeight());
      setReady(true);
      onReady?.(monaco, editor);
    })();

    return () => {
      disposed = true;
      setReady(false);
      contentSizeSubscription?.dispose();
      const model = editorRef.current?.getModel();
      model?.original?.dispose();
      model?.modified?.dispose();
      editorRef.current?.dispose();
      editorRef.current = null;
    };
  }, [
    autoHeightAlignment,
    language,
    normalizedModified,
    normalizedOriginal,
    onModifiedChange,
    onReady,
    options,
    readOnly,
  ]);

  useEffect(() => {
    const model = editorRef.current?.getModel();
    if (!model) return;
    if (model.original.getValue() !== normalizedOriginal) {
      model.original.setValue(normalizedOriginal);
    }
    if (model.modified.getValue() !== normalizedModified) {
      model.modified.setValue(normalizedModified);
    }
  }, [normalizedModified, normalizedOriginal]);

  useEffect(() => {
    const model = editorRef.current?.getModel();
    if (!model) return;
    void setMonacoModelLanguage(model.original, language);
    void setMonacoModelLanguage(model.modified, language);
  }, [language]);

  useEffect(() => {
    const editor = editorRef.current;
    if (!editor || !options) return;
    editor.updateOptions(options);
  }, [options]);

  const height = Math.min(max, Math.max(min, contentHeight));

  return (
    <div className={cn("relative", className)}>
      <div
        ref={containerRef}
        className="w-full overflow-clip rounded-md border text-sm"
        style={{ height }}
      />
      {!ready && (
        <div className="absolute inset-0 flex items-center justify-center rounded-md border bg-background/70">
          <Loader2 className="h-5 w-5 animate-spin text-control-light" />
        </div>
      )}
    </div>
  );
}
