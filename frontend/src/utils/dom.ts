import { onBeforeUnmount, unref, watch, type Ref } from "vue";

export const isDescendantOf = (
  node: Element | null | undefined,
  selector: string
) => {
  while (node) {
    if (node.matches(selector)) {
      return true;
    }
    node = node.parentElement;
  }
  return false;
};

export const findAncestor = (
  node: HTMLElement | SVGElement | null | undefined,
  selector: string
) => {
  while (node) {
    if (node.matches(selector)) {
      return node;
    }
    node = node.parentElement;
  }
  return undefined;
};

export const getScrollParent = (
  element: HTMLElement | null | undefined
): HTMLElement => {
  if (!element) {
    return document.body;
  }

  if (element.scrollHeight > element.clientHeight) {
    return element;
  } else {
    return getScrollParent(element.parentElement);
  }
};

export const nextAnimationFrame = () => {
  return new Promise<number>((resolve) => requestAnimationFrame(resolve));
};

export const usePreventBackAndForward = (
  elemRef: Ref<HTMLElement | null | undefined> | HTMLElement = document.body
) => {
  const preventBackForward = (elem: HTMLElement) => {
    function handleWheel(e: WheelEvent) {
      const maxX = elem.scrollWidth - elem.clientWidth;
      const scrollTarget = elem.scrollLeft + e.deltaX;
      if (scrollTarget < 0 || scrollTarget > maxX) {
        e.preventDefault();
      }
    }

    elem.addEventListener("wheel", handleWheel, {
      passive: false,
    });

    return () => {
      elem.removeEventListener("wheel", handleWheel);
    };
  };

  let unregister: () => void;
  watch(
    () => unref(elemRef),
    (elem) => {
      if (!elem) return;
      if (unregister) {
        unregister();
      }
      unregister = preventBackForward(elem);
    },
    {
      immediate: true,
    }
  );

  onBeforeUnmount(() => {
    if (unregister) {
      unregister();
    }
  });
};
