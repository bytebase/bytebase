import { useAppProject } from "@/hooks/useAppProject";
import type { AsidePanelTab } from "@/modules/sql-editor/store";
import { useSQLEditorStore } from "@/modules/sql-editor/store";
import { useSQLEditorEditorState } from "@/modules/sql-editor/store/editor";
import { TabItem } from "./TabItem";

/**
 * Left gutter of the SQL Editor aside panel. Shows 4 tab buttons
 * (WORKSHEET, SCHEMA, HISTORY, and optionally ACCESS when the current
 * project allows JIT).
 *
 * Replaces frontend/src/views/sql-editor/AsidePanel/GutterBar/GutterBar.vue.
 */
export function GutterBar() {
  const setAsidePanelTab = useSQLEditorStore((s) => s.setAsidePanelTab);
  const projectName = useSQLEditorEditorState((s) => s.project);

  const resolvedProject = useAppProject(projectName);
  const project = projectName ? resolvedProject : undefined;

  const handleClickTab = (target: AsidePanelTab) => {
    setAsidePanelTab(target);
  };

  return (
    <div className="h-full flex flex-col items-stretch justify-between overflow-hidden text-sm p-1">
      <div className="flex flex-col gap-y-1">
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
