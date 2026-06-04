import { create as createProto } from "@bufbuild/protobuf";
import { last } from "lodash-es";
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
import { aiContextEvents } from "../logic";
import { conversationListByConnection, useConversationStore } from "../store";
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
  // ---- AI setting --------------------------------------------------------

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
  // Conversations live in the Zustand `useConversationStore`. We subscribe to
  // the conversation map + per-connection ready flags and derive the current
  // tab's chat (list / ready / selected) with plain React state — no Vue refs.
  const tabInstance = currentTab?.connection.instance;
  const tabDatabase = currentTab?.connection.database;
  const connKey =
    tabInstance && tabDatabase ? `${tabInstance}/${tabDatabase}` : "";

  const conversationsById = useConversationStore((s) => s.conversationsById);
  const readyByConnection = useConversationStore((s) => s.readyByConnection);

  // Fetch the tab's conversations on connection change; the store sets the
  // ready flag and merges results.
  useEffect(() => {
    if (!tabInstance || !tabDatabase) return;
    void useConversationStore.getState().fetchConversationListByConnection({
      instance: tabInstance,
      database: tabDatabase,
    } as never);
  }, [tabInstance, tabDatabase]);

  const chatList = useMemo<Conversation[]>(() => {
    if (!tabInstance || !tabDatabase) return [];
    return Object.values(conversationsById)
      .filter((c) => c.instance === tabInstance && c.database === tabDatabase)
      .sort((a, b) => a.created_ts - b.created_ts);
  }, [conversationsById, tabInstance, tabDatabase]);

  const chatReady = connKey ? !!readyByConnection[connKey] : false;

  // Explicit per-connection selection; falls back to the last conversation
  // once the connection is ready (mirrors the Vue `watch` default).
  const [selectedIdByConn, setSelectedIdByConn] = useState<
    Record<string, string | undefined>
  >({});
  const selectedId = connKey ? selectedIdByConn[connKey] : undefined;
  const chatSelected = useMemo<Conversation | undefined>(() => {
    const explicit = selectedId
      ? chatList.find((c) => c.id === selectedId)
      : undefined;
    if (explicit) return explicit;
    return chatReady ? last(chatList) : undefined;
  }, [selectedId, chatList, chatReady]);
  const setChatSelected = useCallback(
    (next: Conversation | undefined) => {
      if (!connKey) return;
      setSelectedIdByConn((prev) => ({ ...prev, [connKey]: next?.id }));
    },
    [connKey]
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

  // Latest selection map for the event listeners (which fire outside render).
  const selectedIdByConnRef = useRef(selectedIdByConn);
  selectedIdByConnRef.current = selectedIdByConn;

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

  // The listeners fire outside render. They read the live tab via
  // `getCurrentSQLEditorTab()`, await the store's per-connection fetch (so a
  // brand-new tab doesn't create a duplicate empty conversation before its
  // existing ones hydrate), then read/select through the store +
  // `selectedIdByConnRef`.
  useEffect(() => {
    const selectFor = (conn: { instance: string; database: string }) => {
      const ck = `${conn.instance}/${conn.database}`;
      const list = conversationListByConnection(
        useConversationStore.getState(),
        conn
      );
      const selId = selectedIdByConnRef.current[ck];
      const explicit = selId ? list.find((c) => c.id === selId) : undefined;
      return { ck, list, selected: explicit ?? last(list) };
    };

    const offNewConversation = events.on(
      "new-conversation",
      async ({ input }) => {
        const tab = getCurrentSQLEditorTab();
        if (!tab) return;
        const conn = tab.connection;
        await useConversationStore
          .getState()
          .fetchConversationListByConnection(conn);
        setShowHistoryDialog(false);

        const { ck, selected } = selectFor(conn);
        if (!selected || selected.messageList.length !== 0) {
          // Reuse if the current chat is empty, otherwise create a fresh one.
          const c = await useConversationStore
            .getState()
            .createConversation({ name: "", ...conn });
          setSelectedIdByConn((prev) => ({ ...prev, [ck]: c.id }));
        }
        if (input) {
          // rAF mirrors the Vue version — gives `PromptInput`'s
          // pending-pre-input effect a frame to land after the conversation
          // creation settles.
          requestAnimationFrame(() => {
            setPendingPreInput(input);
          });
        }
      }
    );

    const offSendChat = events.on("send-chat", async ({ content, newChat }) => {
      const tab = getCurrentSQLEditorTab();
      if (!tab) return;
      const conn = tab.connection;
      await useConversationStore
        .getState()
        .fetchConversationListByConnection(conn);
      if (newChat) {
        setShowHistoryDialog(false);
        const c = await useConversationStore
          .getState()
          .createConversation({ name: "", ...conn });
        setSelectedIdByConn((prev) => ({
          ...prev,
          [`${conn.instance}/${conn.database}`]: c.id,
        }));
      }
      requestAnimationFrame(() => {
        setPendingSendChat({ content });
      });
    });

    return () => {
      offNewConversation();
      offSendChat();
    };
    // `events` is a stable singleton; `getCurrentSQLEditorTab` /
    // `selectedIdByConnRef` / the store read the latest values at fire time.
  }, [events]);

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
