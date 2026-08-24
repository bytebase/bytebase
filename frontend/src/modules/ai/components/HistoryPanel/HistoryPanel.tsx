import { useTranslation } from "react-i18next";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { ChatView } from "../ChatView/ChatView";
import { useAIContext } from "../context";
import { ConversationList } from "./ConversationList";

/**
 * React port of `plugins/ai/components/HistoryPanel/HistoryPanel.vue`.
 *
 * Side drawer with two columns: the conversation list on the left and
 * the selected conversation rendered in `mode="VIEW"` on the right.
 * The drawer is the only place wide enough to render the full history
 * — `ChatPanel` mounts it once and toggles open via
 * `showHistoryDialog` from the React AIContext.
 */
export function HistoryPanel() {
  const { t } = useTranslation();
  const { showHistoryDialog, setShowHistoryDialog, chat } = useAIContext();

  return (
    <Sheet
      open={showHistoryDialog}
      onOpenChange={(open) => setShowHistoryDialog(open)}
    >
      <SheetContent width="xlarge">
        <SheetHeader>
          <SheetTitle>
            {t("plugin.ai.conversation.history-conversations")}
          </SheetTitle>
        </SheetHeader>
        <SheetBody className="flex-row gap-x-4 overflow-hidden">
          <aside className="hidden w-56 shrink-0 border-control-border border-r pr-4 lg:flex lg:flex-col">
            <ConversationList />
          </aside>
          <div className="flex min-w-0 flex-1 flex-col overflow-hidden rounded-sm bg-control-bg/50 py-2">
            <ChatView mode="VIEW" conversation={chat.selected} />
          </div>
        </SheetBody>
      </SheetContent>
    </Sheet>
  );
}
