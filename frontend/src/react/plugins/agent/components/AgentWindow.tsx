import { Archive, Inbox } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { HumanizeTs } from "@/react/components/HumanizeTs";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogTitle,
  DialogTrigger,
} from "@/react/components/ui/dialog";
import { Input } from "@/react/components/ui/input";
import type { AgentChat as AgentChatRecord } from "../logic/types";
import {
  selectCurrentChat,
  selectHasRunningChat,
  selectOrderedChats,
  useAgentStore,
} from "../store/agent";
import { AgentChat } from "./AgentChat";
import { AgentInput } from "./AgentInput";

const MIN_WIDTH = 300;
const MIN_HEIGHT = 400;
const WINDOW_MARGIN = 16;
const MIN_SIDEBAR_WIDTH = 180;
const MIN_MAIN_PANEL_WIDTH = 240;

const tokenFormatter = new Intl.NumberFormat();

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
  const isViewportResizingRef = useRef(false);
  const viewportResizeFrameRef = useRef(0);

  const [showArchivedOnly, setShowArchivedOnly] = useState(false);
  const [isRenamingCurrentChat, setIsRenamingCurrentChat] = useState(false);
  const [renamingTitle, setRenamingTitle] = useState("");

  // Refs for drag/resize intermediate values (avoid re-renders)
  const isDraggingRef = useRef(false);
  const dragOffsetRef = useRef({ x: 0, y: 0 });
  const isResizingRef = useRef(false);
  const resizeStartRef = useRef({ x: 0, y: 0, w: 0, h: 0 });
  const isSidebarResizingRef = useRef(false);
  const sidebarResizeStartRef = useRef({ x: 0, width: 0 });
  const resizeObserverRef = useRef<ResizeObserver | null>(null);

  // --- Clamping helpers ---

  const maxWidth = useCallback(
    () => Math.max(MIN_WIDTH, viewportSize.width - WINDOW_MARGIN * 2),
    [viewportSize.width]
  );
  const maxHeight = useCallback(
    () => Math.max(MIN_HEIGHT, viewportSize.height - WINDOW_MARGIN * 2),
    [viewportSize.height]
  );

  const clampWidth = useCallback(
    (width: number) =>
      Math.min(maxWidth(), Math.max(MIN_WIDTH, Math.round(width))),
    [maxWidth]
  );
  const clampHeight = useCallback(
    (height: number) =>
      Math.min(maxHeight(), Math.max(MIN_HEIGHT, Math.round(height))),
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

  const isCurrentChatInDisplayedMode = useMemo(
    () => !!currentChat && currentChat.archived === showArchivedOnly,
    [currentChat, showArchivedOnly]
  );

  const currentChatStatusLabel = useMemo(() => {
    if (!currentChat) return t("agent.chat-status-idle");
    if (currentChat.interrupted) return t("agent.chat-status-interrupted");
    switch (currentChat.status) {
      case "running":
        return t("agent.chat-status-running");
      case "awaiting_user":
        return t("agent.chat-status-awaiting-user");
      case "error":
        return t("agent.chat-status-error");
      default:
        return t("agent.chat-status-idle");
    }
  }, [currentChat, t]);

  const currentChatStatusClass = useMemo(() => {
    if (!currentChat || currentChat.status === "idle")
      return "bg-gray-100 text-gray-600";
    if (currentChat.status === "running") return "bg-blue-50 text-blue-600";
    if (currentChat.interrupted || currentChat.status === "error")
      return "bg-red-50 text-red-600";
    return "bg-amber-50 text-amber-600";
  }, [currentChat]);

  const currentChatTokenUsageLabel = useMemo(
    () =>
      t("agent.chat-total-tokens", {
        count: tokenFormatter.format(currentChat?.totalTokensUsed ?? 0),
      }),
    [currentChat?.totalTokensUsed, t]
  );

  const isChatCreationDisabled = hasRunningChat;

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
    const first = displayedChats[0];
    if (!first || !useAgentStore.getState().canSelectChat(first.id))
      return false;
    useAgentStore.getState().setCurrentChat(first.id);
    return useAgentStore.getState().currentChatId === first.id;
  }, [displayedChats]);

  const ensureCurrentChatMatchesDisplayedMode = useCallback(
    (options: { fallbackToActiveWhenEmpty?: boolean } = {}) => {
      const store = useAgentStore.getState();
      const chat = store.chats.find((c) => c.id === store.currentChatId);
      const isInDisplayedMode = !!chat && chat.archived === showArchivedOnly;

      if (isInDisplayedMode) return;
      if (selectFirstDisplayedChat()) return;

      if (showArchivedOnly) {
        if (!options.fallbackToActiveWhenEmpty) return;
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

      if (store.chats.some((c) => c.status === "running")) return;
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

  const toggleArchiveCurrentChat = useCallback(() => {
    const chat = useAgentStore
      .getState()
      .chats.find((c) => c.id === useAgentStore.getState().currentChatId);
    if (!chat) return;
    if (chat.archived) {
      useAgentStore.getState().unarchiveChat(chat.id);
      ensureCurrentChatMatchesDisplayedMode({
        fallbackToActiveWhenEmpty: true,
      });
      return;
    }
    useAgentStore.getState().archiveChat(chat.id);
    ensureCurrentChatMatchesDisplayedMode();
  }, [ensureCurrentChatMatchesDisplayedMode]);

  const deleteCurrentChat = useCallback(() => {
    const store = useAgentStore.getState();
    const chat = store.chats.find((c) => c.id === store.currentChatId);
    if (!chat) return;
    store.deleteChat(chat.id);
    ensureCurrentChatMatchesDisplayedMode({
      fallbackToActiveWhenEmpty: true,
    });
  }, [ensureCurrentChatMatchesDisplayedMode]);

  const toggleChatListMode = useCallback(() => {
    setShowArchivedOnly((prev) => !prev);
  }, []);

  // After showArchivedOnly changes, ensure current chat matches
  useEffect(() => {
    ensureCurrentChatMatchesDisplayedMode();
  }, [showArchivedOnly]);

  // --- Drag handlers ---
  // All drag/resize handlers manipulate DOM directly during movement
  // and only commit to the Zustand store on mouseup, avoiding per-frame
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
    (event: React.MouseEvent) => {
      if (
        event.target instanceof HTMLElement &&
        event.target.closest(
          "[data-agent-window-action], [data-agent-window-resize]"
        )
      ) {
        return;
      }
      syncStoreToDisplayState();
      isDraggingRef.current = true;
      const store = useAgentStore.getState();
      dragOffsetRef.current = {
        x: event.clientX - store.position.x,
        y: event.clientY - store.position.y,
      };

      const onDrag = (e: MouseEvent) => {
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
        useAgentStore.getState().setPosition(x, y);
      };

      const stopDrag = () => {
        isDraggingRef.current = false;
        document.removeEventListener("mousemove", onDrag);
        document.removeEventListener("mouseup", stopDrag);
        dragCleanupRef.current = null;
        useAgentStore.getState().saveWindowState();
      };

      document.addEventListener("mousemove", onDrag);
      document.addEventListener("mouseup", stopDrag);
      dragCleanupRef.current = stopDrag;
    },
    [syncStoreToDisplayState]
  );

  // --- Resize handlers ---

  const startResize = useCallback(
    (event: React.MouseEvent) => {
      event.preventDefault();
      event.stopPropagation();
      syncStoreToDisplayState();
      isResizingRef.current = true;
      const store = useAgentStore.getState();
      resizeStartRef.current = {
        x: event.clientX,
        y: event.clientY,
        w: store.size.width,
        h: store.size.height,
      };

      const onResize = (e: MouseEvent) => {
        if (!isResizingRef.current || !windowRef.current) return;
        const el = windowRef.current;
        const vw = window.innerWidth;
        const vh = window.innerHeight;
        const dx = e.clientX - resizeStartRef.current.x;
        const dy = e.clientY - resizeStartRef.current.y;
        const clW = Math.min(
          Math.max(MIN_WIDTH, vw - WINDOW_MARGIN * 2),
          Math.max(MIN_WIDTH, Math.round(resizeStartRef.current.w + dx))
        );
        const clH = Math.min(
          Math.max(MIN_HEIGHT, vh - WINDOW_MARGIN * 2),
          Math.max(MIN_HEIGHT, Math.round(resizeStartRef.current.h + dy))
        );
        el.style.width = `${clW}px`;
        el.style.height = `${clH}px`;
        // Clamp position
        const maxX = Math.max(WINDOW_MARGIN, vw - clW - WINDOW_MARGIN);
        const maxY = Math.max(WINDOW_MARGIN, vh - clH - WINDOW_MARGIN);
        const curX = parseInt(el.style.left) || 0;
        const curY = parseInt(el.style.top) || 0;
        const clX = Math.min(maxX, Math.max(WINDOW_MARGIN, curX));
        const clY = Math.min(maxY, Math.max(WINDOW_MARGIN, curY));
        el.style.left = `${clX}px`;
        el.style.top = `${clY}px`;
        const store = useAgentStore.getState();
        store.setSize(clW, clH);
        store.setPosition(clX, clY);
      };

      const stopResize = () => {
        isResizingRef.current = false;
        document.removeEventListener("mousemove", onResize);
        document.removeEventListener("mouseup", stopResize);
        resizeCleanupRef.current = null;
        useAgentStore.getState().saveWindowState();
      };

      document.addEventListener("mousemove", onResize);
      document.addEventListener("mouseup", stopResize);
      resizeCleanupRef.current = stopResize;
    },
    [syncStoreToDisplayState]
  );

  // --- Sidebar resize ---

  const startSidebarResize = useCallback(
    (event: React.MouseEvent) => {
      event.preventDefault();
      event.stopPropagation();
      syncStoreToDisplayState();
      isSidebarResizingRef.current = true;
      sidebarResizeStartRef.current = {
        x: event.clientX,
        width: clampedSidebarWidth,
      };

      const onSidebarResize = (e: MouseEvent) => {
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
        document.removeEventListener("mousemove", onSidebarResize);
        document.removeEventListener("mouseup", stopSidebarResize);
        sidebarResizeCleanupRef.current = null;
        useAgentStore.getState().saveWindowState();
      };

      document.addEventListener("mousemove", onSidebarResize);
      document.addEventListener("mouseup", stopSidebarResize);
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
        const clW = Math.min(
          Math.max(MIN_WIDTH, viewportSize.width - WINDOW_MARGIN * 2),
          Math.max(MIN_WIDTH, Math.round(width))
        );
        const clH = Math.min(
          Math.max(MIN_HEIGHT, viewportSize.height - WINDOW_MARGIN * 2),
          Math.max(MIN_HEIGHT, Math.round(height))
        );
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
        className="fixed bottom-4 right-4 z-[1999] flex h-10 w-10 cursor-pointer items-center justify-center rounded-full bg-blue-500 text-white shadow-lg hover:bg-blue-600"
        onClick={() => useAgentStore.getState().restore()}
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          className="h-5 w-5"
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
      document.body
    );
  }

  return createPortal(
    <div
      ref={windowRef}
      data-agent-window
      className="fixed z-[1999] flex flex-col overflow-hidden rounded-lg border border-gray-200 bg-white shadow-xl"
      style={windowStyle}
    >
      {/* Header */}
      <div
        className="flex cursor-move select-none items-center justify-between border-b bg-gray-50 px-3 py-2"
        onMouseDown={startDrag}
      >
        <div className="flex min-w-0 items-center gap-x-2">
          <span className="truncate text-sm font-medium">
            {t("agent.assistant-title")}
          </span>
          <span
            className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${currentChatStatusClass}`}
          >
            {currentChatStatusLabel}
          </span>
          <span className="truncate text-xs text-gray-500">
            {currentChatTokenUsageLabel}
          </span>
        </div>
        <div className="flex items-center gap-x-1">
          <button
            data-agent-window-action
            className="flex h-5 w-5 items-center justify-center rounded-xs text-gray-400 hover:bg-gray-200 hover:text-gray-600"
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
            className="flex h-5 w-5 items-center justify-center rounded-xs text-gray-400 hover:bg-gray-200 hover:text-gray-600"
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
      <div className="flex min-h-0 flex-1 overflow-hidden bg-white">
        {/* Sidebar */}
        <aside
          ref={sidebarRef}
          className="flex shrink-0 flex-col border-r border-gray-200 bg-gray-50"
          style={sidebarStyle}
        >
          {/* Sidebar header */}
          <div className="border-b border-gray-200 px-3 py-3">
            <div className="flex items-center justify-between gap-x-2">
              <div>
                <h2 className="text-xs font-semibold uppercase tracking-wide text-gray-500">
                  {t("agent.chat-list-label")}
                </h2>
              </div>
              <button
                className="rounded-xs border px-2 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-100 disabled:cursor-not-allowed disabled:bg-gray-100 disabled:text-gray-400"
                disabled={isChatCreationDisabled}
                onClick={createChat}
              >
                {t("agent.new-chat")}
              </button>
            </div>
          </div>

          {/* Chat list */}
          <div className="min-h-0 flex-1 overflow-y-auto px-2 py-2">
            <div className="flex flex-col gap-y-1" data-agent-chat-list>
              {displayedChats.map((chat) => (
                <div
                  key={chat.id}
                  className={`w-full rounded-xs px-3 py-2 text-left text-sm transition-colors ${
                    chat.id === currentChatId
                      ? "bg-blue-50 text-blue-700"
                      : "text-gray-700 hover:bg-white"
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
                    <button
                      type="button"
                      className="w-full text-left disabled:cursor-not-allowed disabled:opacity-60"
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
                            ? "text-blue-600/80"
                            : "text-gray-500"
                        }`}
                        data-agent-chat-updated-ts
                      >
                        <HumanizeTs ts={Math.floor(chat.updatedTs / 1000)} />
                      </span>
                    </button>
                  )}
                </div>
              ))}
            </div>
          </div>

          {/* Sidebar footer */}
          <div className="border-t border-gray-200 px-3 py-3">
            <div className="flex flex-col gap-y-2">
              <div
                className="flex flex-wrap gap-x-2 gap-y-2"
                data-agent-chat-sidebar-actions
              >
                {isCurrentChatInDisplayedMode && (
                  <>
                    {showArchivedOnly ? (
                      <button
                        className="rounded-xs border px-2 py-1.5 text-xs font-medium text-gray-600 hover:bg-white"
                        data-agent-unarchive-chat
                        onClick={toggleArchiveCurrentChat}
                      >
                        {t("agent.unarchive-chat")}
                      </button>
                    ) : (
                      <ConfirmDialog
                        message={t("agent.archive-chat-confirmation")}
                        onConfirm={toggleArchiveCurrentChat}
                        triggerClassName="rounded-xs border px-2 py-1.5 text-xs font-medium text-gray-600 hover:bg-white"
                        triggerLabel={t("agent.archive-chat")}
                        triggerDataAttr="data-agent-archive-chat"
                      />
                    )}
                    {showArchivedOnly && (
                      <ConfirmDialog
                        message={t("agent.delete-chat-confirmation")}
                        onConfirm={deleteCurrentChat}
                        triggerClassName="rounded-xs border px-2 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50"
                        triggerLabel={t("agent.delete-chat")}
                        triggerDataAttr="data-agent-delete-chat"
                      />
                    )}
                  </>
                )}
                <button
                  className="ml-auto inline-flex items-center rounded-xs border p-1.5 text-xs font-medium text-gray-600 hover:bg-white"
                  aria-label={
                    showArchivedOnly
                      ? t("agent.archived-only-chats")
                      : t("agent.active-only-chats")
                  }
                  title={
                    showArchivedOnly
                      ? t("agent.archived-only-chats")
                      : t("agent.active-only-chats")
                  }
                  data-agent-chat-list-mode
                  onClick={toggleChatListMode}
                >
                  {showArchivedOnly ? (
                    <Archive className="h-3.5 w-3.5" aria-hidden="true" />
                  ) : (
                    <Inbox className="h-3.5 w-3.5" aria-hidden="true" />
                  )}
                </button>
              </div>
            </div>
          </div>
        </aside>

        {/* Sidebar resize handle */}
        <button
          type="button"
          data-agent-window-action
          data-agent-sidebar-resize
          className="group relative w-1 shrink-0 cursor-col-resize bg-gray-100 transition-colors hover:bg-blue-100"
          onMouseDown={startSidebarResize}
        >
          <span className="pointer-events-none absolute inset-y-0 left-1/2 w-px -translate-x-1/2 bg-transparent transition-colors group-hover:bg-blue-400" />
        </button>

        {/* Main panel */}
        <div className="flex min-w-0 flex-1 flex-col">
          <AgentChat className="min-h-0 flex-1" />
          <AgentInput />
        </div>
      </div>

      {/* Resize handle (SE corner) */}
      <button
        type="button"
        data-agent-window-action
        data-agent-window-resize
        className="absolute bottom-0 right-0 flex h-5 w-5 cursor-se-resize items-end justify-end pb-0.5 pr-0.5 text-gray-300 hover:text-gray-400"
        title={t("agent.resize")}
        onMouseDown={startResize}
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          className="h-3 w-3"
          viewBox="0 0 12 12"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.5"
        >
          <path d="M3.5 8.5h.01M6 6h.01M8.5 3.5h.01" />
          <path d="M3.5 11 11 3.5" />
        </svg>
      </button>
    </div>,
    document.body
  );
}

// --- Confirmation Dialog ---

function ConfirmDialog({
  message,
  onConfirm,
  triggerClassName,
  triggerLabel,
  triggerDataAttr,
}: {
  message: string;
  onConfirm: () => void;
  triggerClassName: string;
  triggerLabel: string;
  triggerDataAttr?: string;
}) {
  const { t } = useTranslation();
  const triggerProps: Record<string, unknown> = {};
  if (triggerDataAttr) triggerProps[triggerDataAttr] = true;

  return (
    <Dialog>
      <DialogTrigger className={triggerClassName} {...triggerProps}>
        {triggerLabel}
      </DialogTrigger>
      <DialogContent className="max-w-sm p-6">
        <DialogTitle className="sr-only">{t("common.confirm")}</DialogTitle>
        <DialogDescription className="mb-4">{message}</DialogDescription>
        <div className="flex justify-end gap-x-2">
          <DialogClose
            render={
              <button className="rounded-xs border px-3 py-1.5 text-sm font-medium text-gray-600 hover:bg-gray-100">
                {t("common.cancel")}
              </button>
            }
          />
          <DialogClose
            render={
              <button
                className="rounded-xs bg-blue-500 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-600"
                onClick={onConfirm}
              >
                {t("common.confirm")}
              </button>
            }
          />
        </div>
      </DialogContent>
    </Dialog>
  );
}
