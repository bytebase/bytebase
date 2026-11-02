import { type RefObject, useCallback } from "react";
import { flushSync } from "react-dom";

const scrollContainerOf = (element: HTMLElement): HTMLElement => {
  for (
    let node = element.parentElement;
    node && node !== document.body;
    node = node.parentElement
  ) {
    const { overflowY } = getComputedStyle(node);
    if (
      (overflowY === "auto" || overflowY === "scroll") &&
      node.scrollHeight > node.clientHeight
    ) {
      return node;
    }
  }
  return (document.scrollingElement as HTMLElement | null) ?? document.body;
};

/**
 * Returns a function that applies a change which shortens the list in `ref`
 * and then scrolls so the list's bottom edge stays where it was on screen.
 * The control at the end of the list stays under the pointer, so undoing the
 * change is one click away.
 */
export function useKeepEndInPlace(
  ref: RefObject<HTMLElement | null>
): (change: () => void) => void {
  return useCallback(
    (change: () => void) => {
      const element = ref.current;
      if (!element) {
        change();
        return;
      }
      const before = element.getBoundingClientRect().bottom;
      flushSync(change);
      const shift = element.getBoundingClientRect().bottom - before;
      if (shift !== 0) scrollContainerOf(element).scrollTop += shift;
    },
    [ref]
  );
}
