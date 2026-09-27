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
  viewportHeight: number,
  options?: {
    minWidth?: number;
    viewportWidth?: number;
  }
): React.CSSProperties {
  const availableBelow = viewportHeight - triggerRect.bottom - DROPDOWN_OFFSET;
  const availableAbove = triggerRect.top - DROPDOWN_OFFSET;
  const shouldOpenUpward =
    availableBelow < dropdownHeight && availableAbove > availableBelow;
  const maxWidth =
    options?.viewportWidth === undefined
      ? undefined
      : Math.max(0, options.viewportWidth - DROPDOWN_OFFSET * 2);
  const width = Math.min(
    Math.max(triggerRect.width, options?.minWidth ?? 0),
    maxWidth ?? Number.POSITIVE_INFINITY
  );
  const left =
    options?.viewportWidth === undefined
      ? triggerRect.left
      : Math.max(
          DROPDOWN_OFFSET,
          Math.min(
            triggerRect.left,
            options.viewportWidth - width - DROPDOWN_OFFSET
          )
        );

  return {
    position: "fixed",
    left,
    width,
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
