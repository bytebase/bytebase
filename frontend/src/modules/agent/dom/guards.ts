const excludedSelector = [
  "[data-agent-sensitive]",
  "[data-agent-window]",
  '[data-bb-layer-family="agent"]',
  '[data-bb-layer-family="critical"]',
  'input[type="password" i]',
  'input[type="hidden" i]',
  "script",
  "style",
  "noscript",
].join(",");

const hiddenSelector =
  '[hidden], [aria-hidden="true"], [inert], [data-base-ui-inert]';

export function isExcluded(element: Element): boolean {
  return !!element.closest(excludedSelector);
}

export function containsExcluded(element: Element): boolean {
  return !!element.querySelector(excludedSelector);
}

export function isDisclosedChild(node: Node): boolean {
  const parent = node.parentElement;
  return (
    !parent?.matches("details:not([open])") ||
    node === parent.querySelector(":scope > summary")
  );
}

export function isPerceivable(element: Element): boolean {
  if (
    !element.isConnected ||
    element.ownerDocument !== document ||
    element.getRootNode() !== document ||
    isExcluded(element) ||
    element.closest(hiddenSelector)
  )
    return false;
  for (let node: Element | null = element; node; node = node.parentElement) {
    if (!isDisclosedChild(node)) return false;
    const { display, visibility, opacity, contentVisibility } =
      window.getComputedStyle(node);
    if (
      display === "none" ||
      visibility === "hidden" ||
      visibility === "collapse" ||
      opacity === "0" ||
      contentVisibility === "hidden"
    )
      return false;
  }
  return true;
}

export function hasDisabledState(element: Element): boolean {
  return (
    element.matches(":disabled") ||
    !!element.closest('[aria-disabled="true"], [disabled], [data-disabled]')
  );
}

export function availableForActivation(element: Element): boolean {
  return (
    isPerceivable(element) &&
    !hasDisabledState(element) &&
    !containsExcluded(element)
  );
}

export function visibleText(element: Element | null | undefined): string {
  if (!element || !isPerceivable(element)) return "";
  return Array.from(element.childNodes, (node) =>
    node.nodeType === Node.TEXT_NODE
      ? isDisclosedChild(node)
        ? (node.textContent ?? "")
        : ""
      : node instanceof Element
        ? visibleText(node)
        : ""
  ).join("");
}
