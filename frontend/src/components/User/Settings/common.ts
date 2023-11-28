import { t } from "@/plugins/i18n";
import { pushNotification } from "@/store";
import { User } from "@/types/proto/v1/auth_service";
import { toClipboard } from "@/utils";

export const copyServiceKeyToClipboardIfNeeded = (user: User) => {
  if (!user.serviceKey) return;

  toClipboard(user.serviceKey);

  pushNotification({
    module: "bytebase",
    style: "INFO",
    title: t("settings.members.service-key-copied"),
  });
};
