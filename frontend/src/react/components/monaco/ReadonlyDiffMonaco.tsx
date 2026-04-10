import type { Language } from "@/types";
import { DiffMonaco } from "./DiffMonaco";

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
  return (
    <DiffMonaco
      className={className}
      language={language}
      max={max}
      min={min}
      modified={modified}
      original={original}
      readOnly
    />
  );
}

export default ReadonlyDiffMonaco;
