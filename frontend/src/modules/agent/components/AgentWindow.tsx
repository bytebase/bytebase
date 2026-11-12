import { Dialog as BaseDialog } from "@base-ui/react/dialog";
import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import {
  Archive,
  Check,
  ChevronLeft,
  Ellipsis,
  EllipsisVertical,
  Folder,
  FolderOpen,
  LoaderCircle,
  MessageSquare,
  Minus,
  Pencil,
  Plus,
  Settings,
  Trash2,
  Undo2,
  X,
} from "lucide-react";
import {
  type ReactNode,
  type RefObject,
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  useSyncExternalStore,
} from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import {
  Panel,
  Group as PanelGroup,
  type PanelImperativeHandle,
  Separator as PanelResizeHandle,
} from "react-resizable-panels";
import { issueServiceClientConnect, planServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import { Button } from "@/components/ui/button";
import {
  Collapsible,
  CollapsiblePanel,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Input } from "@/components/ui/input";
import {
  getLayerRoot,
  usePreserveHigherLayerAccess,
} from "@/components/ui/layer";
import {
  getProjectName,
  projectNamePrefix,
  workspaceNamePrefix,
} from "@/lib/resourceName";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app";
import { State } from "@/types/proto-es/v1/common_pb";
import { GetIssueRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import { GetPlanRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { extractDatabaseResourceName } from "@/utils/v1/database";
import { getAgentAIConfigurationEnabled } from "../logic/aiConfiguration";
import {
  type AgentPageResource,
  getCurrentAgentPageResource,
  getCurrentChatContext,
} from "../logic/chatContext";
import {
  type AgentPageHelpRequest,
  canStartAgentPageHelp,
  isAgentPageHelpRequestCurrent,
  subscribeAgentPageHelp,
} from "../logic/contextualHelp";
import {
  getLandingStarterIdForText,
  type LandingStarterId,
} from "../logic/landingStarters";
import {
  formatResourceMention,
  getDatabaseMentionLabel,
  getDraftResourceMentions,
  getPlanMentionLabel,
  normalizeAgentResourceReference,
  normalizeAgentResourceReferences,
} from "../logic/resourceMention";
import { normalizeAgentResponseActions } from "../logic/responseActions";
import type {
  AgentChat as AgentChatRecord,
  AgentDraft,
  AgentResourceReference,
  AgentRunMode,
} from "../logic/types";
import {
  selectContinueChatForContext,
  selectCurrentChat,
  selectHasRunningChat,
  selectOrderedChats,
  useAgentStore,
} from "../store/agent";
import {
  MIN_HEIGHT,
  MIN_MAIN_PANEL_WIDTH,
  MIN_WIDTH,
  WINDOW_MARGIN,
} from "../window";
import { AgentChat } from "./AgentChat";
import { AgentContextualStart } from "./AgentContextualStart";
import { AgentInput } from "./AgentInput";
import {
  type AgentLauncherPosition,
  AgentMinimizedLauncher,
} from "./AgentMinimizedLauncher";
import { AgentPreferences } from "./AgentPreferences";
import {
  RESIZE_POINTER_MEDIA_QUERY,
  supportsWindowBorderResize,
} from "./resize-capability";
import {
  AgentDialog,
  AgentDialogClose,
  AgentDialogContent,
  AgentDialogDescription,
  AgentDialogTitle,
} from "./ui/AgentDialog";
import {
  AgentDropdownMenu,
  AgentDropdownMenuContent,
  AgentDropdownMenuItem,
  AgentDropdownMenuTrigger,
} from "./ui/AgentDropdownMenu";
import {
  AgentPopover,
  AgentPopoverContent,
  AgentPopoverTitle,
} from "./ui/AgentPopover";
import { AgentTooltip } from "./ui/AgentTooltip";
import {
  type ResizeBounds,
  type ResizeDirection,
  resizeWindowBounds,
} from "./window-resize";

type AnchoredResizeDirection = Extract<ResizeDirection, "n" | "w" | "nw">;

const resizeZoneClasses: Record<AnchoredResizeDirection, string> = {
  n: "left-[12px] top-[-6px] h-[6px] w-[calc(100%-24px)] cursor-n-resize",
  w: "left-[-6px] top-[12px] h-[calc(100%-24px)] w-[6px] cursor-w-resize",
  nw: "left-[-6px] top-[-6px] size-[12px] cursor-nw-resize",
};

// The default 750px agent window leaves a useful reading column beside the
// persistent chat sidebar. Keep the compact rail for genuinely constrained
// desktop widths, rather than applying a browser-level breakpoint.
const COMPACT_WINDOW_MAX_WIDTH = 719;
// Compact remains viable down to the 460px minimum agent-window width. Reserve
// the single-pane mobile flow for genuinely phone-sized agent windows.
const MOBILE_WINDOW_MAX_WIDTH = 479;
const RAIL_WIDTH = 48;
const MIN_CHAT_LIST_WIDTH = 240;
const SIDEBAR_RESIZE_HANDLE_WIDTH = 4;

interface AgentChatGroup {
  context?: string;
  chats: AgentChatRecord[];
}

function groupChatsByContext(chats: AgentChatRecord[]): AgentChatGroup[] {
  const groups = new Map<string, AgentChatGroup>();
  for (const chat of chats) {
    const key = chat.context ?? "";
    const group = groups.get(key);
    if (group) {
      group.chats.push(chat);
      continue;
    }
    groups.set(key, { context: chat.context, chats: [chat] });
  }
  const unscopedGroup = groups.get("");
  const groupRank = (context: string) => {
    if (context.startsWith(workspaceNamePrefix)) return 0;
    if (context.startsWith(projectNamePrefix)) return 1;
    return 2;
  };
  return [
    ...[...groups.entries()]
      .filter(([context]) => context)
      .sort(([left], [right]) => {
        const rankDifference = groupRank(left) - groupRank(right);
        return rankDifference || right.localeCompare(left);
      })
      .map(([, group]) => group),
    ...(unscopedGroup ? [unscopedGroup] : []),
  ];
}

function getChatGroupStateKey(context: string | undefined, archived: boolean) {
  return `${archived ? "archived" : "active"}:${context ?? "unscoped"}`;
}

export function AgentWindow() {
  const { t } = useTranslation();

  const visible = useAgentStore((s) => s.visible);
  const minimized = useAgentStore((s) => s.minimized);
  const size = useAgentStore((s) => s.size);
  const sidebarWidth = useAgentStore((s) => s.sidebarWidth);
  const orderedChats = useAgentStore(selectOrderedChats);
  const currentChat = useAgentStore(selectCurrentChat);
  const currentChatId = useAgentStore((s) => s.currentChatId);
  const queuedChatId = useAgentStore((s) =>
    s.currentChatId && s.queuedUserMessagesByChatId[s.currentChatId]
      ? s.currentChatId
      : null
  );
  const hasRunningChat = useAgentStore(selectHasRunningChat);
  const storedWorkspaceName = useAgentStore((s) => s.workspaceName);
  const storedUserEmail = useAgentStore((s) => s.userEmail);
  const openingBehavior = useAgentStore((s) => s.preferences.openingBehavior);
  const getOrFetchSettingByName = useAppStore((s) => s.getOrFetchSettingByName);
  const aiEnabled = useAppStore((s) => s.serverInfo?.aiEnabled);
  const batchFetchProjects = useAppStore((s) => s.batchFetchProjects);
  const projectsByName = useAppStore((s) => s.projectsByName);
  const instancesByName = useAppStore((s) => s.instancesByName);
  const projectErrorsByName = useAppStore((s) => s.projectErrorsByName);
  const workspace = useAppStore((s) => s.workspace);
  const currentUser = useAppStore((s) => s.currentUser);
  const userEmail =
    currentUser && workspace && currentUser.workspace === workspace.name
      ? currentUser.email
      : "";

  useLayoutEffect(() => {
    useAgentStore.getState().setWorkspaceName(workspace?.name, userEmail);
  }, [hasRunningChat, userEmail, workspace?.name]);

  const windowRef = useRef<HTMLDivElement>(null);
  const bodyRef = useRef<HTMLDivElement>(null);
  const launcherPositionRef = useRef<AgentLauncherPosition | undefined>(
    undefined
  );
  const continuedChatIdRef = useRef<string | null>(null);
  const compactPanelTriggerRef = useRef<HTMLElement | null>(null);
  const renameInputRef = useRef<HTMLInputElement>(null);
  const renameFocusChatIdRef = useRef<string | null>(null);
  const preferencesTriggerRef = useRef<HTMLButtonElement>(null);
  const preferencesReturnFocusRef = useRef(false);
  const [preferencesOpen, setPreferencesOpen] = useState(false);

  useLayoutEffect(() => {
    if (!preferencesOpen && preferencesReturnFocusRef.current) {
      preferencesTriggerRef.current?.focus();
      preferencesReturnFocusRef.current = false;
    }
  }, [preferencesOpen]);

  const [viewportSize, setViewportSize] = useState({
    width: window.innerWidth,
    height: window.innerHeight,
  });
  const [supportsFinePointer, setSupportsFinePointer] = useState(() =>
    supportsWindowBorderResize(window.matchMedia.bind(window))
  );
  const isViewportResizingRef = useRef(false);
  const viewportResizeFrameRef = useRef(0);

  const [showArchivedOnly, setShowArchivedOnly] = useState(false);
  const [contextualStart, setContextualStart] =
    useState<AgentPageResource | null>(null);
  const [contextualDraft, setContextualDraft] = useState<AgentDraft>({
    text: "",
  });
  const contextualPageHelpRef = useRef<AgentPageHelpRequest | null>(null);
  const [contextualPageHelp, setContextualPageHelp] =
    useState<AgentPageHelpRequest | null>(null);
  const canStartPageHelp = useSyncExternalStore(
    useCallback(
      (listener) =>
        contextualPageHelp?.subscribeCanStart?.(listener) ?? (() => {}),
      [contextualPageHelp]
    ),
    useCallback(
      () => !contextualPageHelp || canStartAgentPageHelp(contextualPageHelp),
      [contextualPageHelp]
    )
  );
  const [contextualChatScope, setContextualChatScope] = useState<{
    context: string;
    workspaceName: string;
  } | null>(null);
  const continuableChat = useAgentStore((state) =>
    contextualStart &&
    contextualChatScope?.workspaceName === workspace?.name &&
    state.userEmail === userEmail
      ? selectContinueChatForContext(
          state,
          contextualChatScope?.context,
          contextualChatScope?.workspaceName
        )
      : null
  );
  const [contextualReference, setContextualReference] = useState<
    AgentResourceReference | undefined
  >();
  const [contextualLoading, setContextualLoading] = useState(false);
  const canReferenceContextualResource = useAppStore((state) => {
    if (
      !contextualStart ||
      contextualStart.kind === "project" ||
      (contextualStart.kind === "plan-create" &&
        !contextualStart.databaseName) ||
      contextualStart.kind === "instance" ||
      contextualStart.kind === "instance-create" ||
      contextualStart.kind === "global-masking" ||
      contextualStart.kind === "semantic-types" ||
      contextualStart.kind === "data-classification" ||
      contextualStart.kind === "masking-exemption"
    ) {
      return true;
    }
    if (contextualStart.kind === "plan") {
      if (state.hasWorkspacePermission("bb.plans.get")) {
        return true;
      }
      if (
        !state.projectPoliciesByName[contextualStart.project] ||
        !state.currentUser ||
        state.roles.length === 0 ||
        !state.workspacePolicy
      ) {
        return true;
      }
      const project =
        state.projectsByName[contextualStart.project] ??
        create(ProjectSchema, { name: contextualStart.project });
      return state.hasProjectPermission(project, "bb.plans.get");
    }
    const project = state.projectsByName[contextualStart.project];
    const isDatabaseReference =
      contextualStart.kind === "database" ||
      contextualStart.kind === "database-catalog" ||
      (contextualStart.kind === "plan-create" &&
        Boolean(contextualStart.databaseName));
    const permission = isDatabaseReference
      ? "bb.databases.list"
      : "bb.issues.list";
    const canList =
      state.hasWorkspacePermission(permission) ||
      Boolean(project && state.hasProjectPermission(project, permission));
    if (contextualStart.kind === "issue") return canList;
    return (
      canList &&
      (state.hasWorkspacePermission("bb.projects.get") ||
        Boolean(
          project && state.hasProjectPermission(project, "bb.projects.get")
        ))
    );
  });
  const openedOwnerRef = useRef<{
    workspaceName: string | null;
    userEmail: string | null;
    openingBehavior: "landing" | "continue";
  } | null>(null);
  const wasVisibleRef = useRef(false);
  const resetContextualStart = useCallback(
    (context?: string) => {
      contextualPageHelpRef.current = null;
      setContextualPageHelp(null);
      setContextualDraft({ text: "" });
      let pageResource = getCurrentAgentPageResource();
      const pageContext = pageResource
        ? ("project" in pageResource && pageResource.project) || workspace?.name
        : undefined;
      if (context && context !== pageContext) {
        pageResource = context.startsWith(projectNamePrefix)
          ? { kind: "project", name: context, project: context }
          : undefined;
      }
      setContextualStart(pageResource ?? null);
      setContextualChatScope(
        pageResource && workspace?.name && userEmail
          ? {
              context:
                ("project" in pageResource && pageResource.project) ||
                workspace.name,
              workspaceName: workspace.name,
            }
          : null
      );
      setContextualReference(undefined);
      setContextualLoading(false);
      return pageResource;
    },
    [userEmail, workspace?.name]
  );
  const [compactPanel, setCompactPanel] = useState<"chats" | "archived" | null>(
    null
  );
  const [mobileScreen, setMobileScreen] = useState<
    "chats" | "conversation" | "archived"
  >("chats");
  const [collapsedChatGroups, setCollapsedChatGroups] = useState<Set<string>>(
    () => new Set()
  );
  const [renamingChatId, setRenamingChatId] = useState<string | null>(null);
  const [pendingDeleteChatId, setPendingDeleteChatId] = useState<string | null>(
    null
  );
  const [
    isDeleteAllArchivedChatsDialogOpen,
    setIsDeleteAllArchivedChatsDialogOpen,
  ] = useState(false);
  const [renamingTitle, setRenamingTitle] = useState("");

  useEffect(() => {
    setPreferencesOpen(false);
  }, [storedWorkspaceName, storedUserEmail]);

  useEffect(() => {
    const isOpen = visible && !minimized;
    const store = useAgentStore.getState();
    const selected = store.currentChatId
      ? store.getChat(store.currentChatId)
      : undefined;
    const hasQueuedChat = Boolean(
      store.currentChatId &&
        store.queuedUserMessagesByChatId[store.currentChatId] &&
        !selected?.archived
    );
    const { workspaceName, userEmail: ownerEmail } = store;
    if (workspaceName !== workspace?.name || ownerEmail !== userEmail) return;
    const hasOpenedForOwner =
      openedOwnerRef.current?.workspaceName === workspaceName &&
      openedOwnerRef.current?.userEmail === ownerEmail;
    const openingPreferenceChanged =
      !wasVisibleRef.current &&
      openedOwnerRef.current?.openingBehavior !==
        store.preferences.openingBehavior;
    if (
      isOpen &&
      contextualPageHelpRef.current &&
      isAgentPageHelpRequestCurrent(contextualPageHelpRef.current)
    ) {
      if (!wasVisibleRef.current) {
        setPreferencesOpen(false);
      }
      setShowArchivedOnly(false);
      setRenamingChatId(null);
      setMobileScreen("conversation");
    } else if (isOpen && (!hasOpenedForOwner || openingPreferenceChanged)) {
      setPreferencesOpen(false);
      const pageResource = hasQueuedChat ? undefined : resetContextualStart();
      if (hasQueuedChat) setContextualStart(null);
      const context = pageResource
        ? ("project" in pageResource && pageResource.project) || workspaceName
        : getCurrentChatContext() || workspaceName;
      const lastChat =
        !hasQueuedChat && store.preferences.openingBehavior === "continue"
          ? selectContinueChatForContext(
              store,
              context ?? undefined,
              workspaceName ?? undefined
            )
          : null;
      if (lastChat) {
        continuedChatIdRef.current = lastChat.id;
        store.setCurrentChat(lastChat.id);
        if (useAgentStore.getState().currentChatId === lastChat.id) {
          setContextualStart(null);
        } else {
          continuedChatIdRef.current = null;
        }
      }
      if (!pageResource && !hasQueuedChat && !lastChat) {
        const selected = store.currentChatId
          ? store.getChat(store.currentChatId)
          : undefined;
        const draft = store.getDraft();
        if (
          selected &&
          !selected.archived &&
          !store.isChatRunning(selected.id) &&
          selected.status !== "awaiting_user" &&
          !draft.text &&
          !draft.mentions?.length &&
          !draft.project &&
          !draft.resource &&
          store.getMessages(selected.id).length > 0
        ) {
          store.createChat({
            context: context ?? undefined,
            workspaceName: workspaceName ?? undefined,
            page: {
              path: `${window.location.pathname}${window.location.search}${window.location.hash}`,
              title: document.title,
            },
          });
        }
      }
      if (pageResource || hasQueuedChat || lastChat) {
        setShowArchivedOnly(false);
        setRenamingChatId(null);
        setMobileScreen("conversation");
      }
    } else if (isOpen && hasQueuedChat) {
      setPreferencesOpen(false);
      setContextualStart(null);
      setShowArchivedOnly(false);
      setMobileScreen("conversation");
    }
    if (isOpen && (!hasOpenedForOwner || !wasVisibleRef.current)) {
      openedOwnerRef.current = {
        workspaceName,
        userEmail: ownerEmail,
        openingBehavior: store.preferences.openingBehavior,
      };
    }
    wasVisibleRef.current = visible;
  }, [
    contextualPageHelp,
    currentChat?.archived,
    currentChatId,
    minimized,
    openingBehavior,
    queuedChatId,
    resetContextualStart,
    userEmail,
    visible,
    workspace?.name,
  ]);

  useEffect(() => {
    const receiveRequest = (request: AgentPageHelpRequest) => {
      if (!isAgentPageHelpRequestCurrent(request)) {
        if (contextualPageHelpRef.current === request) {
          contextualPageHelpRef.current = null;
          setContextualPageHelp(null);
          setContextualStart(null);
        }
        return;
      }
      contextualPageHelpRef.current = request;
      setPreferencesOpen(false);
      setContextualPageHelp(request);
      setContextualStart(request.page);
      setContextualDraft({ text: "" });
      setContextualChatScope({
        context:
          ("project" in request.page && request.page.project) ||
          request.workspaceName,
        workspaceName: request.workspaceName,
      });
      setContextualReference(undefined);
      setContextualLoading(false);
      setShowArchivedOnly(false);
      setRenamingChatId(null);
      setMobileScreen("conversation");
      setCompactPanel(null);
    };
    const activeRequest = contextualPageHelpRef.current;
    if (activeRequest && !isAgentPageHelpRequestCurrent(activeRequest)) {
      receiveRequest(activeRequest);
    }
    return subscribeAgentPageHelp(receiveRequest);
  }, [storedUserEmail, storedWorkspaceName]);

  useEffect(() => {
    if (
      !contextualStart ||
      contextualStart.kind === "project" ||
      (contextualStart.kind === "plan-create" &&
        !contextualStart.databaseName) ||
      contextualStart.kind === "instance" ||
      contextualStart.kind === "instance-create" ||
      contextualStart.kind === "global-masking" ||
      contextualStart.kind === "semantic-types" ||
      contextualStart.kind === "data-classification" ||
      contextualStart.kind === "masking-exemption" ||
      !canReferenceContextualResource
    ) {
      return;
    }
    let cancelled = false;
    setContextualReference(undefined);
    setContextualLoading(true);
    const load = async () => {
      if (contextualStart.kind === "issue") {
        const issue = await issueServiceClientConnect.getIssue(
          create(GetIssueRequestSchema, { name: contextualStart.name })
        );
        return normalizeAgentResourceReference({
          kind: "issue",
          name: issue.name,
          project: contextualStart.project,
          label: issue.title,
        });
      }
      if (contextualStart.kind === "plan") {
        const plan = await planServiceClientConnect.getPlan(
          create(GetPlanRequestSchema, { name: contextualStart.name }),
          { contextValues: createContextValues().set(silentContextKey, true) }
        );
        if (plan.name !== contextualStart.name || plan.state !== State.ACTIVE) {
          return undefined;
        }
        return normalizeAgentResourceReference({
          kind: "plan",
          name: plan.name,
          project: contextualStart.project,
          label: getPlanMentionLabel(plan),
        });
      }
      const databaseNames =
        contextualStart.kind === "database" ||
        contextualStart.kind === "database-catalog"
          ? contextualStart.names
          : contextualStart.kind === "plan-create" &&
              contextualStart.databaseName
            ? [contextualStart.databaseName]
            : [];
      for (const name of databaseNames) {
        const database = await useAppStore
          .getState()
          .getOrFetchDatabaseByName(name, true);
        if (
          database.name !== name ||
          database.project !== contextualStart.project
        ) {
          continue;
        }
        return normalizeAgentResourceReference({
          kind: "database",
          name: database.name,
          project: database.project,
          label: getDatabaseMentionLabel(database),
        });
      }
      return undefined;
    };
    void load()
      .then((reference) => {
        if (!cancelled) {
          setContextualReference(reference);
          setContextualLoading(false);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setContextualReference(undefined);
          setContextualLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [canReferenceContextualResource, contextualStart]);

  // Refs for resize intermediate values (avoid re-renders)
  const isResizingRef = useRef(false);
  const resizeStartRef = useRef<{
    x: number;
    y: number;
    bounds: ResizeBounds;
    direction: AnchoredResizeDirection;
  }>({
    x: 0,
    y: 0,
    bounds: { x: 0, y: 0, width: 0, height: 0 },
    direction: "nw",
  });
  const resizeObserverRef = useRef<ResizeObserver | null>(null);

  // --- Clamping helpers ---
  //
  // On wide viewports we keep MIN_WIDTH/MIN_HEIGHT as the preferred floor so
  // the sidebar and main panel both hit their minimums. On narrow viewports
  // (split panes, mobile-like widths), forcing those minimums would render
  // the window partially off-screen — so the floor drops to whatever the
  // viewport can accommodate minus the standard margins. At very narrow
  // widths the main panel gets cramped, but that's strictly better than
  // hiding controls.

  const maxWidth = useCallback(
    () => Math.max(1, viewportSize.width - WINDOW_MARGIN * 2),
    [viewportSize.width]
  );
  const maxHeight = useCallback(
    () => Math.max(1, viewportSize.height - WINDOW_MARGIN * 2),
    [viewportSize.height]
  );

  const clampWidth = useCallback(
    (width: number) => {
      const max = maxWidth();
      const min = Math.min(MIN_WIDTH, max);
      return Math.min(max, Math.max(min, Math.round(width)));
    },
    [maxWidth]
  );
  const clampHeight = useCallback(
    (height: number) => {
      const max = maxHeight();
      const min = Math.min(MIN_HEIGHT, max);
      return Math.min(max, Math.max(min, Math.round(height)));
    },
    [maxHeight]
  );

  const getDisplaySize = useCallback(
    (w: number, h: number) => ({
      width: clampWidth(w),
      height: clampHeight(h),
    }),
    [clampWidth, clampHeight]
  );

  const displaySize = useMemo(
    () => getDisplaySize(size.width, size.height),
    [getDisplaySize, size.height, size.width]
  );

  const clampSidebarWidth = useCallback(
    (width: number, windowWidth = displaySize.width) => {
      const maxSW = Math.max(
        MIN_CHAT_LIST_WIDTH,
        windowWidth -
          RAIL_WIDTH -
          MIN_MAIN_PANEL_WIDTH -
          SIDEBAR_RESIZE_HANDLE_WIDTH
      );
      const minSW = Math.min(MIN_CHAT_LIST_WIDTH, maxSW);
      return Math.min(maxSW, Math.max(minSW, Math.round(width)));
    },
    [displaySize.width]
  );

  const clampedSidebarWidth = useMemo(
    () => clampSidebarWidth(sidebarWidth),
    [clampSidebarWidth, sidebarWidth]
  );

  const windowStyle: React.CSSProperties = {
    right: `${WINDOW_MARGIN}px`,
    bottom: `${WINDOW_MARGIN}px`,
    width: `${displaySize.width}px`,
    height: `${displaySize.height}px`,
  };

  const maxSidebarWidth = Math.max(
    MIN_CHAT_LIST_WIDTH,
    displaySize.width -
      RAIL_WIDTH -
      MIN_MAIN_PANEL_WIDTH -
      SIDEBAR_RESIZE_HANDLE_WIDTH
  );
  const minSidebarWidth = Math.min(MIN_CHAT_LIST_WIDTH, maxSidebarWidth);
  const minMainPanelWidth = Math.min(
    MIN_MAIN_PANEL_WIDTH,
    Math.max(
      1,
      displaySize.width -
        RAIL_WIDTH -
        minSidebarWidth -
        SIDEBAR_RESIZE_HANDLE_WIDTH
    )
  );
  const isMobileWindow = displaySize.width <= MOBILE_WINDOW_MAX_WIDTH;
  const isCompactWindow =
    displaySize.width <= COMPACT_WINDOW_MAX_WIDTH && !isMobileWindow;
  const isCompactDrawerOpen = isCompactWindow && compactPanel !== null;

  // --- Derived chat data ---

  const displayedChats = useMemo(
    () => orderedChats.filter((chat) => chat.archived === showArchivedOnly),
    [orderedChats, showArchivedOnly]
  );
  const displayedChatGroups = useMemo(
    () => groupChatsByContext(displayedChats),
    [displayedChats]
  );
  const missingProjectContexts = useMemo(
    () =>
      [
        ...new Set(
          displayedChatGroups.flatMap((group) =>
            group.context?.startsWith(projectNamePrefix) ? [group.context] : []
          )
        ),
      ].filter(
        (context) => !projectsByName[context] && !projectErrorsByName[context]
      ),
    [displayedChatGroups, projectErrorsByName, projectsByName]
  );
  const activeChats = useMemo(
    () => orderedChats.filter((chat) => !chat.archived),
    [orderedChats]
  );
  const archivedChats = useMemo(
    () => orderedChats.filter((chat) => chat.archived),
    [orderedChats]
  );

  const isArchiveAllDisabled =
    hasRunningChat || showArchivedOnly || activeChats.length === 0;
  const isUnarchiveAllDisabled =
    hasRunningChat || !showArchivedOnly || archivedChats.length === 0;
  const isDeleteAllDisabled =
    hasRunningChat || !showArchivedOnly || archivedChats.length === 0;
  const canResizeWindow = supportsFinePointer;

  useEffect(() => {
    if (!visible || missingProjectContexts.length === 0) return;
    void batchFetchProjects(missingProjectContexts, true).catch(() => {
      // Keep the stable resource-id fallback when a context cannot be loaded.
    });
  }, [batchFetchProjects, missingProjectContexts, visible]);

  const syncStoreToDisplaySize = useCallback(() => {
    const store = useAgentStore.getState();
    store.setSize(displaySize.width, displaySize.height);
  }, [displaySize]);

  // --- Chat helpers ---

  const getCurrentPageSnapshot = useCallback(
    () => ({
      path: window.location.pathname + window.location.hash,
      title: document.title,
    }),
    []
  );

  const getEditableChatTitle = useCallback(
    (chat: AgentChatRecord) => chat.title || t("agent.chat-default-title"),
    [t]
  );

  const getContextLabel = useCallback(
    (context: string) => {
      if (context.startsWith(projectNamePrefix)) {
        return projectsByName[context]?.title || getProjectName(context);
      }
      if (context === workspace?.name) {
        return workspace.title || workspace.name;
      }
      return context;
    },
    [projectsByName, workspace]
  );

  const contextualTitle = contextualStart
    ? contextualStart.kind === "project"
      ? getContextLabel(contextualStart.project)
      : contextualStart.kind === "plan-create"
        ? getContextLabel(contextualStart.project)
        : contextualStart.kind === "instance-create"
          ? contextualStart.project
            ? getContextLabel(contextualStart.project)
            : t("agent.contextual-start.instance-create-title")
          : contextualStart.kind === "global-masking"
            ? t("settings.sidebar.global-masking")
            : contextualStart.kind === "semantic-types"
              ? t("settings.sensitive-data.semantic-types.self")
              : contextualStart.kind === "data-classification"
                ? t("settings.sidebar.data-classification")
                : contextualStart.kind === "masking-exemption"
                  ? getContextLabel(contextualStart.project)
                  : (contextualReference?.label ??
                    (contextualStart.kind === "database" ||
                    contextualStart.kind === "database-catalog"
                      ? extractDatabaseResourceName(contextualStart.names[0])
                          .databaseName
                      : contextualStart.kind === "instance"
                        ? instancesByName?.[contextualStart.name]?.title ||
                          `#${contextualStart.name.split("/").at(-1)}`
                        : `#${contextualStart.name.split("/").at(-1)}`))
    : "";

  const selectFirstDisplayedChat = useCallback(() => {
    const store = useAgentStore.getState();
    const first = selectOrderedChats(store).find(
      (chat) => chat.archived === showArchivedOnly
    );
    if (!first || !useAgentStore.getState().canSelectChat(first.id))
      return false;
    useAgentStore.getState().setCurrentChat(first.id);
    return useAgentStore.getState().currentChatId === first.id;
  }, [showArchivedOnly]);

  const ensureCurrentChatMatchesDisplayedMode = useCallback(
    (
      options: {
        allowCreateWhenEmpty?: boolean;
        fallbackToActiveWhenEmpty?: boolean;
      } = {}
    ) => {
      const store = useAgentStore.getState();
      const chat = store.chats.find((c) => c.id === store.currentChatId);
      if (chat?.status === "running") return;
      const isInDisplayedMode = !!chat && chat.archived === showArchivedOnly;

      if (isInDisplayedMode) return;
      if (selectFirstDisplayedChat()) return;

      if (showArchivedOnly) {
        if (!options.fallbackToActiveWhenEmpty) {
          store.clearCurrentChat();
          return;
        }
        setShowArchivedOnly(false);
        // After toggling, check again with active mode
        const storeNow = useAgentStore.getState();
        const chatNow = storeNow.chats.find(
          (c) => c.id === storeNow.currentChatId
        );
        const isInActiveMode = !!chatNow && !chatNow.archived;
        if (isInActiveMode) return;
        // Try selecting first active chat
        const activeChats = [...storeNow.chats]
          .filter((c) => !c.archived)
          .sort((a, b) => b.updatedTs - a.updatedTs);
        if (activeChats[0] && storeNow.canSelectChat(activeChats[0].id)) {
          storeNow.setCurrentChat(activeChats[0].id);
          return;
        }
      }

      if (options.allowCreateWhenEmpty !== true || showArchivedOnly) {
        store.clearCurrentChat();
        return;
      }

      store.createChat({
        page: getCurrentPageSnapshot(),
        context: getCurrentChatContext(),
        workspaceName: workspace?.name,
      });
    },
    [
      showArchivedOnly,
      selectFirstDisplayedChat,
      getCurrentPageSnapshot,
      workspace?.name,
    ]
  );

  // --- Rename ---

  const beginRenameChat = useCallback(
    (chatId: string) => {
      const chat = useAgentStore.getState().getChat(chatId);
      if (!chat) return;
      setRenamingTitle(getEditableChatTitle(chat));
      setRenamingChatId(chat.id);
    },
    [getEditableChatTitle]
  );

  const cancelRenameChat = useCallback(
    (restoreFocus = false) => {
      renameFocusChatIdRef.current = restoreFocus ? renamingChatId : null;
      setRenamingChatId(null);
      setRenamingTitle("");
    },
    [renamingChatId]
  );

  const commitRenameChat = useCallback(
    (restoreFocus = true) => {
      const chat = useAgentStore.getState().getChat(renamingChatId);
      const title = renamingTitle.trim();
      if (chat && title && title !== getEditableChatTitle(chat)) {
        useAgentStore.getState().renameChat(chat.id, title);
      }
      cancelRenameChat(restoreFocus);
    },
    [renamingTitle, renamingChatId, getEditableChatTitle, cancelRenameChat]
  );

  const onRenameKeydown = useCallback(
    (event: React.KeyboardEvent<HTMLInputElement>) => {
      if (event.nativeEvent.isComposing) return;
      if (event.key === "Escape") {
        event.preventDefault();
        event.stopPropagation();
        cancelRenameChat(true);
        return;
      }
      if (event.key === "Enter") {
        event.preventDefault();
        event.stopPropagation();
        commitRenameChat();
      }
    },
    [cancelRenameChat, commitRenameChat]
  );

  useEffect(() => {
    setRenamingChatId(null);
    setRenamingTitle("");
  }, [currentChatId, storedWorkspaceName, storedUserEmail]);

  useEffect(() => {
    if (!visible || minimized) return;
    if (renamingChatId && renameInputRef.current) {
      renameInputRef.current.focus();
      renameInputRef.current.select();
    } else if (!renamingChatId && renameFocusChatIdRef.current) {
      windowRef.current
        ?.querySelector<HTMLButtonElement>(
          `[data-agent-chat-row="${renameFocusChatIdRef.current}"] [data-agent-chat-menu]`
        )
        ?.focus();
      renameFocusChatIdRef.current = null;
    }
  }, [
    renamingChatId,
    isMobileWindow,
    isCompactWindow,
    compactPanel,
    mobileScreen,
    visible,
    minimized,
  ]);

  useEffect(() => {
    if (
      renamingChatId &&
      !displayedChats.some((chat) => chat.id === renamingChatId)
    ) {
      cancelRenameChat();
    }
  }, [renamingChatId, displayedChats, cancelRenameChat]);

  // --- Chat actions ---

  const setChatGroupOpen = useCallback(
    (
      context: string | undefined,
      open: boolean,
      archived = showArchivedOnly
    ) => {
      const groupKey = getChatGroupStateKey(context, archived);
      setCollapsedChatGroups((previous) => {
        const isOpen = !previous.has(groupKey);
        if (isOpen === open) return previous;

        const next = new Set(previous);
        if (open) {
          next.delete(groupKey);
        } else {
          next.add(groupKey);
        }
        return next;
      });
    },
    [showArchivedOnly]
  );

  const createChat = useCallback(() => {
    const context = getCurrentChatContext();
    const pageResource = resetContextualStart(context);
    setChatGroupOpen(context, true, false);
    if (pageResource) {
      useAgentStore.getState().clearCurrentChat();
    } else {
      useAgentStore.getState().createChat({
        page: getCurrentPageSnapshot(),
        context,
        workspaceName: workspace?.name,
      });
    }
    setShowArchivedOnly(false);
    if (isCompactWindow) setCompactPanel(null);
  }, [
    getCurrentPageSnapshot,
    isCompactWindow,
    resetContextualStart,
    setChatGroupOpen,
    workspace?.name,
  ]);

  const createChatForContext = useCallback(
    (context: string) => {
      const pageResource = resetContextualStart(context);
      setChatGroupOpen(context, true, false);
      if (pageResource) {
        useAgentStore.getState().clearCurrentChat();
      } else {
        useAgentStore.getState().createChat({
          page: getCurrentPageSnapshot(),
          context,
          workspaceName: workspace?.name,
        });
      }
      if (isCompactWindow) setCompactPanel(null);
    },
    [
      getCurrentPageSnapshot,
      isCompactWindow,
      resetContextualStart,
      setChatGroupOpen,
      workspace?.name,
    ]
  );

  const handleChatRowClick = useCallback(
    (chatId: string) => {
      commitRenameChat(false);
      contextualPageHelpRef.current = null;
      setContextualPageHelp(null);
      if (contextualStart) {
        setContextualStart(null);
        if (chatId === useAgentStore.getState().currentChatId) return;
      }
      if (chatId === useAgentStore.getState().currentChatId) {
        return;
      }
      useAgentStore.getState().setCurrentChat(chatId);
      if (isCompactWindow) setCompactPanel(null);
    },
    [isCompactWindow, contextualStart, commitRenameChat]
  );

  const archiveChat = useCallback(
    (chatId: string) => {
      const store = useAgentStore.getState();
      const chat = store.chats.find((c) => c.id === chatId);
      if (!chat || chat.archived) return;
      const wasCurrentChat = store.currentChatId === chat.id;
      store.archiveChat(chat.id);
      if (wasCurrentChat) {
        ensureCurrentChatMatchesDisplayedMode({ allowCreateWhenEmpty: false });
      }
    },
    [ensureCurrentChatMatchesDisplayedMode]
  );

  const unarchiveChat = useCallback(
    (chatId: string) => {
      const store = useAgentStore.getState();
      const chat = store.chats.find((c) => c.id === chatId);
      if (!chat?.archived) return;
      const wasCurrentChat = store.currentChatId === chat.id;
      store.unarchiveChat(chat.id);
      if (wasCurrentChat) {
        ensureCurrentChatMatchesDisplayedMode({ allowCreateWhenEmpty: false });
      }
    },
    [ensureCurrentChatMatchesDisplayedMode]
  );

  const restoreCurrentChat = useCallback(() => {
    const store = useAgentStore.getState();
    const chat = currentChatId ? store.getChat(currentChatId) : undefined;
    if (!chat?.archived || store.currentChatId !== chat.id) return;
    store.unarchiveChat(chat.id);
    setShowArchivedOnly(false);
    setCompactPanel(null);
    setMobileScreen("conversation");
  }, [currentChatId]);

  const archiveAllChats = useCallback(() => {
    if (isArchiveAllDisabled) return;
    const store = useAgentStore.getState();
    const currentChatWasActive = store.chats.some(
      (chat) => chat.id === store.currentChatId && !chat.archived
    );
    const archivedCount = store.archiveAllActiveChats();
    if (archivedCount === 0) return;
    if (currentChatWasActive) {
      cancelRenameChat();
      ensureCurrentChatMatchesDisplayedMode({ allowCreateWhenEmpty: false });
    }
  }, [
    cancelRenameChat,
    ensureCurrentChatMatchesDisplayedMode,
    isArchiveAllDisabled,
  ]);

  const unarchiveAllChats = useCallback(() => {
    if (isUnarchiveAllDisabled) return;
    const store = useAgentStore.getState();
    const currentChatWasArchived = store.chats.some(
      (chat) => chat.id === store.currentChatId && chat.archived
    );
    const unarchivedCount = store.unarchiveAllArchivedChats();
    if (unarchivedCount === 0) return;
    if (currentChatWasArchived) {
      cancelRenameChat();
      ensureCurrentChatMatchesDisplayedMode({ allowCreateWhenEmpty: false });
    }
  }, [
    cancelRenameChat,
    ensureCurrentChatMatchesDisplayedMode,
    isUnarchiveAllDisabled,
  ]);

  const deleteAllArchivedChats = useCallback(() => {
    if (isDeleteAllDisabled) return;
    const store = useAgentStore.getState();
    const currentChatWasArchived = store.chats.some(
      (chat) => chat.id === store.currentChatId && chat.archived
    );
    const deletedCount = store.deleteAllArchivedChats();
    if (deletedCount === 0) return;
    setIsDeleteAllArchivedChatsDialogOpen(false);
    if (currentChatWasArchived) {
      cancelRenameChat();
      ensureCurrentChatMatchesDisplayedMode({ allowCreateWhenEmpty: false });
    }
  }, [
    cancelRenameChat,
    ensureCurrentChatMatchesDisplayedMode,
    isDeleteAllDisabled,
  ]);

  const deleteChat = useCallback(
    (chatId: string) => {
      const store = useAgentStore.getState();
      const chat = store.chats.find((c) => c.id === chatId);
      if (!chat) return;
      const wasCurrentChat = store.currentChatId === chat.id;
      store.deleteChat(chat.id);
      if (wasCurrentChat) {
        ensureCurrentChatMatchesDisplayedMode({ allowCreateWhenEmpty: false });
      }
    },
    [ensureCurrentChatMatchesDisplayedMode]
  );

  const sidebarPanelRef = useRef<PanelImperativeHandle | null>(null);
  const resizeCleanupRef = useRef<(() => void) | null>(null);
  const retryLastTurnRef = useRef<(() => void) | null>(null);
  const retryLastTurn = useCallback(() => retryLastTurnRef.current?.(), []);
  const resendMessageRef = useRef<((messageId: string) => void) | null>(null);
  const resendMessage = useCallback(
    (messageId: string) => resendMessageRef.current?.(messageId),
    []
  );
  const focusInputRef = useRef<((atEnd?: boolean) => void) | null>(null);
  const focusContextualDraftRef = useRef(false);
  useEffect(() => {
    if (contextualStart || !focusContextualDraftRef.current) return;
    focusContextualDraftRef.current = false;
    focusInputRef.current?.(true);
  }, [contextualStart, currentChatId]);
  const selectEmptyChatSuggestion = useCallback(
    (starterId: LandingStarterId) => {
      const store = useAgentStore.getState();
      const chat = store.getChat(store.currentChatId);
      if (
        !chat ||
        chat.status !== "idle" ||
        store.getMessages(chat.id).length ||
        store.getDraft(chat.id).text.trim() ||
        store.queuedUserMessagesByChatId[chat.id]
      ) {
        return;
      }
      store.queueUserMessage({
        chatId: chat.id,
        text: t(`agent.landing-starters.${starterId}`),
        page: getCurrentPageSnapshot(),
        runMode: "read-only",
        starterId,
      });
    },
    [getCurrentPageSnapshot, t]
  );
  const continueResponseAction = useCallback(
    (chatId: string, prompt: string, references: AgentResourceReference[]) => {
      const store = useAgentStore.getState();
      const chat = store.getChat(chatId);
      const draft = store.getDraft(chatId);
      if (
        store.currentChatId !== chatId ||
        !chat ||
        chat.status !== "idle" ||
        chat.archived ||
        draft.text.trim() ||
        draft.project ||
        draft.resource ||
        draft.mentions?.length ||
        store.queuedUserMessagesByChatId[chatId]
      ) {
        return;
      }
      const action = normalizeAgentResponseActions([
        { kind: "continue", prompt },
      ])[0];
      if (action?.kind !== "continue") return;
      const boundReferences = normalizeAgentResourceReferences(references);
      if (boundReferences.length !== references.length) return;

      const prefix = boundReferences
        .map((reference) => formatResourceMention(reference.label))
        .join(" ");
      store.queueUserMessage({
        chatId,
        text: `${prefix}${prefix ? " " : ""}${action.prompt}`,
        page: getCurrentPageSnapshot(),
        runMode: "default",
        ...(boundReferences.length > 0
          ? { resourceReferences: boundReferences }
          : {}),
      });
    },
    [getCurrentPageSnapshot]
  );
  const startContextualChat = useCallback(
    (
      prompt: string,
      reference?: AgentResourceReference,
      runMode: AgentRunMode = "read-only",
      resources?: AgentResourceReference[]
    ) => {
      if (!contextualStart) return;
      const pageHelp = contextualPageHelpRef.current;
      if (pageHelp) {
        if (!canStartAgentPageHelp(pageHelp)) return;
        if (pageHelp.onStart) {
          if (resources?.length) return;
          let started = false;
          try {
            started = pageHelp.onStart(prompt);
          } catch {
            return;
          }
          if (!started) return;
          const chat = selectCurrentChat(useAgentStore.getState());
          if (chat?.context) setChatGroupOpen(chat.context, true, false);
          contextualPageHelpRef.current = null;
          setContextualPageHelp(null);
          setContextualStart(null);
          setContextualDraft({ text: "" });
          setShowArchivedOnly(false);
          if (isCompactWindow) setCompactPanel(null);
          if (isMobileWindow) setMobileScreen("conversation");
          return;
        }
      }
      const requiresReference =
        contextualStart.kind === "database" ||
        contextualStart.kind === "database-catalog" ||
        contextualStart.kind === "plan" ||
        contextualStart.kind === "issue" ||
        (contextualStart.kind === "plan-create" &&
          Boolean(contextualStart.databaseName));
      if (requiresReference && !reference) return;
      const requestedReferences = resources ?? (reference ? [reference] : []);
      const references =
        requestedReferences.length > 0
          ? normalizeAgentResourceReferences(requestedReferences)
          : [];
      if (references.length !== requestedReferences.length) return;
      if (
        reference &&
        !references.some(
          (resource) =>
            resource.kind === reference.kind &&
            resource.name === reference.name &&
            resource.project === reference.project
        )
      )
        return;
      const chatContext =
        ("project" in contextualStart && contextualStart.project) ||
        pageHelp?.workspaceName ||
        getCurrentChatContext();
      if (!chatContext) return;
      const store = useAgentStore.getState();
      const page = getCurrentPageSnapshot();
      const chat = store.createChat({
        page,
        context: chatContext,
        workspaceName: workspace?.name,
      });
      if (useAgentStore.getState().currentChatId !== chat.id) return;
      const hasBoundMention =
        runMode === "default" &&
        reference &&
        getDraftResourceMentions(contextualDraft).some(
          ({ resource }) =>
            resource.kind === reference.kind &&
            resource.name === reference.name &&
            resource.project === reference.project
        );
      const mention =
        reference && !hasBoundMention
          ? formatResourceMention(reference.label)
          : "";
      if (prompt) {
        store.queueUserMessage({
          chatId: chat.id,
          text: mention ? `${mention} ${prompt}` : prompt,
          page,
          runMode,
          ...(references[0] ? { resourceReference: references[0] } : {}),
          ...(references.length > 1 ? { resourceReferences: references } : {}),
          ...(contextualStart.kind === "project" && runMode === "read-only"
            ? {
                starterId: getLandingStarterIdForText({
                  context: chatContext,
                  text: prompt,
                  translate: t,
                }),
              }
            : {}),
        });
      } else {
        store.setDraft(chat.id, {
          text: `${mention} `,
          ...(reference
            ? {
                mentions: [
                  {
                    resource: reference,
                    range: { start: 0, end: mention.length },
                  },
                ],
              }
            : {}),
        });
        focusContextualDraftRef.current = true;
      }
      setChatGroupOpen(chatContext, true, false);
      setShowArchivedOnly(false);
      contextualPageHelpRef.current = null;
      setContextualPageHelp(null);
      setContextualStart(null);
      setContextualDraft({ text: "" });
      if (isCompactWindow) setCompactPanel(null);
      if (isMobileWindow) setMobileScreen("conversation");
    },
    [
      contextualStart,
      contextualDraft,
      getCurrentPageSnapshot,
      isCompactWindow,
      isMobileWindow,
      setChatGroupOpen,
      t,
      workspace?.name,
    ]
  );

  const continueCurrentChat = useCallback(() => {
    if (
      !contextualStart ||
      !contextualChatScope ||
      contextualChatScope.workspaceName !== workspace?.name
    ) {
      return;
    }
    const store = useAgentStore.getState();
    if (store.userEmail !== userEmail) return;
    const chat = selectContinueChatForContext(
      store,
      contextualChatScope.context,
      contextualChatScope.workspaceName
    );
    if (!chat) return;
    continuedChatIdRef.current = chat.id;
    store.setCurrentChat(chat.id);
    if (useAgentStore.getState().currentChatId !== chat.id) {
      continuedChatIdRef.current = null;
      return;
    }
    setChatGroupOpen(contextualChatScope.context, true, false);
    setShowArchivedOnly(false);
    setContextualStart(null);
    setContextualDraft({ text: "" });
    if (isCompactWindow) setCompactPanel(null);
    if (isMobileWindow) setMobileScreen("conversation");
  }, [
    contextualChatScope,
    contextualStart,
    isCompactWindow,
    isMobileWindow,
    setChatGroupOpen,
    userEmail,
    workspace?.name,
  ]);

  // Cleanup resize listeners on unmount.
  useEffect(() => {
    return () => {
      resizeCleanupRef.current?.();
    };
  }, []);

  // --- Resize handlers ---

  const startResize = useCallback(
    (
      direction: AnchoredResizeDirection,
      event: React.PointerEvent<HTMLDivElement>
    ) => {
      if (!canResizeWindow || event.button !== 0) return;
      event.preventDefault();
      event.stopPropagation();
      syncStoreToDisplaySize();
      isResizingRef.current = true;
      resizeStartRef.current = {
        x: event.clientX,
        y: event.clientY,
        bounds: {
          x: window.innerWidth - displaySize.width - WINDOW_MARGIN,
          y: window.innerHeight - displaySize.height - WINDOW_MARGIN,
          width: displaySize.width,
          height: displaySize.height,
        },
        direction,
      };

      const onResize = (e: PointerEvent) => {
        if (!isResizingRef.current || !windowRef.current) return;
        const el = windowRef.current;
        const nextBounds = resizeWindowBounds({
          direction: resizeStartRef.current.direction,
          startBounds: resizeStartRef.current.bounds,
          deltaX: e.clientX - resizeStartRef.current.x,
          deltaY: e.clientY - resizeStartRef.current.y,
          constraints: {
            minWidth: MIN_WIDTH,
            minHeight: MIN_HEIGHT,
            viewportWidth: window.innerWidth,
            viewportHeight: window.innerHeight,
            margin: WINDOW_MARGIN,
          },
        });
        el.style.width = `${nextBounds.width}px`;
        el.style.height = `${nextBounds.height}px`;
        useAgentStore.getState().setSize(nextBounds.width, nextBounds.height);
      };

      const stopResize = () => {
        isResizingRef.current = false;
        document.removeEventListener("pointermove", onResize);
        document.removeEventListener("pointerup", stopResize);
        document.removeEventListener("pointercancel", stopResize);
        resizeCleanupRef.current = null;
        useAgentStore.getState().saveWindowState();
      };

      document.addEventListener("pointermove", onResize);
      document.addEventListener("pointerup", stopResize);
      document.addEventListener("pointercancel", stopResize);
      resizeCleanupRef.current = stopResize;
    },
    [canResizeWindow, displaySize, syncStoreToDisplaySize]
  );

  // --- Viewport resize ---

  useEffect(() => {
    const handler = () => {
      isViewportResizingRef.current = true;
      setViewportSize({
        width: window.innerWidth,
        height: window.innerHeight,
      });
      cancelAnimationFrame(viewportResizeFrameRef.current);
      viewportResizeFrameRef.current = window.requestAnimationFrame(() => {
        isViewportResizingRef.current = false;
      });
    };
    window.addEventListener("resize", handler);
    return () => {
      window.removeEventListener("resize", handler);
      cancelAnimationFrame(viewportResizeFrameRef.current);
    };
  }, []);

  useEffect(() => {
    const mediaQuery = window.matchMedia(RESIZE_POINTER_MEDIA_QUERY);
    const update = () => {
      setSupportsFinePointer(
        supportsWindowBorderResize(window.matchMedia.bind(window))
      );
    };
    update();
    if (typeof mediaQuery.addEventListener === "function") {
      mediaQuery.addEventListener("change", update);
      return () => {
        mediaQuery.removeEventListener("change", update);
      };
    }
    mediaQuery.addListener(update);
    return () => {
      mediaQuery.removeListener(update);
    };
  }, []);

  // --- ResizeObserver ---

  useEffect(() => {
    const el = windowRef.current;
    if (!el) {
      resizeObserverRef.current?.disconnect();
      return;
    }

    const observer = new ResizeObserver(([entry]) => {
      if (!entry || isResizingRef.current || isViewportResizingRef.current) {
        return;
      }
      const target = entry.target as HTMLElement;
      const width = target.offsetWidth;
      const height = target.offsetHeight;
      const store = useAgentStore.getState();
      if (width === store.size.width && height === store.size.height) return;

      useAgentStore.setState((state) => {
        // Same viewport-aware floor as clampWidth/clampHeight.
        const maxW = Math.max(1, viewportSize.width - WINDOW_MARGIN * 2);
        const minW = Math.min(MIN_WIDTH, maxW);
        const maxH = Math.max(1, viewportSize.height - WINDOW_MARGIN * 2);
        const minH = Math.min(MIN_HEIGHT, maxH);
        const clW = Math.min(maxW, Math.max(minW, Math.round(width)));
        const clH = Math.min(maxH, Math.max(minH, Math.round(height)));
        state.size.width = clW;
        state.size.height = clH;
      });
      useAgentStore.getState().saveWindowState();
    });

    observer.observe(el);
    resizeObserverRef.current = observer;
    return () => observer.disconnect();
  }, [visible, minimized, viewportSize.width, viewportSize.height]);

  // --- Sync sidebar width when window width changes ---

  useEffect(() => {
    if (isCompactWindow || isMobileWindow) return;
    const store = useAgentStore.getState();
    const windowWidth = displaySize.width;
    const maxSW = Math.max(
      MIN_CHAT_LIST_WIDTH,
      windowWidth -
        RAIL_WIDTH -
        MIN_MAIN_PANEL_WIDTH -
        SIDEBAR_RESIZE_HANDLE_WIDTH
    );
    const minSW = Math.min(MIN_CHAT_LIST_WIDTH, maxSW);
    const clamped = Math.min(
      maxSW,
      Math.max(minSW, Math.round(store.sidebarWidth))
    );
    if (clamped !== store.sidebarWidth) {
      useAgentStore.setState((state) => {
        state.sidebarWidth = clamped;
      });
      store.saveWindowState();
    }
  }, [displaySize.width, isCompactWindow, isMobileWindow]);

  useEffect(() => {
    if (isCompactWindow || isMobileWindow) return;
    const panel = sidebarPanelRef.current;
    if (!panel) return;
    if (Math.round(panel.getSize().inPixels) !== clampedSidebarWidth) {
      panel.resize(clampedSidebarWidth);
    }
  }, [clampedSidebarWidth, isCompactWindow, isMobileWindow]);

  useEffect(() => {
    if (!isCompactWindow) setCompactPanel(null);
  }, [isCompactWindow]);

  useLayoutEffect(() => {
    if (visible && isMobileWindow) setMobileScreen("conversation");
  }, [isMobileWindow, visible]);

  const saveSidebarWidth = useCallback(() => {
    const width = sidebarPanelRef.current?.getSize().inPixels;
    if (!width || !Number.isFinite(width)) return;
    useAgentStore.getState().setSidebarWidth(clampSidebarWidth(width));
    useAgentStore.getState().saveWindowState();
  }, [clampSidebarWidth]);

  useEffect(() => {
    if (!visible || !currentChatId || contextualStart) return;
    // Restoring a chat must not bind it to a page visited after the landing opened.
    if (continuedChatIdRef.current === currentChatId) {
      continuedChatIdRef.current = null;
      return;
    }
    if (getCurrentAgentPageResource()) return;
    useAgentStore
      .getState()
      .ensureCurrentChat(
        getCurrentPageSnapshot(),
        getCurrentChatContext(),
        workspace?.name
      );
  }, [
    contextualStart,
    currentChatId,
    getCurrentPageSnapshot,
    visible,
    workspace?.name,
  ]);

  useEffect(() => {
    if (!visible || !currentChatId) return;
    const markAIConfigurationRequired = () => {
      useAgentStore.getState().setChatStatus(currentChatId, "error", {
        requiresAIConfiguration: true,
      });
    };
    if (aiEnabled === false) {
      markAIConfigurationRequired();
      return;
    }

    let cancelled = false;
    void getOrFetchSettingByName(Setting_SettingName.AI, true).then(
      (setting) => {
        if (!cancelled && getAgentAIConfigurationEnabled(setting) === false) {
          markAIConfigurationRequired();
        }
      }
    );

    return () => {
      cancelled = true;
    };
  }, [aiEnabled, currentChatId, getOrFetchSettingByName, visible]);

  const openMobileChatList = useCallback(() => {
    setShowArchivedOnly(false);
    setMobileScreen("chats");
  }, []);

  const openMobileArchivedChats = useCallback(() => {
    setShowArchivedOnly(true);
    setMobileScreen("archived");
  }, []);

  const returnToMobileChatList = useCallback(() => {
    setMobileScreen(showArchivedOnly ? "archived" : "chats");
  }, [showArchivedOnly]);

  const createMobileChat = useCallback(
    (context?: string) => {
      if (context) {
        createChatForContext(context);
      } else {
        createChat();
      }
      setMobileScreen("conversation");
    },
    [createChat, createChatForContext]
  );

  const selectMobileChat = useCallback(
    (chatId: string) => {
      if (!useAgentStore.getState().canSelectChat(chatId)) return;
      commitRenameChat(false);
      contextualPageHelpRef.current = null;
      setContextualPageHelp(null);
      setContextualStart(null);
      useAgentStore.getState().setCurrentChat(chatId);
      setMobileScreen("conversation");
    },
    [commitRenameChat]
  );

  const renderChatRow = (chat: AgentChatRecord) => (
    <div
      key={chat.id}
      className={cn(
        "group/agent-chat flex min-h-10 w-full min-w-0 items-center gap-x-2 py-1.5 pr-3 pl-5 text-left text-sm transition-colors",
        !contextualStart && chat.id === currentChatId
          ? "bg-accent/10 text-accent"
          : "text-control hover:bg-control-bg-hover"
      )}
      data-agent-chat-row={chat.id}
      onBlur={(event) => {
        if (
          renamingChatId === chat.id &&
          !event.currentTarget.contains(event.relatedTarget)
        ) {
          commitRenameChat(false);
        }
      }}
    >
      {renamingChatId === chat.id ? (
        <>
          <Input
            ref={renameInputRef}
            size="sm"
            value={renamingTitle}
            onChange={(event) => setRenamingTitle(event.target.value)}
            className="min-w-0 flex-1"
            aria-label={t("agent.rename-chat-placeholder")}
            placeholder={t("agent.rename-chat-placeholder")}
            data-agent-inline-rename-input
            data-agent-chat-title-input
            onKeyDown={onRenameKeydown}
          />
          <div className="flex shrink-0 items-center gap-x-1">
            <AgentTooltip content={t("common.save")}>
              <Button
                appearance="secondary"
                size="xs"
                aria-label={t("common.save")}
                data-agent-save-chat-title
                disabled={
                  !renamingTitle.trim() ||
                  renamingTitle.trim() === getEditableChatTitle(chat)
                }
                onClick={() => commitRenameChat()}
              >
                <Check className="size-3.5" aria-hidden="true" />
              </Button>
            </AgentTooltip>
            <AgentTooltip content={t("common.cancel")}>
              <Button
                appearance="secondary"
                size="xs"
                aria-label={t("common.cancel")}
                data-agent-cancel-chat-title
                onClick={() => cancelRenameChat(true)}
              >
                <X className="size-3.5" aria-hidden="true" />
              </Button>
            </AgentTooltip>
          </div>
        </>
      ) : (
        <>
          <Button
            appearance="secondary"
            size="md"
            type="button"
            title={getEditableChatTitle(chat)}
            className="h-auto min-w-0 flex-1 justify-start p-0 text-left font-normal hover:bg-transparent disabled:cursor-not-allowed disabled:opacity-60"
            disabled={!useAgentStore.getState().canSelectChat(chat.id)}
            aria-current={
              !contextualStart && chat.id === currentChatId ? "true" : undefined
            }
            onMouseDown={(event) => {
              // Keep the rows in place until the pending chat click completes.
              if (renamingChatId && event.button === 0) event.preventDefault();
            }}
            onClick={(event) => {
              isMobileWindow
                ? selectMobileChat(chat.id)
                : handleChatRowClick(chat.id);
              if (renamingChatId) event.currentTarget.focus();
            }}
          >
            <div
              className="flex w-full min-w-0 items-center gap-2"
              data-agent-chat-title
            >
              {chat.status === "running" && (
                <LoaderCircle
                  className="size-3.5 shrink-0 animate-spin text-accent"
                  role="img"
                  aria-label={t("common.running")}
                />
              )}
              <span className="min-w-0 truncate">
                {getEditableChatTitle(chat)}
              </span>
            </div>
          </Button>
          <ChatRowMenu
            title={getEditableChatTitle(chat)}
            archived={chat.archived}
            onRename={() => beginRenameChat(chat.id)}
            onArchive={() => archiveChat(chat.id)}
            onRestore={() => unarchiveChat(chat.id)}
            onDelete={() => setPendingDeleteChatId(chat.id)}
          />
        </>
      )}
    </div>
  );

  const renderConversation = () =>
    contextualStart ? (
      <AgentContextualStart
        page={contextualStart}
        draft={contextualDraft}
        onDraftChange={setContextualDraft}
        context={contextualChatScope?.context}
        contextTitle={
          contextualChatScope
            ? getContextLabel(contextualChatScope.context)
            : undefined
        }
        workspace={
          workspace
            ? { name: workspace.name, title: workspace.title }
            : undefined
        }
        canMentionResources={!contextualPageHelp?.onStart}
        title={contextualTitle}
        reference={
          canReferenceContextualResource ? contextualReference : undefined
        }
        loading={canReferenceContextualResource && contextualLoading}
        maskingRuleEditor={contextualPageHelp?.maskingRuleEditor}
        canStart={canStartPageHelp}
        hasCurrentChat={!contextualPageHelp?.onStart && !!continuableChat}
        showAIConfigurationRecovery={
          aiEnabled === false || currentChat?.requiresAIConfiguration === true
        }
        onStart={startContextualChat}
        onContinue={continueCurrentChat}
      />
    ) : (
      <>
        <AgentChat
          className="min-h-0 flex-1"
          contextLabel={
            currentChat?.context
              ? getContextLabel(currentChat.context)
              : workspace?.title || t("agent.context-workspace")
          }
          onRetry={retryLastTurn}
          onResend={resendMessage}
          onSelectSuggestion={selectEmptyChatSuggestion}
          onContinueAction={continueResponseAction}
        />
        {currentChat?.archived ? (
          <div className="flex shrink-0 flex-wrap items-center justify-between gap-3 border-t border-block-border p-3">
            <p role="status" className="text-sm text-control-light">
              {t("agent.chat-archived-notice")}
            </p>
            <Button
              appearance="secondary"
              size="sm"
              onClick={restoreCurrentChat}
            >
              <Undo2 className="size-4" aria-hidden="true" />
              {t("agent.unarchive-chat")}
            </Button>
          </div>
        ) : (
          <AgentInput
            retryActionRef={retryLastTurnRef}
            resendActionRef={resendMessageRef}
            focusActionRef={focusInputRef}
          />
        )}
      </>
    );

  const renderRail = () => (
    <nav
      className="flex size-full flex-col items-center gap-y-2 border-r border-block-border bg-control-bg py-3"
      data-agent-compact-rail
      aria-label={t("agent.chat-list-label")}
    >
      <AgentTooltip content={t("agent.chat-list-label")}>
        <Button
          appearance="secondary"
          size="xs"
          className={cn(
            "hover:bg-control-bg-hover",
            (isCompactWindow ? compactPanel === "chats" : !showArchivedOnly)
              ? "bg-accent/10 text-accent"
              : "text-control-light"
          )}
          aria-label={t("agent.chat-list-label")}
          aria-current={
            (isCompactWindow ? compactPanel === "chats" : !showArchivedOnly)
              ? "page"
              : undefined
          }
          onClick={(event) => {
            setShowArchivedOnly(false);
            if (isCompactWindow) {
              compactPanelTriggerRef.current = event.currentTarget;
              setCompactPanel("chats");
            } else {
              setCompactPanel(null);
            }
          }}
        >
          <MessageSquare className="size-3.5" aria-hidden="true" />
        </Button>
      </AgentTooltip>
      <AgentTooltip content={t("agent.new-chat")}>
        <Button
          appearance="secondary"
          size="xs"
          className="text-control-light hover:bg-control-bg-hover"
          aria-label={t("agent.new-chat")}
          onClick={createChat}
        >
          <Plus className="size-3.5" aria-hidden="true" />
        </Button>
      </AgentTooltip>
      <AgentTooltip content={t("agent.archived-only-chats")}>
        <Button
          appearance="secondary"
          size="xs"
          className={cn(
            "hover:bg-control-bg-hover",
            (isCompactWindow ? compactPanel === "archived" : showArchivedOnly)
              ? "bg-accent/10 text-accent"
              : "text-control-light"
          )}
          aria-label={t("agent.archived-only-chats")}
          aria-current={
            (isCompactWindow ? compactPanel === "archived" : showArchivedOnly)
              ? "page"
              : undefined
          }
          onClick={(event) => {
            setShowArchivedOnly(true);
            if (isCompactWindow) {
              compactPanelTriggerRef.current = event.currentTarget;
              setCompactPanel("archived");
            } else {
              setCompactPanel(null);
            }
          }}
        >
          <Archive className="size-3.5" aria-hidden="true" />
        </Button>
      </AgentTooltip>
    </nav>
  );

  // --- Render ---

  if (
    !visible ||
    !userEmail ||
    storedWorkspaceName !== workspace?.name ||
    storedUserEmail !== userEmail
  ) {
    return null;
  }

  if (minimized) {
    return createPortal(
      <AgentMinimizedLauncher
        initialPosition={launcherPositionRef.current}
        onPositionChange={(position) => {
          launcherPositionRef.current = position;
        }}
        onOpen={() => useAgentStore.getState().open()}
      />,
      getLayerRoot("agent")
    );
  }

  return createPortal(
    <div
      ref={windowRef}
      data-agent-window
      className="fixed overflow-visible"
      style={windowStyle}
    >
      <AgentDialog
        open={isDeleteAllArchivedChatsDialogOpen}
        onOpenChange={setIsDeleteAllArchivedChatsDialogOpen}
      >
        <AgentDialogContent className="max-w-sm p-6">
          <AgentDialogTitle className="sr-only">
            {t("common.confirm")}
          </AgentDialogTitle>
          <AgentDialogDescription>
            {t("agent.delete-all-chats-confirmation")}
          </AgentDialogDescription>
          <div className="mt-6 flex justify-end gap-x-2">
            <AgentDialogClose
              render={
                <Button appearance="outline" type="button">
                  {t("common.cancel")}
                </Button>
              }
            />
            <AgentDialogClose
              render={
                <Button type="button" variant="destructive">
                  {t("common.confirm")}
                </Button>
              }
              onClick={deleteAllArchivedChats}
            />
          </div>
        </AgentDialogContent>
      </AgentDialog>
      <AgentDialog
        open={pendingDeleteChatId !== null}
        onOpenChange={(open) => {
          if (!open) setPendingDeleteChatId(null);
        }}
      >
        <AgentDialogContent className="max-w-sm p-6">
          <AgentDialogTitle className="sr-only">
            {t("common.confirm")}
          </AgentDialogTitle>
          <AgentDialogDescription>
            {t("agent.delete-chat-confirmation")}
          </AgentDialogDescription>
          <div className="mt-6 flex justify-end gap-x-2">
            <AgentDialogClose
              render={
                <Button appearance="outline" type="button">
                  {t("common.cancel")}
                </Button>
              }
            />
            <AgentDialogClose
              render={
                <Button type="button" variant="destructive">
                  {t("common.confirm")}
                </Button>
              }
              onClick={() => {
                if (pendingDeleteChatId) deleteChat(pendingDeleteChatId);
              }}
            />
          </div>
        </AgentDialogContent>
      </AgentDialog>
      <BaseDialog.Root
        open={isCompactDrawerOpen}
        modal="trap-focus"
        onOpenChange={(open) => {
          if (!open) setCompactPanel(null);
        }}
      >
        <div className="flex size-full flex-col overflow-hidden rounded-sm border border-block-border bg-background shadow-xl">
          {/* Header */}
          <div
            data-agent-window-header
            aria-hidden={isCompactDrawerOpen || undefined}
            inert={isCompactDrawerOpen || undefined}
            className="relative flex select-none items-center justify-between border-b bg-control-bg px-3 py-2"
          >
            <div className="flex min-w-0 items-center gap-x-2">
              {preferencesOpen && (
                <AgentTooltip content={t("agent.preferences.back")}>
                  <Button
                    appearance="secondary"
                    size="xs"
                    data-agent-window-action
                    className="text-control-light hover:bg-control-bg-hover pointer-coarse:min-h-11 pointer-coarse:min-w-11"
                    aria-label={t("agent.preferences.back")}
                    onClick={(event) => {
                      event.stopPropagation();
                      preferencesReturnFocusRef.current = true;
                      setPreferencesOpen(false);
                    }}
                  >
                    <ChevronLeft className="size-3.5" aria-hidden="true" />
                  </Button>
                </AgentTooltip>
              )}
              {!preferencesOpen &&
                isMobileWindow &&
                mobileScreen === "conversation" && (
                  <Button
                    appearance="secondary"
                    size="xs"
                    data-agent-window-action
                    className="-ml-1 text-control-light hover:bg-control-bg-hover"
                    aria-label={t("agent.chat-list-label")}
                    onClick={(event) => {
                      event.stopPropagation();
                      returnToMobileChatList();
                    }}
                  >
                    <ChevronLeft className="size-4" aria-hidden="true" />
                  </Button>
                )}
              {!preferencesOpen &&
                isMobileWindow &&
                mobileScreen === "archived" && (
                  <Button
                    appearance="secondary"
                    size="xs"
                    data-agent-window-action
                    className="-ml-1 text-control-light hover:bg-control-bg-hover"
                    aria-label={t("agent.chat-list-label")}
                    onClick={(event) => {
                      event.stopPropagation();
                      openMobileChatList();
                    }}
                  >
                    <ChevronLeft className="size-4" aria-hidden="true" />
                  </Button>
                )}
              <h2 className="truncate text-sm font-medium">
                {!preferencesOpen && isMobileWindow && mobileScreen === "chats"
                  ? t("agent.chat-list-label")
                  : !preferencesOpen &&
                      isMobileWindow &&
                      mobileScreen === "archived"
                    ? t("agent.archived-only-chats")
                    : t("agent.assistant-title")}
              </h2>
            </div>
            <div className="flex items-center gap-x-1">
              {!preferencesOpen &&
                isMobileWindow &&
                mobileScreen === "chats" && (
                  <>
                    <AgentTooltip content={t("agent.archived-only-chats")}>
                      <Button
                        appearance="secondary"
                        size="xs"
                        data-agent-window-action
                        className="text-control-light hover:bg-control-bg-hover"
                        aria-label={t("agent.archived-only-chats")}
                        onClick={(event) => {
                          event.stopPropagation();
                          openMobileArchivedChats();
                        }}
                      >
                        <Archive className="size-3.5" aria-hidden="true" />
                      </Button>
                    </AgentTooltip>
                    <AgentTooltip content={t("agent.new-chat")}>
                      <Button
                        appearance="secondary"
                        size="xs"
                        data-agent-window-action
                        className="text-control-light hover:bg-control-bg-hover"
                        aria-label={t("agent.new-chat")}
                        onClick={(event) => {
                          event.stopPropagation();
                          createMobileChat();
                        }}
                      >
                        <Plus className="size-3.5" aria-hidden="true" />
                      </Button>
                    </AgentTooltip>
                  </>
                )}
              <AgentTooltip content={t("agent.preferences.title")}>
                <Button
                  ref={preferencesTriggerRef}
                  appearance="secondary"
                  size="xs"
                  data-agent-window-action
                  className={cn(
                    "text-control-placeholder hover:bg-control-bg-hover hover:text-control-light",
                    preferencesOpen && "hidden"
                  )}
                  hidden={preferencesOpen}
                  aria-label={t("agent.preferences.title")}
                  aria-pressed={preferencesOpen}
                  onClick={(event) => {
                    event.stopPropagation();
                    setCompactPanel(null);
                    setPreferencesOpen((open) => !open);
                  }}
                >
                  <Settings className="size-3.5" aria-hidden="true" />
                </Button>
              </AgentTooltip>
              <Button
                appearance="secondary"
                size="xs"
                data-agent-window-action
                className={cn(
                  "flex items-center justify-center rounded-xs text-control-placeholder hover:bg-control-bg-hover hover:text-control-light",
                  preferencesOpen &&
                    "pointer-coarse:min-h-11 pointer-coarse:min-w-11",
                  preferencesOpen && isMobileWindow && "pointer-coarse:hidden"
                )}
                title={t("agent.minimize")}
                onClick={(e) => {
                  e.stopPropagation();
                  useAgentStore.getState().minimize();
                }}
              >
                <Minus className="size-3.5" aria-hidden="true" />
              </Button>
              <Button
                appearance="secondary"
                size="xs"
                data-agent-window-action
                className={cn(
                  "flex items-center justify-center rounded-xs text-control-placeholder hover:bg-control-bg-hover hover:text-control-light",
                  preferencesOpen &&
                    "pointer-coarse:min-h-11 pointer-coarse:min-w-11"
                )}
                title={t("agent.close")}
                onClick={(e) => {
                  e.stopPropagation();
                  useAgentStore.getState().toggle();
                }}
              >
                <X className="size-3.5" aria-hidden="true" />
              </Button>
            </div>
          </div>

          {/* Body */}
          <div
            ref={bodyRef}
            className="relative min-h-0 flex-1"
            data-agent-window-body
          >
            {isMobileWindow ? (
              mobileScreen === "conversation" ? (
                <div
                  className={cn(
                    "flex size-full flex-col bg-background",
                    preferencesOpen && "hidden"
                  )}
                  hidden={preferencesOpen}
                  inert={preferencesOpen || undefined}
                  data-agent-mobile-conversation
                >
                  {renderConversation()}
                </div>
              ) : (
                <div
                  className={cn(
                    "size-full overflow-y-auto bg-control-bg",
                    preferencesOpen && "hidden"
                  )}
                  hidden={preferencesOpen}
                  inert={preferencesOpen || undefined}
                  data-agent-mobile-chat-list
                >
                  <div className="flex flex-col" data-agent-chat-list>
                    {displayedChatGroups.map((group) => {
                      const groupKey = getChatGroupStateKey(
                        group.context,
                        showArchivedOnly
                      );
                      const groupIsOpen = !collapsedChatGroups.has(groupKey);

                      return (
                        <Collapsible
                          key={group.context ?? "unscoped"}
                          open={groupIsOpen}
                          onOpenChange={(open) =>
                            setChatGroupOpen(group.context, open)
                          }
                          data-agent-chat-group={group.context ?? "unscoped"}
                        >
                          <div
                            className="flex w-full min-w-0 items-center justify-between gap-x-2 pt-2 pb-1 text-sm font-medium text-control-light hover:bg-control-bg-hover"
                            data-agent-chat-group-header={
                              group.context ?? "unscoped"
                            }
                          >
                            <CollapsibleTrigger
                              className="h-7 w-auto min-w-0 flex-1 justify-start gap-x-2 rounded-none pr-1 pl-4 text-sm font-medium text-control-light hover:bg-transparent"
                              title={
                                group.context
                                  ? getContextLabel(group.context)
                                  : t("agent.other-chats")
                              }
                              data-agent-chat-group-toggle={
                                group.context ?? "unscoped"
                              }
                            >
                              {groupIsOpen ? (
                                <FolderOpen
                                  className="size-4 shrink-0"
                                  aria-hidden="true"
                                />
                              ) : (
                                <Folder
                                  className="size-4 shrink-0"
                                  aria-hidden="true"
                                />
                              )}
                              <span className="truncate">
                                {group.context
                                  ? getContextLabel(group.context)
                                  : t("agent.other-chats")}
                              </span>
                            </CollapsibleTrigger>
                            {group.context && !showArchivedOnly && (
                              <AgentTooltip
                                content={`${t("agent.new-chat")}: ${getContextLabel(group.context)}`}
                              >
                                <Button
                                  appearance="secondary"
                                  size="xs"
                                  className="mr-2 shrink-0 text-control-light hover:bg-control/25"
                                  aria-label={`${t("agent.new-chat")}: ${getContextLabel(group.context)}`}
                                  data-agent-new-chat-for-context={
                                    group.context
                                  }
                                  onClick={() =>
                                    createMobileChat(group.context)
                                  }
                                >
                                  <Plus
                                    className="size-3.5"
                                    aria-hidden="true"
                                  />
                                </Button>
                              </AgentTooltip>
                            )}
                          </div>
                          <CollapsiblePanel
                            className="p-0"
                            data-agent-chat-group-children={
                              group.context ?? "unscoped"
                            }
                          >
                            {group.chats.map(renderChatRow)}
                          </CollapsiblePanel>
                        </Collapsible>
                      );
                    })}
                  </div>
                </div>
              )
            ) : (
              <div
                className={cn("flex size-full", preferencesOpen && "hidden")}
                hidden={preferencesOpen}
                inert={preferencesOpen || undefined}
                data-agent-desktop-layout
              >
                <div className="w-12 shrink-0">{renderRail()}</div>
                <PanelGroup
                  orientation="horizontal"
                  className="relative z-0 min-w-0 flex-1 bg-background"
                  data-agent-chat-panels
                  aria-hidden={isCompactDrawerOpen || undefined}
                  inert={isCompactDrawerOpen || undefined}
                  onLayoutChanged={(_, meta) => {
                    if (!isCompactWindow && meta.isUserInteraction)
                      saveSidebarWidth();
                  }}
                >
                  {/* Sidebar */}
                  <Panel
                    key={isCompactWindow ? "compact" : "roomy"}
                    id="agent-chat-sidebar"
                    panelRef={sidebarPanelRef}
                    defaultSize={isCompactWindow ? 0 : clampedSidebarWidth}
                    minSize={isCompactWindow ? 0 : minSidebarWidth}
                    maxSize={isCompactWindow ? 0 : maxSidebarWidth}
                    groupResizeBehavior="preserve-pixel-size"
                    className="relative"
                  >
                    {(!isCompactWindow || compactPanel) && (
                      <CompactDrawerPortal
                        compact={isCompactWindow}
                        container={bodyRef}
                        finalFocus={compactPanelTriggerRef}
                        title={
                          showArchivedOnly
                            ? t("agent.archived-only-chats")
                            : t("agent.chat-list-label")
                        }
                      >
                        <aside
                          className={cn(
                            "flex size-full flex-col bg-control-bg",
                            isCompactWindow && "shadow-lg"
                          )}
                          data-agent-chat-sidebar={
                            isCompactWindow ? undefined : true
                          }
                          data-agent-compact-drawer={
                            isCompactWindow ? true : undefined
                          }
                        >
                          {/* Sidebar header */}
                          <div className="border-b border-block-border px-3 py-3">
                            <div className="flex items-center justify-between gap-x-2">
                              <h2 className="text-xs font-semibold uppercase tracking-wide text-control-light">
                                {showArchivedOnly
                                  ? t("agent.archived-only-chats")
                                  : t("agent.chat-list-label")}
                              </h2>
                              {isCompactWindow ? (
                                <AgentDialogClose
                                  render={
                                    <Button
                                      appearance="secondary"
                                      size="xs"
                                      className="text-control-light hover:bg-control-bg-hover"
                                      aria-label={t("agent.close")}
                                    >
                                      &#10005;
                                    </Button>
                                  }
                                />
                              ) : (
                                <div
                                  className="flex items-center gap-x-1"
                                  data-agent-chat-sidebar-actions
                                >
                                  {!showArchivedOnly && (
                                    <AgentTooltip content={t("agent.new-chat")}>
                                      <Button
                                        appearance="secondary"
                                        size="xs"
                                        className="px-1 text-control-light hover:bg-control-bg-hover"
                                        aria-label={t("agent.new-chat")}
                                        onClick={createChat}
                                      >
                                        <Plus
                                          className="size-3.5"
                                          aria-hidden="true"
                                        />
                                      </Button>
                                    </AgentTooltip>
                                  )}
                                  <AgentDropdownMenu>
                                    <AgentTooltip content={t("common.more")}>
                                      <AgentDropdownMenuTrigger
                                        className="inline-flex size-6 items-center justify-center rounded-xs text-control-light outline-hidden hover:bg-control-bg-hover focus-visible:ring-2 focus-visible:ring-accent disabled:pointer-events-none disabled:opacity-50"
                                        aria-label={t("common.more")}
                                        onClick={(event) =>
                                          event.stopPropagation()
                                        }
                                      >
                                        <EllipsisVertical
                                          className="size-3.5"
                                          aria-hidden="true"
                                        />
                                      </AgentDropdownMenuTrigger>
                                    </AgentTooltip>
                                    <AgentDropdownMenuContent>
                                      {showArchivedOnly ? (
                                        <>
                                          <AgentDropdownMenuItem
                                            data-agent-unarchive-all-chats
                                            disabled={isUnarchiveAllDisabled}
                                            onClick={(event) => {
                                              event.stopPropagation();
                                              unarchiveAllChats();
                                            }}
                                          >
                                            {t("agent.unarchive-all-chats")}
                                          </AgentDropdownMenuItem>
                                          <AgentDropdownMenuItem
                                            data-agent-delete-all-chats
                                            className="text-error data-highlighted:bg-error/10"
                                            disabled={isDeleteAllDisabled}
                                            onClick={(event) => {
                                              event.stopPropagation();
                                              setIsDeleteAllArchivedChatsDialogOpen(
                                                true
                                              );
                                            }}
                                          >
                                            {t("agent.delete-all-chats")}
                                          </AgentDropdownMenuItem>
                                        </>
                                      ) : (
                                        <AgentDropdownMenuItem
                                          data-agent-archive-all-chats
                                          className="text-error data-highlighted:bg-error/10"
                                          disabled={isArchiveAllDisabled}
                                          onClick={(event) => {
                                            event.stopPropagation();
                                            archiveAllChats();
                                          }}
                                        >
                                          {t("agent.archive-all-chats")}
                                        </AgentDropdownMenuItem>
                                      )}
                                    </AgentDropdownMenuContent>
                                  </AgentDropdownMenu>
                                </div>
                              )}
                            </div>
                          </div>

                          {/* Chat list */}
                          <div className="min-h-0 flex-1 overflow-y-auto">
                            <div className="flex flex-col" data-agent-chat-list>
                              {displayedChatGroups.map((group) => {
                                const groupKey = getChatGroupStateKey(
                                  group.context,
                                  showArchivedOnly
                                );
                                const groupIsOpen =
                                  !collapsedChatGroups.has(groupKey);

                                return (
                                  <Collapsible
                                    key={group.context ?? "unscoped"}
                                    open={groupIsOpen}
                                    onOpenChange={(open) =>
                                      setChatGroupOpen(group.context, open)
                                    }
                                    data-agent-chat-group={
                                      group.context ?? "unscoped"
                                    }
                                  >
                                    <div
                                      className="flex w-full min-w-0 items-center justify-between gap-x-2 pt-2 pb-1 text-sm font-medium text-control-light hover:bg-control-bg-hover"
                                      data-agent-chat-group-header={
                                        group.context ?? "unscoped"
                                      }
                                    >
                                      <CollapsibleTrigger
                                        className="h-7 w-auto min-w-0 flex-1 justify-start gap-x-2 rounded-none pr-1 pl-4 text-sm font-medium text-control-light hover:bg-transparent"
                                        title={
                                          group.context
                                            ? getContextLabel(group.context)
                                            : t("agent.other-chats")
                                        }
                                        data-agent-chat-group-toggle={
                                          group.context ?? "unscoped"
                                        }
                                      >
                                        {groupIsOpen ? (
                                          <FolderOpen
                                            className="size-4 shrink-0"
                                            aria-hidden="true"
                                          />
                                        ) : (
                                          <Folder
                                            className="size-4 shrink-0"
                                            aria-hidden="true"
                                          />
                                        )}
                                        <span className="truncate">
                                          {group.context
                                            ? getContextLabel(group.context)
                                            : t("agent.other-chats")}
                                        </span>
                                      </CollapsibleTrigger>
                                      {group.context && !showArchivedOnly && (
                                        <AgentTooltip
                                          content={`${t("agent.new-chat")}: ${getContextLabel(group.context)}`}
                                        >
                                          <Button
                                            appearance="secondary"
                                            size="xs"
                                            className="mr-2 shrink-0 text-control-light hover:bg-control/25"
                                            aria-label={`${t("agent.new-chat")}: ${getContextLabel(group.context)}`}
                                            data-agent-new-chat-for-context={
                                              group.context
                                            }
                                            onClick={() =>
                                              createChatForContext(
                                                group.context!
                                              )
                                            }
                                          >
                                            <Plus
                                              className="size-3.5"
                                              aria-hidden="true"
                                            />
                                          </Button>
                                        </AgentTooltip>
                                      )}
                                    </div>
                                    <CollapsiblePanel
                                      className="p-0"
                                      data-agent-chat-group-children={
                                        group.context ?? "unscoped"
                                      }
                                    >
                                      {group.chats.map(renderChatRow)}
                                    </CollapsiblePanel>
                                  </Collapsible>
                                );
                              })}
                            </div>
                          </div>
                        </aside>
                      </CompactDrawerPortal>
                    )}
                  </Panel>

                  {!isCompactWindow && (
                    <PanelResizeHandle
                      data-agent-sidebar-resize
                      className="group relative w-1 shrink-0 cursor-ew-resize bg-control-bg transition-colors hover:bg-accent/10"
                    >
                      <span className="pointer-events-none absolute inset-y-0 left-1/2 w-px -translate-x-1/2 bg-transparent transition-colors group-hover:bg-accent" />
                    </PanelResizeHandle>
                  )}

                  {/* Main panel */}
                  <Panel id="agent-chat-main" minSize={minMainPanelWidth}>
                    <div className="relative flex size-full min-w-0 flex-col">
                      {renderConversation()}
                    </div>
                  </Panel>
                </PanelGroup>
              </div>
            )}
            {preferencesOpen && <AgentPreferences />}
          </div>
        </div>
      </BaseDialog.Root>

      {canResizeWindow &&
        !isCompactDrawerOpen &&
        (Object.keys(resizeZoneClasses) as AnchoredResizeDirection[]).map(
          (direction) => (
            <div
              key={direction}
              aria-hidden="true"
              data-agent-window-resize-zone={direction}
              className={cn(
                "absolute z-1 [touch-action:none]",
                resizeZoneClasses[direction]
              )}
              onPointerDown={(event) => startResize(direction, event)}
            />
          )
        )}
    </div>,
    getLayerRoot("agent")
  );
}

function CompactDrawerPortal({
  children,
  compact,
  container,
  finalFocus,
  title,
}: Readonly<{
  children: ReactNode;
  compact: boolean;
  container: RefObject<HTMLElement | null>;
  finalFocus: RefObject<HTMLElement | null>;
  title: string;
}>) {
  if (!compact) return children;

  return (
    <CompactDrawerModal
      container={container}
      finalFocus={finalFocus}
      title={title}
    >
      {children}
    </CompactDrawerModal>
  );
}

function CompactDrawerModal({
  children,
  container,
  finalFocus,
  title,
}: Readonly<{
  children: ReactNode;
  container: RefObject<HTMLElement | null>;
  finalFocus: RefObject<HTMLElement | null>;
  title: string;
}>) {
  usePreserveHigherLayerAccess("agent");

  return (
    <BaseDialog.Portal container={container}>
      <BaseDialog.Backdrop
        className="absolute inset-y-0 right-0 left-12 z-1 bg-overlay/50"
        data-agent-compact-overlay
      />
      <BaseDialog.Popup
        className="absolute inset-y-0 left-12 z-2 flex w-80 outline-none"
        data-agent-compact-drawer
        finalFocus={finalFocus}
      >
        <BaseDialog.Title className="sr-only">{title}</BaseDialog.Title>
        {children}
      </BaseDialog.Popup>
    </BaseDialog.Portal>
  );
}

function ChatRowMenu({
  title,
  archived,
  onRename,
  onArchive,
  onRestore,
  onDelete,
}: Readonly<{
  title: string;
  archived: boolean;
  onRename: () => void;
  onArchive: () => void;
  onRestore: () => void;
  onDelete: () => void;
}>) {
  const { t } = useTranslation();
  const [menuOpen, setMenuOpen] = useState(false);
  const [archiveOpen, setArchiveOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const skipMenuFinalFocusRef = useRef(false);

  return (
    <>
      <AgentDropdownMenu
        open={menuOpen}
        onOpenChange={(open) => {
          if (open) skipMenuFinalFocusRef.current = false;
          setMenuOpen(open);
        }}
      >
        <AgentTooltip content={t("common.more")}>
          <AgentDropdownMenuTrigger
            ref={triggerRef}
            render={
              <Button
                appearance="secondary"
                size="xs"
                className={cn(
                  "pointer-events-none shrink-0 text-control-light opacity-0 transition-opacity group-hover/agent-chat:pointer-events-auto group-hover/agent-chat:opacity-100 group-has-[:focus-visible]/agent-chat:pointer-events-auto group-has-[:focus-visible]/agent-chat:opacity-100 pointer-coarse:pointer-events-auto pointer-coarse:opacity-100",
                  (menuOpen || archiveOpen) &&
                    "pointer-events-auto bg-control-bg-hover opacity-100"
                )}
                aria-label={`${t("common.more")}: ${title}`}
                data-agent-chat-menu
                data-agent-archived-chat-menu={archived || undefined}
                onClick={(event) => event.stopPropagation()}
              />
            }
          >
            <Ellipsis className="size-3.5" aria-hidden="true" />
          </AgentDropdownMenuTrigger>
        </AgentTooltip>
        <AgentDropdownMenuContent
          finalFocus={() =>
            skipMenuFinalFocusRef.current ? false : triggerRef.current
          }
        >
          <AgentDropdownMenuItem
            data-agent-rename-chat
            onClick={(event) => {
              event.stopPropagation();
              skipMenuFinalFocusRef.current = true;
              onRename();
            }}
          >
            <Pencil className="size-4" aria-hidden="true" />
            {t("agent.rename-chat")}
          </AgentDropdownMenuItem>
          {archived ? (
            <>
              <AgentDropdownMenuItem
                data-agent-unarchive-chat
                onClick={(event) => {
                  event.stopPropagation();
                  onRestore();
                }}
              >
                <Undo2 className="size-4" aria-hidden="true" />
                {t("agent.unarchive-chat")}
              </AgentDropdownMenuItem>
              <AgentDropdownMenuItem
                data-agent-delete-chat
                className="text-error data-highlighted:bg-error/10"
                onClick={(event) => {
                  event.stopPropagation();
                  skipMenuFinalFocusRef.current = true;
                  onDelete();
                }}
              >
                <Trash2 className="size-4" aria-hidden="true" />
                {t("agent.delete-chat")}
              </AgentDropdownMenuItem>
            </>
          ) : (
            <AgentDropdownMenuItem
              data-agent-archive-chat
              onClick={(event) => {
                event.stopPropagation();
                skipMenuFinalFocusRef.current = true;
                setMenuOpen(false);
                setArchiveOpen(true);
              }}
            >
              <Archive className="size-4" aria-hidden="true" />
              {t("agent.archive-chat")}
            </AgentDropdownMenuItem>
          )}
        </AgentDropdownMenuContent>
      </AgentDropdownMenu>
      <AgentPopover open={archiveOpen} onOpenChange={setArchiveOpen}>
        <AgentPopoverContent
          anchor={triggerRef}
          finalFocus={triggerRef}
          side="bottom"
          align="end"
          className="w-56 p-3"
          data-agent-archive-chat-confirmation
          onClick={(event) => event.stopPropagation()}
        >
          <AgentPopoverTitle className="mb-3 font-normal">
            {t("agent.archive-chat-confirmation")}
          </AgentPopoverTitle>
          <div className="flex justify-end gap-2">
            <Button
              appearance="outline"
              size="sm"
              onClick={() => setArchiveOpen(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button
              size="sm"
              onClick={() => {
                setArchiveOpen(false);
                onArchive();
              }}
            >
              {t("common.archive")}
            </Button>
          </div>
        </AgentPopoverContent>
      </AgentPopover>
    </>
  );
}
