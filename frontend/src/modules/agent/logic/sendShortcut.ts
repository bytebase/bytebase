export function isAgentSendShortcut(
  event: KeyboardEvent,
  sendWithEnter: boolean
): boolean {
  if (
    event.key !== "Enter" ||
    event.isComposing ||
    event.keyCode === 229 ||
    event.shiftKey ||
    event.altKey
  ) {
    return false;
  }
  return sendWithEnter
    ? !event.ctrlKey && !event.metaKey
    : event.ctrlKey || event.metaKey;
}
