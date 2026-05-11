import { useEffect, useState } from "react";
import { BannersWrapper } from "@/react/components/BannersWrapper";
import { router } from "@/router";
import {
  useEnvironmentV1Store,
  useSettingV1Store,
  useSQLEditorWorksheetStore,
} from "@/store";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { RequestDrawerHost } from "./RequestDrawerHost";
import { SQLEditorRouteShell } from "./SQLEditorRouteShell";
import { useSQLEditorAutoSave } from "./useSQLEditorAutoSave";

/**
 * React port of `frontend/src/layouts/SQLEditorLayout.vue`.
 *
 * Top-level shell of the SQL Editor route. Wires up:
 *  - the legacy `#sql-editor-debug` teleport target (kept hidden by
 *    default, used by debug `<li>` strings the inner shells emit).
 *  - the React `<BannersWrapper>` at the top.
 *  - workspace profile + environment fetch on mount, gated by Vue
 *    Router's `isReady()` so the SQL Editor doesn't bootstrap before
 *    initial route resolution.
 *  - `useSQLEditorAutoSave()` — the 2-second debounced worksheet
 *    auto-save extracted from the legacy `provideSQLEditorContext()`.
 *  - the `<SQLEditorRouteShell>` once `ready` flips true.
 */
export function SQLEditorLayout() {
  const settingStore = useSettingV1Store();
  const environmentStore = useEnvironmentV1Store();
  // Mounting the worksheet store eagerly mirrors the legacy
  // `provideSheetContext()` call — it boots the per-view watchers
  // (selectedKeys ↔ active tab) the moment the layout appears.
  useSQLEditorWorksheetStore();

  const [ready, setReady] = useState(false);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      await router.isReady();
      await Promise.all([
        settingStore.getOrFetchSettingByName(
          Setting_SettingName.WORKSPACE_PROFILE
        ),
        environmentStore.fetchEnvironments(),
      ]);
      if (!cancelled) setReady(true);
    })();
    return () => {
      cancelled = true;
    };
  }, [settingStore, environmentStore]);

  useSQLEditorAutoSave();

  return (
    <div className="relative h-screen overflow-hidden flex flex-col">
      {/* Hidden teleport target inherited from the Vue layout. The debug
          probes (Pinia connection, current tab, etc.) write `<li>`
          children into this list when manually unhidden via dev tools.
          The fixed positioning is enough to stack above the editor body
          on its own — the legacy `z-999999` was a layering escape hatch
          that the React layering policy now forbids. */}
      <ul
        id="sql-editor-debug"
        className="hidden text-xs font-mono max-h-[33vh] max-w-[40vw] overflow-auto fixed bottom-0 right-0 p-2 bg-white/50 border border-gray-400"
      />
      <BannersWrapper />
      {ready && (
        <RequestDrawerHost>
          <SQLEditorRouteShell />
        </RequestDrawerHost>
      )}
    </div>
  );
}
