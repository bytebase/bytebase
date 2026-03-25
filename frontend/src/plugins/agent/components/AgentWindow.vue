<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import type { AgentThread } from "../logic/types";
import { useAgentStore } from "../store/agent";
import AgentChat from "./AgentChat.vue";
import AgentInput from "./AgentInput.vue";

const { t } = useI18n();
const route = useRoute();
const agentStore = useAgentStore();

const MIN_WIDTH = 300;
const MIN_HEIGHT = 400;
const WINDOW_MARGIN = 16;

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

const windowStyle = computed(() => ({
  left: `${displayWindowState.value.position.x}px`,
  top: `${displayWindowState.value.position.y}px`,
  width: `${displayWindowState.value.size.width}px`,
  height: `${displayWindowState.value.size.height}px`,
}));

const showArchivedThreads = ref(false);
const displayedThreads = computed(() =>
  agentStore.orderedThreads.filter((thread) => {
    return (
      showArchivedThreads.value ||
      !thread.archived ||
      thread.id === agentStore.currentThreadId
    );
  })
);
const currentThreadStatusLabel = computed(() => {
  const thread = agentStore.currentThread;
  if (!thread) {
    return t("agent.thread-status-idle");
  }
  if (thread.interrupted) {
    return t("agent.thread-status-interrupted");
  }
  switch (thread.status) {
    case "running":
      return t("agent.thread-status-running");
    case "awaiting_user":
      return t("agent.thread-status-awaiting-user");
    case "error":
      return t("agent.thread-status-error");
    default:
      return t("agent.thread-status-idle");
  }
});
const currentThreadStatusClass = computed(() => {
  const thread = agentStore.currentThread;
  if (!thread || thread.status === "idle") {
    return "bg-gray-100 text-gray-600";
  }
  if (thread.status === "running") {
    return "bg-blue-50 text-blue-600";
  }
  if (thread.interrupted || thread.status === "error") {
    return "bg-red-50 text-red-600";
  }
  return "bg-amber-50 text-amber-600";
});

const tokenFormatter = new Intl.NumberFormat();
const currentThreadTokenUsageLabel = computed(() =>
  t("agent.thread-total-tokens", {
    count: tokenFormatter.format(
      agentStore.currentThread?.totalTokensUsed ?? 0
    ),
  })
);

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

const getThreadLabel = (thread: AgentThread) => {
  const baseLabel = thread.title
    ? thread.title
    : `${t("agent.thread-default-title")} · ${new Date(
        thread.createdTs
      ).toLocaleString()}`;
  return thread.archived
    ? `${baseLabel} (${t("agent.thread-archived-label")})`
    : baseLabel;
};

function toggleArchivedThreads() {
  showArchivedThreads.value = !showArchivedThreads.value;
}

function renameCurrentThread() {
  const thread = agentStore.currentThread;
  if (!thread) {
    return;
  }
  const nextTitle = window.prompt(
    t("agent.rename-thread-prompt"),
    thread.title || getThreadLabel(thread)
  );
  if (nextTitle === null) {
    return;
  }
  agentStore.renameThread(thread.id, nextTitle);
}

function toggleArchiveCurrentThread() {
  const thread = agentStore.currentThread;
  if (!thread) {
    return;
  }
  if (thread.archived) {
    agentStore.unarchiveThread(thread.id);
    return;
  }
  if (!window.confirm(t("agent.archive-thread-confirmation"))) {
    return;
  }
  agentStore.archiveThread(thread.id);
}

function deleteCurrentThread() {
  const thread = agentStore.currentThread;
  if (!thread || !window.confirm(t("agent.delete-thread-confirmation"))) {
    return;
  }
  agentStore.deleteThread(thread.id);
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

watch(windowRef, () => {
  observeWindowSize();
});

function createThread() {
  agentStore.createThread({ page: getCurrentPageSnapshot() });
}

function resetCurrentThread() {
  agentStore.clearConversation();
}

function selectThread(event: Event) {
  const target = event.target as HTMLSelectElement;
  agentStore.setCurrentThread(target.value);
}

onMounted(() => {
  agentStore.loadWindowState();
  handleViewportResize();
  observeWindowSize();
  window.addEventListener("resize", handleViewportResize);
});

onBeforeUnmount(() => {
  stopDrag();
  stopResize();
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
            :class="currentThreadStatusClass"
          >
            {{ currentThreadStatusLabel }}
          </span>
          <span class="truncate text-xs text-gray-500">
            {{ currentThreadTokenUsageLabel }}
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

      <div class="border-b bg-white px-3 py-2">
        <label class="mb-1 block text-xs font-medium text-gray-500">
          {{ $t("agent.thread-select-label") }}
        </label>
        <div class="flex items-center gap-x-2">
          <select
            class="min-w-0 flex-1 rounded-md border px-2 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:bg-gray-50"
            :value="agentStore.currentThreadId ?? ''"
            @change="selectThread"
          >
            <option
              v-for="thread in displayedThreads"
              :key="thread.id"
              :value="thread.id"
            >
              {{ getThreadLabel(thread) }}
            </option>
          </select>
          <button
            class="rounded-md border px-2 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-50"
            @click="createThread"
          >
            {{ $t("agent.new-thread") }}
          </button>
          <button
            class="rounded-md border px-2 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-50"
            @click="resetCurrentThread"
          >
            {{ $t("agent.reset-thread") }}
          </button>
        </div>
        <div class="mt-2 flex flex-wrap items-center gap-x-2 gap-y-2">
          <button
            class="rounded-md border px-2 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-50"
            @click="renameCurrentThread"
          >
            {{ $t("agent.rename-thread") }}
          </button>
          <button
            class="rounded-md border px-2 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-50"
            @click="toggleArchiveCurrentThread"
          >
            {{
              agentStore.currentThread?.archived
                ? $t("agent.unarchive-thread")
                : $t("agent.archive-thread")
            }}
          </button>
          <button
            class="rounded-md border px-2 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50"
            @click="deleteCurrentThread"
          >
            {{ $t("agent.delete-thread") }}
          </button>
          <button
            class="rounded-md border px-2 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-50"
            @click="toggleArchivedThreads"
          >
            {{
              showArchivedThreads
                ? $t("agent.hide-archived-threads")
                : $t("agent.show-archived-threads")
            }}
          </button>
        </div>
      </div>

      <AgentChat class="min-h-0 flex-1" />
      <AgentInput />

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
