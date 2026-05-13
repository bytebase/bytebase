import { head } from "lodash-es";
import { Loader2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { Input } from "@/react/components/ui/input";
import { useConversationStore } from "../../store";
import type { Conversation } from "../../types";

type Props = {
  readonly conversation: Conversation;
  readonly onCancel: () => void;
  readonly onUpdated: () => void;
};

/**
 * React port of `plugins/ai/components/HistoryPanel/ConversationRenameDialog.vue`.
 *
 * Single-field rename modal. The name initializes to:
 *   1. The current conversation name, or
 *   2. The first message's content (often the user's seed prompt), or
 *   3. The localized "Untitled conversation" placeholder.
 *
 * Mirrors the Vue version exactly. Auto-focuses the input on mount via
 * a ref + rAF so the input is interactive immediately when the dialog
 * portal mounts.
 */
export function ConversationRenameDialog({
  conversation,
  onCancel,
  onUpdated,
}: Props) {
  const { t } = useTranslation();
  const store = useConversationStore();

  const [name, setName] = useState(
    conversation.name ||
      head(conversation.messageList)?.content ||
      t("plugin.ai.conversation.untitled")
  );
  const [loading, setLoading] = useState(false);

  const inputRef = useRef<HTMLInputElement | null>(null);
  useEffect(() => {
    const raf = requestAnimationFrame(() => {
      inputRef.current?.focus();
    });
    return () => cancelAnimationFrame(raf);
  }, []);

  const handleRename = async () => {
    setLoading(true);
    conversation.name = name;
    await store.updateConversation(conversation);
    onUpdated();
  };

  return (
    <Dialog open onOpenChange={(open) => !open && onCancel()}>
      <DialogContent className="w-[18rem] p-6">
        <DialogTitle>{t("plugin.ai.conversation.rename")}</DialogTitle>
        <div className="relative flex flex-col items-start gap-y-2 mt-3">
          <div className="text-sm text-control">{t("common.name")}</div>
          <div className="w-full">
            <Input
              ref={inputRef}
              value={name}
              onChange={(e) => setName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  void handleRename();
                }
              }}
              className="w-full"
            />
          </div>
          <div className="w-full flex items-center justify-end gap-x-2 mt-4 pt-2 border-t">
            <Button variant="outline" onClick={onCancel}>
              {t("common.cancel")}
            </Button>
            <Button onClick={() => void handleRename()} disabled={loading}>
              {t("common.update")}
            </Button>
          </div>
          {loading && (
            <div className="absolute inset-0 bg-background/50 flex items-center justify-center rounded-sm">
              <Loader2 className="size-5 animate-spin" />
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
