import { ChevronLeft, Table as TableIcon } from "lucide-react";
import { useState } from "react";
import { Button } from "@/react/components/ui/button";
import { useVueState } from "@/react/hooks/useVueState";
import { useConnectionOfCurrentSQLEditorTab } from "@/react/stores/sqlEditor/tab-vue-state";
import { useDBSchemaV1Store } from "@/store";
import { PanelSearchBox } from "../common/PanelSearchBox";
import { ColumnsTable } from "../common/tables/ColumnsTable";
import { useViewStateNav } from "../common/useViewStateNav";
import { ExternalTablesTable } from "./ExternalTablesTable";

/**
 * React port of `frontend/src/views/sql-editor/EditorPanel/Panels/ExternalTablesPanel/*`.
 * Single Columns detail tab — no viewer needed, fully unblocked from
 * the CodeViewer/AI carve-out that gates other panels.
 */
export function ExternalTablesPanel() {
  const dbSchemaStore = useDBSchemaV1Store();
  const { database } = useConnectionOfCurrentSQLEditorTab();
  const databaseName = useVueState(() => database.value.name);
  const db = useVueState(() => database.value);
  const databaseMetadata = useVueState(
    () => dbSchemaStore.getDatabaseMetadata(databaseName ?? ""),
    { deep: true }
  );

  const {
    schema: schemaName,
    detail,
    setDetail,
    clearDetail,
  } = useViewStateNav();

  const [tableKeyword, setTableKeyword] = useState("");
  const [columnKeyword, setColumnKeyword] = useState("");

  const schema = databaseMetadata?.schemas.find((s) => s.name === schemaName);
  const externalTable = schema?.externalTables.find(
    (t) => t.name === detail?.externalTable
  );

  if (!db || !databaseMetadata || !schema) return null;

  return (
    <div className="px-2 py-2 gap-y-2 h-full overflow-hidden flex flex-col">
      {!externalTable ? (
        <>
          <div className="w-full flex flex-row gap-x-2 justify-end items-center">
            <PanelSearchBox value={tableKeyword} onChange={setTableKeyword} />
          </div>
          <div className="flex-1 min-h-0">
            <ExternalTablesTable
              database={databaseMetadata}
              schema={schema}
              externalTables={schema.externalTables}
              keyword={tableKeyword}
              onSelect={({ externalTable: target }) =>
                setDetail({ externalTable: target.name })
              }
            />
          </div>
        </>
      ) : (
        <>
          <div className="w-full h-9 flex flex-row gap-x-2 justify-between items-center">
            <Button
              variant="ghost"
              className="h-8 px-1 text-sm"
              onClick={() => clearDetail()}
            >
              <ChevronLeft className="size-5" />
              <TableIcon className="size-4 text-control" />
              <span className="truncate">{externalTable.name}</span>
            </Button>
            <PanelSearchBox value={columnKeyword} onChange={setColumnKeyword} />
          </div>
          <div className="flex-1 min-h-0">
            <ColumnsTable
              db={db}
              database={databaseMetadata}
              schema={schema}
              columns={externalTable.columns}
              flavor="external-table"
              keyword={columnKeyword}
            />
          </div>
        </>
      )}
    </div>
  );
}
