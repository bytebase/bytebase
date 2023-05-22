import { t } from "@/plugins/i18n";
import { toClipboard } from "@soerenmartius/vue3-clipboard";

import { User } from "@/types/proto/v1/auth_service";
import { pushNotification } from "@/store";

export const copyServiceKeyToClipboardIfNeeded = (user: User) => {
  if (!user.serviceKey) return;

  toClipboard(user.serviceKey);

  pushNotification({
    module: "bytebase",
    style: "INFO",
    title: t("settings.members.service-key-copied"),
  });
};
