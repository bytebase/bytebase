import { Inbox } from "lucide-react";
import { useTranslation } from "react-i18next";

/**
 * React port of `plugins/ai/components/ChatView/EmptyView.vue`.
 * Empty-state placeholder shown when a `mode="VIEW"` conversation has
 * no messages. Replaces naive-ui's `NEmpty` with a simple inline icon +
 * caption — the only place in the AI plugin that used `NEmpty` so
 * keeping naive-ui just for this would be wasteful.
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
