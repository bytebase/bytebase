import { isDescendantOf } from "@/utils";
import { useEventListener } from "@vueuse/core";

const FOCUS_ATTRACTIVE_KEYS: (RegExp | string)[] = [
  /^[A-Za-z0-9\s.`~!@#$%^&*()-=[\]\\;',./_+{}|:"<>?]$/,
  "ArrowUp",
  "ArrowDown",
  "ArrowLeft",
  "ArrowRight",
  "Backspace",
  "Delete",
  "Home",
  "End",
];

const shouldAttractFocus = (e: KeyboardEvent) => {
  if (e.metaKey || e.ctrlKey) {
    // A downgrade for safety since key strokes with cmd or ctrl might
    // probably be a system-wide keyboard shortcut.
    return false;
  }
  return FOCUS_ATTRACTIVE_KEYS.some((rule) => {
    if (typeof rule === "string") {
      return e.key === rule;
    }
    return e.key.match(rule);
  });
};

export const useAttractFocus = () => {
  useEventListener("keydown", (e: KeyboardEvent) => {
    const sourceElement = e.target as Element;
    const tag = sourceElement.tagName.toLowerCase();
    if (tag === "input") {
      // Don't rob the focus in other text boxes
      return;
    }
    if (tag === "textarea") {
      if (isDescendantOf(sourceElement, ".active-editor")) {
        // The active editor is already focused
        return;
      }
    }
    if (!shouldAttractFocus(e)) {
      return;
    }

    const target = document.querySelector(
      ".active-editor textarea"
    ) as HTMLTextAreaElement | null;
    // monaco-editor will consume the focus event with key stroke
    target?.focus();
  });
};
