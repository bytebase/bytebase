<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useAgentStore } from "../store/agent";
import AgentChat from "./AgentChat.vue";
import AgentInput from "./AgentInput.vue";

const agentStore = useAgentStore();

const MIN_WIDTH = 300;
const MIN_HEIGHT = 400;
const MAX_WIDTH = 800;
const MAX_HEIGHT = 800;

const windowStyle = computed(() => ({
  left: `${agentStore.position.x}px`,
  top: `${agentStore.position.y}px`,
  width: `${agentStore.size.width}px`,
  height: `${agentStore.size.height}px`,
}));

// Drag logic
const isDragging = ref(false);
const dragOffset = ref({ x: 0, y: 0 });

function startDrag(e: MouseEvent) {
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

function startResize(e: MouseEvent) {
  e.preventDefault();
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
  agentStore.size.width = Math.min(
    MAX_WIDTH,
    Math.max(MIN_WIDTH, resizeStart.value.w + dx)
  );
  agentStore.size.height = Math.min(
    MAX_HEIGHT,
    Math.max(MIN_HEIGHT, resizeStart.value.h + dy)
  );
}

function stopResize() {
  isResizing.value = false;
  document.removeEventListener("mousemove", onResize);
  document.removeEventListener("mouseup", stopResize);
  agentStore.saveWindowState();
}

onMounted(() => {
  agentStore.loadWindowState();
});
</script>

<template>
  <Teleport to="body">
    <!-- Minimized button -->
    <div
      v-if="agentStore.visible && agentStore.minimized"
      data-agent-window
      class="fixed z-[1999] bottom-4 right-4 flex h-10 w-10 cursor-pointer items-center justify-center rounded-full bg-blue-500 text-white shadow-lg hover:bg-blue-600"
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
      data-agent-window
      class="fixed z-[1999] flex flex-col rounded-lg border border-gray-200 shadow-xl bg-white overflow-hidden"
      :style="windowStyle"
    >
      <!-- Title bar -->
      <div
        class="flex items-center justify-between px-3 py-2 border-b bg-gray-50 cursor-move select-none"
        @mousedown="startDrag"
      >
        <span class="text-sm font-medium">{{ $t("agent.assistant-title") }}</span>
        <div class="flex items-center gap-x-1">
          <button
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
            class="flex h-5 w-5 items-center justify-center rounded text-gray-400 hover:bg-gray-200 hover:text-gray-600"
            :title="$t('agent.minimize')"
            @click.stop="agentStore.minimize()"
          >
            &#8722;
          </button>
          <button
            class="flex h-5 w-5 items-center justify-center rounded text-gray-400 hover:bg-gray-200 hover:text-gray-600"
            :title="$t('agent.close')"
            @click.stop="agentStore.toggle()"
          >
            &#10005;
          </button>
        </div>
      </div>

      <!-- Chat -->
      <AgentChat class="flex-1 min-h-0" />

      <!-- Input -->
      <AgentInput />

      <!-- Resize handle -->
      <div
        class="absolute bottom-0 right-0 w-3 h-3 cursor-se-resize"
        @mousedown="startResize"
      />
    </div>
  </Teleport>
</template>
