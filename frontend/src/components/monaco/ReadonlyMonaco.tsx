import type * as monaco from "monaco-editor";
import type { Language } from "@/types";
import { MonacoEditor } from "./MonacoEditor";
import type {
  AdviceOption,
  IStandaloneEditorConstructionOptions,
  LineHighlightOption,
  MonacoModule,
} from "./types";

export interface ReadonlyMonacoProps {
  advices?: AdviceOption[];
  autoHeight?: boolean;
  content: string;
  className?: string;
  filename?: string;
  language?: Language;
  lineHighlights?: LineHighlightOption[];
  max?: number;
  min?: number;
  options?: IStandaloneEditorConstructionOptions;
  onReady?: (
    monaco: MonacoModule,
    editor: monaco.editor.IStandaloneCodeEditor
  ) => void;
}

export function ReadonlyMonaco({
  advices = [],
  autoHeight = true,
  content,
  className = "",
  filename,
  language = "sql",
  lineHighlights = [],
  max = 600,
  min = 120,
  options,
  onReady,
}: ReadonlyMonacoProps) {
  return (
    <MonacoEditor
      advices={advices}
      autoHeight={autoHeight}
      className={className}
      content={content}
      filename={filename}
      language={language}
      lineHighlights={lineHighlights}
      max={max}
      min={min}
      options={options}
      onReady={onReady}
      readOnly
    />
  );
}

export default ReadonlyMonaco;
