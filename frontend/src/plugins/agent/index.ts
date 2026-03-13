import { onMounted, onUnmounted } from "vue";
import { useAgentStore } from "./store/agent";

export { default as AgentWindow } from "./components/AgentWindow.vue";
export { useAgentStore } from "./store/agent";

export function useAgentShortcut() {
  const store = useAgentStore();

  function handleKeydown(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === "A") {
      e.preventDefault();
      store.toggle();
    }
  }

  onMounted(() => {
    window.addEventListener("keydown", handleKeydown);
  });

  onUnmounted(() => {
    window.removeEventListener("keydown", handleKeydown);
  });
}
