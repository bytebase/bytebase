export const ReactShellBridgeEvent = {
  localeChange: "bb.react-locale-change",
  quickstartReset: "bb.react-quickstart-reset",
} as const;

export const VueShellBridgeEvent = {
  notification: "bb.vue-notification",
} as const;

export type ReactShellBridgeEventName =
  (typeof ReactShellBridgeEvent)[keyof typeof ReactShellBridgeEvent];

export type ReactQuickstartResetDetail = {
  keys: string[];
};

export function emitReactLocaleChange(lang: string) {
  window.dispatchEvent(
    new CustomEvent(ReactShellBridgeEvent.localeChange, { detail: lang })
  );
}

export function emitReactQuickstartReset(detail: ReactQuickstartResetDetail) {
  window.dispatchEvent(
    new CustomEvent(ReactShellBridgeEvent.quickstartReset, { detail })
  );
}
