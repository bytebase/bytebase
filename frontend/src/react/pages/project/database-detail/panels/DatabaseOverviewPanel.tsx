import { useEffect, useMemo, useState } from "react";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { useDBSchemaV1Store } from "@/store";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { getDatabaseProject, hasProjectPermissionV2 } from "@/utils";
import { DatabaseObjectExplorerBridge } from "../legacy/DatabaseObjectExplorerBridge";
import { DatabaseOverviewInfoBridge } from "../legacy/DatabaseOverviewInfoBridge";

export function DatabaseOverviewPanel({
  database,
  hasSchemaPermission,
}: {
  database: Database;
  hasSchemaPermission?: boolean;
}) {
  const dbSchemaStore = useDBSchemaV1Store();
  const schemaList = useVueState(() =>
    dbSchemaStore.getSchemaList(database.name)
  );
  const routeSchema = useVueState(() => {
    const schema = router.currentRoute.value.query.schema;
    return typeof schema === "string" ? schema : "";
  });
  const [selectedSchemaName, setSelectedSchemaName] = useState("");
  const [tableSearchKeyword, setTableSearchKeyword] = useState("");
  const [externalTableSearchKeyword, setExternalTableSearchKeyword] =
    useState("");
  const schemaSelectionReady =
    schemaList.length === 0 ||
    schemaList.some((schema) => schema.name === selectedSchemaName);

  const allowViewSchema = useMemo(() => {
    if (typeof hasSchemaPermission === "boolean") {
      return hasSchemaPermission;
    }
    const project = getDatabaseProject(database);
    return project
      ? hasProjectPermissionV2(project, "bb.databases.getSchema")
      : false;
  }, [database, hasSchemaPermission]);

  useEffect(() => {
    if (schemaList.length === 0) {
      setSelectedSchemaName("");
      return;
    }

    const nextSchemaName =
      (routeSchema && schemaList.some((schema) => schema.name === routeSchema)
        ? routeSchema
        : schemaList.find((schema) => schema.name.toLowerCase() === "public")
            ?.name) ||
      schemaList[0]?.name ||
      "";

    setSelectedSchemaName((current) =>
      current === nextSchemaName ? current : nextSchemaName
    );
  }, [routeSchema, schemaList]);

  useEffect(() => {
    setTableSearchKeyword("");
    setExternalTableSearchKeyword("");
  }, [database.name]);

  useEffect(() => {
    if (!allowViewSchema || !schemaSelectionReady) {
      return;
    }

    const currentQuery = router.currentRoute.value.query;
    const currentSchema =
      typeof currentQuery.schema === "string" ? currentQuery.schema : "";

    if (currentSchema === selectedSchemaName) {
      return;
    }

    void router.replace({
      query: {
        ...currentQuery,
        schema: selectedSchemaName || undefined,
      },
    });
  }, [allowViewSchema, schemaSelectionReady, selectedSchemaName]);

  return (
    <div>
      <DatabaseOverviewInfoBridge database={database} className="pb-6" />

      {allowViewSchema && (
        <div className="border-t border-block-border pt-6">
          <DatabaseObjectExplorerBridge
            database={database}
            loading={schemaList.length > 0 && !selectedSchemaName}
            selectedSchemaName={selectedSchemaName}
            tableSearchKeyword={tableSearchKeyword}
            externalTableSearchKeyword={externalTableSearchKeyword}
            onSelectedSchemaNameChange={setSelectedSchemaName}
            onTableSearchKeywordChange={setTableSearchKeyword}
            onExternalTableSearchKeywordChange={setExternalTableSearchKeyword}
          />
        </div>
      )}
    </div>
  );
}
