import { head } from "lodash-es";
import { Loader2, PencilIcon, PlusIcon, TrashIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import scrollIntoView from "scroll-into-view-if-needed";
import {
  AlertDialog,
  AlertDialogClose,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/react/components/ui/alert-dialog";
import { Button } from "@/react/components/ui/button";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useSQLEditorTabStore } from "@/store";
import { useConversationStore } from "../../store";
import type { Conversation } from "../../types";
import { useAIContext } from "../context";
import { ConversationRenameDialog } from "./ConversationRenameDialog";

/**
 * React port of `plugins/ai/components/HistoryPanel/ConversationList.vue`.
 *
 * Per-tab list of historical conversations with select / rename / delete
 * actions and a sticky "New conversation" footer.
 *
 * Behavioural pitfalls preserved from the Vue source:
 *   - On tab switch (`(instance, database)` change) the rename dialog is
 *     dismissed — a half-completed rename for a previous tab's
 *     conversation shouldn't survive a context change.
 *   - When the selected conversation changes the row scrolls into view
 *     (`scrollIntoView({ scrollMode: "if-needed" })`) so the user can
 *     spot the active conversation in a long list. rAF defers to the
 *     next paint so the freshly-inserted node is measurable.
 *   - On delete, the next-selected conversation matches the Vue
 *     `list[index]` heuristic (try to keep the cursor near where it
 *     was). Falls back to undefined when the list empties.
 */
export function ConversationList() {
  const { t } = useTranslation();
  const tabStore = useSQLEditorTabStore();
  const store = useConversationStore();
  const { events, chat } = useAIContext();
  const { list, ready, selected, setSelected } = chat;

  const [rename, setRename] = useState<Conversation | undefined>(undefined);
  const [deleteCandidate, setDeleteCandidate] = useState<
    Conversation | undefined
  >(undefined);

  // Dismiss the rename dialog whenever the active tab's connection
  // changes — a stale dialog tied to a different tab's conversation
  // would mutate the wrong record on Save.
  const connectionKey = useVueState(() => {
    const tab = tabStore.currentTab;
    if (!tab) return "";
    return `${tab.connection.instance}|${tab.connection.database}`;
  });
  useEffect(() => {
    setRename(undefined);
    setDeleteCandidate(undefined);
  }, [connectionKey]);

  // Scroll the selected row into view when it changes.
  useEffect(() => {
    if (!selected?.id || list.length === 0) return;
    const raf = requestAnimationFrame(() => {
      const elem = document.querySelector(
        `[data-conversation-id="${selected.id}"]`
      );
      if (elem) scrollIntoView(elem, { scrollMode: "if-needed" });
    });
    return () => cancelAnimationFrame(raf);
  }, [selected?.id, list]);

  const handleConfirmDelete = async () => {
    if (!deleteCandidate) return;
    const index = list.findIndex((c) => c.id === selected?.id);
    await store.deleteConversation(deleteCandidate.id);
    setDeleteCandidate(undefined);
    setSelected(list[index] ?? undefined);
  };

  return (
    <div className="h-full overflow-hidden flex flex-col">
      <div className="flex-1 overflow-y-auto p-1 flex flex-col gap-y-2">
        {ready ? (
          <>
            {list.map((conversation) => {
              const isActive = selected?.id === conversation.id;
              const previewTitle =
                head(conversation.messageList)?.content ||
                t("plugin.ai.conversation.untitled");
              return (
                <div
                  key={conversation.id}
                  data-conversation-id={conversation.id}
                  className={cn(
                    "flex items-start gap-x-0.5 border rounded-md py-2 pl-2 pr-0.5 hover:bg-indigo-50 hover:border-indigo-400 cursor-pointer",
                    isActive && "bg-indigo-100 border-indigo-400"
                  )}
                  onClick={() => setSelected(conversation)}
                >
                  {conversation.name ? (
                    <div className="text-sm flex-1 whitespace-pre-wrap wrap-break-word break-all">
                      {conversation.name}
                    </div>
                  ) : (
                    <div className="text-sm flex-1 truncate text-gray-500 italic">
                      {previewTitle}
                    </div>
                  )}
                  <div className="flex items-center gap-x-px">
                    <button
                      type="button"
                      className="flex items-center p-0.5 border border-transparent rounded-sm text-gray-500 hover:text-accent hover:bg-indigo-50 hover:border-accent cursor-pointer"
                      onClick={(e) => {
                        e.stopPropagation();
                        setRename(conversation);
                      }}
                      aria-label={t("plugin.ai.conversation.rename")}
                    >
                      <PencilIcon className="size-3" />
                    </button>
                    <button
                      type="button"
                      className="flex items-center p-0.5 border border-transparent rounded-sm text-gray-500 hover:text-accent hover:bg-indigo-50 hover:border-accent cursor-pointer"
                      onClick={(e) => {
                        e.stopPropagation();
                        setDeleteCandidate(conversation);
                      }}
                      aria-label={t("plugin.ai.conversation.delete")}
                    >
                      <TrashIcon className="size-3" />
                    </button>
                  </div>
                </div>
              );
            })}

            <Button
              variant="outline"
              size="sm"
              className="sticky bottom-0 flex items-center justify-center gap-x-1"
              onClick={() => events.emit("new-conversation", { input: "" })}
            >
              <PlusIcon className="size-4" />
              <span className="pr-2">
                {t("plugin.ai.conversation.new-conversation")}
              </span>
            </Button>
          </>
        ) : (
          <Loader2 className="self-center mt-8 size-5 animate-spin" />
        )}
      </div>

      {rename && (
        <ConversationRenameDialog
          conversation={rename}
          onCancel={() => setRename(undefined)}
          onUpdated={() => setRename(undefined)}
        />
      )}

      <AlertDialog
        open={!!deleteCandidate}
        onOpenChange={(open) => !open && setDeleteCandidate(undefined)}
      >
        <AlertDialogContent>
          <AlertDialogTitle>
            {t("plugin.ai.conversation.delete")}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t("bbkit.confirm-button.sure-to-delete")}
          </AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogClose
              render={<Button variant="outline">{t("common.cancel")}</Button>}
            />
            <Button
              variant="destructive"
              onClick={() => void handleConfirmDelete()}
            >
              {t("common.delete")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
