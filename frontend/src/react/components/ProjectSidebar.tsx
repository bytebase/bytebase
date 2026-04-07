import {
  ChevronDown,
  ChevronRight,
  CircleDot,
  Database,
  Settings,
  ShieldCheck,
  Users,
  Workflow,
} from "lucide-react";
import type { ElementType } from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { effectScope } from "vue";
import logoFull from "@/assets/logo-full.svg";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import {
  PROJECT_V1_ROUTE_ACCESS_GRANTS,
  PROJECT_V1_ROUTE_AUDIT_LOGS,
  PROJECT_V1_ROUTE_DATABASE_GROUPS,
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_DETAIL,
  PROJECT_V1_ROUTE_EXPORT_CENTER,
  PROJECT_V1_ROUTE_GITOPS,
  PROJECT_V1_ROUTE_ISSUES,
  PROJECT_V1_ROUTE_MASKING_EXEMPTION,
  PROJECT_V1_ROUTE_MEMBERS,
  PROJECT_V1_ROUTE_PLANS,
  PROJECT_V1_ROUTE_RELEASES,
  PROJECT_V1_ROUTE_SERVICE_ACCOUNTS,
  PROJECT_V1_ROUTE_SETTINGS,
  PROJECT_V1_ROUTE_SYNC_SCHEMA,
  PROJECT_V1_ROUTE_WEBHOOKS,
  PROJECT_V1_ROUTE_WORKLOAD_IDENTITIES,
} from "@/router/dashboard/projectV1";
import { useRecentVisit } from "@/router/useRecentVisit";
import {
  useActuatorV1Store,
  useProjectV1Store,
  useWorkspaceV1Store,
} from "@/store";
import { getProjectName, projectNamePrefix } from "@/store/modules/v1/common";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface SidebarItem {
  title: string;
  path?: string;
  icon?: ElementType;
  hide?: boolean;
  type: "div";
  expand?: boolean;
  children?: SidebarItem[];
}

// ---------------------------------------------------------------------------
// Active-route detection
// ---------------------------------------------------------------------------

function getItemClass(item: SidebarItem, currentRouteName: string): string[] {
  const isActive =
    item.path === currentRouteName ||
    currentRouteName.startsWith(`${item.path}.`);
  if (isActive) {
    return ["router-link-active", "bg-link-hover"];
  }
  return [];
}

// ---------------------------------------------------------------------------
// Sidebar item list builder
// ---------------------------------------------------------------------------

function useSidebarItems(): SidebarItem[] {
  const { t } = useTranslation();

  const isDefault = useVueState(() => {
    const projectId =
      (router.currentRoute.value.params.projectId as string | undefined) ?? "";
    const projectName = projectId ? `${projectNamePrefix}${projectId}` : "";
    const defaultProject =
      useActuatorV1Store().serverInfo?.defaultProject ?? "";
    return !!defaultProject && projectName === defaultProject;
  });

  return useMemo(
    (): SidebarItem[] => [
      {
        title: t("common.issues"),
        path: PROJECT_V1_ROUTE_ISSUES,
        icon: CircleDot,
        type: "div",
        hide: isDefault,
      },
      {
        title: "CI/CD",
        icon: Workflow,
        type: "div",
        expand: true,
        hide: isDefault,
        children: [
          {
            title: t("plan.plans"),
            path: PROJECT_V1_ROUTE_PLANS,
            type: "div",
          },
          {
            title: t("release.releases"),
            path: PROJECT_V1_ROUTE_RELEASES,
            type: "div",
          },
          {
            title: t("gitops.self"),
            path: PROJECT_V1_ROUTE_GITOPS,
            type: "div",
          },
        ],
      },
      {
        title: t("common.database"),
        icon: Database,
        type: "div",
        expand: true,
        children: [
          {
            title: t("common.databases"),
            path: PROJECT_V1_ROUTE_DATABASES,
            type: "div",
          },
          {
            title: t("common.groups"),
            path: PROJECT_V1_ROUTE_DATABASE_GROUPS,
            type: "div",
          },
          {
            title: t("database.sync-schema.title"),
            path: PROJECT_V1_ROUTE_SYNC_SCHEMA,
            type: "div",
          },
        ],
      },
      {
        title: t("settings.sidebar.data-access"),
        icon: ShieldCheck,
        type: "div",
        hide: isDefault,
        expand: true,
        children: [
          {
            title: t("sql-editor.access-grants"),
            path: PROJECT_V1_ROUTE_ACCESS_GRANTS,
            type: "div",
          },
          {
            title: t("project.masking-exemption.self"),
            path: PROJECT_V1_ROUTE_MASKING_EXEMPTION,
            type: "div",
          },
          {
            title: t("export-center.data-export"),
            path: PROJECT_V1_ROUTE_EXPORT_CENTER,
            type: "div",
          },
        ],
      },
      {
        title: t("common.manage"),
        icon: Users,
        type: "div",
        hide: isDefault,
        expand: true,
        children: [
          {
            title: t("common.members", { count: 2 }),
            path: PROJECT_V1_ROUTE_MEMBERS,
            type: "div",
          },
          {
            title: t("settings.members.service-accounts"),
            path: PROJECT_V1_ROUTE_SERVICE_ACCOUNTS,
            type: "div",
          },
          {
            title: t("settings.members.workload-identities"),
            path: PROJECT_V1_ROUTE_WORKLOAD_IDENTITIES,
            type: "div",
          },
          {
            title: t("common.webhooks"),
            path: PROJECT_V1_ROUTE_WEBHOOKS,
            type: "div",
          },
          {
            title: t("settings.sidebar.audit-log"),
            path: PROJECT_V1_ROUTE_AUDIT_LOGS,
            type: "div",
          },
        ],
      },
      {
        title: t("common.setting"),
        icon: Settings,
        path: PROJECT_V1_ROUTE_SETTINGS,
        type: "div",
        hide: isDefault,
      },
    ],
    [t, isDefault]
  );
}

// ---------------------------------------------------------------------------
// Filter logic
// ---------------------------------------------------------------------------

function filterSidebarList(items: SidebarItem[]): SidebarItem[] {
  return items
    .map((item) => ({
      ...item,
      children: (item.children ?? []).filter((child) => !child.hide),
    }))
    .filter((item) => {
      if (item.hide) return false;
      if (item.children && item.children.length > 0) return true;
      return !!item.path;
    });
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

const parentRouteClass =
  "group flex items-center px-2 py-1.5 leading-normal font-medium rounded-xs text-gray-700 outline-item text-sm!";
const childRouteClass =
  "group w-full flex items-center pl-9 pr-2 py-1 outline-item mb-0.5 rounded-xs";

export function ProjectSidebar() {
  const rawItems = useSidebarItems();
  const filteredItems = useMemo(() => filterSidebarList(rawItems), [rawItems]);

  const currentRouteName = useVueState(
    () => router.currentRoute.value.name?.toString() ?? ""
  );

  const projectId = useVueState(
    () =>
      (router.currentRoute.value.params.projectId as string | undefined) ?? ""
  );

  const customLogo = useVueState(
    () => useWorkspaceV1Store().currentWorkspace?.logo ?? ""
  );

  const projectStore = useProjectV1Store();

  // Create a Vue effectScope so we can call the Vue composable useRecentVisit.
  const recordVisitRef = useRef<((path: string) => void) | null>(null);
  useEffect(() => {
    const scope = effectScope();
    scope.run(() => {
      const { record } = useRecentVisit();
      recordVisitRef.current = record;
    });
    return () => scope.stop();
  }, []);

  // Ensure the project is fetched into the store cache.
  // ProjectV1Layout also fetches it, but this guards against race conditions.
  useEffect(() => {
    if (projectId) {
      projectStore.getOrFetchProjectByName(
        `${projectNamePrefix}${projectId}`,
        true
      );
    }
  }, [projectId, projectStore]);

  const project = useVueState(() => {
    const pid =
      (router.currentRoute.value.params.projectId as string | undefined) ?? "";
    const projectName = pid ? `${projectNamePrefix}${pid}` : "";
    return projectStore.getProjectByName(projectName);
  });

  // -- Expand / collapse state -----------------------------------------------
  const [expandedSet, setExpandedSet] = useState<Set<string>>(new Set());
  const manualToggledRef = useRef<Set<string>>(new Set());
  const autoExpandedRef = useRef<Set<string>>(new Set());
  const prevFilteredRef = useRef<SidebarItem[] | null>(null);

  const expandForActiveRoute = useCallback(
    (items: SidebarItem[]) => {
      setExpandedSet((prev) => {
        const next = new Set(prev);

        for (const key of autoExpandedRef.current) {
          next.delete(key);
        }
        autoExpandedRef.current.clear();

        for (let i = 0; i < items.length; i++) {
          const item = items[i];
          const key = `${i}`;
          if (!item.children || item.children.length === 0) continue;
          if (manualToggledRef.current.has(key)) continue;
          if (item.expand) {
            next.add(key);
            continue;
          }
          const hasActiveChild = item.children.some(
            (child) => getItemClass(child, currentRouteName).length > 0
          );
          if (hasActiveChild && !next.has(key)) {
            next.add(key);
            autoExpandedRef.current.add(key);
          }
        }

        return next;
      });
    },
    [currentRouteName]
  );

  useEffect(() => {
    if (prevFilteredRef.current !== filteredItems) {
      prevFilteredRef.current = filteredItems;
      manualToggledRef.current.clear();
      autoExpandedRef.current.clear();
      setExpandedSet(new Set());
    }
  }, [filteredItems]);

  useEffect(() => {
    expandForActiveRoute(filteredItems);
  }, [currentRouteName, filteredItems, expandForActiveRoute]);

  // -- Navigation ------------------------------------------------------------

  const navigateToHome = useCallback(() => {
    router.push({
      name: PROJECT_V1_ROUTE_DETAIL,
      params: { projectId },
    });
  }, [projectId]);

  const onGroupClick = useCallback((item: SidebarItem, key: string) => {
    if (item.children && item.children.length > 0) {
      manualToggledRef.current.add(key);
      autoExpandedRef.current.delete(key);
      setExpandedSet((prev) => {
        const next = new Set(prev);
        if (next.has(key)) {
          next.delete(key);
        } else {
          next.add(key);
        }
        return next;
      });
    }
  }, []);

  const resolveHref = useCallback(
    (path: string) =>
      router.resolve({
        name: path,
        params: { projectId: getProjectName(project.name) },
      }).fullPath,
    [project.name]
  );

  const handleItemClick = useCallback(
    (e: React.MouseEvent, path: string) => {
      const route = router.resolve({
        name: path,
        params: { projectId: getProjectName(project.name) },
      });
      recordVisitRef.current?.(route.fullPath);
      if (e.ctrlKey || e.metaKey) {
        // Let the browser's native <a> Ctrl/Meta-click handle new-tab opening.
        return;
      }
      e.preventDefault();
      router.push(route);
    },
    [project.name]
  );

  // -- Logo ------------------------------------------------------------------

  const logoSrc = customLogo || logoFull;

  // -- Render ----------------------------------------------------------------

  const renderChildren = (children: SidebarItem[], parentIndex: number) => {
    return (
      <div className="flex flex-col gap-y-1 mt-1">
        {children.map((child, j) => {
          const classes = getItemClass(child, currentRouteName);
          if (child.path) {
            return (
              <a
                key={`${parentIndex}-${j}`}
                href={resolveHref(child.path)}
                className={`${childRouteClass} cursor-pointer no-underline text-inherit ${classes.join(" ")}`}
                onClick={(e) => handleItemClick(e, child.path!)}
              >
                {child.title}
              </a>
            );
          }
          return null;
        })}
      </div>
    );
  };

  const renderItem = (item: SidebarItem, index: number) => {
    const key = `${index}`;
    const Icon = item.icon;
    const hasChildren = item.children && item.children.length > 0;
    const isExpanded = expandedSet.has(key);

    // Leaf item (no children) — navigable
    if (!hasChildren && item.path) {
      const classes = getItemClass(item, currentRouteName);
      return (
        <a
          key={index}
          href={resolveHref(item.path)}
          className={`${parentRouteClass} cursor-pointer no-underline text-inherit ${classes.join(" ")}`}
          onClick={(e) => handleItemClick(e, item.path!)}
        >
          {Icon && <Icon className="mr-2 w-5 h-5 text-gray-500" />}
          {item.title}
        </a>
      );
    }

    // Group item with children
    const classes = getItemClass(item, currentRouteName);
    return (
      <div key={index}>
        <div
          className={`${parentRouteClass} cursor-pointer ${classes.join(" ")}`}
          onClick={() => onGroupClick(item, key)}
        >
          {Icon && <Icon className="mr-2 w-5 h-5 text-gray-500" />}
          {item.title}
          {hasChildren && (
            <div className="ml-auto text-gray-500">
              {isExpanded ? (
                <ChevronDown className="w-4 h-4" />
              ) : (
                <ChevronRight className="w-4 h-4" />
              )}
            </div>
          )}
        </div>
        {hasChildren && isExpanded && renderChildren(item.children!, index)}
      </div>
    );
  };

  return (
    <nav className="flex-1 flex flex-col overflow-y-hidden border-r border-block-border">
      <div
        className="p-2 shrink-0 m-auto cursor-pointer"
        onClick={navigateToHome}
      >
        <img src={logoSrc} alt="Bytebase" className="max-w-44" />
      </div>
      <div className="flex-1 overflow-y-auto px-2.5 flex flex-col gap-y-1">
        {filteredItems.map((item, i) => renderItem(item, i))}
      </div>
    </nav>
  );
}
