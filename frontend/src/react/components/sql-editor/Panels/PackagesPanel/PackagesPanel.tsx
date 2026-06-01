import { ChevronLeft, Package } from "lucide-react";
import { useState } from "react";
import { Button } from "@/react/components/ui/button";
import { useAppDatabaseMetadata } from "@/react/hooks/useAppDatabaseMetadata";
import { useConnectionOfCurrentSQLEditorTab } from "@/react/hooks/useSQLEditorBridge";
import {
  extractKeyWithPosition,
  keyWithPosition,
} from "@/react/lib/keyWithPosition";
import { CodeViewer } from "../common/CodeViewer";
import { PanelSearchBox } from "../common/PanelSearchBox";
import { useViewStateNav } from "../common/useViewStateNav";
import { PackagesTable } from "./PackagesTable";

export function PackagesPanel() {
  const { database } = useConnectionOfCurrentSQLEditorTab();
  const databaseName = database.name;
  const db = database;
  const databaseMetadata = useAppDatabaseMetadata(databaseName ?? "", {
    autoFetch: false,
  });

  const {
    schema: schemaName,
    detail,
    setDetail,
    clearDetail,
  } = useViewStateNav();

  const [keyword, setKeyword] = useState("");

  const schema = databaseMetadata.schemas.find((s) => s.name === schemaName);
  const [packName, packPosition] = extractKeyWithPosition(
    detail?.package ?? ""
  );
  const pack = schema?.packages.find(
    (p, i) => p.name === packName && i === packPosition
  );

  if (!db || !schema) return null;

  if (pack) {
    return (
      <CodeViewer
        db={db}
        title={pack.name}
        code={pack.definition}
        onBack={() => clearDetail()}
        titlePrefix={
          <Button
            variant="ghost"
            className="h-8 px-1 text-sm"
            onClick={() => clearDetail()}
          >
            <ChevronLeft className="size-5" />
            <Package className="size-4 text-control" />
            <span className="truncate">{pack.name}</span>
          </Button>
        }
      />
    );
  }

  return (
    <div className="h-full overflow-hidden flex flex-col">
      <div className="w-full h-11 py-2 px-2 border-b border-block-border flex flex-row gap-x-2 justify-end items-center">
        <PanelSearchBox value={keyword} onChange={setKeyword} />
      </div>
      <div className="flex-1 min-h-0">
        <PackagesTable
          database={databaseMetadata}
          schema={schema}
          packages={schema.packages}
          keyword={keyword}
          onSelect={({ pack: target, position }) =>
            setDetail({ package: keyWithPosition(target.name, position) })
          }
        />
      </div>
    </div>
  );
}
