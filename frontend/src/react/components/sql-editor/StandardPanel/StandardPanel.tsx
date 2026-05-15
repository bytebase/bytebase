import { Loader2 } from "lucide-react";
import { Suspense } from "react";
import {
  Panel,
  Group as PanelGroup,
  Separator as PanelResizeHandle,
} from "react-resizable-panels";
import { useShallow } from "zustand/react/shallow";
import { AIChatToSQL, AIContextProvider } from "@/plugins/ai/react";
import { resizeHandleClass } from "@/react/components/SchemaEditorLite/resize";
import { ResultPanel } from "@/react/components/sql-editor/ResultPanel/ResultPanel";
import { useVueState } from "@/react/hooks/useVueState";
import {
  selectEditorPanelSize,
  useSQLEditorStore,
} from "@/react/stores/sqlEditor";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorTabStore,
} from "@/react/stores/sqlEditor/tab-vue-state";
import { instanceV1HasReadonlyMode } from "@/utils";
import { EditorMain } from "./EditorMain";

// Lazy-load EditorMain ↔ Pane callback to keep parity with Vue's
// async-imported `AIChatToSQL`. The original `<Suspense>` fallback is
// the matrix-style spinner; we mirror that with `<Loader2 />`.
const AIPaneFallback = () => (
  <div className="w-full h-full grow flex flex-col items-center justify-center">
    <Loader2 className="size-6 animate-spin text-control-light" />
  </div>
);

/**
 * React port of `frontend/src/views/sql-editor/EditorPanel/StandardPanel/StandardPanel.vue`.
 *
 * Worksheet-mode editor host. Layout, ordered top-to-bottom:
 *   1. Optional outer vertical split — editor / `<ResultPanel>`.
 *      Only rendered when the underlying instance supports read-only
 *      queries (mirrors the Vue `showResultPanel` gate that prevents
 *      the editor from being squeezed into an arbitrary top pane when
 *      the result pane wouldn't render anyway).
 *   2. Inner horizontal split — `<EditorMain>` / `<AIChatToSQL>`.
 *      The AI side pane is now a React tree (Stage 22 port). `<AIContextProvider>`
 *      wraps the React `<AIChatToSQL>` to re-establish the per-tab AI
 *      state — the Vue `<VueMount component={AIChatToSQLBridgeHost}>`
 *      bridge is gone.
 *
 * State source: `useSQLEditorTabStore` for tab + disconnect state,
 * `useConnectionOfCurrentSQLEditorTab` for the current instance,
 * `useSQLEditorStore` (zustand) for AI-panel visibility / sizing.
 */
export function StandardPanel() {
  const tabStore = useSQLEditorTabStore();
  const handleEditorPanelResize = useSQLEditorStore(
    (s) => s.handleEditorPanelResize
  );
  const setAsidePanelTab = useSQLEditorStore((s) => s.setAsidePanelTab);
  const setShowConnectionPanel = useSQLEditorStore(
    (s) => s.setShowConnectionPanel
  );
  const { instance: instanceRef } = useConnectionOfCurrentSQLEditorTab();

  const tab = useVueState(() => tabStore.currentTab);
  const isDisconnected = useVueState(() => tabStore.isDisconnected);
  const showAIPanel = useSQLEditorStore((s) => s.showAIPanel);
  const editorPanelSize = useSQLEditorStore(useShallow(selectEditorPanelSize));
  const instanceHasReadonly = useVueState(() =>
    instanceV1HasReadonlyMode(instanceRef.value)
  );

  if (tab && tab.mode !== "WORKSHEET") {
    return null;
  }

  const showResultPanel = !isDisconnected && instanceHasReadonly;

  const handleAiPanelResize = (sizePct: number) => {
    // react-resizable-panels reports a `PanelSize` struct
    // ({ asPercentage, inPixels }) on resize. Pinia stores a 0-1
    // fraction (`{size: 0.7, ...}`); convert and forward — the store's
    // setter writes `1 - size` to localStorage.
    if (!Number.isFinite(sizePct)) return;
    handleEditorPanelResize(sizePct / 100);
  };

  const handleChangeConnection = () => {
    setAsidePanelTab("SCHEMA");
    setShowConnectionPanel(true);
  };

  const editorWithAi = (
    <PanelGroup orientation="horizontal" className="h-full">
      <Panel
        defaultSize={`${editorPanelSize.size * 100}%`}
        minSize={`${editorPanelSize.min * 100}%`}
        maxSize={`${editorPanelSize.max * 100}%`}
        onResize={(size) => {
          // Only live-sync when the AI pane is visible. Without this
          // guard, the resize fires on initial layout when AI is hidden
          // and the editor occupies 100%, clobbering the persisted size.
          if (showAIPanel) handleAiPanelResize(size.asPercentage);
        }}
      >
        <EditorMain onChangeConnection={handleChangeConnection} />
      </Panel>
      {showAIPanel && tab && (
        <>
          <PanelResizeHandle className={resizeHandleClass("vertical")} />
          <Panel
            defaultSize={`${(1 - editorPanelSize.size) * 100}%`}
            minSize="10%"
          >
            <div className="h-full overflow-hidden flex flex-col">
              <Suspense fallback={<AIPaneFallback />}>
                <AIContextProvider>
                  <AIChatToSQL />
                </AIContextProvider>
              </Suspense>
            </div>
          </Panel>
        </>
      )}
    </PanelGroup>
  );

  if (!showResultPanel) {
    return <div className="h-full">{editorWithAi}</div>;
  }

  return (
    <PanelGroup orientation="vertical" className="h-full">
      <Panel defaultSize="60%" minSize="20%" maxSize="80%">
        {editorWithAi}
      </Panel>
      <PanelResizeHandle className={resizeHandleClass("horizontal")} />
      <Panel defaultSize="40%" minSize="20%">
        <div className="relative h-full">
          <ResultPanel />
        </div>
      </Panel>
    </PanelGroup>
  );
}
