import { Loader2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { ResultView } from "@/react/components/sql-editor/ResultView";
import { Button } from "@/react/components/ui/button";
import { useSQLEditorTabStore } from "@/store";
import type {
  SQLEditorDatabaseQueryContext,
  SQLEditorQueryParams,
} from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

export interface DatabaseQueryContextProps {
  database: Database;
  context: SQLEditorDatabaseQueryContext;
}

/**
 * React port of `DatabaseQueryContext.vue`. Three render paths:
 * - EXECUTING — spinner + elapsed-time + cancel button
 * - CANCELLED — re-execute button
 * - default  — `<ResultView>` with the executed result set
 *
 * Auto-runs the query when status flips to `PENDING`. Mirrors the Vue
 * watcher behavior — same `useExecuteSQL().runQuery` entrypoint that
 * `EditorMain.tsx` already uses from React.
 */
export function DatabaseQueryContext({
  database,
  context,
}: DatabaseQueryContextProps) {
  const { t } = useTranslation();
  const tabStore = useSQLEditorTabStore();
  const { runQuery } = useExecuteSQL();
  const isExecuting = context.status === "EXECUTING";

  // Trigger run on status === PENDING. Pinia's reactivity already
  // surfaces context.status changes through props; the effect dep array
  // re-fires whenever the status flips back to PENDING (which the
  // re-execute button below does).
  const lastRanContextRef = useRef<string | null>(null);
  useEffect(() => {
    if (context.status !== "PENDING") return;
    const key = `${database.name}:${context.id}`;
    if (lastRanContextRef.current === key) return;
    lastRanContextRef.current = key;
    void runQuery(database, context);
  }, [database, context, runQuery, context.status]);

  const elapsed = useElapsed(isExecuting, context.beginTimestampMS);

  const cancelQuery = () => {
    context.abortController?.abort();
    tabStore.updateDatabaseQueryContext({
      database: database.name,
      contextId: context.id,
      context: { status: "CANCELLED" },
    });
  };

  const execQuery = (params: SQLEditorQueryParams) => {
    const next = tabStore.updateDatabaseQueryContext({
      database: database.name,
      contextId: context.id,
      context: { params },
    });
    if (!next) return;
    // Reset so the auto-run effect re-fires for this context.
    lastRanContextRef.current = null;
    void runQuery(database, next);
  };

  if (isExecuting) {
    return (
      <div className="w-full h-full flex flex-col justify-center items-center text-sm gap-y-1 bg-white/80 dark:bg-black/80">
        <div className="flex items-center gap-x-1">
          <Loader2 className="size-5 animate-spin mr-1" />
          <span>{t("sql-editor.executing-query")}</span>
          <span>-</span>
          <span className="font-mono">{elapsed}</span>
        </div>
        <div>
          <Button size="sm" variant="outline" onClick={cancelQuery}>
            {t("common.cancel")}
          </Button>
        </div>
      </div>
    );
  }

  if (context.status === "CANCELLED") {
    return (
      <div className="w-full h-full flex flex-col justify-center items-center text-sm gap-y-1 bg-white/80 dark:bg-black/80">
        <Button
          size="sm"
          variant="outline"
          onClick={() => execQuery(context.params)}
        >
          {t("sql-editor.execute-query")}
        </Button>
      </div>
    );
  }

  return (
    <ResultView
      executeParams={context.params}
      database={database}
      resultSet={context.resultSet}
    />
  );
}

/**
 * Returns a humanized elapsed-time string ("3.2s") that ticks once per
 * second while `running` is true. Replaces the Vue `useCurrentTimestamp`
 * + computed combo. Stops/clears the interval on unmount or when
 * `running` flips to false.
 */
function useElapsed(running: boolean, beginMS: number | undefined): string {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!running) return;
    setNow(Date.now());
    const handle = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(handle);
  }, [running]);
  if (!running || !beginMS) return "";
  return `${((now - beginMS) / 1000).toFixed(1)}s`;
}
