import type { NotificationCreate } from "@/types/notification";

export const ReactShellBridgeEvent = {
  localeChange: "bb.react-locale-change",
  notification: "bb.react-notification",
  quickstartReset: "bb.react-quickstart-reset",
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

export function emitReactNotification(notification: NotificationCreate) {
  window.dispatchEvent(
    new CustomEvent(ReactShellBridgeEvent.notification, {
      detail: notification,
    })
  );
}

export function emitReactQuickstartReset(detail: ReactQuickstartResetDetail) {
  window.dispatchEvent(
    new CustomEvent(ReactShellBridgeEvent.quickstartReset, { detail })
  );
}
