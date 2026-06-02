import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useShallow } from "zustand/react/shallow";
import {
  getProjectResourceId,
  isConnectAlreadyExists,
  isDefaultProjectName,
  useAppStore,
} from "@/react/stores/app";
import type { AppFeatures } from "@/types/appProfile";
import type { Permission } from "@/types/iam/permission";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  isValidEnvironmentName,
  NULL_ENVIRONMENT_NAME,
  nullEnvironment,
  unknownEnvironment,
} from "@/types/v1/environment";
import { unknownUser } from "@/types/v1/user";
import {
  storageKeyRecentProjects,
  workspaceCacheScope,
} from "@/utils/storage-keys";

export { isConnectAlreadyExists };

export function useOptionalCurrentUser() {
  const user = useAppStore((state) => state.currentUser);
  const loadCurrentUser = useAppStore((state) => state.loadCurrentUser);
  useEffect(() => {
    void loadCurrentUser();
  }, [loadCurrentUser]);
  return user;
}

export function useCurrentUser() {
  const user = useOptionalCurrentUser();
  // Stabilize the fallback identity: a fresh unknownUser() each render would
  // change identity while the user is unresolved, retriggering identity-keyed
  // effects in callers (e.g. ProfilePage, TwoFactorSetupPage) and risking a
  // render loop.
  return useMemo(() => user ?? unknownUser(), [user]);
}

export function useWorkspace() {
  const workspace = useAppStore((state) => state.workspace);
  const loadWorkspace = useAppStore((state) => state.loadWorkspace);
  useEffect(() => {
    void loadWorkspace();
  }, [loadWorkspace]);
  return workspace;
}

export function useWorkspaceList() {
  const workspaceList = useAppStore((state) => state.workspaceList);
  const loadWorkspaceList = useAppStore((state) => state.loadWorkspaceList);
  useEffect(() => {
    void loadWorkspaceList();
  }, [loadWorkspaceList]);
  return workspaceList;
}

export function useSwitchWorkspace() {
  return useAppStore((state) => state.switchWorkspace);
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

export function useSubscriptionState() {
  const subscription = useAppStore((state) => state.subscription);
  const loadSubscription = useAppStore((state) => state.loadSubscription);
  const uploadLicense = useAppStore((state) => state.uploadLicense);
  const currentPlan = useAppStore((state) => state.currentPlan());
  const isFreePlan = useAppStore((state) => state.isFreePlan());
  const isTrialing = useAppStore((state) => state.isTrialing());
  const isExpired = useAppStore((state) => state.isExpired());
  const daysBeforeExpire = useAppStore((state) => state.daysBeforeExpire());
  const trialingDays = useAppStore((state) => state.trialingDays());
  const showTrial = useAppStore((state) => state.showTrial());
  const expireAt = useAppStore((state) => state.expireAt());
  const instanceCountLimit = useAppStore((state) => state.instanceCountLimit());
  const userCountLimit = useAppStore((state) => state.userCountLimit());
  const instanceLicenseCount = useAppStore((state) =>
    state.instanceLicenseCount()
  );
  const hasUnifiedInstanceLicense = useAppStore((state) =>
    state.hasUnifiedInstanceLicense()
  );
  useEffect(() => {
    void loadSubscription();
  }, [loadSubscription]);
  return {
    subscription,
    uploadLicense,
    currentPlan,
    isFreePlan,
    isTrialing,
    isExpired,
    daysBeforeExpire,
    trialingDays,
    showTrial,
    expireAt,
    instanceCountLimit,
    userCountLimit,
    instanceLicenseCount,
    hasUnifiedInstanceLicense,
  };
}

export function useServerInfo() {
  const serverInfo = useAppStore((state) => state.serverInfo);
  const loadServerInfo = useAppStore((state) => state.loadServerInfo);
  useEffect(() => {
    void loadServerInfo();
  }, [loadServerInfo]);
  return serverInfo;
}

export function useServerState() {
  const serverInfo = useAppStore((state) => state.serverInfo);
  const loadServerInfo = useAppStore((state) => state.loadServerInfo);
  const isSaaSMode = useAppStore((state) => state.isSaaSMode());
  const workspaceResourceName = useAppStore((state) =>
    state.workspaceResourceName()
  );
  const externalUrl = useAppStore((state) => state.externalUrl());
  const needConfigureExternalUrl = useAppStore((state) =>
    state.needConfigureExternalUrl()
  );
  const version = useAppStore((state) => state.version());
  const changelogURL = useAppStore((state) => state.changelogURL());
  const activatedInstanceCount = useAppStore((state) =>
    state.activatedInstanceCount()
  );
  const totalInstanceCount = useAppStore((state) => state.totalInstanceCount());
  const userCountInIam = useAppStore((state) => state.userCountInIam());
  const activeVcsUserCount = useAppStore((state) => state.activeVcsUserCount());
  useEffect(() => {
    void loadServerInfo();
  }, [loadServerInfo]);
  return {
    serverInfo,
    isSaaSMode,
    workspaceResourceName,
    externalUrl,
    needConfigureExternalUrl,
    version,
    changelogURL,
    activatedInstanceCount,
    totalInstanceCount,
    userCountInIam,
    activeVcsUserCount,
  };
}

export function useWorkspaceResourceName() {
  return useServerState().workspaceResourceName;
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

export function useWorkspaceProfile() {
  const workspaceProfile = useAppStore((state) => state.workspaceProfile);
  const loadWorkspaceProfile = useAppStore(
    (state) => state.loadWorkspaceProfile
  );
  useEffect(() => {
    void loadWorkspaceProfile();
  }, [loadWorkspaceProfile]);
  return workspaceProfile;
}

export function useEnvironmentList() {
  const environmentList = useAppStore((state) => state.environmentList);
  const loadEnvironmentList = useAppStore((state) => state.loadEnvironmentList);
  useEffect(() => {
    void loadEnvironmentList();
  }, [loadEnvironmentList]);
  return environmentList;
}

export function useEnvironment(name: string | undefined) {
  const environmentList = useEnvironmentList();
  return useMemo(() => {
    if (!name || name === NULL_ENVIRONMENT_NAME) {
      return nullEnvironment();
    }
    const environment = environmentList.find((env) => env.name === name);
    if (environment) {
      return environment;
    }
    if (!isValidEnvironmentName(name)) {
      return unknownEnvironment();
    }
    const id = name.replace(/^environments\//, "");
    return {
      ...unknownEnvironment(),
      id,
      name,
      title: id,
    };
  }, [environmentList, name]);
}

export function usePlanFeature(feature: PlanFeature) {
  const loadSubscription = useAppStore((state) => state.loadSubscription);
  const hasFeature = useAppStore((state) => state.hasFeature(feature));
  useEffect(() => {
    void loadSubscription();
  }, [loadSubscription]);
  return hasFeature;
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

export function useProjectPermission(
  project: Project | undefined,
  permission: Permission
) {
  return useAppStore((state) =>
    project ? state.hasProjectPermission(project, permission) : false
  );
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

export function useInstance(name: string | undefined) {
  const instance = useAppStore((state) =>
    name ? state.instancesByName[name] : undefined
  );
  const fetchInstance = useAppStore((state) => state.fetchInstance);
  useEffect(() => {
    if (name) {
      void fetchInstance(name);
    }
  }, [fetchInstance, name]);
  return instance;
}

export function useGroups() {
  const groupsByName = useAppStore((state) => state.groupsByName);
  return useMemo(
    () =>
      Object.values(groupsByName).sort((a, b) => a.name.localeCompare(b.name)),
    [groupsByName]
  );
}

export function useGroupByIdentifier(id: string) {
  return useAppStore((state) => state.getGroupByIdentifier(id));
}

export function useUserByIdentifier(identifier: string | undefined) {
  return useAppStore((state) =>
    identifier ? state.getUserByIdentifier(identifier) : undefined
  );
}

export function useRoleByName(name: string | undefined) {
  return useAppStore((state) => (name ? state.getRoleByName(name) : undefined));
}

export function useReleaseByName(name: string | undefined) {
  return useAppStore((state) =>
    name ? state.getReleaseByName(name) : undefined
  );
}

export function useRevisionByName(name: string | undefined) {
  return useAppStore((state) =>
    name ? state.getRevisionByName(name) : undefined
  );
}

export function useChangelogByName(name: string | undefined) {
  return useAppStore((state) =>
    name ? state.getChangelogByName(name) : undefined
  );
}

export function useServiceAccount(name: string) {
  return useAppStore(useShallow((state) => state.getServiceAccount(name)));
}

export function useWorkloadIdentity(name: string) {
  return useAppStore(useShallow((state) => state.getWorkloadIdentity(name)));
}

export function useIdentityProviderList() {
  return useAppStore(useShallow((state) => state.identityProviderList()));
}

export function useIdentityProvider(name: string) {
  return useAppStore((state) => state.identityProvidersByName[name]);
}

export function useAccessGrant(name: string) {
  return useAppStore((state) => state.accessGrantsByName[name]);
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

function readRecentProjectNames(scope: string, email: string) {
  if (!email) return [];
  try {
    const raw = localStorage.getItem(storageKeyRecentProjects(scope, email));
    const parsed = raw ? JSON.parse(raw) : [];
    return Array.isArray(parsed)
      ? parsed.filter((name): name is string => typeof name === "string")
      : [];
  } catch {
    return [];
  }
}

export function useRecentProjects() {
  const currentUser = useOptionalCurrentUser();
  const isSaaS = useIsSaaSMode();
  const batchFetchProjects = useAppStore((state) => state.batchFetchProjects);
  const projectsByName = useAppStore((state) => state.projectsByName);
  const [projectNames, setProjectNames] = useState<string[]>([]);

  const scope = workspaceCacheScope(isSaaS, currentUser?.workspace ?? "");
  const refreshNames = useCallback(() => {
    setProjectNames(readRecentProjectNames(scope, currentUser?.email ?? ""));
  }, [scope, currentUser?.email]);

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
  useOptionalCurrentUser();
  const recordRecentVisit = useAppStore((state) => state.recordRecentVisit);
  const removeRecentVisit = useAppStore((state) => state.removeRecentVisit);
  return {
    record: recordRecentVisit,
    remove: removeRecentVisit,
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
