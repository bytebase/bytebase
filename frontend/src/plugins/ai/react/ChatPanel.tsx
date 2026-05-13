import { create as createProto } from "@bufbuild/protobuf";
import { head } from "lodash-es";
import { Loader2 } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { sqlServiceClientConnect } from "@/connect";
import { useVueState } from "@/react/hooks/useVueState";
import { useSQLEditorTabStore } from "@/store";
import {
  type AICompletionRequest_Message,
  AICompletionRequest_MessageSchema,
  AICompletionRequestSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import { nextAnimationFrame } from "@/utils";
import * as promptUtils from "../logic/prompt";
import { useConversationStore } from "../store";
import { ActionBar } from "./ActionBar";
import { ChatView } from "./ChatView/ChatView";
import { useAIContext } from "./context";
import { DynamicSuggestions } from "./DynamicSuggestions";
import { HistoryPanel } from "./HistoryPanel/HistoryPanel";
import { PromptInput } from "./PromptInput";

/**
 * React port of `plugins/ai/components/ChatPanel.vue`.
 *
 * The chat surface: ActionBar on top, ChatView in the middle (or a
 * spinner while the per-tab fetch lands), DynamicSuggestions +
 * PromptInput at the bottom, and the HistoryPanel drawer mounted once.
 *
 * `requestAI(query)` is the orchestrator:
 *   1. Push a USER message. On the FIRST message of a conversation,
 *      prepend the schema declaration so the model has context. Subsequent
 *      messages use the bare query.
 *   2. Push an AI message in `LOADING` state.
 *   3. Call `sqlServiceClientConnect.aICompletion` with the full history.
 *   4. Update the AI message with the response (DONE) or error (FAILED).
 *   5. On FAILED, emit `error` on `aiContextEvents` for the host to
 *      surface (e.g. toast).
 *
 * Two `flush: "post"` watch blocks from the Vue version translate to
 * `useEffect` + rAF in React:
 *   - Auto-create an empty conversation when the per-tab fetch resolves
 *     to an empty list.
 *   - Fire `requestAI` when the `send-chat` event handler in the
 *     provider stashes a `pendingSendChat` payload.
 */
export function ChatPanel() {
  const tabStore = useSQLEditorTabStore();
  const store = useConversationStore();
  // Reactive bridge for `tabStore.currentTab` — Pinia mutates the tab
  // proxy in place when the connection changes, so a plain
  // `tabStore.currentTab` read at render time would miss updates.
  const hasCurrentTab = useVueState(() => tabStore.currentTab != null);

  const context = useAIContext();
  const {
    aiSetting,
    chat,
    setShowHistoryDialog,
    pendingSendChat,
    setPendingSendChat,
    events,
  } = context;
  const { list: conversationList, ready, selected } = chat;

  const [loading, setLoading] = useState(false);

  // Reactive tab/connection signal so we can hide history on
  // (instance, database) change. Reading the connection fields via
  // `useVueState` gives us a stable string key the effect can depend on.
  const connectionKey = useVueState(() => {
    const tab = tabStore.currentTab;
    if (!tab) return "";
    return `${tab.connection.instance}|${tab.connection.database}`;
  });
  useEffect(() => {
    setShowHistoryDialog(false);
  }, [connectionKey, setShowHistoryDialog]);

  // Pull the latest `requestAI` into a ref so the pending-send-chat
  // effect doesn't need to depend on its identity (the callback closes
  // over `selected`, `aiSetting`, etc. — wiring those as effect deps
  // would re-run the effect on every conversation tweak).
  const requestAIRef = useRef<(query: string) => Promise<void>>(async () => {});

  const requestAI = useCallback(
    async (query: string) => {
      const conversation = selected;
      if (!conversation) return;
      const tab = tabStore.currentTab;
      if (!tab) return;

      const { messageList } = conversation;
      if (messageList.length === 0) {
        const engine = context.engine;
        const databaseMetadata = context.databaseMetadata;
        const schema = context.schema;
        const prompts: string[] = [
          promptUtils.declaration(databaseMetadata, engine, schema),
          query,
        ];
        const prompt = prompts.join("\n");
        await store.createMessage({
          conversation_id: conversation.id,
          content: query,
          prompt,
          author: "USER",
          error: "",
          status: "DONE",
        });
        console.debug("[AI Assistant] init chat:", prompt);
      } else {
        await store.createMessage({
          conversation_id: conversation.id,
          content: query,
          prompt: query,
          author: "USER",
          error: "",
          status: "DONE",
        });
      }

      const answer = await store.createMessage({
        author: "AI",
        prompt: "",
        content: "",
        error: "",
        conversation_id: conversation.id,
        status: "LOADING",
      });
      const messages: AICompletionRequest_Message[] =
        conversation.messageList.map((message) =>
          createProto(AICompletionRequest_MessageSchema, {
            role: message.author === "USER" ? "user" : "assistant",
            content: message.prompt,
          })
        );
      setLoading(true);
      try {
        const request = createProto(AICompletionRequestSchema, { messages });
        const response = await sqlServiceClientConnect.aICompletion(request);
        const text = head(
          head(response.candidates)?.content?.parts
        )?.text?.trim();
        console.debug("[AI Assistant] answer:", text);
        if (text) {
          answer.content = text;
          answer.prompt = text;
        }
        answer.status = "DONE";
      } catch (err) {
        answer.error = String(err);
        answer.status = "FAILED";
      } finally {
        setLoading(false);
        await store.updateMessage(answer);
        if (answer.status === "FAILED") {
          events.emit("error", answer.error);
        }
      }
    },
    [
      selected,
      tabStore,
      store,
      context.engine,
      context.databaseMetadata,
      context.schema,
      events,
    ]
  );
  requestAIRef.current = requestAI;

  // Auto-create an empty conversation when the per-tab fetch resolves
  // to an empty list. Mirrors the Vue `watch([ready, conversationList],
  // ..., { immediate: true })` — `requestAnimationFrame` defers to the
  // next paint so any concurrent provider-side `new-conversation` flow
  // gets a chance to claim the slot first.
  useEffect(() => {
    if (!ready) return;
    if (conversationList.length > 0) return;
    void store.createConversation({
      name: "",
      instance: tabStore.currentTab?.connection.instance ?? "",
      database: tabStore.currentTab?.connection.database ?? "",
    });
    // We intentionally watch only the boolean transition + the empty
    // condition, not the full `conversationList` reference — Vue's
    // version reacts on identity; the React version reacts on the
    // length so we don't fire each time a new message arrives.
  }, [ready, conversationList.length, store, tabStore]);

  // Fire `requestAI` when a pending send-chat lands. The Vue version
  // used `watch(..., { flush: "post" })` — we approximate by waiting
  // for the next animation frame so the conversation creation in the
  // provider's `send-chat` handler has settled.
  useEffect(() => {
    if (!ready) return;
    if (!pendingSendChat) return;
    let cancelled = false;
    void (async () => {
      await nextAnimationFrame();
      if (cancelled) return;
      const payload = pendingSendChat;
      setPendingSendChat(undefined);
      if (!payload) return;
      void requestAIRef.current(payload.content);
    })();
    return () => {
      cancelled = true;
    };
  }, [ready, pendingSendChat, setPendingSendChat]);

  if (!aiSetting.enabled) return null;

  return (
    <div className="w-full h-full flex-1 flex flex-col overflow-hidden">
      <ActionBar />

      {ready ? (
        <ChatView conversation={selected} />
      ) : (
        <div className="flex-1 overflow-hidden relative flex items-center justify-center">
          <Loader2 className="size-5 animate-spin text-control-light" />
        </div>
      )}

      <div className="px-2 pb-2 pt-1 flex flex-col gap-1">
        <DynamicSuggestions onEnter={(value) => void requestAI(value)} />
        {hasCurrentTab && (
          <PromptInput
            disabled={loading}
            onEnter={(value) => void requestAI(value)}
          />
        )}
      </div>

      <HistoryPanel />
    </div>
  );
}
