import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
  useSQLEditorTabStore,
} from "@/store";
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
  const tabStore = useSQLEditorTabStore();
  const dbSchemaStore = useDBSchemaV1Store();
  const connection = useConnectionOfCurrentSQLEditorTab();

  const engine = useVueState(() => connection.instance.value.engine);
  const databaseName = useVueState(() => connection.database.value.name);
  const tabSchema = useVueState(() => tabStore.currentTab?.connection.schema);
  const schemas = useVueState(
    () => dbSchemaStore.getDatabaseMetadata(databaseName).schemas
  );

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
    if (!tabStore.currentTab) return;
    tabStore.currentTab.connection.schema =
      next === SchemaOptionValueUnspecified ? undefined : next;
  };

  // Seed from URL query parameter (mirrors Vue watchEffect)
  const queryParam = useVueState(
    () => router.currentRoute.value.query.schema as string | undefined
  );
  useEffect(() => {
    if (queryParam) handleChange(queryParam);
    // handleChange is defined inline and stable for this effect
  }, [queryParam]);

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
