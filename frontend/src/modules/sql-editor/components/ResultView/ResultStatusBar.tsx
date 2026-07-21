import { ChevronRightIcon } from "lucide-react";
import { type Ref, useLayoutEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { EngineIcon } from "@/components/EngineIcon";
import { Button } from "@/components/ui/button";
import { CopyButton } from "@/components/ui/copy-button";
import { EllipsisText } from "@/components/ui/ellipsis-text";
import { cn } from "@/lib/utils";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  extractDatabaseResourceName,
  getDatabaseEnvironment,
  getInstanceResource,
} from "@/utils/v1/database";
import { instanceV1Name } from "@/utils/v1/instance";

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
        <RichDatabaseName
          ref={databaseRef}
          database={database}
          hidden={hideDatabase}
          className="max-w-[45%] shrink"
        />
        <div
          ref={statementRef}
          className="flex min-w-0 flex-1 items-center gap-x-1"
          data-testid="result-status-statement"
        >
          <EllipsisText
            text={statement}
            className="min-w-0 max-w-full truncate"
          />
          <CopyButton
            content={statement}
            size="xs"
            appearance="secondary"
            className="h-auto shrink-0 px-0 text-control-light hover:bg-transparent hover:text-control"
          />
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

/**
 * Inline simplified renderer mirroring the visible output of Vue
 * `RichDatabaseName` with default props: engine icon + instance title +
 * chevron + environment + database. Skips the popover-on-hover branch
 * (only used by `tooltip="instance"` callers; result-view doesn't set it).
 */
export function RichDatabaseName({
  ref,
  database,
  hidden = false,
  className,
}: Readonly<{
  ref?: Ref<HTMLDivElement>;
  database: Database;
  hidden?: boolean;
  className?: string;
}>) {
  const instance = getInstanceResource(database);
  const environment = getDatabaseEnvironment(database);
  const { databaseName } = extractDatabaseResourceName(database.name);
  return (
    <div
      ref={ref}
      className={cn(
        "flex min-w-0 items-center gap-x-1 overflow-hidden whitespace-nowrap",
        className,
        hidden && "hidden"
      )}
      data-testid="result-status-database"
    >
      <EngineIcon engine={instance.engine} className="size-4 shrink-0" />
      <span className="truncate">{instanceV1Name(instance)}</span>
      <ChevronRightIcon className="size-3 shrink-0" />
      <span className="flex min-w-0 items-center gap-x-1 overflow-hidden">
        <span className="truncate text-control-light">{environment.title}</span>
        <span className="truncate">{databaseName}</span>
      </span>
    </div>
  );
}
