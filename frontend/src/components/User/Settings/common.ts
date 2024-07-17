import { t } from "@/plugins/i18n";
import { pushNotification } from "@/store";
import type { ComposedUser } from "@/types";
import { toClipboard } from "@/utils";

export const copyServiceKeyToClipboardIfNeeded = (user: ComposedUser) => {
  if (!user.serviceKey) return;

  toClipboard(user.serviceKey);

  pushNotification({
    module: "bytebase",
    style: "INFO",
    title: t("settings.members.service-key-copied"),
  });
};
