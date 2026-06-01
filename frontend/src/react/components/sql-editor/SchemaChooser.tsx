import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useAppDatabaseMetadata } from "@/react/hooks/useAppDatabaseMetadata";
import { useConnectionOfCurrentSQLEditorTab } from "@/react/hooks/useSQLEditorBridge";
import { useVueRoute } from "@/react/hooks/useVueRoute";
import {
  getSQLEditorTabsState,
  useSQLEditorTabState,
} from "@/react/stores/sqlEditor/tab";
import { instanceAllowsSchemaScopedQuery } from "@/utils";
import { ConnectChooser } from "./ConnectChooser";

const SchemaOptionValueUnspecified = "-1";

/**
 * Replaces frontend/src/views/sql-editor/EditorCommon/SchemaChooser.vue.
 * Visible only for engines that support schema-scoped queries.
 * Selected schema persists to the current tab's connection.schema.
 */
export function SchemaChooser() {
  const { t } = useTranslation();
  const { instance, database } = useConnectionOfCurrentSQLEditorTab();

  const engine = instance.engine;
  const databaseName = database.name;
  const tabSchema = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.connection.schema
  );
  // Parent SchemaPane (E4 migration) drives the metadata fetch; here we
  // only need the cached read.
  const { schemas } = useAppDatabaseMetadata(databaseName, {
    autoFetch: false,
  });

  const show = instanceAllowsSchemaScopedQuery(engine);

  const options = useMemo(() => {
    const opts = schemas.map((schema) => ({
      value: schema.name,
      label: schema.name || t("db.schema.default"),
    }));
    return [
      {
        value: SchemaOptionValueUnspecified,
        label: t("database.schema.unspecified"),
      },
      ...opts,
    ];
  }, [schemas, t]);

  const value =
    tabSchema === undefined ? SchemaOptionValueUnspecified : tabSchema;
  const isChosen = value !== SchemaOptionValueUnspecified;

  const handleChange = (next: string) => {
    const tabsState = getSQLEditorTabsState();
    const currentTab = tabsState.tabsById.get(tabsState.currentTabId);
    if (!currentTab) return;
    tabsState.updateCurrentTab({
      connection: {
        ...currentTab.connection,
        schema: next === SchemaOptionValueUnspecified ? undefined : next,
      },
    });
  };

  // Seed from URL query parameter on mount and whenever the query param OR
  // the active tab changes. Mirrors Vue's watchEffect, which implicitly
  // tracked both `route.query.schema` and `tab.value` (the latter via the
  // setter's reactive reads) so that switching to a new tab with the URL
  // query still present re-seeded the new tab's connection.schema.
  const queryParam = useVueRoute().query.schema as string | undefined;
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
      placeholder={t("database.schema.select")}
    />
  );
}
