import { FilterIcon } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useExecuteSQL } from "@/hooks/useExecuteSQL";
import { DatabaseQueryContext } from "@/modules/sql-editor/components/ResultPanel/DatabaseQueryContext";
import { useConnectionOfCurrentSQLEditorTab } from "@/modules/sql-editor/hooks/useSQLEditorState";
import {
  getSQLEditorTabsState,
  useSQLEditorTabState,
} from "@/modules/sql-editor/store/tab";
import {
  getDataExplorerFilterPlaceholderKey,
  getDataExplorerQueryPrefix,
  getDataExplorerStatement,
} from "./query";

export function DataExplorerPanel() {
  const { t } = useTranslation();
  const { connection, database, instance } =
    useConnectionOfCurrentSQLEditorTab();
  const { execute } = useExecuteSQL();
  const tab = useSQLEditorTabState((state) =>
    state.tabsById.get(state.currentTabId)
  );
  const context = useSQLEditorTabState(
    (state) =>
      state.tabsById
        .get(state.currentTabId)
        ?.databaseQueryContexts?.get(database.name)?.[0]
  );
  const [filter, setFilter] = useState(tab?.dataExplorer?.filter ?? "");
  const initializedRef = useRef(false);
  const queryTarget = useMemo(
    () => ({
      engine: instance.engine,
      schema: connection.schema ?? "",
      table: connection.table ?? "",
    }),
    [connection.schema, connection.table, instance.engine]
  );
  const prefix = getDataExplorerQueryPrefix(queryTarget);
  const filterPlaceholder = t(
    getDataExplorerFilterPlaceholderKey(instance.engine)
  );
  const isExecuting = context?.status === "EXECUTING";

  const updateExplorerState = useCallback(
    (patch: { filter?: string; initialized?: boolean; resetRow?: boolean }) => {
      if (!tab?.dataExplorer) return;
      getSQLEditorTabsState().updateTab(tab.id, {
        dataExplorer: {
          ...tab.dataExplorer,
          ...(patch.filter !== undefined ? { filter: patch.filter } : {}),
          ...(patch.initialized !== undefined
            ? { initialized: patch.initialized }
            : {}),
          ...(patch.resetRow ? { selectedRowKey: undefined } : {}),
        },
      });
    },
    [tab]
  );

  const run = useCallback(
    async (suffix: string) => {
      const statement = getDataExplorerStatement(queryTarget, suffix);
      if (!tab || !statement) return;

      const tabsState = getSQLEditorTabsState();
      for (const existing of tab.databaseQueryContexts?.get(database.name) ??
        []) {
        existing.abortController?.abort();
      }
      tabsState.deleteDatabaseQueryContext(database.name);
      updateExplorerState({
        filter: suffix,
        initialized: true,
        resetRow: true,
      });
      await execute({
        connection,
        statement,
        engine: instance.engine,
        explain: false,
        selection: tab.editorState.selection,
      });
    },
    [
      connection,
      database.name,
      execute,
      instance.engine,
      queryTarget,
      tab,
      updateExplorerState,
    ]
  );

  useEffect(() => {
    if (!tab?.dataExplorer || tab.dataExplorer.initialized) return;
    if (!prefix) return;
    if (initializedRef.current) return;
    initializedRef.current = true;
    void run(tab.dataExplorer.filter);
  }, [prefix, run, tab]);

  const handleFilterChange = (value: string) => {
    setFilter(value);
    updateExplorerState({ filter: value });
  };

  if (!tab || !prefix) return null;

  return (
    <div className="h-full min-h-0 flex flex-col bg-background">
      <form
        className="shrink-0 flex items-center gap-x-2 border-b border-block-border p-2"
        onSubmit={(event) => {
          event.preventDefault();
          if (isExecuting) return;
          void run(filter);
        }}
      >
        <code className="shrink-0 whitespace-nowrap text-sm text-main">
          {prefix}
        </code>
        <Input
          size="md"
          className="min-w-0 flex-1"
          autoComplete="off"
          value={filter}
          disabled={isExecuting}
          aria-label={filterPlaceholder}
          placeholder={filterPlaceholder}
          onChange={(event) => handleFilterChange(event.target.value)}
        />
        <Button type="submit" size="md" disabled={isExecuting}>
          <FilterIcon className="size-4" />
          {t("sql-editor.apply-filter")}
        </Button>
      </form>
      <div className="flex-1 min-h-0 overflow-hidden">
        {context && (
          <DatabaseQueryContext
            database={database}
            context={context}
            presentation="DATA_EXPLORER"
          />
        )}
      </div>
    </div>
  );
}
