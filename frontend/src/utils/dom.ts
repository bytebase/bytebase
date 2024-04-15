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
