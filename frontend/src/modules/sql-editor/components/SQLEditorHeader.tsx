import { useCallback, useEffect } from "react";
import { useTranslation } from "react-i18next";
import {
  SQL_EDITOR_HOME_MODULE,
  SQL_EDITOR_PROJECT_MODULE,
  useNavigate,
} from "@/app/router";
import { BytebaseLogo } from "@/components/BytebaseLogo";
import { HeaderBreadcrumb } from "@/components/header/HeaderBreadcrumb";
import { ProfileMenuTrigger } from "@/components/header/ProfileMenuTrigger";
import {
  useRecentVisit,
  useSwitchWorkspace,
  useWorkspace,
} from "@/hooks/useAppState";
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
  const workspace = useWorkspace();
  const switchWorkspace = useSwitchWorkspace();
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
        window.alert(t("sql-editor.tab.unsaved-worksheet"));
        return false;
      }
    }
    return true;
  }, [t]);

  const handleSelectWorkspace = useCallback(
    (workspaceName: string, event: React.MouseEvent<HTMLElement>) => {
      const route = navigate.resolve({
        name: SQL_EDITOR_HOME_MODULE,
      });
      record(route.fullPath);

      if (workspaceName === workspace?.name) {
        if (event.ctrlKey || event.metaKey) {
          window.open(route.fullPath, "_blank");
        } else {
          void navigate.push({ name: SQL_EDITOR_HOME_MODULE });
        }
        return;
      }

      void switchWorkspace(workspaceName, false).then(() => {
        globalThis.location.assign(route.fullPath);
      });
    },
    [navigate, record, switchWorkspace, workspace?.name]
  );

  return (
    <header className="h-12 shrink-0 border-b border-block-border bg-background px-3 flex items-center justify-between gap-x-4">
      <div className="min-w-0 flex items-center gap-x-4">
        <BytebaseLogo
          builtinTheme={isDarkTheme(theme) ? "dark" : "light"}
          className="h-9 md:h-10"
        />
        <HeaderBreadcrumb
          projectId={projectId}
          currentProjectName={projectName}
          projectSwitchExcludeDefaultProject={!allowAccessDefaultProject}
          onBeforeSwitchWorkspace={handleBeforeSwitchWorkspace}
          onSelectWorkspace={handleSelectWorkspace}
          onSelectProject={handleSelectProject}
        />
      </div>
      <ProfileMenuTrigger size="medium" link />
    </header>
  );
}
