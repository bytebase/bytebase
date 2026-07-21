import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSchemaEditorContext } from "./context";
import { DatabaseEditor } from "./Panels/DatabaseEditor";
import { FunctionEditor } from "./Panels/FunctionEditor";
import { ProcedureEditor } from "./Panels/ProcedureEditor";
import { TableEditor } from "./Panels/TableEditor";
import { ViewEditor } from "./Panels/ViewEditor";
import { TabsContainer } from "./TabsContainer";

export function EditorPanel() {
  const { t } = useTranslation();
  const { tabs } = useSchemaEditorContext();
  const { currentTab } = tabs;

  // Per-tab schema selection. User choices are stored by tab ID so they
  // don't leak across tabs. Falls back to the tab's initial selectedSchema
  // (set when navigating from a schema node in the tree).
  const [selectedSchemas, setSelectedSchemas] = useState<
    Record<string, string | undefined>
  >({});

  const selectedSchemaName = currentTab
    ? (selectedSchemas[currentTab.id] ??
      (currentTab.type === "database" ? currentTab.selectedSchema : undefined))
    : undefined;

  const handleSchemaNameChange = useCallback(
    (name: string | undefined) => {
      if (currentTab) {
        setSelectedSchemas((prev) => ({ ...prev, [currentTab.id]: name }));
      }
    },
    [currentTab]
  );

  return (
    <main className="flex size-full flex-col overflow-y-hidden">
      <TabsContainer />
      <div className="flex-1 overflow-y-hidden" key={currentTab?.id ?? "empty"}>
        {!currentTab && (
          <div className="flex size-full items-center justify-center text-sm text-control-light">
            {t("schema-editor.select-object-hint")}
          </div>
        )}
        {currentTab?.type === "database" && (
          <DatabaseEditor
            db={currentTab.database}
            database={currentTab.metadata.database}
            selectedSchemaName={selectedSchemaName}
            onSelectedSchemaNameChange={handleSchemaNameChange}
          />
        )}
        {currentTab?.type === "table" && (
          <TableEditor
            db={currentTab.database}
            database={currentTab.metadata.database}
            schema={currentTab.metadata.schema}
            table={currentTab.metadata.table}
          />
        )}
        {currentTab?.type === "view" && (
          <ViewEditor
            db={currentTab.database}
            database={currentTab.metadata.database}
            schema={currentTab.metadata.schema}
            view={currentTab.metadata.view}
          />
        )}
        {currentTab?.type === "procedure" && (
          <ProcedureEditor
            db={currentTab.database}
            database={currentTab.metadata.database}
            schema={currentTab.metadata.schema}
            procedure={currentTab.metadata.procedure}
          />
        )}
        {currentTab?.type === "function" && (
          <FunctionEditor
            db={currentTab.database}
            database={currentTab.metadata.database}
            schema={currentTab.metadata.schema}
            func={currentTab.metadata.function}
          />
        )}
      </div>
    </main>
  );
}
