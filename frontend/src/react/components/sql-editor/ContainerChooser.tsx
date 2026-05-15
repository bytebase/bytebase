import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useVueState } from "@/react/hooks/useVueState";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorTabStore,
} from "@/react/stores/sqlEditor/tab-vue-state";
import { router } from "@/router";
import { useDBSchemaV1Store } from "@/store";
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
  const tabStore = useSQLEditorTabStore();
  const dbSchemaStore = useDBSchemaV1Store();
  const connection = useConnectionOfCurrentSQLEditorTab();

  const engine = useVueState(() => connection.instance.value.engine);
  const databaseName = useVueState(() => connection.database.value.name);
  const tabTable = useVueState(() => tabStore.currentTab?.connection.table);
  const schemas = useVueState(
    () => dbSchemaStore.getDatabaseMetadata(databaseName).schemas
  );

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
    if (!tabStore.currentTab) return;
    tabStore.currentTab.connection.table =
      next === OptionValueUnspecified ? undefined : next;
  };

  // Seed from URL query parameter on mount and whenever the query param OR
  // the active tab changes. Mirrors Vue's watchEffect, which implicitly
  // tracked both `route.query.table` and `tab.value` (the latter via the
  // setter's reactive reads) so that switching to a new tab with the URL
  // query still present re-seeded the new tab's connection.table.
  const queryParam = useVueState(
    () => router.currentRoute.value.query.table as string | undefined
  );
  const currentTabId = useVueState(() => tabStore.currentTab?.id);
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
