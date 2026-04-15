const DROPDOWN_OFFSET = 4;

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

export { DROPDOWN_OFFSET, getPortalDropdownStyle };
