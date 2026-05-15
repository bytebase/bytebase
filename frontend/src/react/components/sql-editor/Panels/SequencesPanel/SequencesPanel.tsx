import { useState } from "react";
import { useVueState } from "@/react/hooks/useVueState";
import { useConnectionOfCurrentSQLEditorTab } from "@/react/stores/sqlEditor/tab-vue-state";
import { useDBSchemaV1Store } from "@/store";
import { keyWithPosition } from "@/views/sql-editor/EditorCommon/utils";
import { PanelSearchBox } from "../common/PanelSearchBox";
import { useViewStateNav } from "../common/useViewStateNav";
import { SequencesTable } from "./SequencesTable";

/**
 * React port of `frontend/src/views/sql-editor/EditorPanel/Panels/SequencesPanel/*`.
 * List-only panel — clicking a row updates `viewState.detail.sequence`
 * to scroll that row into view (no separate detail surface).
 */
export function SequencesPanel() {
  const dbSchemaStore = useDBSchemaV1Store();
  const { database } = useConnectionOfCurrentSQLEditorTab();
  const databaseName = useVueState(() => database.value.name);
  const databaseMetadata = useVueState(
    () => dbSchemaStore.getDatabaseMetadata(databaseName ?? ""),
    { deep: true }
  );

  const { schema: schemaName, setDetail } = useViewStateNav();

  const [keyword, setKeyword] = useState("");

  const schema = databaseMetadata?.schemas.find((s) => s.name === schemaName);
  if (!databaseMetadata || !schema) return null;

  return (
    <div className="h-full overflow-hidden flex flex-col">
      <div className="w-full h-11 py-2 px-2 border-b border-block-border flex flex-row gap-x-2 justify-end items-center">
        <PanelSearchBox value={keyword} onChange={setKeyword} />
      </div>
      <div className="flex-1 min-h-0">
        <SequencesTable
          database={databaseMetadata}
          schema={schema}
          sequences={schema.sequences}
          keyword={keyword}
          onSelect={({ sequence, position }) =>
            setDetail({
              sequence: keyWithPosition(sequence.name, position),
            })
          }
        />
      </div>
    </div>
  );
}
