import type { Language } from "@/types";
import { MonacoEditor } from "./MonacoEditor";
import type {
  AdviceOption,
  IStandaloneEditorConstructionOptions,
  LineHighlightOption,
} from "./types";

export interface ReadonlyMonacoProps {
  advices?: AdviceOption[];
  content: string;
  className?: string;
  filename?: string;
  language?: Language;
  lineHighlights?: LineHighlightOption[];
  max?: number;
  min?: number;
  options?: IStandaloneEditorConstructionOptions;
}

export function ReadonlyMonaco({
  advices = [],
  content,
  className = "",
  filename,
  language = "sql",
  lineHighlights = [],
  max = 600,
  min = 120,
  options,
}: ReadonlyMonacoProps) {
  return (
    <MonacoEditor
      advices={advices}
      className={className}
      content={content}
      filename={filename}
      language={language}
      lineHighlights={lineHighlights}
      max={max}
      min={min}
      options={options}
      readOnly
    />
  );
}

export default ReadonlyMonaco;
