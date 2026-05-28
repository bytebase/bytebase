import logoIcon from "@/assets/logo-icon.svg";
import { Separator } from "@/react/components/ui/separator";
import { usePiniaBridge } from "@/react/hooks/usePiniaBridge";
import type { AsidePanelTab } from "@/react/stores/sqlEditor";
import { useSQLEditorStore } from "@/react/stores/sqlEditor";
import { useSQLEditorEditorState } from "@/react/stores/sqlEditor/editor";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_DETAIL } from "@/router/dashboard/projectV1";
import { WORKSPACE_ROUTE_LANDING } from "@/router/dashboard/workspaceRoutes";
import { useProjectV1Store } from "@/store";
import { TabItem } from "./TabItem";

/**
 * Left gutter of the SQL Editor aside panel. Shows the Bytebase logo at
 * the top and 4 tab buttons (WORKSHEET, SCHEMA, HISTORY, and optionally
 * ACCESS when the current project allows JIT).
 *
 * Replaces frontend/src/views/sql-editor/AsidePanel/GutterBar/GutterBar.vue.
 */
export function GutterBar() {
  const projectStore = useProjectV1Store();
  const setAsidePanelTab = useSQLEditorStore((s) => s.setAsidePanelTab);
  const projectName = useSQLEditorEditorState((s) => s.project);

  const project = usePiniaBridge(() => {
    return projectName ? projectStore.getProjectByName(projectName) : undefined;
  });

  const routeProjectParam = usePiniaBridge(
    () => router.currentRoute.value.params.project as string | undefined
  );

  const logoHref = routeProjectParam
    ? router.resolve({
        name: PROJECT_V1_ROUTE_DETAIL,
        params: { projectId: routeProjectParam },
      }).href
    : router.resolve({ name: WORKSPACE_ROUTE_LANDING }).href;

  const handleClickTab = (target: AsidePanelTab) => {
    setAsidePanelTab(target);
  };

  return (
    <div className="h-full flex flex-col items-stretch justify-between overflow-hidden text-sm p-1">
      <div className="flex flex-col gap-y-1">
        <div className="flex flex-col justify-center items-center pb-1">
          <a href={logoHref} target="_blank" rel="noopener noreferrer">
            <img className="w-9 h-auto" src={logoIcon} alt="Bytebase" />
          </a>
        </div>
        <Separator />
        <TabItem tab="WORKSHEET" onClick={() => handleClickTab("WORKSHEET")} />
        <TabItem tab="SCHEMA" onClick={() => handleClickTab("SCHEMA")} />
        <TabItem tab="HISTORY" onClick={() => handleClickTab("HISTORY")} />
        {project?.allowJustInTimeAccess && (
          <TabItem tab="ACCESS" onClick={() => handleClickTab("ACCESS")} />
        )}
      </div>
      <div className="flex flex-col justify-end items-center" />
    </div>
  );
}
