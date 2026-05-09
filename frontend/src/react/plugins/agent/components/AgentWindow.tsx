import { Archive, EllipsisVertical, Plus, Trash2, Undo2 } from "lucide-react";
import {
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { HumanizeTs } from "@/react/components/HumanizeTs";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { getLayerRoot } from "@/react/components/ui/layer";
import { cn } from "@/react/lib/utils";
import type { AgentChat as AgentChatRecord } from "../logic/types";
import {
  selectCurrentChat,
  selectHasRunningChat,
  selectOrderedChats,
  useAgentStore,
} from "../store/agent";
import {
  MIN_HEIGHT,
  MIN_MAIN_PANEL_WIDTH,
  MIN_SIDEBAR_WIDTH,
  MIN_WIDTH,
  WINDOW_MARGIN,
} from "../window";
import { AgentChat } from "./AgentChat";
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
  AgentDropdownMenuSeparator,
  AgentDropdownMenuTrigger,
} from "./ui/AgentDropdownMenu";
import { AgentTooltip } from "./ui/AgentTooltip";
import {
  type ResizeBounds,
  type ResizeDirection,
  resizeWindowBounds,
} from "./window-resize";

const resizeZoneClasses: Record<ResizeDirection, string> = {
  n: "left-[12px] top-[-6px] h-[6px] w-[calc(100%-24px)] cursor-n-resize",
  s: "bottom-[-6px] left-[12px] h-[6px] w-[calc(100%-24px)] cursor-s-resize",
  e: "right-[-6px] top-[12px] h-[calc(100%-24px)] w-[6px] cursor-e-resize",
  w: "left-[-6px] top-[12px] h-[calc(100%-24px)] w-[6px] cursor-w-resize",
  ne: "right-[-6px] top-[-6px] size-[12px] cursor-ne-resize",
  nw: "left-[-6px] top-[-6px] size-[12px] cursor-nw-resize",
  se: "bottom-[-6px] right-[-6px] size-[12px] cursor-se-resize",
  sw: "bottom-[-6px] left-[-6px] size-[12px] cursor-sw-resize",
};

export function AgentWindow() {
  const { t } = useTranslation();

  const visible = useAgentStore((s) => s.visible);
  const minimized = useAgentStore((s) => s.minimized);
  const position = useAgentStore((s) => s.position);
  const size = useAgentStore((s) => s.size);
  const sidebarWidth = useAgentStore((s) => s.sidebarWidth);
  const orderedChats = useAgentStore(selectOrderedChats);
  const currentChat = useAgentStore(selectCurrentChat);
  const currentChatId = useAgentStore((s) => s.currentChatId);
  const hasRunningChat = useAgentStore(selectHasRunningChat);

  const windowRef = useRef<HTMLDivElement>(null);
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
  const [isRenamingCurrentChat, setIsRenamingCurrentChat] = useState(false);
  const [
    isDeleteAllArchivedChatsDialogOpen,
    setIsDeleteAllArchivedChatsDialogOpen,
  ] = useState(false);
  const [renamingTitle, setRenamingTitle] = useState("");

  // Refs for drag/resize intermediate values (avoid re-renders)
  const isDraggingRef = useRef(false);
  const dragOffsetRef = useRef({ x: 0, y: 0 });
  const dragPositionRef = useRef<{ x: number; y: number } | null>(null);
  const isResizingRef = useRef(false);
  const resizeStartRef = useRef<{
    x: number;
    y: number;
    bounds: ResizeBounds;
    direction: ResizeDirection;
  }>({
    x: 0,
    y: 0,
    bounds: { x: 0, y: 0, width: 0, height: 0 },
    direction: "se",
  });
  const isSidebarResizingRef = useRef(false);
  const sidebarResizeStartRef = useRef({ x: 0, width: 0 });
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

  const getDisplayPosition = useCallback(
    (x: number, y: number, displaySize?: { width: number; height: number }) => {
      const sz = displaySize ?? getDisplaySize(size.width, size.height);
      const maxX = Math.max(
        WINDOW_MARGIN,
        viewportSize.width - sz.width - WINDOW_MARGIN
      );
      const maxY = Math.max(
        WINDOW_MARGIN,
        viewportSize.height - sz.height - WINDOW_MARGIN
      );
      return {
        x: Math.min(maxX, Math.max(WINDOW_MARGIN, Math.round(x))),
        y: Math.min(maxY, Math.max(WINDOW_MARGIN, Math.round(y))),
      };
    },
    [
      getDisplaySize,
      size.width,
      size.height,
      viewportSize.width,
      viewportSize.height,
    ]
  );

  const displayWindowState = useMemo(() => {
    const sz = getDisplaySize(size.width, size.height);
    const pos = getDisplayPosition(position.x, position.y, sz);
    return { position: pos, size: sz };
  }, [
    getDisplaySize,
    getDisplayPosition,
    size.width,
    size.height,
    position.x,
    position.y,
  ]);

  const clampSidebarWidth = useCallback(
    (width: number, windowWidth = displayWindowState.size.width) => {
      const maxSW = Math.max(
        MIN_SIDEBAR_WIDTH,
        windowWidth - MIN_MAIN_PANEL_WIDTH
      );
      const minSW = Math.min(MIN_SIDEBAR_WIDTH, maxSW);
      return Math.min(maxSW, Math.max(minSW, Math.round(width)));
    },
    [displayWindowState.size.width]
  );

  const clampedSidebarWidth = useMemo(
    () => clampSidebarWidth(sidebarWidth),
    [clampSidebarWidth, sidebarWidth]
  );

  const windowStyle: React.CSSProperties = {
    left: `${displayWindowState.position.x}px`,
    top: `${displayWindowState.position.y}px`,
    width: `${displayWindowState.size.width}px`,
    height: `${displayWindowState.size.height}px`,
  };

  const sidebarStyle: React.CSSProperties = {
    width: `${clampedSidebarWidth}px`,
  };

  // --- Derived chat data ---

  const displayedChats = useMemo(
    () => orderedChats.filter((chat) => chat.archived === showArchivedOnly),
    [orderedChats, showArchivedOnly]
  );
  const activeChats = useMemo(
    () => orderedChats.filter((chat) => !chat.archived),
    [orderedChats]
  );
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

  const syncStoreToDisplayState = useCallback(() => {
    const store = useAgentStore.getState();
    store.setSize(
      displayWindowState.size.width,
      displayWindowState.size.height
    );
    store.setPosition(
      displayWindowState.position.x,
      displayWindowState.position.y
    );
  }, [displayWindowState]);

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
      store.createChat({ page: getCurrentPageSnapshot() });
    },
    [showArchivedOnly, selectFirstDisplayedChat, getCurrentPageSnapshot]
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

  const createChat = useCallback(() => {
    if (isChatCreationDisabled) return;
    useAgentStore.getState().createChat({ page: getCurrentPageSnapshot() });
  }, [isChatCreationDisabled, getCurrentPageSnapshot]);

  const handleChatRowClick = useCallback(
    (chatId: string) => {
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
    },
    [isRenamingCurrentChat, beginRenameCurrentChat, cancelRenameCurrentChat]
  );

  const archiveChat = useCallback(
    (chatId: string) => {
      const chat = useAgentStore.getState().chats.find((c) => c.id === chatId);
      if (!chat || chat.archived) return;
      useAgentStore.getState().archiveChat(chat.id);
      ensureCurrentChatMatchesDisplayedMode({
        allowCreateWhenEmpty: false,
      });
    },
    [ensureCurrentChatMatchesDisplayedMode]
  );

  const unarchiveChat = useCallback(
    (chatId: string) => {
      const chat = useAgentStore.getState().chats.find((c) => c.id === chatId);
      if (!chat?.archived) return;
      useAgentStore.getState().unarchiveChat(chat.id);
      ensureCurrentChatMatchesDisplayedMode({
        allowCreateWhenEmpty: false,
      });
    },
    [ensureCurrentChatMatchesDisplayedMode]
  );

  const archiveAllChats = useCallback(() => {
    if (isArchiveAllDisabled) return;
    const archivedCount = useAgentStore.getState().archiveAllActiveChats();
    if (archivedCount === 0) return;
    cancelRenameCurrentChat();
    ensureCurrentChatMatchesDisplayedMode({
      allowCreateWhenEmpty: false,
    });
  }, [
    cancelRenameCurrentChat,
    ensureCurrentChatMatchesDisplayedMode,
    isArchiveAllDisabled,
  ]);

  const unarchiveAllChats = useCallback(() => {
    if (isUnarchiveAllDisabled) return;
    const unarchivedCount = useAgentStore
      .getState()
      .unarchiveAllArchivedChats();
    if (unarchivedCount === 0) return;
    cancelRenameCurrentChat();
    ensureCurrentChatMatchesDisplayedMode({
      allowCreateWhenEmpty: false,
    });
  }, [
    cancelRenameCurrentChat,
    ensureCurrentChatMatchesDisplayedMode,
    isUnarchiveAllDisabled,
  ]);

  const deleteAllArchivedChats = useCallback(() => {
    if (isDeleteAllDisabled) return;
    const deletedCount = useAgentStore.getState().deleteAllArchivedChats();
    if (deletedCount === 0) return;
    cancelRenameCurrentChat();
    setIsDeleteAllArchivedChatsDialogOpen(false);
    ensureCurrentChatMatchesDisplayedMode({
      allowCreateWhenEmpty: false,
    });
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
      store.deleteChat(chat.id);
      ensureCurrentChatMatchesDisplayedMode({
        allowCreateWhenEmpty: false,
      });
    },
    [ensureCurrentChatMatchesDisplayedMode]
  );

  const toggleChatListMode = useCallback(() => {
    setShowArchivedOnly((prev) => !prev);
  }, []);

  // After showArchivedOnly changes, ensure current chat matches
  useEffect(() => {
    ensureCurrentChatMatchesDisplayedMode();
  }, [showArchivedOnly]);

  // --- Drag handlers ---
  // All drag/resize handlers manipulate DOM directly during movement
  // and only commit to the Zustand store on pointerup, avoiding per-frame
  // immer drafts + React re-renders.

  const sidebarRef = useRef<HTMLElement>(null);
  const dragCleanupRef = useRef<(() => void) | null>(null);
  const resizeCleanupRef = useRef<(() => void) | null>(null);
  const sidebarResizeCleanupRef = useRef<(() => void) | null>(null);

  // Cleanup drag/resize listeners on unmount
  useEffect(() => {
    return () => {
      dragCleanupRef.current?.();
      resizeCleanupRef.current?.();
      sidebarResizeCleanupRef.current?.();
    };
  }, []);

  const startDrag = useCallback(
    (event: React.PointerEvent<HTMLDivElement>) => {
      if (event.button !== 0) return;
      if (
        event.target instanceof HTMLElement &&
        event.target.closest("[data-agent-window-action]")
      ) {
        return;
      }
      event.preventDefault();
      syncStoreToDisplayState();
      isDraggingRef.current = true;
      const store = useAgentStore.getState();
      dragOffsetRef.current = {
        x: event.clientX - store.position.x,
        y: event.clientY - store.position.y,
      };
      dragPositionRef.current = {
        x: store.position.x,
        y: store.position.y,
      };

      const onDrag = (e: PointerEvent) => {
        if (!isDraggingRef.current || !windowRef.current) return;
        const el = windowRef.current;
        const vw = window.innerWidth;
        const vh = window.innerHeight;
        const w = el.offsetWidth;
        const h = el.offsetHeight;
        const maxX = Math.max(WINDOW_MARGIN, vw - w - WINDOW_MARGIN);
        const maxY = Math.max(WINDOW_MARGIN, vh - h - WINDOW_MARGIN);
        const x = Math.min(
          maxX,
          Math.max(WINDOW_MARGIN, e.clientX - dragOffsetRef.current.x)
        );
        const y = Math.min(
          maxY,
          Math.max(WINDOW_MARGIN, e.clientY - dragOffsetRef.current.y)
        );
        el.style.left = `${x}px`;
        el.style.top = `${y}px`;
        dragPositionRef.current = { x, y };
      };

      const stopDrag = () => {
        isDraggingRef.current = false;
        document.removeEventListener("pointermove", onDrag);
        document.removeEventListener("pointerup", stopDrag);
        document.removeEventListener("pointercancel", stopDrag);
        dragCleanupRef.current = null;
        if (dragPositionRef.current) {
          useAgentStore
            .getState()
            .setPosition(dragPositionRef.current.x, dragPositionRef.current.y);
        }
        dragPositionRef.current = null;
        useAgentStore.getState().saveWindowState();
      };

      document.addEventListener("pointermove", onDrag);
      document.addEventListener("pointerup", stopDrag);
      document.addEventListener("pointercancel", stopDrag);
      dragCleanupRef.current = stopDrag;
    },
    [syncStoreToDisplayState]
  );

  // --- Resize handlers ---

  const startResize = useCallback(
    (direction: ResizeDirection, event: React.PointerEvent<HTMLDivElement>) => {
      if (!canResizeWindow || event.button !== 0) return;
      event.preventDefault();
      event.stopPropagation();
      syncStoreToDisplayState();
      isResizingRef.current = true;
      const store = useAgentStore.getState();
      resizeStartRef.current = {
        x: event.clientX,
        y: event.clientY,
        bounds: {
          x: store.position.x,
          y: store.position.y,
          width: store.size.width,
          height: store.size.height,
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
        el.style.left = `${nextBounds.x}px`;
        el.style.top = `${nextBounds.y}px`;
        el.style.width = `${nextBounds.width}px`;
        el.style.height = `${nextBounds.height}px`;
        const store = useAgentStore.getState();
        store.setSize(nextBounds.width, nextBounds.height);
        store.setPosition(nextBounds.x, nextBounds.y);
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
    [canResizeWindow, syncStoreToDisplayState]
  );

  // --- Sidebar resize ---

  const startSidebarResize = useCallback(
    (event: React.PointerEvent<HTMLButtonElement>) => {
      if (event.button !== 0) return;
      event.preventDefault();
      event.stopPropagation();
      syncStoreToDisplayState();
      isSidebarResizingRef.current = true;
      sidebarResizeStartRef.current = {
        x: event.clientX,
        width: clampedSidebarWidth,
      };

      const onSidebarResize = (e: PointerEvent) => {
        if (!isSidebarResizingRef.current || !sidebarRef.current) return;
        const dx = e.clientX - sidebarResizeStartRef.current.x;
        const newWidth = sidebarResizeStartRef.current.width + dx;
        const windowWidth = windowRef.current?.offsetWidth ?? 0;
        const maxSW = Math.max(
          MIN_SIDEBAR_WIDTH,
          windowWidth - MIN_MAIN_PANEL_WIDTH
        );
        const minSW = Math.min(MIN_SIDEBAR_WIDTH, maxSW);
        const clamped = Math.min(maxSW, Math.max(minSW, Math.round(newWidth)));
        sidebarRef.current.style.width = `${clamped}px`;
        useAgentStore.getState().setSidebarWidth(clamped);
      };

      const stopSidebarResize = () => {
        isSidebarResizingRef.current = false;
        document.removeEventListener("pointermove", onSidebarResize);
        document.removeEventListener("pointerup", stopSidebarResize);
        document.removeEventListener("pointercancel", stopSidebarResize);
        sidebarResizeCleanupRef.current = null;
        useAgentStore.getState().saveWindowState();
      };

      document.addEventListener("pointermove", onSidebarResize);
      document.addEventListener("pointerup", stopSidebarResize);
      document.addEventListener("pointercancel", stopSidebarResize);
      sidebarResizeCleanupRef.current = stopSidebarResize;
    },
    [syncStoreToDisplayState, clampedSidebarWidth]
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
        const maxX = Math.max(
          WINDOW_MARGIN,
          viewportSize.width - clW - WINDOW_MARGIN
        );
        const maxY = Math.max(
          WINDOW_MARGIN,
          viewportSize.height - clH - WINDOW_MARGIN
        );
        state.position.x = Math.min(
          maxX,
          Math.max(WINDOW_MARGIN, Math.round(state.position.x))
        );
        state.position.y = Math.min(
          maxY,
          Math.max(WINDOW_MARGIN, Math.round(state.position.y))
        );
      });
      useAgentStore.getState().saveWindowState();
    });

    observer.observe(el);
    resizeObserverRef.current = observer;
    return () => observer.disconnect();
  }, [visible, minimized, viewportSize.width, viewportSize.height]);

  // --- Sync sidebar width when window width changes ---

  useEffect(() => {
    const store = useAgentStore.getState();
    const windowWidth = displayWindowState.size.width;
    const maxSW = Math.max(
      MIN_SIDEBAR_WIDTH,
      windowWidth - MIN_MAIN_PANEL_WIDTH
    );
    const minSW = Math.min(MIN_SIDEBAR_WIDTH, maxSW);
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
  }, [displayWindowState.size.width]);

  // --- Mount: load window state ---

  useEffect(() => {
    useAgentStore.getState().loadWindowState();
  }, []);

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
                <Button variant="outline" type="button">
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
      <div className="flex size-full flex-col overflow-hidden rounded-lg border border-block-border bg-background shadow-xl">
        {/* Header */}
        <div
          data-agent-window-header
          className="flex cursor-move select-none items-center justify-between border-b bg-control-bg px-3 py-2 [touch-action:none]"
          onPointerDown={startDrag}
        >
          <div className="flex min-w-0 items-center gap-x-2">
            <span className="truncate text-sm font-medium">
              {t("agent.assistant-title")}
            </span>
          </div>
          <div className="flex items-center gap-x-1">
            <button
              data-agent-window-action
              className="flex size-5 items-center justify-center rounded-xs text-control-placeholder hover:bg-control-bg-hover hover:text-control-light"
              title={t("agent.minimize")}
              onClick={(e) => {
                e.stopPropagation();
                useAgentStore.getState().minimize();
              }}
            >
              &#8722;
            </button>
            <button
              data-agent-window-action
              className="flex size-5 items-center justify-center rounded-xs text-control-placeholder hover:bg-control-bg-hover hover:text-control-light"
              title={t("agent.close")}
              onClick={(e) => {
                e.stopPropagation();
                useAgentStore.getState().toggle();
              }}
            >
              &#10005;
            </button>
          </div>
        </div>

        {/* Body */}
        <div className="flex min-h-0 flex-1 overflow-hidden bg-background">
          {/* Sidebar */}
          <aside
            ref={sidebarRef}
            className="flex shrink-0 flex-col border-r border-block-border bg-control-bg"
            style={sidebarStyle}
          >
            {/* Sidebar header */}
            <div className="border-b border-block-border px-3 py-3">
              <div className="flex items-center justify-between gap-x-2">
                <div>
                  <h2 className="text-xs font-semibold uppercase tracking-wide text-control-light">
                    {t("agent.chat-list-label")}
                  </h2>
                </div>
                <div
                  className="flex items-center gap-x-2"
                  data-agent-chat-sidebar-actions
                >
                  <AgentTooltip content={t("agent.new-chat")}>
                    <Button
                      variant="outline"
                      size="sm"
                      className="text-control-light"
                      aria-label={t("agent.new-chat")}
                      disabled={isChatCreationDisabled}
                      onClick={createChat}
                    >
                      <Plus className="size-4" aria-hidden="true" />
                    </Button>
                  </AgentTooltip>
                  <AgentDropdownMenu>
                    <AgentTooltip content={t("common.more")}>
                      <AgentDropdownMenuTrigger
                        className="inline-flex size-7 items-center justify-center rounded-xs border border-control-border bg-transparent text-control-light outline-hidden hover:bg-control-bg focus-visible:ring-2 focus-visible:ring-accent disabled:pointer-events-none disabled:opacity-50"
                        aria-label={t("common.more")}
                        onClick={(event) => event.stopPropagation()}
                      >
                        <EllipsisVertical
                          className="size-4"
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
                            className="text-error data-highlighted:bg-red-50"
                            disabled={isDeleteAllDisabled}
                            onClick={(event) => {
                              event.stopPropagation();
                              setIsDeleteAllArchivedChatsDialogOpen(true);
                            }}
                          >
                            {t("agent.delete-all-chats")}
                          </AgentDropdownMenuItem>
                        </>
                      ) : (
                        <AgentDropdownMenuItem
                          data-agent-archive-all-chats
                          disabled={isArchiveAllDisabled}
                          onClick={(event) => {
                            event.stopPropagation();
                            archiveAllChats();
                          }}
                        >
                          {t("agent.archive-all-chats")}
                        </AgentDropdownMenuItem>
                      )}
                      <AgentDropdownMenuSeparator />
                      <AgentDropdownMenuItem
                        data-agent-chat-list-mode
                        onClick={(event) => {
                          event.stopPropagation();
                          toggleChatListMode();
                        }}
                      >
                        {showArchivedOnly
                          ? t("agent.active-only-chats")
                          : t("agent.archived-only-chats")}
                      </AgentDropdownMenuItem>
                    </AgentDropdownMenuContent>
                  </AgentDropdownMenu>
                </div>
              </div>
            </div>

            {/* Chat list */}
            <div className="min-h-0 flex-1 overflow-y-auto px-2 py-2">
              <div className="flex flex-col gap-y-1" data-agent-chat-list>
                {displayedChats.map((chat) => (
                  <div
                    key={chat.id}
                    className={`group w-full rounded-xs px-3 py-2 text-left text-sm transition-colors ${
                      chat.id === currentChatId
                        ? "bg-accent/10 text-accent"
                        : "text-control hover:bg-background"
                    }`}
                    data-agent-chat-row={chat.id}
                  >
                    {chat.id === currentChatId && isRenamingCurrentChat ? (
                      <Input
                        ref={renameInputRef}
                        value={renamingTitle}
                        onChange={(e) => setRenamingTitle(e.target.value)}
                        className="h-7 text-sm"
                        placeholder={t("agent.rename-chat-placeholder")}
                        data-agent-inline-rename-input
                        onBlur={commitRenameCurrentChat}
                        onKeyDown={onRenameKeydown}
                      />
                    ) : (
                      <div className="flex items-start gap-x-2">
                        <button
                          type="button"
                          className="min-w-0 flex-1 text-left disabled:cursor-not-allowed disabled:opacity-60"
                          disabled={
                            !useAgentStore.getState().canSelectChat(chat.id)
                          }
                          aria-current={
                            chat.id === currentChatId ? "true" : undefined
                          }
                          onClick={() => handleChatRowClick(chat.id)}
                        >
                          <div
                            className="truncate font-medium"
                            data-agent-chat-title
                          >
                            {getChatLabel(chat)}
                          </div>
                          <span
                            className={`mt-1 block truncate text-xs ${
                              chat.id === currentChatId
                                ? "text-accent/80"
                                : "text-control-light"
                            }`}
                            data-agent-chat-updated-ts
                          >
                            <HumanizeTs
                              ts={Math.floor(chat.updatedTs / 1000)}
                            />
                          </span>
                        </button>
                        <div className="pointer-events-none flex shrink-0 items-center gap-x-2 opacity-0 transition-opacity group-hover:pointer-events-auto group-hover:opacity-100 group-focus-within:pointer-events-auto group-focus-within:opacity-100">
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
                                message={t("agent.delete-chat-confirmation")}
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
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          </aside>

          {/* Sidebar resize handle */}
          <button
            type="button"
            data-agent-window-action
            data-agent-sidebar-resize
            className="group relative w-1 shrink-0 cursor-col-resize bg-control-bg transition-colors hover:bg-accent/10 [touch-action:none]"
            onPointerDown={startSidebarResize}
          >
            <span className="pointer-events-none absolute inset-y-0 left-1/2 w-px -translate-x-1/2 bg-transparent transition-colors group-hover:bg-accent" />
          </button>

          {/* Main panel */}
          <div className="flex min-w-0 flex-1 flex-col">
            <AgentChat className="min-h-0 flex-1" />
            <AgentInput />
          </div>
        </div>
      </div>

      {canResizeWindow &&
        (Object.keys(resizeZoneClasses) as ResizeDirection[]).map(
          (direction) => (
            <div
              key={direction}
              aria-hidden="true"
              data-agent-window-resize-zone={direction}
              className={cn(
                "absolute z-[1] [touch-action:none]",
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
              variant="ghost"
              size="xs"
              className="text-control-light hover:bg-background"
              aria-label={triggerLabel}
              onClick={(event) => event.stopPropagation()}
              {...triggerProps}
            />
          ) : (
            <button
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
              <button className="rounded-xs border px-3 py-1.5 text-sm font-medium text-control-light hover:bg-control-bg">
                {t("common.cancel")}
              </button>
            }
          />
          <AgentDialogClose
            render={
              <button
                className="rounded-xs bg-accent px-3 py-1.5 text-sm font-medium text-accent-text hover:bg-accent-hover"
                onClick={onConfirm}
              >
                {t("common.confirm")}
              </button>
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
    <AgentTooltip content={tooltip}>
      <Button
        variant="ghost"
        size="xs"
        className="text-control-light hover:bg-background"
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
