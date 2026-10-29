import { Dialog as BaseDialog } from "@base-ui/react/dialog";
import { create } from "@bufbuild/protobuf";
import {
  Archive,
  ChevronLeft,
  EllipsisVertical,
  Folder,
  FolderOpen,
  MessageSquare,
  Plus,
  Trash2,
  Undo2,
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
} from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import {
  Panel,
  Group as PanelGroup,
  type PanelImperativeHandle,
  Separator as PanelResizeHandle,
} from "react-resizable-panels";
import { issueServiceClientConnect } from "@/api";
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
import { GetIssueRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { extractDatabaseResourceName } from "@/utils/v1/database";
import { getAgentAIConfigurationEnabled } from "../logic/aiConfiguration";
import {
  type AgentPageResource,
  getCurrentAgentPageResource,
  getCurrentChatContext,
} from "../logic/chatContext";
import type { LandingStarterId } from "../logic/landingStarters";
import {
  formatResourceMention,
  getDatabaseMentionLabel,
  normalizeAgentResourceReference,
} from "../logic/resourceMention";
import type {
  AgentChat as AgentChatRecord,
  AgentResourceReference,
} from "../logic/types";
import {
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
  RESIZE_POINTER_MEDIA_QUERY,
  supportsWindowBorderResize,
} from "./resize-capability";
import {
  AgentDialog,
  AgentDialogClose,
  AgentDialogContent,
  AgentDialogDescription,
  AgentDialogTitle,
  AgentDialogTrigger,
} from "./ui/AgentDialog";
import {
  AgentDropdownMenu,
  AgentDropdownMenuContent,
  AgentDropdownMenuItem,
  AgentDropdownMenuTrigger,
} from "./ui/AgentDropdownMenu";
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
  const hasRunningChat = useAgentStore(selectHasRunningChat);
  const getOrFetchSettingByName = useAppStore((s) => s.getOrFetchSettingByName);
  const batchFetchProjects = useAppStore((s) => s.batchFetchProjects);
  const projectsByName = useAppStore((s) => s.projectsByName);
  const projectErrorsByName = useAppStore((s) => s.projectErrorsByName);
  const workspace = useAppStore((s) => s.workspace);

  useLayoutEffect(() => {
    useAgentStore.getState().setWorkspaceName(workspace?.name);
  }, [hasRunningChat, workspace?.name]);

  const windowRef = useRef<HTMLDivElement>(null);
  const bodyRef = useRef<HTMLDivElement>(null);
  const compactPanelTriggerRef = useRef<HTMLElement | null>(null);
  const renameInputRef = useRef<HTMLInputElement>(null);

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
  const [contextualReference, setContextualReference] = useState<
    AgentResourceReference | undefined
  >();
  const [contextualLoading, setContextualLoading] = useState(false);
  const canReferenceContextualResource = useAppStore((state) => {
    if (!contextualStart || contextualStart.kind === "project") return true;
    const project = state.projectsByName[contextualStart.project];
    const permission =
      contextualStart.kind === "database"
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
  const openingRef = useRef(false);
  const [compactPanel, setCompactPanel] = useState<"chats" | "archived" | null>(
    null
  );
  const [mobileScreen, setMobileScreen] = useState<
    "chats" | "conversation" | "archived"
  >("chats");
  const [collapsedChatGroups, setCollapsedChatGroups] = useState<Set<string>>(
    () => new Set()
  );
  const [isRenamingCurrentChat, setIsRenamingCurrentChat] = useState(false);
  const [pendingDeleteChatId, setPendingDeleteChatId] = useState<string | null>(
    null
  );
  const [
    isDeleteAllArchivedChatsDialogOpen,
    setIsDeleteAllArchivedChatsDialogOpen,
  ] = useState(false);
  const [renamingTitle, setRenamingTitle] = useState("");

  useEffect(() => {
    const isOpen = visible && !minimized;
    if (isOpen && !openingRef.current) {
      const pageResource = getCurrentAgentPageResource();
      setContextualStart(pageResource ?? null);
      setContextualReference(undefined);
      setContextualLoading(false);
      if (pageResource) {
        setShowArchivedOnly(false);
        setIsRenamingCurrentChat(false);
        setMobileScreen("conversation");
      }
    }
    openingRef.current = isOpen;
  }, [minimized, visible]);

  useEffect(() => {
    if (
      !contextualStart ||
      contextualStart.kind === "project" ||
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
      for (const name of contextualStart.names) {
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
  const canContinueCurrentChat =
    !!currentChat &&
    !currentChat.archived &&
    useAgentStore.getState().canSelectChat(currentChat.id);
  const archivedChats = useMemo(
    () => orderedChats.filter((chat) => chat.archived),
    [orderedChats]
  );

  const isChatCreationDisabled = hasRunningChat;
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

  const getChatLabel = useCallback(
    (chat: AgentChatRecord) => {
      const baseLabel = getEditableChatTitle(chat);
      return chat.archived
        ? `${baseLabel} (${t("agent.chat-archived-label")})`
        : baseLabel;
    },
    [getEditableChatTitle, t]
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
      : (contextualReference?.label ??
        (contextualStart.kind === "database"
          ? extractDatabaseResourceName(contextualStart.names[0]).databaseName
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

      if (store.chats.some((c) => c.status === "running")) {
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

  const beginRenameCurrentChat = useCallback(() => {
    if (!currentChat) return;
    setRenamingTitle(getEditableChatTitle(currentChat));
    setIsRenamingCurrentChat(true);
  }, [currentChat, getEditableChatTitle]);

  const cancelRenameCurrentChat = useCallback(() => {
    setIsRenamingCurrentChat(false);
    setRenamingTitle("");
  }, []);

  const commitRenameCurrentChat = useCallback(() => {
    const chat = useAgentStore
      .getState()
      .chats.find((c) => c.id === useAgentStore.getState().currentChatId);
    const title = renamingTitle.trim();
    if (!chat || !isRenamingCurrentChat) return;
    if (!title) {
      cancelRenameCurrentChat();
      return;
    }
    useAgentStore.getState().renameChat(chat.id, title);
    cancelRenameCurrentChat();
  }, [renamingTitle, isRenamingCurrentChat, cancelRenameCurrentChat]);

  const onRenameKeydown = useCallback(
    (event: React.KeyboardEvent<HTMLInputElement>) => {
      if (event.nativeEvent.isComposing) return;
      if (event.key === "Escape") {
        event.preventDefault();
        cancelRenameCurrentChat();
        return;
      }
      if (event.key === "Enter") {
        event.preventDefault();
        commitRenameCurrentChat();
      }
    },
    [cancelRenameCurrentChat, commitRenameCurrentChat]
  );

  // Cancel rename when currentChatId changes
  useEffect(() => {
    if (isRenamingCurrentChat) {
      cancelRenameCurrentChat();
    }
    // Only react to currentChatId changes, not the rename state itself
  }, [currentChatId]);

  // Focus rename input when it appears
  useEffect(() => {
    if (isRenamingCurrentChat && renameInputRef.current) {
      renameInputRef.current.focus();
      renameInputRef.current.select();
    }
  }, [isRenamingCurrentChat]);

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
    if (isChatCreationDisabled) return;
    setContextualStart(null);
    const context = getCurrentChatContext();
    setChatGroupOpen(context, true, false);
    useAgentStore.getState().createChat({
      page: getCurrentPageSnapshot(),
      context,
      workspaceName: workspace?.name,
    });
    setShowArchivedOnly(false);
    if (isCompactWindow) setCompactPanel(null);
  }, [
    isChatCreationDisabled,
    getCurrentPageSnapshot,
    isCompactWindow,
    setChatGroupOpen,
    workspace?.name,
  ]);

  const createChatForContext = useCallback(
    (context: string) => {
      if (isChatCreationDisabled) return;
      setContextualStart(null);
      setChatGroupOpen(context, true, false);
      useAgentStore.getState().createChat({
        page: getCurrentPageSnapshot(),
        context,
        workspaceName: workspace?.name,
      });
      if (isCompactWindow) setCompactPanel(null);
    },
    [
      isChatCreationDisabled,
      getCurrentPageSnapshot,
      isCompactWindow,
      setChatGroupOpen,
      workspace?.name,
    ]
  );

  const handleChatRowClick = useCallback(
    (chatId: string) => {
      if (contextualStart) {
        setContextualStart(null);
        if (chatId === useAgentStore.getState().currentChatId) return;
      }
      if (chatId === useAgentStore.getState().currentChatId) {
        if (!isRenamingCurrentChat) {
          beginRenameCurrentChat();
        }
        return;
      }
      if (isRenamingCurrentChat) {
        cancelRenameCurrentChat();
      }
      useAgentStore.getState().setCurrentChat(chatId);
      if (isCompactWindow) setCompactPanel(null);
    },
    [
      isCompactWindow,
      contextualStart,
      isRenamingCurrentChat,
      beginRenameCurrentChat,
      cancelRenameCurrentChat,
    ]
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

  const archiveAllChats = useCallback(() => {
    if (isArchiveAllDisabled) return;
    const store = useAgentStore.getState();
    const currentChatWasActive = store.chats.some(
      (chat) => chat.id === store.currentChatId && !chat.archived
    );
    const archivedCount = store.archiveAllActiveChats();
    if (archivedCount === 0) return;
    if (currentChatWasActive) {
      cancelRenameCurrentChat();
      ensureCurrentChatMatchesDisplayedMode({ allowCreateWhenEmpty: false });
    }
  }, [
    cancelRenameCurrentChat,
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
      cancelRenameCurrentChat();
      ensureCurrentChatMatchesDisplayedMode({ allowCreateWhenEmpty: false });
    }
  }, [
    cancelRenameCurrentChat,
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
      cancelRenameCurrentChat();
      ensureCurrentChatMatchesDisplayedMode({ allowCreateWhenEmpty: false });
    }
  }, [
    cancelRenameCurrentChat,
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
  const focusInputRef = useRef<(() => void) | null>(null);
  const focusContextualDraftRef = useRef(false);
  useEffect(() => {
    if (contextualStart || !focusContextualDraftRef.current) return;
    focusContextualDraftRef.current = false;
    focusInputRef.current?.();
  }, [contextualStart, currentChatId]);
  const selectEmptyChatSuggestion = useCallback(
    (starterId: LandingStarterId) => {
      const store = useAgentStore.getState();
      const chat = store.getChat(store.currentChatId);
      if (
        !chat ||
        chat.status !== "idle" ||
        store.getMessages(chat.id).length ||
        store.getDraft(chat.id).text.trim()
      ) {
        return;
      }
      store.setDraft(chat.id, {
        text: t(`agent.landing-starters.${starterId}`),
      });
      focusInputRef.current?.();
    },
    [t]
  );
  const startContextualChat = useCallback(
    (prompt: string, reference?: AgentResourceReference) => {
      if (
        !contextualStart ||
        useAgentStore.getState().chats.some((chat) => chat.status === "running")
      ) {
        return;
      }
      if (contextualStart.kind !== "project" && !reference) return;
      const store = useAgentStore.getState();
      const chat = store.createChat({
        page: getCurrentPageSnapshot(),
        context: contextualStart.project,
        workspaceName: workspace?.name,
      });
      if (useAgentStore.getState().currentChatId !== chat.id) return;
      const mention = reference ? formatResourceMention(reference.label) : "";
      store.setDraft(chat.id, {
        text: mention ? `${mention}${prompt ? ` ${prompt}` : " "}` : prompt,
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
      setChatGroupOpen(contextualStart.project, true, false);
      setShowArchivedOnly(false);
      focusContextualDraftRef.current = true;
      setContextualStart(null);
      if (isCompactWindow) setCompactPanel(null);
      if (isMobileWindow) setMobileScreen("conversation");
    },
    [
      contextualStart,
      getCurrentPageSnapshot,
      isCompactWindow,
      isMobileWindow,
      setChatGroupOpen,
      workspace?.name,
    ]
  );

  const continueCurrentChat = useCallback(() => {
    const store = useAgentStore.getState();
    const current = store.getChat(store.currentChatId);
    if (!current || current.archived || !store.canSelectChat(current.id))
      return;
    setContextualStart(null);
  }, []);

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
    if (!visible || !currentChatId || getCurrentAgentPageResource()) return;
    useAgentStore
      .getState()
      .ensureCurrentChat(
        getCurrentPageSnapshot(),
        getCurrentChatContext(),
        workspace?.name
      );
  }, [currentChatId, getCurrentPageSnapshot, visible, workspace?.name]);

  useEffect(() => {
    if (!visible || !currentChatId) return;

    let cancelled = false;
    void getOrFetchSettingByName(Setting_SettingName.AI, true).then(
      (setting) => {
        if (cancelled || getAgentAIConfigurationEnabled(setting) !== false) {
          return;
        }
        useAgentStore.getState().setChatStatus(currentChatId, "error", {
          requiresAIConfiguration: true,
        });
      }
    );

    return () => {
      cancelled = true;
    };
  }, [currentChatId, getOrFetchSettingByName, visible]);

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
      cancelRenameCurrentChat();
      setContextualStart(null);
      useAgentStore.getState().setCurrentChat(chatId);
      setMobileScreen("conversation");
    },
    [cancelRenameCurrentChat]
  );

  const renderConversation = () =>
    contextualStart ? (
      <AgentContextualStart
        page={contextualStart}
        title={contextualTitle}
        reference={
          canReferenceContextualResource ? contextualReference : undefined
        }
        loading={canReferenceContextualResource && contextualLoading}
        canStart={!isChatCreationDisabled}
        hasCurrentChat={canContinueCurrentChat}
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
          onSelectSuggestion={selectEmptyChatSuggestion}
        />
        <AgentInput
          retryActionRef={retryLastTurnRef}
          focusActionRef={focusInputRef}
        />
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
          disabled={isChatCreationDisabled}
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

  if (!visible) return null;

  if (minimized) {
    return createPortal(
      <div
        data-agent-window
        className="fixed bottom-4 right-4 flex size-10 cursor-pointer items-center justify-center rounded-full bg-accent text-accent-text shadow-lg hover:bg-accent-hover"
        onClick={() => useAgentStore.getState().restore()}
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          className="size-5"
          viewBox="0 0 20 20"
          fill="currentColor"
        >
          <path
            fillRule="evenodd"
            d="M18 10c0 3.866-3.582 7-8 7a8.841 8.841 0 01-4.083-.98L2 17l1.338-3.123C2.493 12.767 2 11.434 2 10c0-3.866 3.582-7 8-7s8 3.134 8 7zM7 9H5v2h2V9zm8 0h-2v2h2V9zm-4 0H9v2h2V9z"
            clipRule="evenodd"
          />
        </svg>
      </div>,
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
              {isMobileWindow && mobileScreen === "conversation" && (
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
              {isMobileWindow && mobileScreen === "archived" && (
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
              <span className="truncate text-sm font-medium">
                {isMobileWindow && mobileScreen === "chats"
                  ? t("agent.chat-list-label")
                  : isMobileWindow && mobileScreen === "archived"
                    ? t("agent.archived-only-chats")
                    : t("agent.assistant-title")}
              </span>
            </div>
            <div className="flex items-center gap-x-1">
              {isMobileWindow && mobileScreen === "chats" && (
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
                      disabled={isChatCreationDisabled}
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
              <Button
                appearance="secondary"
                size="xs"
                data-agent-window-action
                className="flex items-center justify-center rounded-xs text-control-placeholder hover:bg-control-bg-hover hover:text-control-light"
                title={t("agent.minimize")}
                onClick={(e) => {
                  e.stopPropagation();
                  useAgentStore.getState().minimize();
                }}
              >
                &#8722;
              </Button>
              <Button
                appearance="secondary"
                size="xs"
                data-agent-window-action
                className="flex items-center justify-center rounded-xs text-control-placeholder hover:bg-control-bg-hover hover:text-control-light"
                title={t("agent.close")}
                onClick={(e) => {
                  e.stopPropagation();
                  useAgentStore.getState().toggle();
                }}
              >
                &#10005;
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
                  className="flex size-full flex-col bg-background"
                  data-agent-mobile-conversation
                >
                  {renderConversation()}
                </div>
              ) : (
                <div
                  className="size-full overflow-y-auto bg-control-bg"
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
                                  disabled={isChatCreationDisabled}
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
                            {group.chats.map((chat) => (
                              <div
                                key={chat.id}
                                className={cn(
                                  "flex w-full items-center gap-x-2 py-2 pr-3 pl-5 text-sm transition-colors",
                                  !contextualStart && chat.id === currentChatId
                                    ? "bg-accent/10 text-accent"
                                    : "text-control hover:bg-control-bg-hover"
                                )}
                                data-agent-chat-row={chat.id}
                              >
                                <Button
                                  appearance="secondary"
                                  size="md"
                                  type="button"
                                  className="h-auto min-w-0 flex-1 justify-start p-0 text-left font-normal hover:bg-transparent"
                                  title={getChatLabel(chat)}
                                  disabled={
                                    !useAgentStore
                                      .getState()
                                      .canSelectChat(chat.id)
                                  }
                                  aria-current={
                                    !contextualStart &&
                                    chat.id === currentChatId
                                      ? "true"
                                      : undefined
                                  }
                                  onClick={() => selectMobileChat(chat.id)}
                                >
                                  <span className="truncate">
                                    {getChatLabel(chat)}
                                  </span>
                                </Button>
                                {chat.archived ? (
                                  <>
                                    <SidebarIconButton
                                      tooltip={t("agent.unarchive-chat")}
                                      ariaLabel={t("agent.unarchive-chat")}
                                      dataAttr="data-agent-unarchive-chat"
                                      onClick={() => unarchiveChat(chat.id)}
                                    >
                                      <Undo2
                                        className="size-3.5"
                                        aria-hidden="true"
                                      />
                                    </SidebarIconButton>
                                    <ConfirmDialog
                                      message={t(
                                        "agent.delete-chat-confirmation"
                                      )}
                                      onConfirm={() => deleteChat(chat.id)}
                                      triggerLabel={t("agent.delete-chat")}
                                      triggerDataAttr="data-agent-delete-chat"
                                      triggerVariant="icon"
                                    >
                                      <Trash2
                                        className="size-3.5"
                                        aria-hidden="true"
                                      />
                                    </ConfirmDialog>
                                  </>
                                ) : (
                                  <SidebarIconButton
                                    tooltip={t("agent.archive-chat")}
                                    ariaLabel={t("agent.archive-chat")}
                                    dataAttr="data-agent-archive-chat"
                                    onClick={() => archiveChat(chat.id)}
                                  >
                                    <Archive
                                      className="size-3.5"
                                      aria-hidden="true"
                                    />
                                  </SidebarIconButton>
                                )}
                              </div>
                            ))}
                          </CollapsiblePanel>
                        </Collapsible>
                      );
                    })}
                  </div>
                </div>
              )
            ) : (
              <div className="flex size-full" data-agent-desktop-layout>
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
                                        disabled={isChatCreationDisabled}
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
                                            disabled={isChatCreationDisabled}
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
                                      {group.chats.map((chat) => (
                                        <div
                                          key={chat.id}
                                          className={cn(
                                            "group w-full py-2 pr-3 pl-5 text-left text-sm transition-colors",
                                            !contextualStart &&
                                              chat.id === currentChatId
                                              ? "bg-accent/10 text-accent"
                                              : "text-control hover:bg-control-bg-hover"
                                          )}
                                          data-agent-chat-row={chat.id}
                                        >
                                          {!contextualStart &&
                                          chat.id === currentChatId &&
                                          isRenamingCurrentChat ? (
                                            <Input
                                              ref={renameInputRef}
                                              value={renamingTitle}
                                              onChange={(e) =>
                                                setRenamingTitle(e.target.value)
                                              }
                                              className="h-7 text-sm"
                                              placeholder={t(
                                                "agent.rename-chat-placeholder"
                                              )}
                                              data-agent-inline-rename-input
                                              onBlur={commitRenameCurrentChat}
                                              onKeyDown={onRenameKeydown}
                                            />
                                          ) : (
                                            <div className="relative flex min-w-0 items-center">
                                              <Button
                                                appearance="secondary"
                                                size="md"
                                                type="button"
                                                title={getChatLabel(chat)}
                                                className={cn(
                                                  "h-auto min-w-0 flex-1 flex-col items-start p-0 text-left font-normal hover:bg-transparent disabled:cursor-not-allowed disabled:opacity-60",
                                                  chat.archived
                                                    ? "pr-7"
                                                    : "group-hover:pr-7 group-focus-within:pr-7"
                                                )}
                                                disabled={
                                                  !useAgentStore
                                                    .getState()
                                                    .canSelectChat(chat.id)
                                                }
                                                aria-current={
                                                  !contextualStart &&
                                                  chat.id === currentChatId
                                                    ? "true"
                                                    : undefined
                                                }
                                                onClick={() =>
                                                  handleChatRowClick(chat.id)
                                                }
                                              >
                                                <div
                                                  className="w-full truncate"
                                                  data-agent-chat-title
                                                >
                                                  {getChatLabel(chat)}
                                                </div>
                                              </Button>
                                              {chat.archived ? (
                                                <AgentDropdownMenu>
                                                  <AgentDropdownMenuTrigger
                                                    className="absolute top-1/2 right-0 inline-flex size-6 -translate-y-1/2 items-center justify-center rounded-xs text-control-light outline-hidden hover:bg-control-bg-hover focus-visible:ring-2 focus-visible:ring-accent"
                                                    aria-label={t(
                                                      "common.more"
                                                    )}
                                                    data-agent-archived-chat-menu
                                                    onClick={(event) =>
                                                      event.stopPropagation()
                                                    }
                                                  >
                                                    <EllipsisVertical
                                                      className="size-3.5"
                                                      aria-hidden="true"
                                                    />
                                                  </AgentDropdownMenuTrigger>
                                                  <AgentDropdownMenuContent>
                                                    <AgentDropdownMenuItem
                                                      data-agent-unarchive-chat
                                                      onClick={(event) => {
                                                        event.stopPropagation();
                                                        unarchiveChat(chat.id);
                                                      }}
                                                    >
                                                      <Undo2
                                                        className="size-4"
                                                        aria-hidden="true"
                                                      />
                                                      {t(
                                                        "agent.unarchive-chat"
                                                      )}
                                                    </AgentDropdownMenuItem>
                                                    <AgentDropdownMenuItem
                                                      data-agent-delete-chat
                                                      className="text-error data-highlighted:bg-error/10"
                                                      onClick={(event) => {
                                                        event.stopPropagation();
                                                        setPendingDeleteChatId(
                                                          chat.id
                                                        );
                                                      }}
                                                    >
                                                      <Trash2
                                                        className="size-4"
                                                        aria-hidden="true"
                                                      />
                                                      {t("agent.delete-chat")}
                                                    </AgentDropdownMenuItem>
                                                  </AgentDropdownMenuContent>
                                                </AgentDropdownMenu>
                                              ) : (
                                                <div className="pointer-events-none absolute right-0 top-1/2 -translate-y-1/2 opacity-0 transition-opacity group-hover:pointer-events-auto group-hover:opacity-100 group-focus-within:pointer-events-auto group-focus-within:opacity-100">
                                                  <SidebarIconButton
                                                    tooltip={t(
                                                      "agent.archive-chat"
                                                    )}
                                                    ariaLabel={t(
                                                      "agent.archive-chat"
                                                    )}
                                                    dataAttr="data-agent-archive-chat"
                                                    onClick={() =>
                                                      archiveChat(chat.id)
                                                    }
                                                  >
                                                    <Archive
                                                      className="size-3.5"
                                                      aria-hidden="true"
                                                    />
                                                  </SidebarIconButton>
                                                </div>
                                              )}
                                            </div>
                                          )}
                                        </div>
                                      ))}
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

// --- Confirmation Dialog ---

function ConfirmDialog({
  children,
  message,
  onConfirm,
  triggerLabel,
  triggerDataAttr,
  triggerVariant = "text",
}: {
  children?: ReactNode;
  message: string;
  onConfirm: () => void;
  triggerLabel: string;
  triggerDataAttr?: string;
  triggerVariant?: "icon" | "text";
}) {
  const { t } = useTranslation();
  const triggerProps: Record<string, unknown> = {};
  if (triggerDataAttr) triggerProps[triggerDataAttr] = true;

  return (
    <AgentDialog>
      <AgentDialogTrigger
        render={
          triggerVariant === "icon" ? (
            <Button
              appearance="secondary"
              size="xs"
              className="text-control-light hover:bg-background"
              aria-label={triggerLabel}
              onClick={(event) => event.stopPropagation()}
              {...triggerProps}
            />
          ) : (
            <Button
              appearance="secondary"
              size="sm"
              className="rounded-xs border px-2 py-1.5 text-xs font-medium text-control-light hover:bg-background"
              onClick={(event) => event.stopPropagation()}
              {...triggerProps}
            />
          )
        }
      >
        {triggerVariant === "icon" ? (
          <AgentTooltip content={triggerLabel}>{children}</AgentTooltip>
        ) : (
          triggerLabel
        )}
      </AgentDialogTrigger>
      <AgentDialogContent className="max-w-sm p-6">
        <AgentDialogTitle className="sr-only">
          {t("common.confirm")}
        </AgentDialogTitle>
        <AgentDialogDescription className="mb-4">
          {message}
        </AgentDialogDescription>
        <div className="flex justify-end gap-x-2">
          <AgentDialogClose
            render={
              <Button
                appearance="secondary"
                size="md"
                className="h-auto rounded-xs border px-3 py-1.5 text-sm font-medium text-control-light hover:bg-control-bg"
              >
                {t("common.cancel")}
              </Button>
            }
          />
          <AgentDialogClose
            render={
              <Button
                appearance="secondary"
                size="md"
                className="h-auto rounded-xs bg-accent px-3 py-1.5 text-sm font-medium text-accent-text hover:bg-accent-hover"
                onClick={onConfirm}
              >
                {t("common.confirm")}
              </Button>
            }
          />
        </div>
      </AgentDialogContent>
    </AgentDialog>
  );
}

type SidebarIconButtonProps = Readonly<{
  ariaLabel: string;
  children: ReactNode;
  dataAttr?: string;
  onClick: () => void;
  tooltip: string;
}>;

function SidebarIconButton({
  ariaLabel,
  children,
  dataAttr,
  onClick,
  tooltip,
}: SidebarIconButtonProps) {
  const dataProps: Record<string, unknown> = {};
  if (dataAttr) dataProps[dataAttr] = true;

  return (
    <AgentTooltip content={tooltip} delayDuration={500}>
      <Button
        appearance="secondary"
        size="xs"
        className="text-control-light hover:bg-control-bg-hover"
        aria-label={ariaLabel}
        onClick={(event) => {
          event.stopPropagation();
          onClick();
        }}
        {...dataProps}
      >
        {children}
      </Button>
    </AgentTooltip>
  );
}
