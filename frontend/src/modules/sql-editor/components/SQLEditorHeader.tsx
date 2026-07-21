import { useCallback } from "react";
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
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { isDarkTheme } from "./theme/derive";
import { useSQLEditorTheme } from "./theme/SQLEditorThemeScope";

export function SQLEditorHeader() {
  const navigate = useNavigate();
  const { record } = useRecentVisit();
  const theme = useSQLEditorTheme();
  const projectName = useSQLEditorEditorState((s) => s.project);
  const maybeSwitchProject = useSQLEditorStore((s) => s.maybeSwitchProject);
  const projectId = isValidProjectName(projectName)
    ? getProjectName(projectName)
    : undefined;

  const handleSelectProject = useCallback(
    (project: Project, event: React.MouseEvent<HTMLElement>) => {
      const route = navigate.resolve({
        name: SQL_EDITOR_PROJECT_MODULE,
        params: {
          project: getProjectName(project.name),
        },
      });
      record(route.fullPath);

      if (event.ctrlKey || event.metaKey) {
        window.open(route.fullPath, "_blank");
      } else {
        void maybeSwitchProject(project.name);
      }
    },
    [maybeSwitchProject, navigate, record]
  );

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
          onSelectProject={handleSelectProject}
        />
      </div>
      <ProfileMenuTrigger size="medium" link />
    </header>
  );
}
