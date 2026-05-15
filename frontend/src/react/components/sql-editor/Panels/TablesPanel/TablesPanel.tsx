import { useState } from "react";
import { useVueState } from "@/react/hooks/useVueState";
import { useConnectionOfCurrentSQLEditorTab } from "@/react/stores/sqlEditor/tab-vue-state";
import { useDBSchemaV1Store } from "@/store";
import { PanelSearchBox } from "../common/PanelSearchBox";
import { useViewStateNav } from "../common/useViewStateNav";
import { TableDetail } from "./TableDetail";
import { TablesTable } from "./TablesTable";

export function TablesPanel() {
  const dbSchemaStore = useDBSchemaV1Store();
  const { database } = useConnectionOfCurrentSQLEditorTab();
  const databaseName = useVueState(() => database.value.name);
  const db = useVueState(() => database.value);
  const databaseMetadata = useVueState(
    () => dbSchemaStore.getDatabaseMetadata(databaseName ?? ""),
    { deep: true }
  );

  const { schema: schemaName, detail, setDetail } = useViewStateNav();

  const [keyword, setKeyword] = useState("");

  const schema = databaseMetadata?.schemas.find((s) => s.name === schemaName);
  const table = schema?.tables.find((t) => t.name === detail?.table);

  if (!db || !databaseMetadata || !schema) return null;

  if (table) {
    return (
      <TableDetail
        db={db}
        database={databaseMetadata}
        schema={schema}
        table={table}
      />
    );
  }

  return (
    <div className="px-2 py-2 gap-y-2 h-full overflow-hidden flex flex-col">
      <div className="w-full flex flex-row gap-x-2 justify-end items-center">
        <PanelSearchBox value={keyword} onChange={setKeyword} />
      </div>
      <div className="flex-1 min-h-0">
        <TablesTable
          db={db}
          database={databaseMetadata}
          schema={schema}
          tables={schema.tables}
          keyword={keyword}
          onSelect={({ table: target }) => setDetail({ table: target.name })}
        />
      </div>
    </div>
  );
}
