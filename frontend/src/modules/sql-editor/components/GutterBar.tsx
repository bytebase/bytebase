import { useEffect } from "react";
import { useAppProject } from "@/hooks/useAppProject";
import type { AsidePanelTab } from "@/modules/sql-editor/store";
import { useSQLEditorStore } from "@/modules/sql-editor/store";
import { useSQLEditorEditorState } from "@/modules/sql-editor/store/editor";
import { TabItem } from "./TabItem";

/**
 * Left gutter of the SQL Editor aside panel. Shows 4 tab buttons
 * (SAVED_QUERY, SCHEMA, HISTORY, and optionally ACCESS when the current
 * project allows JIT).
 *
 * Replaces frontend/src/views/sql-editor/AsidePanel/GutterBar/GutterBar.vue.
 */
export function GutterBar() {
  const asidePanelTab = useSQLEditorStore((s) => s.asidePanelTab);
  const setAsidePanelTab = useSQLEditorStore((s) => s.setAsidePanelTab);
  const projectName = useSQLEditorEditorState((s) => s.project);

  const resolvedProject = useAppProject(projectName);
  const project = projectName ? resolvedProject : undefined;

  useEffect(() => {
    if (asidePanelTab === "ACCESS" && !project?.allowJustInTimeAccess) {
      setAsidePanelTab("SAVED_QUERY");
    }
  }, [asidePanelTab, project?.allowJustInTimeAccess, setAsidePanelTab]);

  const handleClickTab = (target: AsidePanelTab) => {
    setAsidePanelTab(target);
  };

  return (
    <div className="h-full flex flex-col items-stretch justify-between overflow-hidden text-sm p-1">
      <div className="flex flex-col gap-y-1">
        <TabItem
          tab="SAVED_QUERY"
          onClick={() => handleClickTab("SAVED_QUERY")}
        />
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
