import { useState } from "react";
import { useAppDatabaseMetadata } from "@/react/hooks/useAppDatabaseMetadata";
import { useConnectionOfCurrentSQLEditorTab } from "@/react/hooks/useSQLEditorBridge";
import { PanelSearchBox } from "../common/PanelSearchBox";
import { useViewStateNav } from "../common/useViewStateNav";
import { ViewDetail } from "./ViewDetail";
import { ViewsTable } from "./ViewsTable";

export function ViewsPanel() {
  const { database } = useConnectionOfCurrentSQLEditorTab();
  const databaseName = database.name;
  const db = database;
  const databaseMetadata = useAppDatabaseMetadata(databaseName ?? "", {
    autoFetch: false,
  });

  const { schema: schemaName, detail, setDetail } = useViewStateNav();

  const [keyword, setKeyword] = useState("");

  const schema = databaseMetadata.schemas.find((s) => s.name === schemaName);
  const view = schema?.views.find((v) => v.name === detail?.view);

  if (!db || !schema) return null;

  if (view) {
    return (
      <ViewDetail
        db={db}
        database={databaseMetadata}
        schema={schema}
        view={view}
      />
    );
  }

  return (
    <div className="h-full overflow-hidden flex flex-col">
      <div className="w-full h-11 py-2 px-2 border-b border-block-border flex flex-row gap-x-2 justify-end items-center">
        <PanelSearchBox value={keyword} onChange={setKeyword} />
      </div>
      <div className="flex-1 min-h-0">
        <ViewsTable
          database={databaseMetadata}
          schema={schema}
          views={schema.views}
          keyword={keyword}
          onSelect={({ view: target }) => setDetail({ view: target.name })}
        />
      </div>
    </div>
  );
}
