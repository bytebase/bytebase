import {
  CircleDot,
  Database,
  GalleryHorizontalEnd,
  GripVertical,
  Layers,
  Link,
  Settings,
  ShieldCheck,
  SquareStack,
  Users,
  Workflow,
} from "lucide-react";
import {
  type DragEvent,
  type FC,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { ProjectSwitchDialog } from "@/react/components/header/ProjectSwitchDialog";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { useCurrentUser, useServerState } from "@/react/hooks/useAppState";
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
  WORKSPACE_ROUTE_IDENTITY_PROVIDERS,
  WORKSPACE_ROUTE_IM,
  WORKSPACE_ROUTE_MCP,
  WORKSPACE_ROUTE_MEMBERS,
  WORKSPACE_ROUTE_MY_ISSUES,
  WORKSPACE_ROUTE_RISK_ASSESSMENT,
  WORKSPACE_ROUTE_ROLES,
  WORKSPACE_ROUTE_SEMANTIC_TYPES,
  WORKSPACE_ROUTE_SERVICE_ACCOUNTS,
  WORKSPACE_ROUTE_SQL_REVIEW,
  WORKSPACE_ROUTE_USERS,
  WORKSPACE_ROUTE_WORKLOAD_IDENTITIES,
} from "@/router/dashboard/workspaceRoutes";
import {
  SETTING_ROUTE_WORKSPACE_GENERAL,
  SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
} from "@/router/dashboard/workspaceSetting";
import { useRecentVisit } from "@/router/useRecentVisit";
import { useProjectV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { UNKNOWN_PROJECT_NAME } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { extractProjectResourceName, storageKeyQuickAccess } from "@/utils";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface QuickLinkDef {
  id: string;
  title?: string;
  route?: string;
  icon: FC<{ className?: string }>;
}

// ---------------------------------------------------------------------------
// Build the full quick-link catalogue (mirrors useDashboardSidebar + useQuickLink)
// ---------------------------------------------------------------------------

function useFullQuickLinkList(): QuickLinkDef[] {
  const { t } = useTranslation();
  const { isSaaSMode } = useServerState();

  return useMemo(() => {
    // Two special items always first
    const list: QuickLinkDef[] = [
      {
        id: "visit-projects",
        title: t("landing.quick-link.visit-prjects"),
        icon: GalleryHorizontalEnd,
      },
      {
        id: "visit-issues",
        title: t("landing.quick-link.visit-issues"),
        icon: CircleDot,
      },
    ];

    // Sidebar route items (flattened, excluding landing itself)
    const sidebarItems: {
      id: string;
      titleKey: string;
      icon: FC<{ className?: string }>;
      hide?: boolean;
    }[] = [
      {
        id: PROJECT_V1_ROUTE_DASHBOARD,
        titleKey: "common.projects",
        icon: GalleryHorizontalEnd,
      },
      {
        id: INSTANCE_ROUTE_DASHBOARD,
        titleKey: "common.instances",
        icon: Layers,
      },
      {
        id: DATABASE_ROUTE_DASHBOARD,
        titleKey: "common.databases",
        icon: Database,
      },
      {
        id: ENVIRONMENT_V1_ROUTE_DASHBOARD,
        titleKey: "common.environments",
        icon: SquareStack,
      },
      // IAM & Admin
      {
        id: WORKSPACE_ROUTE_USERS,
        titleKey: isSaaSMode
          ? "settings.sidebar.members-and-groups"
          : "settings.sidebar.users-and-groups",
        icon: Users,
      },
      {
        id: WORKSPACE_ROUTE_SERVICE_ACCOUNTS,
        titleKey: "settings.members.service-accounts",
        icon: Users,
        hide: isSaaSMode,
      },
      {
        id: WORKSPACE_ROUTE_WORKLOAD_IDENTITIES,
        titleKey: "settings.members.workload-identities",
        icon: Users,
        hide: isSaaSMode,
      },
      {
        id: WORKSPACE_ROUTE_MEMBERS,
        titleKey: "settings.sidebar.members",
        icon: Users,
        hide: isSaaSMode,
      },
      {
        id: WORKSPACE_ROUTE_ROLES,
        titleKey: "settings.sidebar.custom-roles",
        icon: Users,
      },
      {
        id: WORKSPACE_ROUTE_IDENTITY_PROVIDERS,
        titleKey: "settings.sidebar.sso",
        icon: Users,
      },
      {
        id: WORKSPACE_ROUTE_AUDIT_LOG,
        titleKey: "settings.sidebar.audit-log",
        icon: Users,
      },
      // CI/CD
      {
        id: WORKSPACE_ROUTE_SQL_REVIEW,
        titleKey: "sql-review.title",
        icon: Workflow,
      },
      {
        id: WORKSPACE_ROUTE_RISK_ASSESSMENT,
        titleKey: "custom-approval.risk.self",
        icon: Workflow,
      },
      {
        id: WORKSPACE_ROUTE_CUSTOM_APPROVAL,
        titleKey: "custom-approval.self",
        icon: Workflow,
      },
      // Data Access
      {
        id: WORKSPACE_ROUTE_SEMANTIC_TYPES,
        titleKey: "settings.sensitive-data.semantic-types.self",
        icon: ShieldCheck,
      },
      {
        id: WORKSPACE_ROUTE_DATA_CLASSIFICATION,
        titleKey: "settings.sidebar.data-classification",
        icon: ShieldCheck,
      },
      {
        id: WORKSPACE_ROUTE_GLOBAL_MASKING,
        titleKey: "settings.sidebar.global-masking",
        icon: ShieldCheck,
      },
      // Integration
      {
        id: WORKSPACE_ROUTE_IM,
        titleKey: "settings.sidebar.im-integration",
        icon: Link,
      },
      {
        id: WORKSPACE_ROUTE_MCP,
        titleKey: "settings.sidebar.mcp",
        icon: Link,
      },
      // Settings
      {
        id: SETTING_ROUTE_WORKSPACE_GENERAL,
        titleKey: "settings.sidebar.general",
        icon: Settings,
      },
      {
        id: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
        titleKey: "settings.sidebar.subscription",
        icon: Settings,
      },
    ];

    for (const item of sidebarItems) {
      if (item.hide) continue;
      list.push({
        id: item.id,
        title: t(item.titleKey),
        route: item.id,
        icon: item.icon,
      });
    }

    return list;
  }, [t, isSaaSMode]);
}

// ---------------------------------------------------------------------------
// localStorage-backed quick-access config (same keys as Vue version)
// ---------------------------------------------------------------------------

const DEFAULT_CONFIG = [
  "visit-issues",
  "visit-projects",
  WORKSPACE_ROUTE_USERS,
  WORKSPACE_ROUTE_SQL_REVIEW,
];

function useQuickAccessConfig(email: string) {
  const key = storageKeyQuickAccess(email);

  const read = useCallback((): string[] => {
    try {
      const raw = localStorage.getItem(key);
      if (raw) return JSON.parse(raw);
    } catch {
      // ignore
    }
    return [...DEFAULT_CONFIG];
  }, [key]);

  const [config, setConfig] = useState<string[]>(read);

  // Sync when email (and thus key) changes
  useEffect(() => {
    setConfig(read());
  }, [read]);

  const persist = useCallback(
    (next: string[]) => {
      setConfig(next);
      localStorage.setItem(key, JSON.stringify(next));
    },
    [key]
  );

  return { config, setConfig: persist };
}

// ---------------------------------------------------------------------------
// Last visited project
// ---------------------------------------------------------------------------

function useLastVisitedProject() {
  const lastVisitProjectPath = useVueState(
    () => useRecentVisit().lastVisitProjectPath.value
  );
  const [project, setProject] = useState<Project | undefined>();

  useEffect(() => {
    if (!lastVisitProjectPath) {
      setProject(undefined);
      return;
    }

    const projectName = `${projectNamePrefix}${extractProjectResourceName(lastVisitProjectPath)}`;
    const projectStore = useProjectV1Store();
    let cancelled = false;
    projectStore.getOrFetchProjectByName(projectName).then((p) => {
      if (cancelled) return;
      setProject(p.name === UNKNOWN_PROJECT_NAME ? undefined : p);
    });
    return () => {
      cancelled = true;
    };
  }, [lastVisitProjectPath]);

  return { project, projectPath: lastVisitProjectPath };
}

// ---------------------------------------------------------------------------
// Config Drawer
// ---------------------------------------------------------------------------

function ConfigSheet({
  open,
  onClose,
  fullList,
  config,
  setConfig,
}: {
  open: boolean;
  onClose: () => void;
  fullList: QuickLinkDef[];
  config: string[];
  setConfig: (c: string[]) => void;
}) {
  const { t } = useTranslation();

  const dragItem = useRef<number | null>(null);
  const dragOver = useRef<number | null>(null);

  const selected = useMemo(
    () =>
      config
        .map((id) => fullList.find((l) => l.id === id))
        .filter(Boolean) as QuickLinkDef[],
    [config, fullList]
  );

  const unselected = useMemo(
    () => fullList.filter((l) => !config.includes(l.id)),
    [config, fullList]
  );

  const handleDragStart = (index: number) => {
    dragItem.current = index;
  };

  const handleDragEnter = (index: number) => {
    dragOver.current = index;
  };

  const handleDragEnd = () => {
    if (dragItem.current === null || dragOver.current === null) return;
    // Reorder using the filtered selected IDs, not raw config indices,
    // so stale/hidden IDs in config don't cause index mismatches.
    const selectedIds = selected.map((s) => s.id);
    const [removed] = selectedIds.splice(dragItem.current, 1);
    selectedIds.splice(dragOver.current, 0, removed);
    setConfig(selectedIds);
    dragItem.current = null;
    dragOver.current = null;
  };

  const uncheck = (id: string) => {
    setConfig(config.filter((c) => c !== id));
  };

  const check = (id: string) => {
    setConfig([...config, id]);
  };

  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="narrow">
        <SheetHeader>
          <SheetTitle>{t("landing.quick-link.manage")}</SheetTitle>
        </SheetHeader>
        <SheetBody className="px-2 py-2">
          {selected.map((item, i) => (
            <div
              key={item.id}
              draggable
              onDragStart={() => handleDragStart(i)}
              onDragEnter={() => handleDragEnter(i)}
              onDragEnd={handleDragEnd}
              onDragOver={(e: DragEvent) => e.preventDefault()}
              className="flex items-center justify-between p-2 hover:bg-gray-100 rounded-sm cursor-grab"
            >
              <div className="flex items-center gap-x-2">
                <Checkbox
                  checked={true}
                  disabled={selected.length <= 1}
                  onCheckedChange={() => uncheck(item.id)}
                />
                <item.icon className="w-5 h-5 text-gray-500" />
                {item.title}
              </div>
              <GripVertical className="w-5 h-5 text-gray-500" />
            </div>
          ))}

          {unselected.length > 0 && <div className="border-t my-2" />}

          {unselected.map((item) => (
            <div
              key={item.id}
              className="flex items-center gap-x-2 p-2 hover:bg-gray-100 rounded-sm cursor-pointer"
              onClick={() => check(item.id)}
            >
              <Checkbox checked={false} />
              <item.icon className="w-5 h-5 text-gray-500" />
              {item.title}
            </div>
          ))}
        </SheetBody>
      </SheetContent>
    </Sheet>
  );
}

// ---------------------------------------------------------------------------
// LandingPage
// ---------------------------------------------------------------------------

export function LandingPage(_: Record<string, never> = {}) {
  const { t } = useTranslation();
  const [showConfigDrawer, setShowConfigDrawer] = useState(false);
  const [showProjectSwitchDialog, setShowProjectSwitchDialog] = useState(false);

  const email = useCurrentUser()?.email ?? "";
  const { version, changelogURL } = useServerState();

  const fullList = useFullQuickLinkList();
  const { config, setConfig } = useQuickAccessConfig(email);
  const { project: lastProject, projectPath: lastVisitProjectPath } =
    useLastVisitedProject();

  const quickLinkList = useMemo(
    () =>
      config
        .map((id) => fullList.find((l) => l.id === id))
        .filter(Boolean) as QuickLinkDef[],
    [config, fullList]
  );

  const handleClick = useCallback((link: QuickLinkDef) => {
    if (link.route) {
      router.push({ name: link.route });
      return;
    }
    switch (link.id) {
      case "visit-projects":
        setShowProjectSwitchDialog(true);
        break;
      case "visit-issues":
        router.push({ name: WORKSPACE_ROUTE_MY_ISSUES });
        break;
    }
  }, []);

  return (
    <>
      <div className="py-4 px-4 flex flex-col h-full items-center relative">
        <div className="flex-1" />
        <div className="flex-[60%] flex flex-col gap-y-6">
          <div className="flex items-baseline gap-x-4">
            <div className="flex items-start gap-x-1">
              <div className="font-semibold text-2xl">
                {t("landing.quick-link.self")}
              </div>
              <button
                className="p-1 rounded-xs hover:bg-gray-100"
                onClick={() => setShowConfigDrawer(true)}
              >
                <Settings className="w-4 h-4 text-gray-500" />
              </button>
            </div>
            {lastProject && lastVisitProjectPath && (
              <a
                className="underline normal-link cursor-pointer"
                onClick={() => router.push({ path: lastVisitProjectPath })}
              >
                {t("landing.last-visit")}: {lastProject.title}
              </a>
            )}
          </div>

          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
            {quickLinkList.map((link) => {
              const tileClass =
                "flex justify-center items-center gap-x-2 cursor-pointer border rounded-sm px-4 py-5 bg-white hover:bg-gray-100 no-underline text-main";
              if (link.route) {
                const href = router.resolve({
                  name: link.route,
                }).href;
                return (
                  <a key={link.id} href={href} className={tileClass}>
                    <link.icon className="w-5 h-5 text-gray-500" />
                    {link.title}
                  </a>
                );
              }
              return (
                <button
                  key={link.id}
                  type="button"
                  className={tileClass}
                  onClick={() => handleClick(link)}
                >
                  <link.icon className="w-5 h-5 text-gray-500" />
                  {link.title}
                </button>
              );
            })}
          </div>

          <div className="flex flex-col gap-y-2">
            {changelogURL && (
              <a
                className="underline normal-link"
                target="_blank"
                rel="noopener noreferrer"
                href={changelogURL}
              >
                {t("landing.changelog-for-version", { version })}
              </a>
            )}
          </div>
        </div>
      </div>

      <ConfigSheet
        open={showConfigDrawer}
        onClose={() => setShowConfigDrawer(false)}
        fullList={fullList}
        config={config}
        setConfig={setConfig}
      />

      <ProjectSwitchDialog
        open={showProjectSwitchDialog}
        onClose={() => setShowProjectSwitchDialog(false)}
      />
    </>
  );
}
