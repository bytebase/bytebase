import { defineStore } from "pinia";
import { ref, watch } from "vue";
import type { Message } from "../logic/types";

const MESSAGES_KEY = "bb-agent-messages";

export const useAgentStore = defineStore("agent", () => {
  // Window state
  const visible = ref(false);
  const position = ref({
    x: window.innerWidth - 420,
    y: window.innerHeight - 520,
  });
  const size = ref({ width: 400, height: 500 });
  const minimized = ref(false);

  // Conversation
  const messages = ref<Message[]>([]);
  const loading = ref(false);
  const abortController = ref<AbortController | null>(null);

  function toggle() {
    visible.value = !visible.value;
    if (visible.value) minimized.value = false;
  }

  function minimize() {
    minimized.value = true;
  }

  function restore() {
    minimized.value = false;
  }

  function addMessage(message: Message) {
    messages.value.push(message);
  }

  function clearMessages() {
    messages.value = [];
    localStorage.removeItem(MESSAGES_KEY);
  }

  function clearConversation() {
    clearMessages();
    cancel();
  }

  function cancel() {
    abortController.value?.abort();
    abortController.value = null;
    loading.value = false;
  }

  function saveWindowState() {
    localStorage.setItem(
      "bb-agent-window",
      JSON.stringify({ position: position.value, size: size.value })
    );
  }

  function saveMessages() {
    localStorage.setItem(MESSAGES_KEY, JSON.stringify(messages.value));
  }

  function loadMessages() {
    const saved = localStorage.getItem(MESSAGES_KEY);
    if (saved) {
      try {
        const parsed = JSON.parse(saved);
        if (Array.isArray(parsed)) {
          messages.value = parsed;
        }
      } catch {
        // Ignore corrupted data
        localStorage.removeItem(MESSAGES_KEY);
      }
    }
  }

  function loadWindowState() {
    const saved = localStorage.getItem("bb-agent-window");
    if (saved) {
      const state = JSON.parse(saved);
      if (state.position) position.value = state.position;
      if (state.size) size.value = state.size;
    }
  }

  // Persist messages on every change
  watch(messages, saveMessages, { deep: true });

  // Load persisted messages on init
  loadMessages();

  return {
    visible,
    position,
    size,
    minimized,
    messages,
    loading,
    abortController,
    toggle,
    minimize,
    restore,
    addMessage,
    clearMessages,
    clearConversation,
    cancel,
    saveWindowState,
    loadWindowState,
  };
});
