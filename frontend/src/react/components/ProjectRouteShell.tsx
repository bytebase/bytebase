import { LoaderCircle } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { IAMRemindDialog } from "@/react/components/IAMRemindDialog";
import { RoutePermissionGuardShell } from "@/react/components/RoutePermissionGuardShell";
import { Alert, AlertDescription } from "@/react/components/ui/alert";
import type { ReactRouteShellTargets } from "@/react/dashboard-shell";
import { useNotify } from "@/react/hooks/useAppState";
import {
  PROJECT_V1_ROUTE_DETAIL,
  useCurrentRoute,
  useNavigate,
  WORKSPACE_ROUTE_LANDING,
} from "@/react/router";
import { projectResourceNameFromId, useAppStore } from "@/react/stores/app";
import { State } from "@/types/proto-es/v1/common_pb";
import { setDocumentTitle } from "@/utils";

interface ProjectRouteShellProps {
  projectId: string;
  routeKey?: string;
  onReady?: (targets: ReactRouteShellTargets | null) => void;
}

export function ProjectRouteShell({
  projectId,
  routeKey,
  onReady,
}: ProjectRouteShellProps) {
  const { t } = useTranslation();
  const route = useCurrentRoute();
  const navigate = useNavigate();
  const navigateRef = useRef(navigate);
  navigateRef.current = navigate;
  const notify = useNotify();
  const projectName = projectResourceNameFromId(projectId);
  const project = useAppStore((state) => state.projectsByName[projectName]);
  const defaultProjectName = useAppStore(
    (state) => state.serverInfo?.defaultProject ?? ""
  );
  const allowEdit = useAppStore((state) =>
    project
      ? project.state !== State.DELETED &&
        state.hasProjectPermission(project, "bb.projects.update")
      : false
  );
  const fetchProject = useAppStore((state) => state.fetchProject);
  const loadServerInfo = useAppStore((state) => state.loadServerInfo);
  const loadCurrentUser = useAppStore((state) => state.loadCurrentUser);
  const loadSubscription = useAppStore((state) => state.loadSubscription);
  const loadProjectIamPolicy = useAppStore(
    (state) => state.loadProjectIamPolicy
  );
  const setRecentProject = useAppStore((state) => state.setRecentProject);
  const removeRecentVisit = useAppStore((state) => state.removeRecentVisit);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    let stale = false;
    setReady(false);
    onReady?.(null);

    const load = async () => {
      await Promise.all([
        loadCurrentUser(),
        loadServerInfo(),
        loadSubscription(),
      ]);
      const nextProject = await fetchProject(projectName);
      if (stale) return;
      if (!nextProject) {
        const projectRoute = navigateRef.current.resolve({
          name: PROJECT_V1_ROUTE_DETAIL,
          params: { projectId },
        });
        removeRecentVisit(projectRoute.fullPath);
        const error = useAppStore.getState().projectErrorsByName[projectName];
        if (error) {
          notify({
            module: "bytebase",
            style: "CRITICAL",
            title: `Failed to fetch project ${projectId}`,
            description: error.message,
          });
        }
        void navigateRef.current.replace({ name: WORKSPACE_ROUTE_LANDING });
        return;
      }
      setRecentProject(nextProject.name);
      await loadProjectIamPolicy(nextProject.name);
      if (!stale) {
        setReady(true);
      }
    };

    void load();
    return () => {
      stale = true;
      onReady?.(null);
    };
  }, [
    fetchProject,
    loadCurrentUser,
    loadProjectIamPolicy,
    loadServerInfo,
    loadSubscription,
    notify,
    onReady,
    projectId,
    projectName,
    removeRecentVisit,
    setRecentProject,
  ]);

  useEffect(() => {
    if (!ready || !project) return;
    if (route.title) {
      setDocumentTitle(route.title, project.title);
    } else {
      setDocumentTitle(project.title);
    }
  }, [project, ready, route.title]);

  const routeProps = useMemo(
    () => ({
      projectId,
      allowEdit,
    }),
    [allowEdit, projectId]
  );

  if (!ready || !project) {
    return (
      <div className="fixed inset-0 bg-white flex flex-col items-center justify-center">
        <LoaderCircle className="size-5 animate-spin text-control-light" />
      </div>
    );
  }

  const isDefaultProject = project.name === defaultProjectName;

  return (
    <>
      {project.state === State.DELETED && (
        <div className="h-8 w-full text-base font-medium bg-control text-white flex justify-center items-center py-2 mb-4">
          {t("common.archived")}
        </div>
      )}
      {isDefaultProject && (
        <div className="m-4">
          <h1 className="mb-4 text-xl font-bold leading-6 text-main truncate">
            {t("database.unassigned-databases")}
          </h1>
          <Alert variant="info" className="mb-4">
            <AlertDescription>
              {t("project.overview.info-slot-content")}
            </AlertDescription>
          </Alert>
        </div>
      )}
      <RoutePermissionGuardShell
        project={project}
        routeKey={routeKey}
        className="m-4"
        targetClassName="h-full min-h-0"
        onReady={(target) =>
          onReady?.({
            content: target,
            routeProps,
          })
        }
      />
      <IAMRemindDialog project={project} />
    </>
  );
}
