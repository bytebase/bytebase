import { create as createProto } from "@bufbuild/protobuf";
import type { ReactNode } from "react";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useVueState } from "@/react/hooks/useVueState";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
  useSettingV1Store,
  useSQLEditorTabStore,
} from "@/store";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { DatabaseMetadata } from "@/types/proto-es/v1/database_service_pb";
import type { AISetting } from "@/types/proto-es/v1/setting_service_pb";
import {
  AISettingSchema,
  Setting_SettingName,
} from "@/types/proto-es/v1/setting_service_pb";
import { wrapRefAsPromise } from "@/utils";
import { aiContextEvents, useChatByTab } from "../logic";
import { useConversationStore } from "../store";
import type { AIContextEvents, Conversation } from "../types";

/**
 * React-shaped slice of the AI plugin's per-tab state. Mirrors the Vue
 * `AIChatInfo` (refs) with plain values + an explicit setter so consumers
 * don't have to know about Vue's reactivity.
 */
export type ReactAIChatInfo = {
  list: Conversation[];
  ready: boolean;
  selected: Conversation | undefined;
  setSelected: (next: Conversation | undefined) => void;
};

/**
 * React-shaped AI plugin context. Replaces the Vue `AIContext` defined in
 * `plugins/ai/types/context.ts` for components under `plugins/ai/react/`.
 * Same fields, plain values, paired setters for mutable state.
 */
export type ReactAIContext = {
  aiSetting: AISetting;
  engine: Engine | undefined;
  databaseMetadata: DatabaseMetadata | undefined;
  schema: string | undefined;

  showHistoryDialog: boolean;
  setShowHistoryDialog: (next: boolean) => void;

  chat: ReactAIChatInfo;

  /**
   * One-shot trigger written by the `send-chat` event handler — the
   * downstream `ChatPanel` effect reads this and dispatches the actual
   * AICompletion request, then clears it via `setPendingSendChat(undefined)`.
   * Same pattern as the Vue `pendingSendChat` ref.
   */
  pendingSendChat: { content: string } | undefined;
  setPendingSendChat: (next: { content: string } | undefined) => void;

  /**
   * One-shot trigger written by the `new-conversation` event handler when
   * the host pre-fills a seed prompt (e.g. `OpenAIButton.tsx` "Ask AI
   * about this query"). `PromptInput` reads + clears it on mount /
   * pending-pre-input change.
   */
  pendingPreInput: string | undefined;
  setPendingPreInput: (next: string | undefined) => void;

  events: AIContextEvents;
};

const AIContextReactContext = createContext<ReactAIContext | null>(null);

export function useAIContext(): ReactAIContext {
  const ctx = useContext(AIContextReactContext);
  if (!ctx) {
    throw new Error(
      "useAIContext (React) must be used inside <AIContextProvider>"
    );
  }
  return ctx;
}

const EMPTY_AI_SETTING: AISetting = createProto(AISettingSchema, {});

export function AIContextProvider({ children }: { children: ReactNode }) {
  // ---- Vue-side state bridged via useVueState ----------------------------

  const settingV1Store = useSettingV1Store();
  // The original Vue ProvideAIContext fetched the AI setting on mount.
  // Mirror that here. `getOrFetchSettingByName` is idempotent; firing it
  // again on remount is cheap.
  useEffect(() => {
    void settingV1Store.getOrFetchSettingByName(
      Setting_SettingName.AI,
      /* silent */ true
    );
  }, [settingV1Store]);

  const aiSetting = useVueState<AISetting>(() => {
    const setting = settingV1Store.getSettingByName(Setting_SettingName.AI);
    if (setting?.value?.value?.case === "ai") {
      return setting.value.value.value;
    }
    return EMPTY_AI_SETTING;
  });

  const { instance: instanceRef, database: databaseRef } =
    useConnectionOfCurrentSQLEditorTab();

  const engine = useVueState<Engine | undefined>(
    () => instanceRef.value.engine
  );

  // Fetch database metadata on database change (mirrors Vue
  // `useMetadata`'s `watchEffect` + cache read). `useDBSchemaV1Store`
  // dedupes concurrent fetches, so firing this from the React effect
  // doesn't race with another caller. We read the cache via
  // `useVueState` so cache hydration triggers a re-render.
  const databaseName = useVueState(() => databaseRef.value.name);
  const dbSchemaStore = useDBSchemaV1Store();
  useEffect(() => {
    if (!databaseName) return;
    void dbSchemaStore.getOrFetchDatabaseMetadata({ database: databaseName });
  }, [databaseName, dbSchemaStore]);
  const databaseMetadata = useVueState<DatabaseMetadata | undefined>(() =>
    databaseName ? dbSchemaStore.getDatabaseMetadata(databaseName) : undefined
  );

  const tabStore = useSQLEditorTabStore();
  const schema = useVueState<string | undefined>(
    () => tabStore.currentTab?.connection.schema
  );

  // ---- Per-tab chat info -------------------------------------------------
  //
  // `useChatByTab()` returns a Vue `ComputedRef<AIChatInfo>` where the
  // inner `AIChatInfo` itself holds Vue refs (`list`, `ready`, `selected`).
  // We hold the ComputedRef across renders and read each inner ref via
  // its own `useVueState` getter — that way React subscribes to the right
  // dep granularity (chat list mutations don't churn `ready`, and so on).
  const chatByTabRef = useChatByTab();

  const chatList = useVueState<Conversation[]>(
    () => chatByTabRef.value.list.value,
    { deep: true }
  );
  const chatReady = useVueState<boolean>(() => chatByTabRef.value.ready.value);
  const chatSelected = useVueState<Conversation | undefined>(
    () => chatByTabRef.value.selected.value
  );
  const setChatSelected = useCallback(
    (next: Conversation | undefined) => {
      chatByTabRef.value.selected.value = next;
    },
    [chatByTabRef]
  );
  const chat = useMemo<ReactAIChatInfo>(
    () => ({
      list: chatList,
      ready: chatReady,
      selected: chatSelected,
      setSelected: setChatSelected,
    }),
    [chatList, chatReady, chatSelected, setChatSelected]
  );

  // ---- React-side state --------------------------------------------------

  const [showHistoryDialog, setShowHistoryDialog] = useState(false);
  const [pendingSendChat, setPendingSendChat] = useState<
    { content: string } | undefined
  >(undefined);
  const [pendingPreInput, setPendingPreInput] = useState<string | undefined>(
    undefined
  );

  // Stable reference to `aiContextEvents` — singleton, but keep typed.
  const events: AIContextEvents = aiContextEvents;

  // ---- Event listeners (mirror ProvideAIContext.vue) ---------------------

  const conversationStore = useConversationStore();

  // We need access to the latest chat refs from inside the listeners.
  // `chatByTabRef` is stable across renders (it's a ComputedRef captured
  // by useMemo([]) inside `useChatByTab`), but `tab` is a live Pinia
  // value we read via `tabStore.currentTab` at fire time.
  useEffect(() => {
    const offNewConversation = events.on(
      "new-conversation",
      async ({ input }) => {
        const tab = tabStore.currentTab;
        if (!tab) return;
        // Wait until the per-tab chat fetch has resolved before deciding
        // whether to reuse or create. Without this guard a brand-new tab
        // sees `selected === undefined` and creates a duplicate empty
        // conversation when one would have hydrated a beat later.
        await wrapRefAsPromise(chatByTabRef.value.ready, /* expected */ true);
        setShowHistoryDialog(false);

        const sel = chatByTabRef.value.selected.value;
        if (!sel || sel.messageList.length !== 0) {
          // Reuse if the current chat is empty, otherwise create a fresh one.
          const c = await conversationStore.createConversation({
            name: "",
            ...tab.connection,
          });
          chatByTabRef.value.selected.value = c;
        }
        if (input) {
          // rAF mirrors the Vue version — gives `PromptInput`'s
          // pending-pre-input effect a frame to land after the conversation
          // creation settles. Without it the seed text can be dropped if
          // the effect fires before `selected` updates.
          requestAnimationFrame(() => {
            setPendingPreInput(input);
          });
        }
      }
    );

    const offSendChat = events.on("send-chat", async ({ content, newChat }) => {
      const tab = tabStore.currentTab;
      if (!tab) return;
      await wrapRefAsPromise(chatByTabRef.value.ready, /* expected */ true);
      if (newChat) {
        setShowHistoryDialog(false);
        const c = await conversationStore.createConversation({
          name: "",
          ...tab.connection,
        });
        chatByTabRef.value.selected.value = c;
      }
      requestAnimationFrame(() => {
        setPendingSendChat({ content });
      });
    });

    return () => {
      offNewConversation();
      offSendChat();
    };
    // `events`, `tabStore`, `chatByTabRef`, `conversationStore` are stable
    // singletons / Pinia store references — fine to depend on them.
  }, [events, tabStore, chatByTabRef, conversationStore]);

  // ---- Memoized value bundle --------------------------------------------

  const value = useMemo<ReactAIContext>(
    () => ({
      aiSetting,
      engine,
      databaseMetadata,
      schema,
      showHistoryDialog,
      setShowHistoryDialog,
      chat,
      pendingSendChat,
      setPendingSendChat,
      pendingPreInput,
      setPendingPreInput,
      events,
    }),
    [
      aiSetting,
      engine,
      databaseMetadata,
      schema,
      showHistoryDialog,
      chat,
      pendingSendChat,
      pendingPreInput,
      events,
    ]
  );

  return (
    <AIContextReactContext.Provider value={value}>
      {children}
    </AIContextReactContext.Provider>
  );
}
