import { ClockIcon, PlusIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useAIContext } from "./context";

/**
 * React port of `plugins/ai/components/ActionBar.vue`.
 * Top bar above the chat: "AI Assistant" heading on the left,
 * History + New-conversation buttons on the right.
 */
export function ActionBar() {
  const { t } = useTranslation();
  const { events, setShowHistoryDialog } = useAIContext();

  return (
    <div className="w-full flex flex-wrap gap-y-1 justify-between sm:items-center px-1 py-1 border-b bg-background">
      <div className="action-left gap-x-2 flex overflow-x-auto sm:overflow-x-hidden items-center pl-1">
        <h3 className="text-sm font-medium">{t("plugin.ai.ai-assistant")}</h3>
      </div>
      <div className="action-right gap-x-1 flex overflow-x-auto sm:overflow-x-hidden sm:justify-end items-center">
        <Tooltip
          content={t("plugin.ai.conversation.view-history-conversations")}
          side="bottom"
        >
          <Button
            variant="ghost"
            size="sm"
            className="h-7 px-1.5"
            onClick={() => setShowHistoryDialog(true)}
            aria-label={t("plugin.ai.conversation.view-history-conversations")}
          >
            <ClockIcon className="size-4" />
          </Button>
        </Tooltip>
        <Tooltip
          content={t("plugin.ai.conversation.new-conversation")}
          side="bottom"
        >
          <Button
            variant="ghost"
            size="sm"
            className="h-7 px-1.5"
            onClick={() => events.emit("new-conversation", { input: "" })}
            aria-label={t("plugin.ai.conversation.new-conversation")}
          >
            <PlusIcon className="size-4" />
          </Button>
        </Tooltip>
      </div>
    </div>
  );
}
