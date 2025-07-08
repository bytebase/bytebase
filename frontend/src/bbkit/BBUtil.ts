export function hashCode(s: string): number {
  let hash = 0;
  for (let i = 0; i < s.length; i++) {
    hash = (hash << 5) - hash + s.charCodeAt(i);
    hash |= 0; // Convert to 32bit integer
  }
  return hash;
}

export function isAncestorOf(
  maybeAncestor: HTMLElement,
  maybeDescendant: HTMLElement
): boolean {
  let element: HTMLElement | null = maybeDescendant;
  while (element) {
    if (element === maybeAncestor) return true;
    element = element.parentElement;
  }
  return false;
}
