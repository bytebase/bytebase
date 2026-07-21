/**
 * Fire a React-compatible change event on an input or textarea element.
 *
 * React 18 tracks input value mutations via the native prototype descriptor
 * setter rather than by diffing the DOM value attribute. Dispatching a plain
 * `input`/`change` Event without first setting the value through the descriptor
 * means React never sees the new value and the `onChange` handler never fires.
 *
 * This helper applies the descriptor setter trick so that React's synthetic
 * event system correctly receives the new value — this is NOT optional in
 * React 18 when driving inputs via raw DOM events without @testing-library/react.
 */
export function nativeChange(
  el: HTMLInputElement | HTMLTextAreaElement,
  value: string
): void {
  const proto =
    el instanceof HTMLTextAreaElement
      ? HTMLTextAreaElement.prototype
      : HTMLInputElement.prototype;
  const descriptor = Object.getOwnPropertyDescriptor(proto, "value");
  descriptor?.set?.call(el, value);
  el.dispatchEvent(new Event("input", { bubbles: true }));
  el.dispatchEvent(new Event("change", { bubbles: true }));
}
