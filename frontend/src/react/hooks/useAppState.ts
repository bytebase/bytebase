import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  getProjectResourceId,
  isConnectAlreadyExists,
  isDefaultProjectName,
  useAppStore,
} from "@/react/stores/app";
import type { AppFeatures } from "@/types/appProfile";
import type { Permission } from "@/types/iam/permission";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { storageKeyRecentProjects } from "@/utils/storage-keys";

export { isConnectAlreadyExists };

export function useCurrentUser() {
  const user = useAppStore((state) => state.currentUser);
  const loadCurrentUser = useAppStore((state) => state.loadCurrentUser);
  useEffect(() => {
    void loadCurrentUser();
  }, [loadCurrentUser]);
  return user;
}

export function useWorkspace() {
  const workspace = useAppStore((state) => state.workspace);
  const loadWorkspace = useAppStore((state) => state.loadWorkspace);
  useEffect(() => {
    void loadWorkspace();
  }, [loadWorkspace]);
  return workspace;
}

export function useSubscription() {
  const subscription = useAppStore((state) => state.subscription);
  const loadSubscription = useAppStore((state) => state.loadSubscription);
  const uploadLicense = useAppStore((state) => state.uploadLicense);
  useEffect(() => {
    void loadSubscription();
  }, [loadSubscription]);
  return { subscription, uploadLicense };
}

export function useServerInfo() {
  const serverInfo = useAppStore((state) => state.serverInfo);
  const loadServerInfo = useAppStore((state) => state.loadServerInfo);
  useEffect(() => {
    void loadServerInfo();
  }, [loadServerInfo]);
  return serverInfo;
}

export function useIsSaaSMode() {
  return useServerInfo()?.saas ?? false;
}

export function useAppFeature<T extends keyof AppFeatures>(feature: T) {
  const value = useAppStore((state) => state.appFeatures[feature]);
  const loadWorkspaceProfile = useAppStore(
    (state) => state.loadWorkspaceProfile
  );
  useEffect(() => {
    void loadWorkspaceProfile();
  }, [loadWorkspaceProfile]);
  return value;
}

export function useWorkspacePermission(permission: Permission) {
  const allowed = useAppStore((state) =>
    state.hasWorkspacePermission(permission)
  );
  const loadWorkspacePermissionState = useAppStore(
    (state) => state.loadWorkspacePermissionState
  );
  useEffect(() => {
    void loadWorkspacePermissionState();
  }, [loadWorkspacePermissionState]);
  return allowed;
}

export function useProject(name: string | undefined) {
  const project = useAppStore((state) =>
    name ? state.projectsByName[name] : undefined
  );
  const fetchProject = useAppStore((state) => state.fetchProject);
  useEffect(() => {
    if (name) {
      void fetchProject(name);
    }
  }, [fetchProject, name]);
  return project;
}

export function useProjectList(query: string) {
  const searchProjects = useAppStore((state) => state.searchProjects);
  const [projects, setProjects] = useState<Project[]>([]);
  const [pageToken, setPageToken] = useState("");
  const pageTokenRef = useRef("");
  const [isLoading, setIsLoading] = useState(true);
  const [isFetchingMore, setIsFetchingMore] = useState(false);
  const requestIdRef = useRef(0);
  const [pageSize, setPageSize] = useState(() => {
    const value = Number(localStorage.getItem("bb.project-switch.page-size"));
    return [50, 100, 200, 500].includes(value) ? value : 50;
  });

  const fetchPage = useCallback(
    async (mode: "refresh" | "append") => {
      if (mode === "refresh") {
        setIsLoading(true);
      } else {
        setIsFetchingMore(true);
      }
      const requestId = ++requestIdRef.current;
      try {
        const response = await searchProjects({
          pageSize,
          pageToken: mode === "refresh" ? "" : pageTokenRef.current,
          query,
        });
        if (requestId !== requestIdRef.current) {
          return;
        }
        pageTokenRef.current = response.nextPageToken ?? "";
        setPageToken(response.nextPageToken ?? "");
        setProjects((previous) =>
          mode === "refresh"
            ? response.projects
            : [...previous, ...response.projects]
        );
      } finally {
        if (requestId === requestIdRef.current) {
          if (mode === "refresh") {
            setIsLoading(false);
          } else {
            setIsFetchingMore(false);
          }
        }
      }
    },
    [pageSize, query, searchProjects]
  );

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void fetchPage("refresh");
    }, 200);
    return () => window.clearTimeout(timer);
  }, [fetchPage]);

  const onPageSizeChange = useCallback((next: number) => {
    localStorage.setItem("bb.project-switch.page-size", String(next));
    setPageSize(next);
  }, []);

  return {
    projects,
    isLoading,
    isFetchingMore,
    hasMore: Boolean(pageToken),
    loadMore: () => void fetchPage("append"),
    pageSize,
    pageSizeOptions: [50, 100, 200, 500],
    onPageSizeChange,
  };
}

function readRecentProjectNames(email: string) {
  if (!email) return [];
  try {
    const raw = localStorage.getItem(storageKeyRecentProjects(email));
    const parsed = raw ? JSON.parse(raw) : [];
    return Array.isArray(parsed)
      ? parsed.filter((name): name is string => typeof name === "string")
      : [];
  } catch {
    return [];
  }
}

export function useRecentProjects() {
  const currentUser = useCurrentUser();
  const batchFetchProjects = useAppStore((state) => state.batchFetchProjects);
  const projectsByName = useAppStore((state) => state.projectsByName);
  const [projectNames, setProjectNames] = useState<string[]>([]);

  const refreshNames = useCallback(() => {
    setProjectNames(readRecentProjectNames(currentUser?.email ?? ""));
  }, [currentUser?.email]);

  useEffect(() => {
    refreshNames();
  }, [refreshNames]);

  useEffect(() => {
    if (projectNames.length > 0) {
      void batchFetchProjects(projectNames);
    }
  }, [batchFetchProjects, projectNames]);

  const projects = useMemo(() => {
    return projectNames
      .map((name) => projectsByName[name])
      .filter((project): project is Project => Boolean(project))
      .filter((project) => !isDefaultProjectName(project.name));
  }, [projectNames, projectsByName]);

  return { projects, refresh: refreshNames };
}

export function useRecentVisit() {
  useCurrentUser();
  const recordRecentVisit = useAppStore((state) => state.recordRecentVisit);
  return {
    record: recordRecentVisit,
  };
}

export function useNotify() {
  return useAppStore((state) => state.notify);
}

export function useQuickstartReset() {
  return useAppStore((state) => state.resetQuickstartProgress);
}

export function useCreateProject() {
  const createProject = useAppStore((state) => state.createProject);
  const setRecentProject = useAppStore((state) => state.setRecentProject);
  return {
    createProject,
    setRecentProject,
  };
}

export function projectMatchesKeyword(project: Project, keyword: string) {
  const normalized = keyword.trim().toLowerCase();
  if (!normalized) return true;
  return (
    project.title.toLowerCase().includes(normalized) ||
    project.name.toLowerCase().includes(normalized) ||
    getProjectResourceId(project).toLowerCase().includes(normalized)
  );
}
