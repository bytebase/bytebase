import {
  PROJECT_V1_ROUTE_DETAIL,
  WORKSPACE_ROUTE_LANDING,
} from "@/app/router/handles";
import logoIcon from "@/assets/logo-icon.svg";
import { RouterLink } from "@/components/RouterLink";
import { Separator } from "@/components/ui/separator";
import { useAppProject } from "@/hooks/useAppProject";
import { useReactiveRoute } from "@/hooks/useReactiveRoute";
import type { AsidePanelTab } from "@/modules/sql-editor/store";
import { useSQLEditorStore } from "@/modules/sql-editor/store";
import { useSQLEditorEditorState } from "@/modules/sql-editor/store/editor";
import { TabItem } from "./TabItem";

/**
 * Left gutter of the SQL Editor aside panel. Shows the Bytebase logo at
 * the top and 4 tab buttons (WORKSHEET, SCHEMA, HISTORY, and optionally
 * ACCESS when the current project allows JIT).
 *
 * Replaces frontend/src/views/sql-editor/AsidePanel/GutterBar/GutterBar.vue.
 */
export function GutterBar() {
  const setAsidePanelTab = useSQLEditorStore((s) => s.setAsidePanelTab);
  const projectName = useSQLEditorEditorState((s) => s.project);

  const resolvedProject = useAppProject(projectName);
  const project = projectName ? resolvedProject : undefined;

  const routeProjectParam = useReactiveRoute().params.project as
    | string
    | undefined;

  const logoRoute = routeProjectParam
    ? {
        name: PROJECT_V1_ROUTE_DETAIL,
        params: { projectId: routeProjectParam },
      }
    : { name: WORKSPACE_ROUTE_LANDING };

  const handleClickTab = (target: AsidePanelTab) => {
    setAsidePanelTab(target);
  };

  return (
    <div className="h-full flex flex-col items-stretch justify-between overflow-hidden text-sm p-1">
      <div className="flex flex-col gap-y-1">
        <div className="flex flex-col justify-center items-center pb-1">
          <RouterLink to={logoRoute} target="_blank" rel="noopener noreferrer">
            <img className="w-9 h-auto" src={logoIcon} alt="Bytebase" />
          </RouterLink>
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
