const DROPDOWN_OFFSET = 4;

function shouldIgnorePortalDropdownScroll(
  target: EventTarget | null,
  dropdownElement: HTMLElement | null
): boolean {
  return target instanceof Node && dropdownElement?.contains(target) === true;
}

function getPortalDropdownStyle(
  triggerRect: Pick<DOMRect, "top" | "left" | "width" | "bottom">,
  dropdownHeight: number,
  viewportHeight: number
): React.CSSProperties {
  const availableBelow = viewportHeight - triggerRect.bottom - DROPDOWN_OFFSET;
  const availableAbove = triggerRect.top - DROPDOWN_OFFSET;
  const shouldOpenUpward =
    availableBelow < dropdownHeight && availableAbove > availableBelow;

  return {
    position: "fixed",
    left: triggerRect.left,
    width: triggerRect.width,
    ...(shouldOpenUpward
      ? { bottom: viewportHeight - triggerRect.top + DROPDOWN_OFFSET }
      : { top: triggerRect.bottom + DROPDOWN_OFFSET }),
  };
}

function isPortalDropdownStyleEqual(
  previous: React.CSSProperties,
  next: React.CSSProperties
): boolean {
  return (
    previous.position === next.position &&
    previous.left === next.left &&
    previous.width === next.width &&
    previous.top === next.top &&
    previous.bottom === next.bottom
  );
}

export {
  DROPDOWN_OFFSET,
  getPortalDropdownStyle,
  isPortalDropdownStyleEqual,
  shouldIgnorePortalDropdownScroll,
};
