import { useCallback, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useAppDatabaseMetadata } from "@/hooks/useAppDatabaseMetadata";
import { useReactiveRoute } from "@/hooks/useReactiveRoute";
import { useConnectionOfCurrentSQLEditorTab } from "@/modules/sql-editor/hooks/useSQLEditorState";
import {
  getSQLEditorTabsState,
  useSQLEditorTabState,
} from "@/modules/sql-editor/store/tab";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { ConnectChooser } from "./ConnectChooser";

const OptionValueUnspecified = "-1";
const ignoredRouteTableByTab = new Map<string, string>();

type Props = {
  readonly openSignal?: number;
  readonly variant?: "connection" | "run";
};

/**
 * Replaces frontend/src/views/sql-editor/EditorCommon/ContainerChooser.vue.
 * Visible only for CosmosDB databases (container = table in CosmosDB).
 * Selected container persists to the current tab's connection.table.
 */
export function ContainerChooser({
  openSignal,
  variant = "connection",
}: Props) {
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

  const containers = useMemo(
    () => schemas.flatMap((schema) => schema.tables.map((table) => table.name)),
    [schemas]
  );

  const options = useMemo(() => {
    const opts = [
      {
        value: OptionValueUnspecified,
        label: t("database.container.unspecified"),
      },
    ];
    for (const container of containers) {
      opts.push({ value: container, label: container });
    }
    return opts;
  }, [containers, t]);

  const value = tabTable === undefined ? OptionValueUnspecified : tabTable;
  const isChosen = value !== OptionValueUnspecified;

  const queryParam = useReactiveRoute().query.table as string | undefined;
  const currentTabId = useSQLEditorTabState((s) => s.currentTabId);
  const handleChange = useCallback(
    (next: string) => {
      const tabsState = getSQLEditorTabsState();
      const currentTab = tabsState.tabsById.get(tabsState.currentTabId);
      if (!currentTab) return;
      if (next === OptionValueUnspecified && queryParam) {
        ignoredRouteTableByTab.set(currentTabId, queryParam);
      } else {
        ignoredRouteTableByTab.delete(currentTabId);
      }
      tabsState.updateCurrentTab({
        connection: {
          ...currentTab.connection,
          table: next === OptionValueUnspecified ? undefined : next,
        },
      });
    },
    [currentTabId, queryParam]
  );

  // Seed from URL query parameter on mount and whenever the query param OR
  // the active tab changes. Mirrors Vue's watchEffect, which implicitly
  // tracked both `route.query.table` and `tab.value` (the latter via the
  // setter's reactive reads) so that switching to a new tab with the URL
  // query still present re-seeded the new tab's connection.table.
  useEffect(() => {
    if (!queryParam) {
      ignoredRouteTableByTab.delete(currentTabId);
      return;
    }
    if (ignoredRouteTableByTab.get(currentTabId) === queryParam) return;
    handleChange(queryParam);
  }, [queryParam, currentTabId, handleChange]);

  if (!show) return null;

  return (
    <ConnectChooser
      value={value}
      onChange={handleChange}
      options={options}
      isChosen={isChosen}
      placeholder={t("database.container.select")}
      dropdownMinWidth={variant === "connection" ? 192 : undefined}
      openSignal={openSignal}
      triggerVariant={variant}
    />
  );
}
