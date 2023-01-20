export type AdviceOption = {
  severity: "ERROR" | "WARNING";
  message: string;
  source?: string;
  startLineNumber: number; // starts from 1
  startColumn: number; // starts from 1
  endLineNumber: number; // starts from 1
  endColumn: number; // starts from 1
};
