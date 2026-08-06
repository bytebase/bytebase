import { useLayoutEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { DatabaseTargetDisplay } from "@/components/DatabaseTargetDisplay";
import { Button } from "@/components/ui/button";
import { CopyButton } from "@/components/ui/copy-button";
import { EllipsisText } from "@/components/ui/ellipsis-text";
import { cn } from "@/lib/utils";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

type ResultStatusBarProps = Readonly<{
  database: Database;
  statement: string;
  queryTime: string;
  showVisualizeButton?: boolean;
  onVisualizeExplain?: () => void;
}>;

export function ResultStatusBar({
  database,
  statement,
  queryTime,
  showVisualizeButton = false,
  onVisualizeExplain,
}: ResultStatusBarProps) {
  const { t } = useTranslation();
  const hasStatement = statement.trim() !== "";
  const statusLeftRef = useRef<HTMLDivElement>(null);
  const databaseRef = useRef<HTMLDivElement>(null);
  const statementRef = useRef<HTMLDivElement>(null);
  const databaseWidthRef = useRef(0);
  const [hideDatabase, setHideDatabase] = useState(false);

  useLayoutEffect(() => {
    const update = () => {
      const statusLeft = statusLeftRef.current;
      const databaseLabel = databaseRef.current;
      const statementLabel = statementRef.current;
      if (!statusLeft || !databaseLabel || !statementLabel) return;

      const databaseWidth =
        databaseLabel.getBoundingClientRect().width ||
        databaseLabel.clientWidth;
      if (databaseWidth > 0) {
        databaseWidthRef.current = databaseWidth;
      }

      const statementText = statementLabel.querySelector("span");
      const statementWidth = Math.max(
        statementLabel.scrollWidth,
        statementText?.scrollWidth ?? 0
      );
      setHideDatabase(
        databaseWidthRef.current > 0 &&
          statementWidth + databaseWidthRef.current > statusLeft.clientWidth
      );
    };

    update();
    const observer = new ResizeObserver(update);
    if (statusLeftRef.current) observer.observe(statusLeftRef.current);
    if (databaseRef.current) observer.observe(databaseRef.current);
    if (statementRef.current) observer.observe(statementRef.current);
    return () => observer.disconnect();
  }, [statement]);

  return (
    <div className="w-full min-w-0 flex items-center justify-between text-xs mt-1 gap-x-4 text-control-light">
      <div
        ref={statusLeftRef}
        className="flex min-w-0 flex-1 items-center gap-x-2 overflow-hidden"
        data-testid="result-status-left"
      >
        <div
          ref={databaseRef}
          className={cn(
            "min-w-0 max-w-[45%] shrink overflow-hidden whitespace-nowrap",
            hideDatabase && "hidden"
          )}
          data-testid="result-status-database"
        >
          <DatabaseTargetDisplay
            database={database}
            showEnvironment
            className="max-w-full"
          />
        </div>
        <div
          ref={statementRef}
          className="flex min-w-0 flex-1 items-center gap-x-1"
          data-testid="result-status-statement"
        >
          <EllipsisText
            text={statement}
            className="min-w-0 max-w-full truncate"
          />
          {hasStatement && (
            <CopyButton
              content={statement}
              size="xs"
              appearance="secondary"
              className="h-auto shrink-0 px-0 text-control-light hover:bg-transparent hover:text-control"
            />
          )}
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-x-2">
        {showVisualizeButton && (
          <Button
            size="sm"
            appearance="link"
            className="h-auto px-0 text-xs"
            onClick={onVisualizeExplain}
          >
            {t("sql-editor.visualize-explain")}
          </Button>
        )}
        <span>
          {t("sql-editor.query-time")}: {queryTime}
        </span>
      </div>
    </div>
  );
}
