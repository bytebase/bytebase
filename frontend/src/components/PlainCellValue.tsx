import type { ReactNode } from "react";

interface PlainCellValueProps {
  value: string | null | undefined;
  children?: ReactNode;
}

export const getCellDisplayText = (value: string | null | undefined) => {
  if (value === undefined) return "UNSET";
  if (value === null) return "NULL";
  return value;
};

export function PlainCellValue({ value, children }: PlainCellValueProps) {
  if (value === undefined || value === null) {
    return (
      <span className="text-control-placeholder italic">
        {children ?? getCellDisplayText(value)}
      </span>
    );
  }
  if (value.length === 0) {
    return <br style={{ minWidth: "1rem", display: "inline-flex" }} />;
  }
  return <>{children ?? value}</>;
}
