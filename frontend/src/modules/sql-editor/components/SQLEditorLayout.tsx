import { type ReactNode, useEffect, useState } from "react";
import { router } from "@/app/router";
import { BannersWrapper } from "@/components/BannersWrapper";
import { useEnsureWorkspaceCommonData } from "@/hooks/useEnsureWorkspaceCommonData";
import { provideSheetContext } from "@/modules/sql-editor/model/Sheet";
import { RequestDrawerHost } from "./RequestDrawerHost";
import { SQLEditorRouteShell } from "./SQLEditorRouteShell";
import { SQLEditorThemeScope } from "./theme/SQLEditorThemeScope";
import { useMonacoThemeController } from "./theme/useMonacoThemeController";
import { useSQLEditorOverlayTheme } from "./theme/useSQLEditorOverlayTheme";
import { useWorkspaceSQLEditorTheme } from "./theme/useWorkspaceSQLEditorTheme";
import { useSQLEditorAutoSave } from "./useSQLEditorAutoSave";

function SQLEditorThemeRoot({ children }: Readonly<{ children: ReactNode }>) {
  const theme = useWorkspaceSQLEditorTheme();
  // Live-applies the theme on switch. It deliberately does NOT call setTheme on
  // mount — editors theme themselves at construction via options.theme — so it
  // never races Monaco construction (see useMonacoThemeController).
  useMonacoThemeController();
  // Themes all portaled overlays (dialogs, drawer, Select/Popover popups) that
  // render outside the SQL Editor DOM subtree.
  useSQLEditorOverlayTheme();
  return (
    <SQLEditorThemeScope theme={theme} asContents>
      {children}
    </SQLEditorThemeScope>
  );
}

/**
 * Top-level shell of the SQL Editor route. Wires up:
 *  - the `#sql-editor-debug` portal target (kept hidden by default, used
 *    by the debug `<li>` strings `SQLEditorHomePage` emits).
 *  - the React `<BannersWrapper>` at the top.
 *  - workspace-scope common data via `useEnsureWorkspaceCommonData()` —
 *    the same hook DashboardFrameShell uses. Idempotent loaders make it
 *    safe to call from every top-level shell.
 *  - `useSQLEditorAutoSave()` — the 2-second debounced saved query
 *    auto-save.
 *  - the `<SQLEditorRouteShell>` once `ready` flips true.
 */
export function SQLEditorLayout() {
  // Boots the per-view watchers (selectedKeys ↔ active tab) the moment
  // the layout appears. The sheet-context singleton is module-level and
  // lazy — calling it here ensures the watchers are wired before any
  // child component reads from it.
  provideSheetContext();

  const commonDataReady = useEnsureWorkspaceCommonData();
  const [routerReady, setRouterReady] = useState(false);

  useEffect(() => {
    let cancelled = false;
    void router.isReady().then(() => {
      if (!cancelled) setRouterReady(true);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  const ready = commonDataReady && routerReady;

  useSQLEditorAutoSave();

  return (
    <div className="relative h-screen overflow-hidden flex flex-col">
      {/* Hidden debug target. The debug probes (disconnected state, current
          tab id and connection) write `<li>` children into this list;
          unhide it via dev tools to inspect them. The fixed positioning is
          enough to stack above the editor body on its own — the React
          layering policy forbids a raw global `z-index` here. */}
      <ul
        id="sql-editor-debug"
        className="hidden text-xs font-mono max-h-[33vh] max-w-[40vw] overflow-auto fixed bottom-0 right-0 p-2 bg-background/50 border border-control-border"
      />
      <BannersWrapper />
      {ready && (
        <SQLEditorThemeRoot>
          <RequestDrawerHost>
            <SQLEditorRouteShell />
          </RequestDrawerHost>
        </SQLEditorThemeRoot>
      )}
    </div>
  );
}
