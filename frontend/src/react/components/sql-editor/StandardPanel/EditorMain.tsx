import { useCallback, useEffect } from "react";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { ReadonlyModeNotSupported } from "@/react/components/sql-editor/ReadonlyModeNotSupported";
import { useVueState } from "@/react/hooks/useVueState";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorTabStore,
} from "@/react/stores/sqlEditor/tab-vue-state";
import { useDatabaseV1Store } from "@/store";
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
 *
 * Calls the existing Vue `useExecuteSQL` composable directly: its body
 * is Pinia-store-driven and the only Vue-specific bits (`reactive`,
 * `markRaw`) behave as no-ops outside Vue's reactivity system.
 */
export function EditorMain({ onChangeConnection }: EditorMainProps) {
  const tabStore = useSQLEditorTabStore();
  const databaseStore = useDatabaseV1Store();
  const { instance } = useConnectionOfCurrentSQLEditorTab();

  const tabId = useVueState(() => tabStore.currentTab?.id);
  const isDisconnected = useVueState(() => tabStore.isDisconnected);
  const engine = useVueState(() => instance.value.engine);

  const allowReadonlyMode =
    !isDisconnected && instanceV1HasReadonlyMode(engine ?? Engine.MYSQL);

  const { execute } = useExecuteSQL();

  const handleExecute = useCallback(
    ({ params, newTab }: { params: SQLEditorQueryParams; newTab: boolean }) => {
      if (newTab) {
        tabStore.cloneTab(tabStore.currentTabId, {
          statement: params.statement,
        });
      }
      requestAnimationFrame(() => {
        void execute(params);
      });
    },
    [tabStore, execute]
  );

  // Run-from-toolbar: pull the active statement that the React
  // `SQLEditor` published to the shared shallowRef.
  const handleExecuteFromActionBar = useCallback(
    (params: SQLEditorQueryParams) => {
      const tab = tabStore.currentTab;
      if (!tab) return;
      const statement =
        activeStatementRef.value || tab.statement || params.statement;
      handleExecute({ params: { ...params, statement }, newTab: false });
    },
    [tabStore, handleExecute]
  );

  // execute-sql event (fired e.g. from the schema-pane "run" affordance)
  useEffect(() => {
    const off = sqlEditorEvents.on(
      "execute-sql",
      async ({ connection, statement, batchQueryContext }) => {
        const database = await databaseStore.getOrFetchDatabaseByName(
          connection.database
        );
        const newTab = tabStore.addTab(
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
  }, [databaseStore, tabStore, execute]);

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
