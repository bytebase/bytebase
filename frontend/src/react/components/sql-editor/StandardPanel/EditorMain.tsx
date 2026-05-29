import { useCallback, useEffect } from "react";
import { ReadonlyModeNotSupported } from "@/react/components/sql-editor/ReadonlyModeNotSupported";
import { useExecuteSQL } from "@/react/hooks/useExecuteSQL";
import { useConnectionOfCurrentSQLEditorTab } from "@/react/hooks/useSQLEditorBridge";
import { useAppStore } from "@/react/stores/app";
import {
  getSQLEditorTabsState,
  useIsDisconnected,
  useSQLEditorTabState,
} from "@/react/stores/sqlEditor/tab";
import type { SQLEditorQueryParams } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { getInstanceResource, instanceV1HasReadonlyMode } from "@/utils";
import { sqlEditorEvents } from "@/views/sql-editor/events";
import { EditorAction } from "../EditorAction";
import { ExecutingHintModal } from "../ExecutingHintModal";
import { SaveSheetModal } from "../SaveSheetModal";
import { Welcome } from "../Welcome";
import { SQLEditor } from "./SQLEditor";
import { activeStatementRef } from "./state";

interface EditorMainProps {
  /** Open the connection drawer (hosted by SQLEditorHomePage). */
  onChangeConnection: () => void;
}

/**
 * React port of `frontend/src/views/sql-editor/EditorPanel/StandardPanel/EditorMain.vue`.
 *
 * Worksheet shell:
 *  - Top toolbar (`EditorAction`, already React)
 *  - Body: `SQLEditor` when there's a tab, `Welcome` otherwise
 *  - Two hidden React modal mounts (`ExecutingHintModal`, `SaveSheetModal`)
 *
 * AI side pane has been hoisted to `StandardPanel.vue` (Vue host) so
 * the cross-framework boundary stays clean — see Stage 17 design.
 */
export function EditorMain({ onChangeConnection }: EditorMainProps) {
  const { instance } = useConnectionOfCurrentSQLEditorTab();

  const tabId = useSQLEditorTabState((s) => s.tabsById.get(s.currentTabId)?.id);
  const isDisconnected = useIsDisconnected();
  const engine = instance.engine;

  const allowReadonlyMode =
    !isDisconnected && instanceV1HasReadonlyMode(engine ?? Engine.MYSQL);

  const { execute } = useExecuteSQL();

  const handleExecute = useCallback(
    ({ params, newTab }: { params: SQLEditorQueryParams; newTab: boolean }) => {
      if (newTab) {
        const tabsState = getSQLEditorTabsState();
        tabsState.cloneTab(tabsState.currentTabId, {
          statement: params.statement,
        });
      }
      requestAnimationFrame(() => {
        void execute(params);
      });
    },
    [execute]
  );

  // Run-from-toolbar: pull the active statement that the React
  // `SQLEditor` published to the shared shallowRef.
  const handleExecuteFromActionBar = useCallback(
    (params: SQLEditorQueryParams) => {
      const tabsState = getSQLEditorTabsState();
      const tab = tabsState.tabsById.get(tabsState.currentTabId);
      if (!tab) return;
      const statement =
        activeStatementRef.value || tab.statement || params.statement;
      handleExecute({ params: { ...params, statement }, newTab: false });
    },
    [handleExecute]
  );

  // execute-sql event (fired e.g. from the schema-pane "run" affordance)
  useEffect(() => {
    const off = sqlEditorEvents.on(
      "execute-sql",
      async ({ connection, statement, batchQueryContext }) => {
        const database = await useAppStore
          .getState()
          .getOrFetchDatabaseByName(connection.database);
        const newTab = getSQLEditorTabsState().addTab(
          { connection, statement, batchQueryContext },
          /* beside */ true
        );
        requestAnimationFrame(() => {
          void execute({
            connection: { ...newTab.connection },
            statement,
            engine: getInstanceResource(database).engine,
            explain: false,
            selection: null,
          });
        });
      }
    );
    return () => {
      off();
    };
  }, [execute]);

  if (!isDisconnected && !allowReadonlyMode) {
    // Connected to an instance without a read-only data source —
    // surface the admin-mode CTA, matching Vue `EditorMain.vue`'s
    // `<ReactPageMount v-else page="ReadonlyModeNotSupported" />` branch.
    return <ReadonlyModeNotSupported />;
  }

  return (
    <div className="h-full flex flex-col overflow-hidden">
      <EditorAction onExecute={handleExecuteFromActionBar} />
      {tabId ? (
        <div className="flex-1 min-h-0">
          <SQLEditor
            onExecute={(params, newTab) => handleExecute({ params, newTab })}
          />
        </div>
      ) : (
        <div className="flex-1 flex flex-col min-h-0">
          <Welcome onChangeConnection={onChangeConnection} />
        </div>
      )}
      {/* Hidden React modal mounts — their Dialogs portal to overlay
          layer; the inline mounts must not occupy flex space. */}
      <div className="hidden">
        <ExecutingHintModal />
        <SaveSheetModal />
      </div>
    </div>
  );
}
