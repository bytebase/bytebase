import { ChevronLeft, FileCode } from "lucide-react";
import { useState } from "react";
import { Button } from "@/react/components/ui/button";
import { useVueState } from "@/react/hooks/useVueState";
import { useConnectionOfCurrentSQLEditorTab } from "@/react/stores/sqlEditor/tab-vue-state";
import { useDBSchemaV1Store } from "@/store";
import {
  extractKeyWithPosition,
  keyWithPosition,
} from "@/views/sql-editor/EditorCommon/utils";
import { CodeViewer } from "../common/CodeViewer";
import { PanelSearchBox } from "../common/PanelSearchBox";
import { useViewStateNav } from "../common/useViewStateNav";
import { ProceduresTable } from "./ProceduresTable";

export function ProceduresPanel() {
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

  const [keyword, setKeyword] = useState("");

  const schema = databaseMetadata?.schemas.find((s) => s.name === schemaName);
  const [procName, procPosition] = extractKeyWithPosition(
    detail?.procedure ?? ""
  );
  const procedure = schema?.procedures.find(
    (p, i) => p.name === procName && i === procPosition
  );

  if (!db || !databaseMetadata || !schema) return null;

  if (procedure) {
    return (
      <CodeViewer
        db={db}
        title={procedure.name}
        code={procedure.definition}
        onBack={() => clearDetail()}
        titlePrefix={
          <Button
            variant="ghost"
            className="h-8 px-1 text-sm"
            onClick={() => clearDetail()}
          >
            <ChevronLeft className="size-5" />
            <FileCode className="size-4 text-control" />
            <span className="truncate">{procedure.name}</span>
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
        <ProceduresTable
          database={databaseMetadata}
          schema={schema}
          procedures={schema.procedures}
          keyword={keyword}
          onSelect={({ procedure: target, position }) =>
            setDetail({
              procedure: keyWithPosition(target.name, position),
            })
          }
        />
      </div>
    </div>
  );
}
