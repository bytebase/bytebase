import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useAppDatabaseMetadata } from "@/react/hooks/useAppDatabaseMetadata";
import { useReactiveRoute } from "@/react/hooks/useReactiveRoute";
import { useConnectionOfCurrentSQLEditorTab } from "@/react/hooks/useSQLEditorBridge";
import {
  getSQLEditorTabsState,
  useSQLEditorTabState,
} from "@/react/stores/sqlEditor/tab";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { ConnectChooser } from "./ConnectChooser";

const OptionValueUnspecified = "-1";

/**
 * Replaces frontend/src/views/sql-editor/EditorCommon/ContainerChooser.vue.
 * Visible only for CosmosDB databases (container = table in CosmosDB).
 * Selected container persists to the current tab's connection.table.
 */
export function ContainerChooser() {
  const { t } = useTranslation();
  const { instance, database } = useConnectionOfCurrentSQLEditorTab();

  const engine = instance.engine;
  const databaseName = database.name;
  const tabTable = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.connection.table
  );
  // Parent SchemaPane (E4 migration) drives the metadata fetch; here we
  // only need the cached read.
  const { schemas } = useAppDatabaseMetadata(databaseName, {
    autoFetch: false,
  });

  const show = engine === Engine.COSMOSDB;

  const options = useMemo(() => {
    const opts = [
      {
        value: OptionValueUnspecified,
        label: t("database.schema.unspecified"),
      },
    ];
    for (const schema of schemas) {
      for (const table of schema.tables) {
        opts.push({ value: table.name, label: table.name });
      }
    }
    return opts;
  }, [schemas, t]);

  const value = tabTable === undefined ? OptionValueUnspecified : tabTable;
  const isChosen = value !== OptionValueUnspecified;

  const handleChange = (next: string) => {
    const tabsState = getSQLEditorTabsState();
    const currentTab = tabsState.tabsById.get(tabsState.currentTabId);
    if (!currentTab) return;
    tabsState.updateCurrentTab({
      connection: {
        ...currentTab.connection,
        table: next === OptionValueUnspecified ? undefined : next,
      },
    });
  };

  // Seed from URL query parameter on mount and whenever the query param OR
  // the active tab changes. Mirrors Vue's watchEffect, which implicitly
  // tracked both `route.query.table` and `tab.value` (the latter via the
  // setter's reactive reads) so that switching to a new tab with the URL
  // query still present re-seeded the new tab's connection.table.
  const queryParam = useReactiveRoute().query.table as string | undefined;
  const currentTabId = useSQLEditorTabState((s) => s.currentTabId);
  useEffect(() => {
    if (queryParam) handleChange(queryParam);
  }, [queryParam, currentTabId]);

  if (!show) return null;

  return (
    <ConnectChooser
      value={value}
      onChange={handleChange}
      options={options}
      isChosen={isChosen}
      placeholder={t("database.table.select")}
    />
  );
}
