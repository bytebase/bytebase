import {
  ChevronDown,
  ChevronRight,
  Database,
  GalleryHorizontalEnd,
  Home,
  Layers,
  Link,
  Settings,
  ShieldCheck,
  SquareStack,
  Users,
  Workflow,
} from "lucide-react";
import type { ElementType } from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import logoFull from "@/assets/logo-full.svg";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import {
  DATABASE_ROUTE_DASHBOARD,
  ENVIRONMENT_V1_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DASHBOARD,
  WORKSPACE_ROUTE_AUDIT_LOG,
  WORKSPACE_ROUTE_CUSTOM_APPROVAL,
  WORKSPACE_ROUTE_DATA_CLASSIFICATION,
  WORKSPACE_ROUTE_GLOBAL_MASKING,
  WORKSPACE_ROUTE_GROUPS,
  WORKSPACE_ROUTE_IDENTITY_PROVIDERS,
  WORKSPACE_ROUTE_IM,
  WORKSPACE_ROUTE_LANDING,
  WORKSPACE_ROUTE_MCP,
  WORKSPACE_ROUTE_MEMBERS,
  WORKSPACE_ROUTE_RISK_CENTER,
  WORKSPACE_ROUTE_ROLES,
  WORKSPACE_ROUTE_SEMANTIC_TYPES,
  WORKSPACE_ROUTE_SERVICE_ACCOUNTS,
  WORKSPACE_ROUTE_SQL_REVIEW,
  WORKSPACE_ROUTE_USER_PROFILE,
  WORKSPACE_ROUTE_USERS,
  WORKSPACE_ROUTE_WORKLOAD_IDENTITIES,
} from "@/router/dashboard/workspaceRoutes";
import {
  SETTING_ROUTE_WORKSPACE_GENERAL,
  SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
} from "@/router/dashboard/workspaceSetting";
import { SQL_EDITOR_HOME_MODULE } from "@/router/sqlEditor";
import {
  useActuatorV1Store,
  useAppFeature,
  useWorkspaceV1Store,
} from "@/store";
import { DatabaseChangeMode } from "@/types/proto-es/v1/setting_service_pb";
import { WorkspaceSwitcher } from "./WorkspaceSwitcher";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface SidebarItem {
  title?: string;
  name?: string;
  path?: string;
  icon?: ElementType;
  hide?: boolean;
  type: "route" | "div" | "divider" | "link";
  expand?: boolean;
  children?: SidebarItem[];
}

// ---------------------------------------------------------------------------
// Active-route detection
// ---------------------------------------------------------------------------

function getItemClass(item: SidebarItem, currentRouteName: string): string[] {
  const isActive =
    item.name === currentRouteName ||
    currentRouteName.startsWith(`${item.name}.`);
  if (isActive) {
    return ["router-link-active", "bg-link-hover"];
  }
  if (
    item.name === WORKSPACE_ROUTE_USERS &&
    currentRouteName === WORKSPACE_ROUTE_USER_PROFILE
  ) {
    return ["router-link-active", "bg-link-hover"];
  }
  return [];
}

// ---------------------------------------------------------------------------
// Sidebar item list builder
// ---------------------------------------------------------------------------

function useSidebarItems(): SidebarItem[] {
  const { t } = useTranslation();
  const isSaaSMode = useVueState(() => useActuatorV1Store().isSaaSMode);

  return useMemo(
    (): SidebarItem[] => [
      {
        title: t("common.home"),
        icon: Home,
        name: WORKSPACE_ROUTE_LANDING,
        type: "route",
      },
      {
        title: t("common.projects"),
        icon: GalleryHorizontalEnd,
        name: PROJECT_V1_ROUTE_DASHBOARD,
        type: "route",
      },
      {
        title: t("common.instances"),
        icon: Layers,
        name: INSTANCE_ROUTE_DASHBOARD,
        type: "route",
      },
      {
        title: t("common.databases"),
        icon: Database,
        name: DATABASE_ROUTE_DASHBOARD,
        type: "route",
      },
      {
        title: t("common.environments"),
        icon: SquareStack,
        name: ENVIRONMENT_V1_ROUTE_DASHBOARD,
        type: "route",
      },
      {
        type: "divider",
        name: "",
      },
      {
        title: t("settings.sidebar.iam-and-admin"),
        icon: Users,
        type: "div",
        children: [
          {
            title: t("common.users"),
            name: WORKSPACE_ROUTE_USERS,
            type: "route",
            hide: isSaaSMode,
          },
          {
            title: t("settings.members.service-accounts"),
            name: WORKSPACE_ROUTE_SERVICE_ACCOUNTS,
            type: "route",
            hide: isSaaSMode,
          },
          {
            title: t("settings.members.workload-identities"),
            name: WORKSPACE_ROUTE_WORKLOAD_IDENTITIES,
            type: "route",
            hide: isSaaSMode,
          },
          {
            title: t("settings.sidebar.members"),
            name: WORKSPACE_ROUTE_MEMBERS,
            type: "route",
          },
          {
            title: t("settings.members.groups.self"),
            name: WORKSPACE_ROUTE_GROUPS,
            type: "route",
          },
          {
            title: t("settings.sidebar.custom-roles"),
            name: WORKSPACE_ROUTE_ROLES,
            type: "route",
          },
          {
            title: t("settings.sidebar.sso"),
            name: WORKSPACE_ROUTE_IDENTITY_PROVIDERS,
            type: "route",
          },
          {
            title: t("settings.sidebar.audit-log"),
            name: WORKSPACE_ROUTE_AUDIT_LOG,
            type: "route",
          },
        ],
      },
      {
        title: "CI/CD",
        icon: Workflow,
        type: "div",
        children: [
          {
            title: t("sql-review.title"),
            name: WORKSPACE_ROUTE_SQL_REVIEW,
            type: "route",
          },
          {
            title: t("custom-approval.risk.self"),
            name: WORKSPACE_ROUTE_RISK_CENTER,
            type: "route",
          },
          {
            title: t("custom-approval.self"),
            name: WORKSPACE_ROUTE_CUSTOM_APPROVAL,
            type: "route",
          },
        ],
      },
      {
        title: t("settings.sidebar.data-access"),
        icon: ShieldCheck,
        type: "div",
        children: [
          {
            title: t("settings.sensitive-data.semantic-types.self"),
            name: WORKSPACE_ROUTE_SEMANTIC_TYPES,
            type: "route",
          },
          {
            title: t("settings.sidebar.data-classification"),
            name: WORKSPACE_ROUTE_DATA_CLASSIFICATION,
            type: "route",
          },
          {
            title: t("settings.sidebar.global-masking"),
            name: WORKSPACE_ROUTE_GLOBAL_MASKING,
            type: "route",
          },
        ],
      },
      {
        title: t("settings.sidebar.integration"),
        icon: Link,
        type: "div",
        children: [
          {
            title: t("settings.sidebar.im-integration"),
            name: WORKSPACE_ROUTE_IM,
            type: "route",
          },
          {
            title: t("settings.sidebar.mcp"),
            name: WORKSPACE_ROUTE_MCP,
            type: "route",
          },
        ],
      },
      {
        title: t("common.settings"),
        icon: Settings,
        type: "div",
        children: [
          {
            title: t("settings.sidebar.general"),
            name: SETTING_ROUTE_WORKSPACE_GENERAL,
            type: "route",
          },
          {
            title: t("settings.sidebar.subscription"),
            name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
            type: "route",
          },
        ],
      },
    ],
    [t, isSaaSMode]
  );
}

// ---------------------------------------------------------------------------
// Filter logic (mirrors CommonSidebar.vue filteredSidebarList)
// ---------------------------------------------------------------------------

function filterSidebarList(items: SidebarItem[]): SidebarItem[] {
  return items
    .map((item) => ({
      ...item,
      children: (item.children ?? []).filter((child) => !child.hide),
    }))
    .filter((item) => {
      if (item.type === "divider") return true;
      if (item.hide) return false;
      if (item.children && item.children.length > 0) return true;
      if (item.type === "div" || item.type === "link") return !!item.path;
      return !!item.path || !!item.name;
    });
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

const parentRouteClass =
  "group flex items-center px-2 py-1.5 leading-normal font-medium rounded-xs text-gray-700 outline-item text-sm!";
const childRouteClass =
  "group w-full flex items-center pl-9 pr-2 py-1 outline-item mb-0.5 rounded-xs";

export function DashboardSidebar() {
  const rawItems = useSidebarItems();
  const filteredItems = useMemo(() => filterSidebarList(rawItems), [rawItems]);

  const currentRouteName = useVueState(
    () => router.currentRoute.value.name?.toString() ?? ""
  );
  const databaseChangeMode = useVueState(
    () => useAppFeature("bb.feature.database-change-mode").value
  );
  const customLogo = useVueState(
    () => useWorkspaceV1Store().currentWorkspace?.logo ?? ""
  );

  // -- Expand / collapse state -----------------------------------------------
  const [expandedSet, setExpandedSet] = useState<Set<string>>(new Set());
  const manualToggledRef = useRef<Set<string>>(new Set());
  const autoExpandedRef = useRef<Set<string>>(new Set());
  const prevFilteredRef = useRef<SidebarItem[] | null>(null);

  const expandForActiveRoute = useCallback(
    (items: SidebarItem[]) => {
      setExpandedSet((prev) => {
        const next = new Set(prev);

        // Remove previous auto-expansions
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

  // When the filtered item list identity changes, reset manual toggles
  useEffect(() => {
    if (prevFilteredRef.current !== filteredItems) {
      prevFilteredRef.current = filteredItems;
      manualToggledRef.current.clear();
      autoExpandedRef.current.clear();
      // Reset and re-expand
      setExpandedSet(new Set());
    }
  }, [filteredItems]);

  // Re-expand on route change or item list change
  useEffect(() => {
    expandForActiveRoute(filteredItems);
  }, [currentRouteName, filteredItems, expandForActiveRoute]);

  // -- Navigation ------------------------------------------------------------

  const navigateToHome = useCallback(() => {
    const target =
      databaseChangeMode === DatabaseChangeMode.EDITOR
        ? SQL_EDITOR_HOME_MODULE
        : WORKSPACE_ROUTE_LANDING;
    router.push({ name: target });
  }, [databaseChangeMode]);

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
    (name: string) => router.resolve({ name }).fullPath,
    []
  );

  const handleRouteClick = useCallback((e: React.MouseEvent, name: string) => {
    // Allow Ctrl/Meta+click to open in new tab naturally
    if (e.ctrlKey || e.metaKey) return;
    e.preventDefault();
    router.push({ name });
  }, []);

  // -- Logo ------------------------------------------------------------------

  const logoSrc = customLogo || logoFull;

  // -- Render ----------------------------------------------------------------

  const renderChildren = (children: SidebarItem[], parentIndex: number) => {
    return (
      <div className="flex flex-col gap-y-1 mt-1">
        {children.map((child, j) => {
          const classes = getItemClass(child, currentRouteName);
          if (child.type === "route" && child.name) {
            return (
              <a
                key={`${parentIndex}-${j}`}
                href={resolveHref(child.name)}
                className={`${childRouteClass} cursor-pointer no-underline text-inherit ${classes.join(" ")}`}
                onClick={(e) => handleRouteClick(e, child.name!)}
              >
                {child.title}
              </a>
            );
          }
          if (child.type === "div") {
            return (
              <div
                key={`${parentIndex}-${j}`}
                className={`${childRouteClass} cursor-pointer ${classes.join(" ")}`}
                onClick={() => onGroupClick(child, `${parentIndex}-${j}`)}
              >
                {child.title}
              </div>
            );
          }
          return null;
        })}
      </div>
    );
  };

  const renderItem = (item: SidebarItem, index: number) => {
    const key = `${index}`;

    if (item.type === "divider") {
      return (
        <div
          key={index}
          className="border-t border-gray-300 my-2.5 mr-4 ml-2"
        />
      );
    }

    if (item.type === "route" && item.name) {
      const classes = getItemClass(item, currentRouteName);
      const Icon = item.icon;
      return (
        <a
          key={index}
          href={resolveHref(item.name)}
          className={`${parentRouteClass} cursor-pointer no-underline text-inherit ${classes.join(" ")}`}
          onClick={(e) => handleRouteClick(e, item.name!)}
        >
          {Icon && <Icon className="mr-2 w-5 h-5 text-gray-500" />}
          {item.title}
        </a>
      );
    }

    if (item.type === "div") {
      const classes = getItemClass(item, currentRouteName);
      const Icon = item.icon;
      const hasChildren = item.children && item.children.length > 0;
      const isExpanded = expandedSet.has(key);
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
    }

    return null;
  };

  return (
    <nav className="flex-1 flex flex-col overflow-y-hidden border-r border-block-border">
      <div
        className="p-2 shrink-0 m-auto cursor-pointer"
        onClick={navigateToHome}
      >
        <img src={logoSrc} alt="Bytebase" className="max-w-44" />
      </div>
      <WorkspaceSwitcher />
      <div className="flex-1 overflow-y-auto px-2.5 flex flex-col gap-y-1">
        {filteredItems.map((item, i) => renderItem(item, i))}
      </div>
    </nav>
  );
}
