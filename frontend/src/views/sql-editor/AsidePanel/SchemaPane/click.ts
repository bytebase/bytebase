import Emittery from "emittery";
import { ref } from "vue";
import { type TreeNode } from "./tree";

export const useClickEvents = () => {
  const DELAY = 250;
  const state = ref<{
    timeout: ReturnType<typeof setTimeout>;
    node: TreeNode;
  }>();
  const events = new Emittery<{
    "single-click": { node: TreeNode };
    "double-click": { node: TreeNode };
  }>();

  const clear = () => {
    if (!state.value) return;
    clearTimeout(state.value.timeout);
    state.value = undefined;
  };
  const queue = (node: TreeNode) => {
    state.value = {
      timeout: setTimeout(() => {
        events.emit("single-click", { node });
        clear();
      }, DELAY),
      node,
    };
  };

  const handleClick = (node: TreeNode) => {
    if (state.value && state.value.node.key === node.key) {
      events.emit("double-click", { node });
      clear();
      return;
    }
    clear();
    queue(node);
  };

  return { events, handleClick };
};
