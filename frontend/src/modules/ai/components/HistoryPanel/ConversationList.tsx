import { head } from "lodash-es";
import { Loader2, PencilIcon, PlusIcon, TrashIcon } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import scrollIntoView from "scroll-into-view-if-needed";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { useCurrentSQLEditorTab } from "@/modules/sql-editor/store/tab";
import { useConversationStore } from "../../store";
import type { Conversation } from "../../types";
import { useAIContext } from "../context";

const resizeTextarea = (textarea: HTMLTextAreaElement) => {
  textarea.style.height = "0px";
  textarea.style.height = `${textarea.scrollHeight}px`;
};

/**
 * React port of `plugins/ai/components/HistoryPanel/ConversationList.vue`.
 *
 * Per-tab list of historical conversations with select / rename / delete
 * actions and a sticky "New conversation" footer.
 *
 * Behavioural pitfalls preserved from the Vue source:
 *   - On tab switch (`(instance, database)` change) the inline rename is
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
  const currentTab = useCurrentSQLEditorTab();
  const store = useConversationStore();
  const { events, chat } = useAIContext();
  const { list, ready, selected, setSelected } = chat;

  const [rename, setRename] = useState<
    { conversation: Conversation; name: string } | undefined
  >(undefined);
  const [deleteCandidate, setDeleteCandidate] = useState<
    Conversation | undefined
  >(undefined);
  const renameTextareaRef = useRef<HTMLTextAreaElement>(null);

  // Dismiss the inline rename whenever the active tab's connection
  // changes — a stale edit tied to a different tab's conversation
  // would mutate the wrong record on Save.
  const connectionKey = currentTab
    ? `${currentTab.connection.instance}|${currentTab.connection.database}`
    : "";
  useEffect(() => {
    setRename(undefined);
    setDeleteCandidate(undefined);
  }, [connectionKey]);

  useEffect(() => {
    if (!rename) return;
    const raf = requestAnimationFrame(() => {
      const textarea = renameTextareaRef.current;
      if (!textarea) return;
      resizeTextarea(textarea);
      textarea.focus();
      textarea.select();
    });
    return () => cancelAnimationFrame(raf);
  }, [rename?.conversation.id]);

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
    const index = list.findIndex((c) => c.id === deleteCandidate.id);
    const remaining = list.filter((c) => c.id !== deleteCandidate.id);
    await store.deleteConversation(deleteCandidate.id);
    setDeleteCandidate(undefined);
    if (selected?.id === deleteCandidate.id) {
      setSelected(remaining[index] ?? remaining[index - 1]);
    }
  };

  const handleRename = async (conversation: Conversation) => {
    if (rename?.conversation.id !== conversation.id) return;
    const name = rename.name;
    setRename(undefined);
    if (name === conversation.name) return;
    await store.updateConversation({ ...conversation, name });
  };

  const startRename = (conversation: Conversation) => {
    setRename({ conversation, name: conversation.name });
  };

  return (
    <div className="h-full overflow-hidden flex flex-col">
      <div className="flex-1 overflow-y-auto p-1 flex flex-col gap-y-2">
        {ready ? (
          <>
            {list.map((conversation) => {
              const isActive = selected?.id === conversation.id;
              const isRenaming = rename?.conversation.id === conversation.id;
              const previewTitle =
                head(conversation.messageList)?.content ||
                t("plugin.ai.conversation.untitled");
              return (
                <div
                  key={conversation.id}
                  data-conversation-id={conversation.id}
                  className={cn(
                    "flex items-start gap-x-0.5 border rounded-md py-2 pl-2 pr-0.5 hover:bg-control-bg hover:border-accent cursor-pointer",
                    isActive && "bg-accent/10 border-accent",
                    isRenaming && "cursor-default"
                  )}
                  onClick={() => setSelected(conversation)}
                >
                  {isRenaming ? (
                    <Textarea
                      ref={renameTextareaRef}
                      size="md"
                      rows={1}
                      value={rename.name}
                      placeholder={previewTitle}
                      aria-label={t("plugin.ai.conversation.rename")}
                      className="min-h-6 min-w-0 flex-1 resize-none overflow-hidden text-sm leading-5 placeholder:italic"
                      style={{ padding: "2px 4px" }}
                      onClick={(event) => event.stopPropagation()}
                      onChange={(event) => {
                        resizeTextarea(event.currentTarget);
                        setRename({
                          conversation,
                          name: event.target.value,
                        });
                      }}
                      onBlur={() => void handleRename(conversation)}
                      onKeyDown={(event) => {
                        event.stopPropagation();
                        if (event.nativeEvent.isComposing) return;
                        if (event.key === "Enter") {
                          event.preventDefault();
                          event.currentTarget.blur();
                        } else if (event.key === "Escape") {
                          event.preventDefault();
                          setRename(undefined);
                        }
                      }}
                    />
                  ) : conversation.name ? (
                    <div className="min-w-0 flex-1 whitespace-pre-wrap wrap-break-word break-all text-sm">
                      {conversation.name}
                    </div>
                  ) : (
                    <div className="min-w-0 flex-1 truncate text-sm text-control-light italic">
                      {previewTitle}
                    </div>
                  )}
                  {!isRenaming && (
                    <div className="flex items-center gap-x-px">
                      <Button
                        type="button"
                        appearance="secondary"
                        size="xs"
                        className="shrink-0 text-control-light hover:text-accent"
                        onPointerDown={(event) => {
                          if (event.button !== 0) return;
                          event.preventDefault();
                          event.stopPropagation();
                          startRename(conversation);
                        }}
                        onClick={(e) => {
                          e.stopPropagation();
                          startRename(conversation);
                        }}
                        aria-label={t("plugin.ai.conversation.rename")}
                      >
                        <PencilIcon className="size-3.5" />
                      </Button>
                      <Popover
                        open={deleteCandidate?.id === conversation.id}
                        onOpenChange={(open) =>
                          setDeleteCandidate(open ? conversation : undefined)
                        }
                      >
                        <PopoverTrigger
                          render={
                            <Button
                              type="button"
                              appearance="secondary"
                              size="xs"
                              className="shrink-0 text-control-light hover:text-error"
                              onClick={(event) => {
                                event.stopPropagation();
                                setDeleteCandidate(conversation);
                              }}
                              aria-label={t("plugin.ai.conversation.delete")}
                            />
                          }
                        >
                          <TrashIcon className="size-3.5" />
                        </PopoverTrigger>
                        <PopoverContent
                          side="bottom"
                          align="end"
                          className="w-64"
                        >
                          <div className="font-medium">
                            {t("plugin.ai.conversation.delete")}
                          </div>
                          <div className="mt-1 text-control-light">
                            {t("bbkit.confirm-button.sure-to-delete")}
                          </div>
                          <div className="mt-3 flex justify-end gap-x-2">
                            <Button
                              appearance="outline"
                              size="sm"
                              onClick={() => setDeleteCandidate(undefined)}
                            >
                              {t("common.cancel")}
                            </Button>
                            <Button
                              variant="destructive"
                              size="sm"
                              onClick={() => void handleConfirmDelete()}
                            >
                              {t("common.delete")}
                            </Button>
                          </div>
                        </PopoverContent>
                      </Popover>
                    </div>
                  )}
                </div>
              );
            })}

            <Button
              appearance="outline"
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
    </div>
  );
}
