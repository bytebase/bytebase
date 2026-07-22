import { useCallback, useEffect } from "react";
import { useTranslation } from "react-i18next";
import {
  SQL_EDITOR_PROJECT_MODULE,
  useNavigate,
  WORKSPACE_ROUTE_LANDING,
} from "@/app/router";
import { BytebaseLogo } from "@/components/BytebaseLogo";
import { HeaderBreadcrumb } from "@/components/header/HeaderBreadcrumb";
import { ProfileMenuTrigger } from "@/components/header/ProfileMenuTrigger";
import { useRecentVisit } from "@/hooks/useAppState";
import { getProjectName, isValidProjectName } from "@/lib/resourceName";
import { useSQLEditorStore } from "@/modules/sql-editor/store";
import { useSQLEditorEditorState } from "@/modules/sql-editor/store/editor";
import { getSQLEditorTabsState } from "@/modules/sql-editor/store/tab";
import { useAppStore } from "@/stores/app";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { defaultProject } from "@/types/v1/project";
import { isDarkTheme } from "./theme/derive";
import { useSQLEditorTheme } from "./theme/SQLEditorThemeScope";

export function SQLEditorHeader() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { record } = useRecentVisit();
  const theme = useSQLEditorTheme();
  const projectName = useSQLEditorEditorState((s) => s.project);
  const maybeSwitchProject = useSQLEditorStore((s) => s.maybeSwitchProject);
  const setRecentProject = useAppStore((s) => s.setRecentProject);
  const loadWorkspacePermissionState = useAppStore(
    (s) => s.loadWorkspacePermissionState
  );
  const defaultProjectName = useAppStore(
    (s) => s.serverInfo?.defaultProject ?? ""
  );
  const allowAccessDefaultProject = useAppStore((s) =>
    defaultProjectName
      ? s.hasProjectPermission(
          defaultProject(defaultProjectName),
          "bb.projects.get"
        )
      : false
  );
  const projectId = isValidProjectName(projectName)
    ? getProjectName(projectName)
    : undefined;

  useEffect(() => {
    void loadWorkspacePermissionState();
  }, [loadWorkspacePermissionState]);

  const handleSelectProject = useCallback(
    (project: Project, event: React.MouseEvent<HTMLElement>) => {
      const route = navigate.resolve({
        name: SQL_EDITOR_PROJECT_MODULE,
        params: {
          project: getProjectName(project.name),
        },
      });
      record(route.fullPath);
      setRecentProject(project.name);

      if (event.ctrlKey || event.metaKey) {
        window.open(route.fullPath, "_blank");
      } else {
        void maybeSwitchProject(project.name);
      }
    },
    [maybeSwitchProject, navigate, record, setRecentProject]
  );

  const handleBeforeSwitchWorkspace = useCallback(() => {
    const tabsState = getSQLEditorTabsState();
    for (const persisted of tabsState.openTmpTabList) {
      const tab = tabsState.tabsById.get(persisted.id);
      if (tab && tab.status !== "CLEAN") {
        return window.confirm(
          `${t("sql-editor.tab.unsaved-worksheet")} ${t("common.leave-without-saving")}`
        );
      }
    }
    return true;
  }, [t]);

  return (
    <header className="h-12 shrink-0 border-b border-block-border bg-background px-3 flex items-center justify-between gap-x-4">
      <div className="min-w-0 flex items-center gap-x-4">
        <BytebaseLogo
          redirect={WORKSPACE_ROUTE_LANDING}
          builtinTheme={isDarkTheme(theme) ? "dark" : "light"}
          className="h-9 md:h-10"
        />
        <HeaderBreadcrumb
          projectId={projectId}
          currentProjectName={projectName}
          projectSwitchExcludeDefaultProject={!allowAccessDefaultProject}
          onBeforeSwitchWorkspace={handleBeforeSwitchWorkspace}
          onSelectProject={handleSelectProject}
        />
      </div>
      <ProfileMenuTrigger size="medium" link />
    </header>
  );
}
