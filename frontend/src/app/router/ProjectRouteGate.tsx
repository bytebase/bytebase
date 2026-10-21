import { LoaderCircle } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { Outlet, useParams } from "react-router";
import {
  PROJECT_V1_ROUTE_DETAIL,
  useCurrentRoute,
  useNavigate,
  WORKSPACE_ROUTE_LANDING,
} from "@/app/router";
import { PROJECT_V1_ROUTE_ISSUES } from "@/app/router/handles";
import {
  PermissionDeniedFallback,
  useComponentPermissionState,
  usePermissionDataReady,
} from "@/components/ComponentPermissionGuard";
import { useNotify } from "@/hooks/useAppState";
import { projectResourceNameFromId, useAppStore } from "@/stores/app";
import { isSaaSOrDev } from "@/utils/featureGates";

// `<Outlet/>`-aware permission gate for `/projects/:projectId/*`. Project-scoped
// permission checks need the loaded `Project` resource, so this loads the
// project (and its IAM policy, via `usePermissionDataReady`) before gating the
// routed leaf on `route.requiredPermissions` — the aggregated parent
// (`bb.projects.get`) + leaf permission lists from the route handles. A user
// lacking a route permission sees the request-role / permission-denied fallback
// instead of mounting the page and failing on individual API calls.
export function ProjectRouteGate() {
  const params = useParams();
  const projectId = params.projectId ?? "";
  const projectName = projectResourceNameFromId(projectId);
  const route = useCurrentRoute();
  const navigate = useNavigate();
  const navigateRef = useRef(navigate);
  navigateRef.current = navigate;
  const notify = useNotify();

  const project = useAppStore((state) => state.projectsByName[projectName]);
  const fetchProject = useAppStore((state) => state.fetchProject);
  const loadCurrentUser = useAppStore((state) => state.loadCurrentUser);
  const loadServerInfo = useAppStore((state) => state.loadServerInfo);
  const setRecentProject = useAppStore((state) => state.setRecentProject);
  const removeRecentVisit = useAppStore((state) => state.removeRecentVisit);
  const [loadedProjectName, setLoadedProjectName] = useState("");
  const [projectLandingEnabled, setProjectLandingEnabled] = useState<
    boolean | undefined
  >();

  useEffect(() => {
    let stale = false;
    setLoadedProjectName("");
    setProjectLandingEnabled(undefined);

    const load = async () => {
      await Promise.all([loadCurrentUser(), loadServerInfo()]);
      const landingEnabled = isSaaSOrDev(useAppStore.getState().isSaaSMode());
      const nextProject = await fetchProject(projectName);
      if (stale) return;
      if (!nextProject) {
        // Project missing or not accessible (e.g. no `bb.projects.get`): drop
        // it from recent visits and bounce to landing.
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
      if (!stale) {
        setProjectLandingEnabled(landingEnabled);
        setLoadedProjectName(projectName);
      }
    };

    void load();
    return () => {
      stale = true;
    };
  }, [
    fetchProject,
    loadCurrentUser,
    loadServerInfo,
    notify,
    projectId,
    projectName,
    removeRecentVisit,
    setRecentProject,
  ]);

  const permissionReady = usePermissionDataReady(project);
  const { missedBasicPermissions, missedPermissions, permitted } =
    useComponentPermissionState({
      permissions: route.requiredPermissions,
      project,
      checkBasicWorkspacePermissions: true,
    });

  const loaded = loadedProjectName === projectName;
  const shouldRedirectProjectLanding =
    loaded &&
    projectLandingEnabled === false &&
    route.name === PROJECT_V1_ROUTE_DETAIL;

  useEffect(() => {
    if (!shouldRedirectProjectLanding) return;
    void navigateRef.current.replace({
      name: PROJECT_V1_ROUTE_ISSUES,
      params: { projectId },
    });
  }, [projectId, shouldRedirectProjectLanding]);

  if (
    !loaded ||
    projectLandingEnabled === undefined ||
    shouldRedirectProjectLanding ||
    !project ||
    !permissionReady
  ) {
    return (
      <div className="flex h-full min-h-0 items-center justify-center">
        <LoaderCircle className="size-5 animate-spin text-control-light" />
      </div>
    );
  }

  if (!permitted) {
    return (
      <PermissionDeniedFallback
        missedBasicPermissions={missedBasicPermissions}
        missedPermissions={missedPermissions}
        project={project}
        className="m-4"
        path={route.fullPath}
        enableRequestRole
      />
    );
  }

  return <Outlet />;
}
