import { LayersIcon, LinkIcon } from "lucide-react";
import { type MouseEvent as ReactMouseEvent, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { router, SQL_EDITOR_PROJECT_MODULE, useNavigate } from "@/app/router";
import {
  PROJECT_V1_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_INSTANCE_CREATE,
} from "@/app/router/handles";
import { BytebaseLogo } from "@/components/BytebaseLogo";
import { ProjectSwitchPanel } from "@/components/header/ProjectSwitchPanel";
import {
  PermissionGuard,
  usePermissionCheck,
} from "@/components/PermissionGuard";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { useAppProject } from "@/hooks/useAppProject";
import { useProjectList } from "@/hooks/useAppState";
import {
  CREATE_PROJECT_PRODUCT_INTRO,
  PRODUCT_INTRO_QUERY_KEY,
} from "@/lib/productIntro";
import { useSQLEditorStore } from "@/modules/sql-editor/store";
import { useSQLEditorEditorState } from "@/modules/sql-editor/store/editor";
import { useAppStore } from "@/stores/app";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { extractProjectResourceName } from "@/utils/v1/project";
import { isDarkTheme } from "./theme/derive";
import { useSQLEditorTheme } from "./theme/SQLEditorThemeScope";
import { WelcomeButton } from "./WelcomeButton";

export type WelcomeProps = {
  /**
   * Called when the user clicks "Connect to a database". Vue parent
   * passes a callback that sets `asidePanelTab = "SCHEMA"` and
   * `showConnectionPanel = true` on the Vue-side SQL Editor context.
   */
  readonly onChangeConnection: () => void;
};

export function Welcome({ onChangeConnection }: WelcomeProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const theme = useSQLEditorTheme();
  const projectName = useSQLEditorEditorState((s) => s.project);
  const defaultProjectName = useAppStore(
    (s) => s.serverInfo?.defaultProject ?? ""
  );
  const maybeSwitchProject = useSQLEditorStore((s) => s.maybeSwitchProject);
  const { projects, isLoading: isLoadingProjects } = useProjectList("", {
    excludeDefault: true,
  });
  const [isProjectSelectorOpen, setProjectSelectorOpen] = useState(false);
  const [databaseState, setDatabaseState] = useState<
    "loading" | "empty" | "available" | "unavailable"
  >("loading");

  const resolvedProject = useAppProject(projectName);
  const project = projectName ? resolvedProject : undefined;
  const isDefaultProject = project?.name === defaultProjectName;

  const [canCreateProject] = usePermissionCheck(["bb.projects.create"]);
  const [canCreateInstance] = usePermissionCheck(
    ["bb.instances.create"],
    project
  );
  const [canConnect] = usePermissionCheck(["bb.sql.select"], project);

  useEffect(() => {
    if (!project?.name) return;

    let cancelled = false;
    setDatabaseState("loading");
    void useAppStore
      .getState()
      .fetchDatabases({
        parent: project.name,
        pageSize: 1,
        silent: true,
      })
      .then(({ databases }) => {
        if (!cancelled) {
          setDatabaseState(databases.length > 0 ? "available" : "empty");
        }
      })
      .catch(() => {
        if (!cancelled) {
          setDatabaseState("unavailable");
        }
      });

    return () => {
      cancelled = true;
    };
  }, [project?.name]);

  const handleCreateProject = () => {
    setProjectSelectorOpen(false);
    router.push({
      name: PROJECT_V1_ROUTE_DASHBOARD,
      query: {
        [PRODUCT_INTRO_QUERY_KEY]: CREATE_PROJECT_PRODUCT_INTRO,
      },
    });
  };

  const handleCreateInstance = () => {
    if (!project?.name) return;
    router.push({
      name: PROJECT_V1_ROUTE_INSTANCE_CREATE,
      params: {
        projectId: extractProjectResourceName(project.name),
      },
    });
  };

  const handleSelectProject = async (
    selectedProject: Project,
    event: ReactMouseEvent<HTMLElement>
  ) => {
    const route = navigate.resolve({
      name: SQL_EDITOR_PROJECT_MODULE,
      params: {
        project: extractProjectResourceName(selectedProject.name),
      },
    });
    if (event.ctrlKey || event.metaKey) {
      window.open(route.fullPath, "_blank");
      return;
    }
    const switchedProject = await maybeSwitchProject(selectedProject.name);
    if (!switchedProject) return;
    await navigate.push(route);
  };

  const createProjectAction = (
    <PermissionGuard permissions={["bb.projects.create"]}>
      <WelcomeButton
        data-testid="create-project"
        disabled={!canCreateProject}
        icon={<LayersIcon className="size-5 text-control-light" />}
        onClick={handleCreateProject}
      >
        {t("quick-action.new-project")}
      </WelcomeButton>
    </PermissionGuard>
  );

  const createInstanceAction = (
    <PermissionGuard permissions={["bb.instances.create"]} project={project}>
      <WelcomeButton
        data-testid="create-project-instance"
        disabled={!canCreateInstance}
        icon={<LayersIcon className="size-5 text-control-light" />}
        onClick={handleCreateInstance}
      >
        {t("sql-editor.add-a-new-instance")}
      </WelcomeButton>
    </PermissionGuard>
  );

  const connectDatabaseAction = (
    <PermissionGuard permissions={["bb.sql.select"]} project={project}>
      <WelcomeButton
        data-testid="connect-database"
        disabled={!canConnect}
        icon={<LinkIcon className="size-5 text-control-light" />}
        onClick={onChangeConnection}
      >
        {t("sql-editor.connect-to-a-database")}
      </WelcomeButton>
    </PermissionGuard>
  );

  return (
    <div className="w-full h-full flex flex-col items-center justify-center gap-y-8">
      <BytebaseLogo
        builtinTheme={isDarkTheme(theme) ? "dark" : "light"}
        className="h-20 w-44"
      />
      <div className="flex items-center flex-wrap gap-4">
        {!projectName && !isLoadingProjects && projects.length === 0
          ? createProjectAction
          : null}
        {!projectName && projects.length > 0 ? (
          <Popover
            open={isProjectSelectorOpen}
            onOpenChange={setProjectSelectorOpen}
          >
            <PopoverTrigger
              render={
                <WelcomeButton
                  data-testid="select-project"
                  icon={<LayersIcon className="size-5 text-control-light" />}
                >
                  {t("project.select")}
                </WelcomeButton>
              }
            />
            <PopoverContent
              align="center"
              sideOffset={8}
              className="w-[24rem] max-w-[calc(100vw-2rem)] p-0! py-3!"
            >
              <ProjectSwitchPanel
                currentProjectName={projectName}
                excludeDefaultProject
                onClose={() => setProjectSelectorOpen(false)}
                onRequestCreate={handleCreateProject}
                onSelectProject={(selectedProject, event) => {
                  void handleSelectProject(selectedProject, event);
                }}
              />
            </PopoverContent>
          </Popover>
        ) : null}
        {project && !isDefaultProject && databaseState !== "loading"
          ? createInstanceAction
          : null}
        {project &&
        (databaseState === "available" || databaseState === "unavailable")
          ? connectDatabaseAction
          : null}
      </div>
    </div>
  );
}
