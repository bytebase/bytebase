import { create as createProto } from "@bufbuild/protobuf";
import type { ReactNode } from "react";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useAppDatabaseMetadata } from "@/react/hooks/useAppDatabaseMetadata";
import { useConnectionOfCurrentSQLEditorTab } from "@/react/hooks/useSQLEditorBridge";
import { useVueState } from "@/react/hooks/useVueState";
import { useAppStore } from "@/react/stores/app";
import {
  getCurrentSQLEditorTab,
  useCurrentSQLEditorTab,
} from "@/react/stores/sqlEditor/tab";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { DatabaseMetadata } from "@/types/proto-es/v1/database_service_pb";
import type { AISetting } from "@/types/proto-es/v1/setting_service_pb";
import {
  AISettingSchema,
  Setting_SettingName,
} from "@/types/proto-es/v1/setting_service_pb";
import { wrapRefAsPromise } from "@/utils";
import { aiContextEvents, getChatByTab } from "../logic";
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

  // The original Vue ProvideAIContext fetched the AI setting on mount.
  // Mirror that here. `getOrFetchSettingByName` is idempotent; firing it
  // again on remount is cheap.
  useEffect(() => {
    void useAppStore
      .getState()
      .getOrFetchSettingByName(Setting_SettingName.AI, /* silent */ true);
  }, []);

  // Subscribe to the setting cache so this re-resolves once the AI setting
  // loads (selector returns the AI value or the stable empty fallback).
  const settingsByName = useAppStore((s) => s.settingsByName);
  const aiSetting = useMemo<AISetting>(() => {
    const setting = useAppStore
      .getState()
      .getSettingByName(Setting_SettingName.AI);
    if (setting?.value?.value?.case === "ai") {
      return setting.value.value.value;
    }
    return EMPTY_AI_SETTING;
  }, [settingsByName]);

  const currentTab = useCurrentSQLEditorTab();

  const { instance, database } = useConnectionOfCurrentSQLEditorTab();
  const engine: Engine | undefined = instance.engine;

  // `useAppDatabaseMetadata` self-fetches and subscribes; the app-store
  // dbSchema slice dedupes concurrent fetches, so this stays cheap even
  // if another mount triggers the same fetch in parallel.
  const databaseName = database.name;
  const fetchedMetadata = useAppDatabaseMetadata(databaseName ?? "");
  const databaseMetadata = databaseName ? fetchedMetadata : undefined;

  const schema: string | undefined = currentTab?.connection.schema;

  // ---- Per-tab chat info -------------------------------------------------
  //
  // `getChatByTab(tab)` returns an `AIChatInfo` whose inner `list`,
  // `ready`, and `selected` are Vue refs (the conversation list is
  // backed by a Pinia store). We select the tab from Zustand, memoize
  // the per-tab chat, and bridge each inner ref via its own
  // `useVueState` getter — subscribing at the right dep granularity
  // (chat list mutations don't churn `ready`, and so on). `deps:
  // [chatInfo]` re-subscribes each watch when the tab's chat is
  // swapped, since the swap arrives through Zustand rather than Vue.
  const chatInfo = useMemo(() => getChatByTab(currentTab), [currentTab]);

  const chatList = useVueState<Conversation[]>(() => chatInfo.list.value, {
    deep: true,
    deps: [chatInfo],
  });
  const chatReady = useVueState<boolean>(() => chatInfo.ready.value, {
    deps: [chatInfo],
  });
  const chatSelected = useVueState<Conversation | undefined>(
    () => chatInfo.selected.value,
    { deps: [chatInfo] }
  );
  const setChatSelected = useCallback(
    (next: Conversation | undefined) => {
      chatInfo.selected.value = next;
    },
    [chatInfo]
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

  // Live handle to the current tab's chat for the event listeners
  // below — they fire outside React's render and must read the latest
  // refs without re-subscribing on every tab switch.
  const chatInfoRef = useRef(chatInfo);
  chatInfoRef.current = chatInfo;

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

  // We need the latest chat refs + tab from inside the listeners.
  // `chatInfoRef` always points at the current tab's chat, and
  // `getCurrentSQLEditorTab()` reads the live Zustand tab at fire time.
  useEffect(() => {
    const offNewConversation = events.on(
      "new-conversation",
      async ({ input }) => {
        const tab = getCurrentSQLEditorTab();
        if (!tab) return;
        // Wait until the per-tab chat fetch has resolved before deciding
        // whether to reuse or create. Without this guard a brand-new tab
        // sees `selected === undefined` and creates a duplicate empty
        // conversation when one would have hydrated a beat later.
        await wrapRefAsPromise(chatInfoRef.current.ready, /* expected */ true);
        setShowHistoryDialog(false);

        const sel = chatInfoRef.current.selected.value;
        if (!sel || sel.messageList.length !== 0) {
          // Reuse if the current chat is empty, otherwise create a fresh one.
          const c = await conversationStore.createConversation({
            name: "",
            ...tab.connection,
          });
          chatInfoRef.current.selected.value = c;
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
      const tab = getCurrentSQLEditorTab();
      if (!tab) return;
      await wrapRefAsPromise(chatInfoRef.current.ready, /* expected */ true);
      if (newChat) {
        setShowHistoryDialog(false);
        const c = await conversationStore.createConversation({
          name: "",
          ...tab.connection,
        });
        chatInfoRef.current.selected.value = c;
      }
      requestAnimationFrame(() => {
        setPendingSendChat({ content });
      });
    });

    return () => {
      offNewConversation();
      offSendChat();
    };
    // `events` and `conversationStore` are stable singletons;
    // `chatInfoRef` / `getCurrentSQLEditorTab` read the latest values
    // at fire time — no need to re-subscribe on tab change.
  }, [events, conversationStore]);

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
