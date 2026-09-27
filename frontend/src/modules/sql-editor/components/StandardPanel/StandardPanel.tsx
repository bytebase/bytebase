import { Loader2 } from "lucide-react";
import type { ReactNode } from "react";
import { Suspense, useEffect, useLayoutEffect, useRef, useState } from "react";
import {
  Panel,
  Group as PanelGroup,
  type PanelImperativeHandle,
  Separator as PanelResizeHandle,
} from "react-resizable-panels";
import { useShallow } from "zustand/react/shallow";
import { AIChatToSQL, AIContextProvider } from "@/modules/ai/components";
import { resizeHandleClass } from "@/modules/schema-editor/resize";
import { ResultPanel } from "@/modules/sql-editor/components/ResultPanel/ResultPanel";
import { useConnectionOfCurrentSQLEditorTab } from "@/modules/sql-editor/hooks/useSQLEditorState";
import {
  MAXIMUM_RESULT_PANEL_SIZE,
  MINIMUM_RESULT_PANEL_SIZE,
  selectEditorPanelSize,
  useSQLEditorStore,
} from "@/modules/sql-editor/store";
import {
  useCurrentSQLEditorTab,
  useIsDisconnected,
} from "@/modules/sql-editor/store/tab";
import { instanceV1HasReadonlyMode } from "@/utils";
import { EditorMain } from "./EditorMain";

/** Panel sizes are CSS percentages; round so 1 - 0.8 reads as "20%". */
const percent = (fraction: number) => `${Math.round(fraction * 1000) / 10}%`;

const AIPaneFallback = () => (
  <div className="w-full h-full grow flex flex-col items-center justify-center">
    <Loader2 className="size-6 animate-spin text-control-light" />
  </div>
);

/**
 * SavedQuery-mode editor host. Layout, ordered top-to-bottom:
 *   1. Optional outer vertical split — editor / `<ResultPanel>`.
 *      Only rendered when the tab is connected and the instance supports
 *      read-only queries (`showResultPanel`), so the editor isn't
 *      squeezed into an arbitrary top pane when the result pane wouldn't
 *      render anyway.
 *   2. Inner horizontal split — `<EditorMain>` / `<AIChatToSQL>`.
 *      `<AIContextProvider>` wraps `<AIChatToSQL>` to provide the per-tab
 *      AI state.
 *
 * State source: tab Zustand selectors for tab + disconnect state,
 * `useConnectionOfCurrentSQLEditorTab` for the current instance,
 * `useSQLEditorStore` (zustand) for AI-panel visibility / sizing.
 */
export function StandardPanel() {
  const handleEditorPanelResize = useSQLEditorStore(
    (s) => s.handleEditorPanelResize
  );
  const setAsidePanelTab = useSQLEditorStore((s) => s.setAsidePanelTab);
  const setShowConnectionPanel = useSQLEditorStore(
    (s) => s.setShowConnectionPanel
  );
  const { instance } = useConnectionOfCurrentSQLEditorTab();

  const tab = useCurrentSQLEditorTab();
  const isDisconnected = useIsDisconnected();
  const showAIPanel = useSQLEditorStore((s) => s.showAIPanel);
  const editorPanelSize = useSQLEditorStore(useShallow(selectEditorPanelSize));
  const setResultPanelMaximized = useSQLEditorStore(
    (s) => s.setResultPanelMaximized
  );
  const instanceHasReadonly = instanceV1HasReadonlyMode(instance);
  // Derived above the early return below so the effect stays unconditional.
  const isSavedQueryTab = !tab || tab.mode === "SAVED_QUERY";
  const showResultPanel =
    isSavedQueryTab && !isDisconnected && instanceHasReadonly;

  // Maximizing is a momentary view of one result, not a remembered preference:
  // it lasts as long as the pane it belongs to. Switching tabs remounts this
  // component and losing the connection takes the pane away; either would
  // otherwise leave a collapsed editor and a hidden sidebar with no control on
  // screen to undo them.
  useEffect(() => {
    if (!showResultPanel) setResultPanelMaximized(false);
    return () => setResultPanelMaximized(false);
  }, [showResultPanel, setResultPanelMaximized]);

  if (!isSavedQueryTab) {
    return null;
  }

  const handleAiPanelResize = (sizePct: number) => {
    // react-resizable-panels reports a `PanelSize` struct
    // ({ asPercentage, inPixels }) on resize. The store keeps a 0-1
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
          <PanelResizeHandle
            className={resizeHandleClass("vertical", "w-0.5")}
          />
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

  return <EditorResultSplit>{editorWithAi}</EditorResultSplit>;
}

/**
 * The editor / result split.
 *
 * Owns the remembered height, read once when this group mounts rather than
 * when `StandardPanel` does: the group alone comes and goes with the
 * connection, and a height dragged before it went away is the one to come
 * back to. Reading it live instead would feed `defaultSize` a new value on
 * every frame of a drag.
 */
function EditorResultSplit({ children }: { children: ReactNode }) {
  const [initialResultPanelSize] = useState(
    () => useSQLEditorStore.getState().resultPanelSize
  );
  const resultPanelMaximized = useSQLEditorStore((s) => s.resultPanelMaximized);
  const handleResultPanelResize = useSQLEditorStore(
    (s) => s.handleResultPanelResize
  );
  // The editor pane is collapsed rather than unmounted, so Monaco keeps its
  // cursor, scroll and undo history while the result pane is maximized.
  const editorPanelRef = useRef<PanelImperativeHandle | null>(null);

  // Before paint, so a group that mounts maximized never shows the editor for
  // a frame first.
  useLayoutEffect(() => {
    const panel = editorPanelRef.current;
    if (!panel) return;
    if (resultPanelMaximized) {
      panel.collapse();
    } else {
      panel.expand();
    }
  }, [resultPanelMaximized]);

  return (
    <PanelGroup orientation="vertical" className="h-full">
      <Panel
        panelRef={editorPanelRef}
        collapsible
        collapsedSize="0%"
        defaultSize={percent(1 - initialResultPanelSize)}
        minSize={percent(1 - MAXIMUM_RESULT_PANEL_SIZE)}
        maxSize={percent(1 - MINIMUM_RESULT_PANEL_SIZE)}
        onResize={(size) => {
          const editorShare = size.asPercentage / 100;
          if (!Number.isFinite(editorShare)) return;
          // Height only. Whether the pane is maximized is the reader's
          // decision, taken through the toggle: deriving it from the layout
          // would let the group's first report — the default size, sent
          // before the effect above can collapse — cancel it on every mount.
          handleResultPanelResize(1 - editorShare);
        }}
      >
        {children}
      </Panel>
      {/* A maximized pane owns the whole area; dragging it back to a
          half-state would leave the editor visible under a sidebar that is
          still hidden. Restore first. */}
      <PanelResizeHandle
        disabled={resultPanelMaximized}
        className={resizeHandleClass("horizontal", "h-0.5")}
      />
      <Panel
        defaultSize={percent(initialResultPanelSize)}
        minSize={percent(MINIMUM_RESULT_PANEL_SIZE)}
      >
        <div className="relative h-full">
          <ResultPanel />
        </div>
      </Panel>
    </PanelGroup>
  );
}
