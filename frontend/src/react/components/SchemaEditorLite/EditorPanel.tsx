import { useState } from "react";
import { useSchemaEditorContext } from "./context";
import { DatabaseEditor } from "./Panels/DatabaseEditor";
import { FunctionEditor } from "./Panels/FunctionEditor";
import { ProcedureEditor } from "./Panels/ProcedureEditor";
import { TableEditor } from "./Panels/TableEditor";
import { ViewEditor } from "./Panels/ViewEditor";
import { TabsContainer } from "./TabsContainer";

export function EditorPanel() {
  const { tabs } = useSchemaEditorContext();
  const { currentTab } = tabs;
  const [selectedSchemaName, setSelectedSchemaName] = useState<
    string | undefined
  >();

  return (
    <main className="flex size-full flex-col overflow-y-hidden">
      <TabsContainer />
      <div className="flex-1 overflow-y-hidden" key={currentTab?.id ?? "empty"}>
        {!currentTab && (
          <div className="flex size-full items-center justify-center text-sm text-control-light">
            Select a database object from the tree to edit
          </div>
        )}
        {currentTab?.type === "database" && (
          <DatabaseEditor
            db={currentTab.database}
            database={currentTab.metadata.database}
            selectedSchemaName={currentTab.selectedSchema ?? selectedSchemaName}
            onSelectedSchemaNameChange={setSelectedSchemaName}
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
