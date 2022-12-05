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
