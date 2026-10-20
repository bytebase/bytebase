import {
  Bot,
  CircleDot,
  Database,
  GalleryHorizontalEnd,
  GripVertical,
  Layers,
  Link,
  Send,
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
import { router } from "@/app/router";
import {
  DATABASE_ROUTE_DASHBOARD,
  ENVIRONMENT_V1_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DASHBOARD,
  SETTING_ROUTE_WORKSPACE_GENERAL,
  SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
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
} from "@/app/router/handles";
import { RouterLink } from "@/components/RouterLink";
import { Alert } from "@/components/ui/alert";
import { Button, buttonVariants } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { QuickLinkButton, QuickLinkLink } from "@/components/ui/quick-link";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Textarea } from "@/components/ui/textarea";
import { WorkspacePageLayout } from "@/components/WorkspacePageLayout";
import {
  useOptionalCurrentUser,
  usePlanFeature,
  useServerState,
  useWorkspace,
  useWorkspacePermission,
} from "@/hooks/useAppState";
import { useRecentVisit } from "@/hooks/useRecentVisit";
import {
  AI_ASSISTANT_PRODUCT_INTRO,
  PRODUCT_INTRO_QUERY_KEY,
} from "@/lib/productIntro";
import { LandingStarterSelector } from "@/modules/agent/components/LandingStarterSelector";
import { getLandingStarterIds } from "@/modules/agent/logic/landingStarters";
import {
  selectHasRunningChat,
  useAgentStore,
} from "@/modules/agent/store/agent";
import { useAppStore } from "@/stores/app";
import { projectNamePrefix } from "@/stores/modules/v1/common";
import { UNKNOWN_PROJECT_NAME } from "@/types";
import { ApprovalStatus, IssueStatus } from "@/types/proto-es/v1/common_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { extractProjectResourceName, storageKeyQuickAccess } from "@/utils";
import { WorkspaceLandingGettingStarted } from "./WorkspaceLandingGettingStarted";

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
// Build the full quick-link catalogue
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
        route: PROJECT_V1_ROUTE_DASHBOARD,
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
// localStorage-backed quick-access config
// ---------------------------------------------------------------------------

const DEFAULT_CONFIG = [
  PROJECT_V1_ROUTE_DASHBOARD,
  WORKSPACE_ROUTE_USERS,
  WORKSPACE_ROUTE_SQL_REVIEW,
  SETTING_ROUTE_WORKSPACE_GENERAL,
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
  const { lastVisitProjectPath } = useRecentVisit();
  const [project, setProject] = useState<Project | undefined>();

  useEffect(() => {
    if (!lastVisitProjectPath) {
      setProject(undefined);
      return;
    }

    const projectName = `${projectNamePrefix}${extractProjectResourceName(lastVisitProjectPath)}`;
    let cancelled = false;
    useAppStore
      .getState()
      .getOrFetchProjectByName(projectName)
      .then((p) => {
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
              className="flex items-center justify-between rounded-sm p-2 hover:bg-control-bg cursor-grab"
            >
              <div className="flex items-center gap-x-2">
                <Checkbox
                  checked={true}
                  disabled={selected.length <= 1}
                  onCheckedChange={() => uncheck(item.id)}
                />
                <item.icon className="w-5 h-5 text-control-light" />
                {item.title}
              </div>
              <GripVertical className="w-5 h-5 text-control-light" />
            </div>
          ))}

          {unselected.length > 0 && (
            <div className="my-2 border-t border-control-border" />
          )}

          {unselected.map((item) => (
            <div
              key={item.id}
              className="flex items-center gap-x-2 rounded-sm p-2 hover:bg-control-bg cursor-pointer"
              onClick={() => check(item.id)}
            >
              <Checkbox checked={false} />
              <item.icon className="w-5 h-5 text-control-light" />
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
  const hasAuditLogFeature = usePlanFeature(PlanFeature.FEATURE_AUDIT_LOG);
  const [showConfigDrawer, setShowConfigDrawer] = useState(false);
  const [aiPrompt, setAiPrompt] = useState("");
  const workspace = useWorkspace();
  const aiEnabled = useAppStore((state) => state.serverInfo?.aiEnabled);
  const canConfigureAI = useWorkspacePermission("bb.settings.set");
  const hasRunningChat = useAgentStore(selectHasRunningChat);
  const hasQueuedMessage = useAgentStore(
    (state) => state.queuedUserMessage !== null
  );

  const currentUser = useOptionalCurrentUser();
  const email = currentUser?.email ?? "";
  const currentUserName = currentUser?.name ?? "";
  const { isSaaSMode, version, changelogURL } = useServerState();
  const canSearchIssues = useWorkspacePermission("bb.issues.get");
  const canSearchAuditLogs = useWorkspacePermission("bb.auditLogs.search");
  const listIssues = useAppStore((state) => state.listIssues);
  const [activeWork, setActiveWork] = useState<
    "loading" | "present" | "empty" | "unknown"
  >("loading");

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
    if (link.id === "visit-issues") {
      router.push({ name: WORKSPACE_ROUTE_MY_ISSUES });
    }
  }, []);

  useEffect(() => {
    if (
      !aiEnabled ||
      !workspace?.name ||
      !canSearchIssues ||
      !currentUserName
    ) {
      setActiveWork("unknown");
      return;
    }
    let active = true;
    setActiveWork("loading");
    void Promise.allSettled([
      listIssues({
        find: {
          project: "projects/-",
          query: "",
          currentApprover: currentUserName,
          statusList: [IssueStatus.OPEN],
          approvalStatus: ApprovalStatus.PENDING,
          orderBy: "update_time desc",
        },
        pageSize: 1,
      }),
      listIssues({
        find: {
          project: "projects/-",
          query: "",
          creator: currentUserName,
          statusList: [IssueStatus.OPEN],
          orderBy: "update_time desc",
        },
        pageSize: 1,
      }),
    ]).then((results) => {
      if (!active) return;
      if (
        results.some(
          (result) =>
            result.status === "fulfilled" && result.value.issues.length > 0
        )
      ) {
        setActiveWork("present");
      } else if (results.every((result) => result.status === "fulfilled")) {
        setActiveWork("empty");
      } else {
        setActiveWork("unknown");
      }
    });
    return () => {
      active = false;
    };
  }, [
    aiEnabled,
    canSearchIssues,
    currentUserName,
    listIssues,
    workspace?.name,
  ]);

  const landingStarterIds = getLandingStarterIds({
    scope: "workspace",
    activeWork,
    canViewRecentActivity: hasAuditLogFeature && canSearchAuditLogs,
  });

  const sendAIMessage = () => {
    const text = aiPrompt.trim();
    if (!text || !aiEnabled || !workspace?.name) return;
    const store = useAgentStore.getState();
    if (selectHasRunningChat(store) || store.queuedUserMessage) return;
    const page = {
      path: `${window.location.pathname}${window.location.search}${window.location.hash}`,
      title: document.title,
    };
    const chat = store.selectOrCreateChatForContext(
      workspace.name,
      page,
      workspace.name
    );
    if (!chat) return;
    store.queueUserMessage({ chatId: chat.id, text, page });
    store.open();
    setAiPrompt("");
  };

  const sendLandingStarter = (
    starterId: (typeof landingStarterIds)[number]
  ) => {
    if (!aiEnabled || !workspace?.name) return;
    const store = useAgentStore.getState();
    if (selectHasRunningChat(store) || store.queuedUserMessage) return;
    const page = {
      path: `${window.location.pathname}${window.location.search}${window.location.hash}`,
      title: document.title,
    };
    const chat = store.selectOrCreateChatForContext(
      workspace.name,
      page,
      workspace.name
    );
    if (!chat) return;
    store.queueUserMessage({
      chatId: chat.id,
      text: t(`agent.landing-starters.${starterId}`),
      page,
      runMode: "read-only",
      starterId,
    });
    store.open();
  };

  return (
    <>
      <WorkspacePageLayout>
        <div className="flex min-w-0 flex-col gap-y-8">
          <header className="min-w-0">
            <h1 className="min-w-0 break-words text-2xl font-bold text-main">
              {workspace?.title ?? t("landing.workspace")}
            </h1>
          </header>

          {aiEnabled && workspace?.name && (
            <form
              className="min-w-0 rounded-sm border border-accent/60 bg-background focus-within:border-accent focus-within:ring-1 focus-within:ring-accent"
              onSubmit={(event) => {
                event.preventDefault();
                sendAIMessage();
              }}
            >
              {!aiPrompt.trim() && (
                <div className="px-3 pt-2">
                  <LandingStarterSelector
                    starterIds={landingStarterIds}
                    disabled={hasRunningChat || hasQueuedMessage}
                    onSelect={sendLandingStarter}
                  />
                </div>
              )}
              <Textarea
                size="md"
                rows={2}
                value={aiPrompt}
                aria-label={t("landing.ai-placeholder")}
                placeholder={`${t("landing.ai-placeholder")} · ${t("agent.input-keyboard-hint")}`}
                className="min-h-20 resize-none border-0 focus:border-0 focus:ring-0"
                onChange={(event) => setAiPrompt(event.target.value)}
                onKeyDown={(event) => {
                  if (
                    event.key === "Enter" &&
                    !event.shiftKey &&
                    !event.altKey &&
                    !event.ctrlKey &&
                    !event.metaKey &&
                    !event.nativeEvent.isComposing
                  ) {
                    event.preventDefault();
                    sendAIMessage();
                  }
                }}
              />
              <div className="flex flex-wrap items-end justify-between gap-2 px-3 pb-3">
                <div className="flex min-w-0 flex-1 items-center text-xs text-control-light">
                  <span className="min-w-0 max-w-full truncate">
                    {t("agent.context")}: {workspace.title}
                  </span>
                </div>
                <Button
                  type="submit"
                  size="sm"
                  disabled={
                    !aiPrompt.trim() || hasRunningChat || hasQueuedMessage
                  }
                >
                  <Send className="size-4" />
                  {t("agent.send")}
                </Button>
              </div>
            </form>
          )}

          {aiEnabled === false && canConfigureAI && !isSaaSMode && (
            <Alert
              role="region"
              aria-label={t("landing.ai-setup.title")}
              showIcon={false}
              className="border-block-border bg-background text-main shadow-none"
            >
              <div className="flex flex-wrap items-center gap-3">
                <span className="flex size-8 shrink-0 items-center justify-center rounded-xs bg-accent/10 text-accent">
                  <Bot className="size-4" aria-hidden="true" />
                </span>
                <div className="min-w-0 flex-1">
                  <h2 className="text-sm font-semibold">
                    {t("landing.ai-setup.title")}
                  </h2>
                  <p className="text-sm text-control-light">
                    {t("landing.ai-setup.description")}
                  </p>
                </div>
                <RouterLink
                  to={{
                    name: SETTING_ROUTE_WORKSPACE_GENERAL,
                    hash: "#ai-assistant",
                    query: {
                      [PRODUCT_INTRO_QUERY_KEY]: AI_ASSISTANT_PRODUCT_INTRO,
                    },
                  }}
                  className={buttonVariants({ size: "sm" })}
                >
                  {t("landing.ai-setup.action")}
                </RouterLink>
              </div>
            </Alert>
          )}

          <WorkspaceLandingGettingStarted />

          <section className="min-w-0">
            <div className="flex flex-wrap items-center justify-between gap-2 border-b border-block-border pb-4">
              <div className="flex min-w-0 flex-wrap items-baseline gap-x-4 gap-y-2">
                <h2 className="text-base font-semibold text-main">
                  {t("landing.quick-link.yours")}
                </h2>
                {lastProject && lastVisitProjectPath && (
                  <RouterLink
                    to={{ path: lastVisitProjectPath }}
                    className="normal-link text-sm underline"
                  >
                    {t("landing.last-visit")}: {lastProject.title}
                  </RouterLink>
                )}
              </div>
              <Button
                type="button"
                appearance="outline"
                size="sm"
                onClick={() => setShowConfigDrawer(true)}
              >
                {t("project.home.customize")}
              </Button>
            </div>
            <div className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
              {quickLinkList.map((link) => {
                if (link.route) {
                  return (
                    <QuickLinkLink
                      key={link.id}
                      to={{ name: link.route }}
                      className="min-w-0 justify-start py-3"
                      icon={<link.icon className="size-4 text-control-light" />}
                    >
                      {link.title}
                    </QuickLinkLink>
                  );
                }
                return (
                  <QuickLinkButton
                    key={link.id}
                    type="button"
                    onClick={() => handleClick(link)}
                    className="min-w-0 justify-start py-3"
                    icon={<link.icon className="size-4 text-control-light" />}
                  >
                    {link.title}
                  </QuickLinkButton>
                );
              })}
            </div>
          </section>

          {changelogURL && (
            <a
              className="normal-link w-fit text-sm underline"
              target="_blank"
              rel="noopener noreferrer"
              href={changelogURL}
            >
              {t("landing.changelog-for-version", { version })}
            </a>
          )}
        </div>
      </WorkspacePageLayout>

      <ConfigSheet
        open={showConfigDrawer}
        onClose={() => setShowConfigDrawer(false)}
        fullList={fullList}
        config={config}
        setConfig={setConfig}
      />
    </>
  );
}
