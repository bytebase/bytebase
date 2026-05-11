import { type IRange, Selection } from "monaco-editor";
import { type ReactNode, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { positionWithOffset } from "@/components/MonacoEditor/utils";
import { activeSQLEditorRef } from "@/react/components/sql-editor/StandardPanel/state";
import { Alert } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import { cn } from "@/react/lib/utils";
import type { SQLEditorQueryParams, SQLResultSetV1 } from "@/types";
import { sqlEditorEvents } from "@/views/sql-editor/events";
import { PostgresError } from "./PostgresError";

interface ErrorViewProps {
  dark: boolean;
  error: string | undefined;
  executeParams?: SQLEditorQueryParams;
  resultSet?: SQLResultSetV1;
  suffix?: ReactNode;
}

export function ErrorView({
  dark,
  error,
  executeParams,
  resultSet,
  suffix,
}: ErrorViewProps) {
  const { t } = useTranslation();

  const failingStatement = useMemo(() => {
    const failed = resultSet?.results?.find((r) => r.error);
    return failed?.statement?.trim();
  }, [resultSet]);

  const errorPosition = useMemo(() => {
    const result = resultSet?.results?.[0];
    if (result?.detailedError.case === "syntaxError") {
      return result.detailedError.value.startPosition;
    }
    return undefined;
  }, [resultSet]);

  const hasErrorPosition = (errorPosition?.line ?? 0) > 0;
  const canShowInEditor = hasErrorPosition || !!failingStatement;

  const findStatementRange = (): IRange | undefined => {
    let statement = failingStatement;
    if (!statement) return undefined;

    const editor = activeSQLEditorRef.value;
    const model = editor?.getModel();
    if (!model) return undefined;

    statement = statement.replace(/\s+LIMIT\s+\d+\s*;?\s*$/i, "").trim();
    let matches = model.findMatches(
      statement,
      false,
      false,
      false,
      null,
      false
    );
    if (matches.length > 0) return matches[0].range;

    matches = model.findMatches(
      `${statement};`,
      false,
      false,
      false,
      null,
      false
    );
    if (matches.length > 0) return matches[0].range;
    return undefined;
  };

  const positionLabel = (() => {
    if (hasErrorPosition && errorPosition) {
      const [line, col] = positionWithOffset(
        errorPosition.line,
        errorPosition.column,
        executeParams?.selection
      );
      return `L${line}:C${col}`;
    }
    return t("sql-editor.show-in-editor");
  })();

  const showInEditor = () => {
    let range: IRange | undefined;
    if (hasErrorPosition && errorPosition) {
      const [line, col] = positionWithOffset(
        errorPosition.line,
        errorPosition.column,
        executeParams?.selection
      );
      range = new Selection(line, col, line, col);
    } else {
      range = findStatementRange();
    }
    if (range) {
      sqlEditorEvents.emit("set-editor-selection", range);
    }
  };

  return (
    <div
      className={cn(
        "w-full text-md font-normal flex flex-col gap-2 text-sm",
        dark ? "text-matrix-green-hover" : "text-control-light"
      )}
    >
      <Alert variant="error" className="w-full">
        <div className="flex items-center gap-2 w-full">
          <span className="flex-1">{error}</span>
          {canShowInEditor && (
            <Button size="sm" variant="ghost" onClick={showInEditor}>
              {positionLabel}
            </Button>
          )}
        </div>
      </Alert>
      {suffix && <div>{suffix}</div>}
      {resultSet && <PostgresError resultSet={resultSet} />}
    </div>
  );
}
