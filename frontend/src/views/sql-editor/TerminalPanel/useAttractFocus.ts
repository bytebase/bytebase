import { useEventListener } from "@vueuse/core";
import { unref } from "vue";
import { MaybeRef } from "@/types";
import { isDescendantOf } from "@/utils";

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

export type AttractFocusOptions = {
  excluded?: { tag: string; selector: string }[];
  targetSelector: string;
};

export const useAttractFocus = (options: MaybeRef<AttractFocusOptions>) => {
  useEventListener("keydown", (e: KeyboardEvent) => {
    const sourceElement = e.target as Element;
    const tag = sourceElement.tagName.toLowerCase();
    if (tag === "input") {
      // Don't rob the focus in other text boxes
      return;
    }
    const { excluded = [], targetSelector } = unref(options);
    for (let i = 0; i < excluded.length; i++) {
      const rule = excluded[i];
      if (tag === rule.tag && isDescendantOf(sourceElement, rule.selector)) {
        return;
      }
    }
    if (!shouldAttractFocus(e)) {
      return;
    }

    const target = document.querySelector(
      targetSelector
    ) as HTMLTextAreaElement | null;
    // monaco-editor will consume the focus event with key stroke11
    target?.focus();
  });
};
