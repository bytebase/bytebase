import { Inbox } from "lucide-react";
import { useTranslation } from "react-i18next";

/**
 * Empty-state placeholder shown when a `mode="VIEW"` conversation has
 * no messages.
 */
export function EmptyView() {
  const { t } = useTranslation();
  return (
    <div className="w-full h-full flex flex-col justify-center items-center">
      <div className="w-[20rem] mx-auto flex flex-col items-center gap-y-2 py-6">
        <Inbox className="size-12 text-control-placeholder" />
        <p className="text-center text-sm text-control-light">
          {t("plugin.ai.conversation.no-message")}
        </p>
      </div>
    </div>
  );
}
