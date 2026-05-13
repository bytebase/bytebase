import { useTranslation } from "react-i18next";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
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
      <SheetContent width="wide" className="w-[calc(100vw-8rem)] max-w-6xl p-0">
        <SheetHeader className="px-4 py-3 border-b">
          <SheetTitle>
            {t("plugin.ai.conversation.history-conversations")}
          </SheetTitle>
        </SheetHeader>
        <div className="flex h-full">
          <aside className="hidden lg:flex lg:flex-col w-[14em] border-l border-b">
            <ConversationList />
          </aside>
          <div className="flex-1 flex flex-col py-2 bg-gray-100 overflow-hidden">
            <ChatView mode="VIEW" conversation={chat.selected} />
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}
