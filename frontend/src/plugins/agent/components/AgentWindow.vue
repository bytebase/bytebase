<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useAgentStore } from "../store/agent";
import AgentChat from "./AgentChat.vue";
import AgentInput from "./AgentInput.vue";

const agentStore = useAgentStore();

const MIN_WIDTH = 300;
const MIN_HEIGHT = 400;
const WINDOW_MARGIN = 16;

const windowRef = ref<HTMLElement | null>(null);

const windowStyle = computed(() => ({
  left: `${agentStore.position.x}px`,
  top: `${agentStore.position.y}px`,
  width: `${agentStore.size.width}px`,
  height: `${agentStore.size.height}px`,
}));

const maxWidth = () =>
  Math.max(MIN_WIDTH, window.innerWidth - WINDOW_MARGIN * 2);
const maxHeight = () =>
  Math.max(MIN_HEIGHT, window.innerHeight - WINDOW_MARGIN * 2);

const clampWidth = (width: number) =>
  Math.min(maxWidth(), Math.max(MIN_WIDTH, Math.round(width)));
const clampHeight = (height: number) =>
  Math.min(maxHeight(), Math.max(MIN_HEIGHT, Math.round(height)));

function syncSize(width: number, height: number) {
  agentStore.size.width = clampWidth(width);
  agentStore.size.height = clampHeight(height);
}

function syncPosition() {
  const maxX = Math.max(
    WINDOW_MARGIN,
    window.innerWidth - agentStore.size.width - WINDOW_MARGIN
  );
  const maxY = Math.max(
    WINDOW_MARGIN,
    window.innerHeight - agentStore.size.height - WINDOW_MARGIN
  );
  agentStore.position.x = Math.min(
    maxX,
    Math.max(WINDOW_MARGIN, Math.round(agentStore.position.x))
  );
  agentStore.position.y = Math.min(
    maxY,
    Math.max(WINDOW_MARGIN, Math.round(agentStore.position.y))
  );
}

function syncWindowState() {
  syncSize(agentStore.size.width, agentStore.size.height);
  syncPosition();
}

// Drag logic
const isDragging = ref(false);
const dragOffset = ref({ x: 0, y: 0 });

function startDrag(e: MouseEvent) {
  if (
    e.target instanceof HTMLElement &&
    e.target.closest("[data-agent-window-action], [data-agent-window-resize]")
  ) {
    return;
  }

  isDragging.value = true;
  dragOffset.value = {
    x: e.clientX - agentStore.position.x,
    y: e.clientY - agentStore.position.y,
  };
  document.addEventListener("mousemove", onDrag);
  document.addEventListener("mouseup", stopDrag);
}

function onDrag(e: MouseEvent) {
  if (!isDragging.value) return;
  agentStore.position.x = e.clientX - dragOffset.value.x;
  agentStore.position.y = e.clientY - dragOffset.value.y;
  syncPosition();
}

function stopDrag() {
  isDragging.value = false;
  document.removeEventListener("mousemove", onDrag);
  document.removeEventListener("mouseup", stopDrag);
  agentStore.saveWindowState();
}

// Resize logic
const isResizing = ref(false);
const resizeStart = ref({ x: 0, y: 0, w: 0, h: 0 });
let resizeObserver: ResizeObserver | null = null;

function startResize(e: MouseEvent) {
  e.preventDefault();
  e.stopPropagation();
  isResizing.value = true;
  resizeStart.value = {
    x: e.clientX,
    y: e.clientY,
    w: agentStore.size.width,
    h: agentStore.size.height,
  };
  document.addEventListener("mousemove", onResize);
  document.addEventListener("mouseup", stopResize);
}

function onResize(e: MouseEvent) {
  if (!isResizing.value) return;
  const dx = e.clientX - resizeStart.value.x;
  const dy = e.clientY - resizeStart.value.y;
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
  if (!windowRef.value) return;

  resizeObserver = new ResizeObserver(([entry]) => {
    if (!entry || isResizing.value) return;

    // `contentRect` excludes borders, but the inline width/height we persist are
    // border-box dimensions. Reading the content box here creates a feedback loop
    // where every observer callback writes a slightly smaller size back.
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
  syncWindowState();
  agentStore.saveWindowState();
}

watch(windowRef, () => {
  observeWindowSize();
});

onMounted(() => {
  agentStore.loadWindowState();
  syncWindowState();
  observeWindowSize();
  window.addEventListener("resize", handleViewportResize);
});

onBeforeUnmount(() => {
  stopDrag();
  stopResize();
  resizeObserver?.disconnect();
  window.removeEventListener("resize", handleViewportResize);
});
</script>

<template>
  <Teleport to="body">
    <!-- Minimized button -->
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

    <!-- Full window -->
    <div
      v-if="agentStore.visible && !agentStore.minimized"
      ref="windowRef"
      data-agent-window
      class="fixed z-[1999] flex resize flex-col overflow-hidden rounded-lg border border-gray-200 bg-white shadow-xl"
      :style="windowStyle"
    >
      <!-- Title bar -->
      <div
        class="cursor-move select-none border-b bg-gray-50 px-3 py-2"
        @mousedown="startDrag"
      >
        <div class="flex items-center justify-between gap-x-2">
          <span class="text-sm font-medium">{{ $t("agent.assistant-title") }}</span>
          <div class="flex items-center gap-x-1">
            <button
              data-agent-window-action
              class="flex h-5 w-5 items-center justify-center rounded text-gray-400 hover:bg-gray-200 hover:text-gray-600"
              :title="$t('agent.new-chat')"
              @click.stop="agentStore.clearMessages()"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                class="h-3.5 w-3.5"
                viewBox="0 0 20 20"
                fill="currentColor"
              >
                <path
                  fill-rule="evenodd"
                  d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z"
                  clip-rule="evenodd"
                />
              </svg>
            </button>
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
      </div>

      <!-- Chat -->
      <AgentChat class="min-h-0 flex-1" />

      <!-- Input -->
      <AgentInput />

      <!-- Resize handle -->
      <button
        type="button"
        data-agent-window-resize
        class="absolute bottom-0 right-0 flex h-5 w-5 cursor-se-resize items-end justify-end pr-0.5 pb-0.5 text-gray-300 hover:text-gray-400"
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
