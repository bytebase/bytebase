<script setup lang="ts">
import { NInput, NPopconfirm } from "naive-ui";
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import HumanizeTs from "@/components/misc/HumanizeTs.vue";
import type { AgentChat as AgentChatRecord } from "../logic/types";
import { useAgentStore } from "../store/agent";
import AgentChat from "./AgentChat.vue";
import AgentInput from "./AgentInput.vue";

const { t } = useI18n();
const route = useRoute();
const agentStore = useAgentStore();

const MIN_WIDTH = 300;
const MIN_HEIGHT = 400;
const WINDOW_MARGIN = 16;
const MIN_SIDEBAR_WIDTH = 180;
const MIN_MAIN_PANEL_WIDTH = 240;

const windowRef = ref<HTMLElement | null>(null);
const viewportSize = ref({
  width: window.innerWidth,
  height: window.innerHeight,
});
const isViewportResizing = ref(false);
let viewportResizeFrame = 0;

const maxWidth = () =>
  Math.max(MIN_WIDTH, viewportSize.value.width - WINDOW_MARGIN * 2);
const maxHeight = () =>
  Math.max(MIN_HEIGHT, viewportSize.value.height - WINDOW_MARGIN * 2);

const clampWidth = (width: number) =>
  Math.min(maxWidth(), Math.max(MIN_WIDTH, Math.round(width)));
const clampHeight = (height: number) =>
  Math.min(maxHeight(), Math.max(MIN_HEIGHT, Math.round(height)));

function getDisplaySize(width: number, height: number) {
  return {
    width: clampWidth(width),
    height: clampHeight(height),
  };
}

function getDisplayPosition(
  x: number,
  y: number,
  size = getDisplaySize(agentStore.size.width, agentStore.size.height)
) {
  const maxX = Math.max(
    WINDOW_MARGIN,
    viewportSize.value.width - size.width - WINDOW_MARGIN
  );
  const maxY = Math.max(
    WINDOW_MARGIN,
    viewportSize.value.height - size.height - WINDOW_MARGIN
  );
  return {
    x: Math.min(maxX, Math.max(WINDOW_MARGIN, Math.round(x))),
    y: Math.min(maxY, Math.max(WINDOW_MARGIN, Math.round(y))),
  };
}

const displayWindowState = computed(() => {
  const size = getDisplaySize(agentStore.size.width, agentStore.size.height);
  const position = getDisplayPosition(
    agentStore.position.x,
    agentStore.position.y,
    size
  );
  return { position, size };
});

const getSidebarWidthBounds = (
  windowWidth = displayWindowState.value.size.width
) => {
  const maxSidebarWidth = Math.max(
    MIN_SIDEBAR_WIDTH,
    windowWidth - MIN_MAIN_PANEL_WIDTH
  );
  return {
    min: Math.min(MIN_SIDEBAR_WIDTH, maxSidebarWidth),
    max: maxSidebarWidth,
  };
};

const clampSidebarWidth = (
  width: number,
  windowWidth = displayWindowState.value.size.width
) => {
  const bounds = getSidebarWidthBounds(windowWidth);
  return Math.min(bounds.max, Math.max(bounds.min, Math.round(width)));
};

const sidebarStyle = computed(() => ({
  width: `${clampSidebarWidth(agentStore.sidebarWidth)}px`,
}));

const windowStyle = computed(() => ({
  left: `${displayWindowState.value.position.x}px`,
  top: `${displayWindowState.value.position.y}px`,
  width: `${displayWindowState.value.size.width}px`,
  height: `${displayWindowState.value.size.height}px`,
}));

const showArchivedChats = ref(false);
const displayedChats = computed(() =>
  agentStore.orderedChats.filter((chat) => {
    return (
      showArchivedChats.value ||
      !chat.archived ||
      chat.id === agentStore.currentChatId
    );
  })
);
const currentChatStatusLabel = computed(() => {
  const chat = agentStore.currentChat;
  if (!chat) {
    return t("agent.chat-status-idle");
  }
  if (chat.interrupted) {
    return t("agent.chat-status-interrupted");
  }
  switch (chat.status) {
    case "running":
      return t("agent.chat-status-running");
    case "awaiting_user":
      return t("agent.chat-status-awaiting-user");
    case "error":
      return t("agent.chat-status-error");
    default:
      return t("agent.chat-status-idle");
  }
});
const currentChatStatusClass = computed(() => {
  const chat = agentStore.currentChat;
  if (!chat || chat.status === "idle") {
    return "bg-gray-100 text-gray-600";
  }
  if (chat.status === "running") {
    return "bg-blue-50 text-blue-600";
  }
  if (chat.interrupted || chat.status === "error") {
    return "bg-red-50 text-red-600";
  }
  return "bg-amber-50 text-amber-600";
});

const tokenFormatter = new Intl.NumberFormat();
const currentChatTokenUsageLabel = computed(() =>
  t("agent.chat-total-tokens", {
    count: tokenFormatter.format(agentStore.currentChat?.totalTokensUsed ?? 0),
  })
);
const isChatSwitchLocked = computed(() => agentStore.hasRunningChat);
const isChatCreationDisabled = computed(() => agentStore.hasRunningChat);

function syncSize(width: number, height: number) {
  const size = getDisplaySize(width, height);
  agentStore.size.width = size.width;
  agentStore.size.height = size.height;
}

function syncPosition() {
  const position = getDisplayPosition(
    agentStore.position.x,
    agentStore.position.y,
    getDisplaySize(agentStore.size.width, agentStore.size.height)
  );
  agentStore.position.x = position.x;
  agentStore.position.y = position.y;
}

function syncSidebarWidth(
  width = agentStore.sidebarWidth,
  windowWidth = displayWindowState.value.size.width
) {
  agentStore.sidebarWidth = clampSidebarWidth(width, windowWidth);
}

function syncStoreToDisplayState() {
  agentStore.size.width = displayWindowState.value.size.width;
  agentStore.size.height = displayWindowState.value.size.height;
  agentStore.position.x = displayWindowState.value.position.x;
  agentStore.position.y = displayWindowState.value.position.y;
}

const getCurrentPageSnapshot = () => ({
  path: route.fullPath,
  title: document.title,
});

const getEditableChatTitle = (chat: AgentChatRecord) =>
  chat.title || t("agent.chat-default-title");

const getChatLabel = (chat: AgentChatRecord) => {
  const baseLabel = getEditableChatTitle(chat);
  return chat.archived
    ? `${baseLabel} (${t("agent.chat-archived-label")})`
    : baseLabel;
};

function toggleArchivedChats() {
  showArchivedChats.value = !showArchivedChats.value;
}

const isRenamingCurrentChat = ref(false);
const renamingTitle = ref("");
const renameInputRef = ref<
  InstanceType<typeof NInput> | InstanceType<typeof NInput>[] | null
>(null);

function focusRenameInput() {
  const input = Array.isArray(renameInputRef.value)
    ? renameInputRef.value[0]
    : renameInputRef.value;
  input?.focus?.();
  input?.select?.();
}

function beginRenameCurrentChat() {
  const chat = agentStore.currentChat;
  if (!chat) {
    return;
  }
  renamingTitle.value = getEditableChatTitle(chat);
  isRenamingCurrentChat.value = true;
  nextTick(() => {
    focusRenameInput();
  });
}

function cancelRenameCurrentChat() {
  isRenamingCurrentChat.value = false;
  renamingTitle.value = "";
}

function commitRenameCurrentChat() {
  const chat = agentStore.currentChat;
  const title = renamingTitle.value.trim();
  if (!chat || !isRenamingCurrentChat.value) {
    return;
  }
  if (!title) {
    cancelRenameCurrentChat();
    return;
  }
  agentStore.renameChat(chat.id, title);
  cancelRenameCurrentChat();
}

function onRenameCurrentChatKeydown(event: KeyboardEvent) {
  if (event.isComposing) {
    return;
  }
  if (event.key === "Escape") {
    event.preventDefault();
    cancelRenameCurrentChat();
    return;
  }
  if (event.key === "Enter") {
    event.preventDefault();
    commitRenameCurrentChat();
  }
}

function toggleArchiveCurrentChat() {
  const chat = agentStore.currentChat;
  if (!chat) {
    return;
  }
  if (chat.archived) {
    agentStore.unarchiveChat(chat.id);
    return;
  }
  agentStore.archiveChat(chat.id);
}

function deleteCurrentChat() {
  const chat = agentStore.currentChat;
  if (!chat) {
    return;
  }
  agentStore.deleteChat(chat.id);
}

const isDragging = ref(false);
const dragOffset = ref({ x: 0, y: 0 });

function startDrag(event: MouseEvent) {
  if (
    event.target instanceof HTMLElement &&
    event.target.closest(
      "[data-agent-window-action], [data-agent-window-resize]"
    )
  ) {
    return;
  }

  syncStoreToDisplayState();
  isDragging.value = true;
  dragOffset.value = {
    x: event.clientX - agentStore.position.x,
    y: event.clientY - agentStore.position.y,
  };
  document.addEventListener("mousemove", onDrag);
  document.addEventListener("mouseup", stopDrag);
}

function onDrag(event: MouseEvent) {
  if (!isDragging.value) {
    return;
  }
  agentStore.position.x = event.clientX - dragOffset.value.x;
  agentStore.position.y = event.clientY - dragOffset.value.y;
  syncPosition();
}

function stopDrag() {
  isDragging.value = false;
  document.removeEventListener("mousemove", onDrag);
  document.removeEventListener("mouseup", stopDrag);
  agentStore.saveWindowState();
}

const isResizing = ref(false);
const resizeStart = ref({ x: 0, y: 0, w: 0, h: 0 });
let resizeObserver: ResizeObserver | null = null;

const isSidebarResizing = ref(false);
const sidebarResizeStart = ref({ x: 0, width: 0 });

function startSidebarResize(event: MouseEvent) {
  event.preventDefault();
  event.stopPropagation();
  syncStoreToDisplayState();
  isSidebarResizing.value = true;
  sidebarResizeStart.value = {
    x: event.clientX,
    width: clampSidebarWidth(agentStore.sidebarWidth),
  };
  document.addEventListener("mousemove", onSidebarResize);
  document.addEventListener("mouseup", stopSidebarResize);
}

function onSidebarResize(event: MouseEvent) {
  if (!isSidebarResizing.value) {
    return;
  }
  const dx = event.clientX - sidebarResizeStart.value.x;
  syncSidebarWidth(sidebarResizeStart.value.width + dx);
}

function stopSidebarResize() {
  isSidebarResizing.value = false;
  document.removeEventListener("mousemove", onSidebarResize);
  document.removeEventListener("mouseup", stopSidebarResize);
  agentStore.saveWindowState();
}

function startResize(event: MouseEvent) {
  event.preventDefault();
  event.stopPropagation();
  syncStoreToDisplayState();
  isResizing.value = true;
  resizeStart.value = {
    x: event.clientX,
    y: event.clientY,
    w: agentStore.size.width,
    h: agentStore.size.height,
  };
  document.addEventListener("mousemove", onResize);
  document.addEventListener("mouseup", stopResize);
}

function onResize(event: MouseEvent) {
  if (!isResizing.value) {
    return;
  }
  const dx = event.clientX - resizeStart.value.x;
  const dy = event.clientY - resizeStart.value.y;
  syncSize(resizeStart.value.w + dx, resizeStart.value.h + dy);
  syncPosition();
}

function stopResize() {
  isResizing.value = false;
  document.removeEventListener("mousemove", onResize);
  document.removeEventListener("mouseup", stopResize);
  agentStore.saveWindowState();
}

function observeWindowSize() {
  resizeObserver?.disconnect();
  if (!windowRef.value) {
    return;
  }

  resizeObserver = new ResizeObserver(([entry]) => {
    if (!entry || isResizing.value || isViewportResizing.value) {
      return;
    }

    const target = entry.target as HTMLElement;
    const width = clampWidth(target.offsetWidth);
    const height = clampHeight(target.offsetHeight);
    if (width === agentStore.size.width && height === agentStore.size.height) {
      return;
    }

    syncSize(width, height);
    syncPosition();
    agentStore.saveWindowState();
  });

  resizeObserver.observe(windowRef.value);
}

function handleViewportResize() {
  isViewportResizing.value = true;
  viewportSize.value = {
    width: window.innerWidth,
    height: window.innerHeight,
  };
  cancelAnimationFrame(viewportResizeFrame);
  viewportResizeFrame = window.requestAnimationFrame(() => {
    isViewportResizing.value = false;
  });
}

watch(
  () => displayWindowState.value.size.width,
  (windowWidth) => {
    const sidebarWidth = clampSidebarWidth(
      agentStore.sidebarWidth,
      windowWidth
    );
    if (sidebarWidth === agentStore.sidebarWidth) {
      return;
    }
    agentStore.sidebarWidth = sidebarWidth;
    agentStore.saveWindowState();
  }
);

watch(windowRef, () => {
  observeWindowSize();
});

function createChat() {
  if (isChatCreationDisabled.value) {
    return;
  }
  agentStore.createChat({ page: getCurrentPageSnapshot() });
}

function handleChatRowClick(chatId: string) {
  if (chatId === agentStore.currentChatId) {
    if (!isRenamingCurrentChat.value) {
      beginRenameCurrentChat();
    }
    return;
  }
  if (isRenamingCurrentChat.value) {
    cancelRenameCurrentChat();
  }
  agentStore.setCurrentChat(chatId);
}

watch(
  () => agentStore.currentChatId,
  () => {
    if (isRenamingCurrentChat.value) {
      cancelRenameCurrentChat();
    }
  }
);

onMounted(() => {
  agentStore.loadWindowState();
  handleViewportResize();
  syncSidebarWidth();
  observeWindowSize();
  window.addEventListener("resize", handleViewportResize);
});

onBeforeUnmount(() => {
  stopDrag();
  stopResize();
  stopSidebarResize();
  resizeObserver?.disconnect();
  cancelAnimationFrame(viewportResizeFrame);
  window.removeEventListener("resize", handleViewportResize);
});
</script>

<template>
  <Teleport to="body">
    <div
      v-if="agentStore.visible && agentStore.minimized"
      data-agent-window
      class="fixed bottom-4 right-4 z-[1999] flex h-10 w-10 cursor-pointer items-center justify-center rounded-full bg-blue-500 text-white shadow-lg hover:bg-blue-600"
      @click="agentStore.restore()"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        class="h-5 w-5"
        viewBox="0 0 20 20"
        fill="currentColor"
      >
        <path
          fill-rule="evenodd"
          d="M18 10c0 3.866-3.582 7-8 7a8.841 8.841 0 01-4.083-.98L2 17l1.338-3.123C2.493 12.767 2 11.434 2 10c0-3.866 3.582-7 8-7s8 3.134 8 7zM7 9H5v2h2V9zm8 0h-2v2h2V9zm-4 0H9v2h2V9z"
          clip-rule="evenodd"
        />
      </svg>
    </div>

    <div
      v-if="agentStore.visible && !agentStore.minimized"
      ref="windowRef"
      data-agent-window
      class="fixed z-[1999] flex flex-col overflow-hidden rounded-lg border border-gray-200 bg-white shadow-xl"
      :style="windowStyle"
    >
      <div
        class="flex cursor-move select-none items-center justify-between border-b bg-gray-50 px-3 py-2"
        @mousedown="startDrag"
      >
        <div class="flex min-w-0 items-center gap-x-2">
          <span class="truncate text-sm font-medium">
            {{ $t("agent.assistant-title") }}
          </span>
          <span
            class="inline-flex rounded-full px-2 py-0.5 text-xs font-medium"
            :class="currentChatStatusClass"
          >
            {{ currentChatStatusLabel }}
          </span>
          <span class="truncate text-xs text-gray-500">
            {{ currentChatTokenUsageLabel }}
          </span>
        </div>
        <div class="flex items-center gap-x-1">
          <button
            data-agent-window-action
            class="flex h-5 w-5 items-center justify-center rounded text-gray-400 hover:bg-gray-200 hover:text-gray-600"
            :title="$t('agent.minimize')"
            @click.stop="agentStore.minimize()"
          >
            &#8722;
          </button>
          <button
            data-agent-window-action
            class="flex h-5 w-5 items-center justify-center rounded text-gray-400 hover:bg-gray-200 hover:text-gray-600"
            :title="$t('agent.close')"
            @click.stop="agentStore.toggle()"
          >
            &#10005;
          </button>
        </div>
      </div>

      <div class="flex min-h-0 flex-1 overflow-hidden bg-white">
        <aside
          class="flex shrink-0 flex-col border-r border-gray-200 bg-gray-50"
          :style="sidebarStyle"
        >
          <div class="border-b border-gray-200 px-3 py-3">
            <div class="flex items-center justify-between gap-x-2">
              <div>
                <h2 class="text-xs font-semibold uppercase tracking-wide text-gray-500">
                  {{ $t("agent.chat-list-label") }}
                </h2>
                <p
                  v-if="isChatSwitchLocked"
                  class="mt-1 text-xs text-amber-600"
                  data-agent-chat-lock-message
                >
                  {{ $t("agent.chat-switch-locked") }}
                </p>
              </div>
              <button
                class="rounded-md border px-2 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-100 disabled:cursor-not-allowed disabled:bg-gray-100 disabled:text-gray-400"
                :disabled="isChatCreationDisabled"
                @click="createChat"
              >
                {{ $t("agent.new-chat") }}
              </button>
            </div>
          </div>

          <div class="min-h-0 flex-1 overflow-y-auto px-2 py-2">
            <div class="flex flex-col gap-y-1" data-agent-chat-list>
              <div
                v-for="chat in displayedChats"
                :key="chat.id"
                class="w-full rounded-md px-3 py-2 text-left text-sm transition-colors"
                :class="
                  chat.id === agentStore.currentChatId
                    ? 'bg-blue-50 text-blue-700'
                    : 'text-gray-700 hover:bg-white'
                "
                :data-agent-chat-row="chat.id"
              >
                <NInput
                  v-if="
                    chat.id === agentStore.currentChatId && isRenamingCurrentChat
                  "
                  ref="renameInputRef"
                  v-model:value="renamingTitle"
                  size="small"
                  :placeholder="$t('agent.rename-chat-placeholder')"
                  data-agent-inline-rename-input
                  @blur="commitRenameCurrentChat"
                  @keydown="onRenameCurrentChatKeydown"
                />
                <button
                  v-else
                  type="button"
                  class="w-full text-left disabled:cursor-not-allowed disabled:opacity-60"
                  :disabled="!agentStore.canSelectChat(chat.id)"
                  :aria-current="
                    chat.id === agentStore.currentChatId ? 'true' : undefined
                  "
                  @click="handleChatRowClick(chat.id)"
                >
                  <div class="truncate font-medium" data-agent-chat-title>
                    {{ getChatLabel(chat) }}
                  </div>
                  <HumanizeTs
                    :ts="Math.floor(chat.updatedTs / 1000)"
                    class="mt-1 block truncate text-xs"
                    :class="
                      chat.id === agentStore.currentChatId
                        ? 'text-blue-600/80'
                        : 'text-gray-500'
                    "
                    data-agent-chat-updated-ts
                  />
                </button>
              </div>
            </div>
          </div>

          <div class="border-t border-gray-200 px-3 py-3">
            <div class="flex flex-col gap-y-2">
              <div class="flex flex-wrap gap-x-2 gap-y-2">
                <button
                  v-if="agentStore.currentChat?.archived"
                  class="rounded-md border px-2 py-1.5 text-xs font-medium text-gray-600 hover:bg-white"
                  @click="toggleArchiveCurrentChat"
                >
                  {{ $t("agent.unarchive-chat") }}
                </button>
                <NPopconfirm
                  v-else
                  @positive-click="toggleArchiveCurrentChat"
                >
                  <template #trigger>
                    <button
                      class="rounded-md border px-2 py-1.5 text-xs font-medium text-gray-600 hover:bg-white"
                    >
                      {{ $t("agent.archive-chat") }}
                    </button>
                  </template>
                  {{ $t("agent.archive-chat-confirmation") }}
                </NPopconfirm>
                <NPopconfirm
                  @positive-click="deleteCurrentChat"
                >
                  <template #trigger>
                    <button
                      class="rounded-md border px-2 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50"
                    >
                      {{ $t("agent.delete-chat") }}
                    </button>
                  </template>
                  {{ $t("agent.delete-chat-confirmation") }}
                </NPopconfirm>
                <button
                  class="rounded-md border px-2 py-1.5 text-xs font-medium text-gray-600 hover:bg-white"
                  @click="toggleArchivedChats"
                >
                  {{
                    showArchivedChats
                      ? $t("agent.hide-archived-chats")
                      : $t("agent.show-archived-chats")
                  }}
                </button>
              </div>
            </div>
          </div>
        </aside>

        <button
          type="button"
          data-agent-window-action
          data-agent-sidebar-resize
          class="group relative w-1 shrink-0 cursor-col-resize bg-gray-100 transition-colors hover:bg-blue-100"
          @mousedown="startSidebarResize"
        >
          <span
            class="pointer-events-none absolute inset-y-0 left-1/2 w-px -translate-x-1/2 bg-transparent transition-colors group-hover:bg-blue-400"
          />
        </button>

        <div class="flex min-w-0 flex-1 flex-col">
          <AgentChat class="min-h-0 flex-1" />
          <AgentInput />
        </div>
      </div>

      <button
        type="button"
        data-agent-window-action
        data-agent-window-resize
        class="absolute bottom-0 right-0 flex h-5 w-5 cursor-se-resize items-end justify-end pb-0.5 pr-0.5 text-gray-300 hover:text-gray-400"
        :title="$t('agent.resize')"
        @mousedown="startResize"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          class="h-3 w-3"
          viewBox="0 0 12 12"
          fill="none"
          stroke="currentColor"
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="1.5"
        >
          <path d="M3.5 8.5h.01M6 6h.01M8.5 3.5h.01" />
          <path d="M3.5 11 11 3.5" />
        </svg>
      </button>
    </div>
  </Teleport>
</template>
