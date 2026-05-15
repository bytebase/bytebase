import { ChevronLeft, Zap } from "lucide-react";
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
import { TriggersTable } from "./TriggersTable";

/**
 * Standalone TriggersPanel. The Vue version reads `viewState.table` to
 * scope the trigger list to a specific table; the React port preserves
 * that behavior.
 */
export function TriggersPanel() {
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
    viewState,
    detail,
    setDetail,
    clearDetail,
  } = useViewStateNav();

  const [keyword, setKeyword] = useState("");

  const schema = databaseMetadata?.schemas.find((s) => s.name === schemaName);
  const table = schema?.tables.find((t) => t.name === viewState?.table);

  const [triggerName, triggerPosition] = extractKeyWithPosition(
    detail?.trigger ?? ""
  );
  const trigger = table?.triggers.find(
    (tr, i) => tr.name === triggerName && i === triggerPosition
  );

  if (!db || !databaseMetadata || !schema) return null;

  if (trigger) {
    return (
      <CodeViewer
        db={db}
        title={trigger.name}
        code={trigger.body}
        onBack={() => clearDetail()}
        titlePrefix={
          <Button
            variant="ghost"
            className="h-8 px-1 text-sm"
            onClick={() => clearDetail()}
          >
            <ChevronLeft className="size-5" />
            <Zap className="size-4 text-control" />
            <span className="truncate">{trigger.name}</span>
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
        <TriggersTable
          table={table}
          triggers={table?.triggers}
          keyword={keyword}
          onSelect={({ trigger: target, position }) =>
            setDetail({ trigger: keyWithPosition(target.name, position) })
          }
        />
      </div>
    </div>
  );
}
