import type { Engine } from "@/types/proto/v1/common";
import type { DatabaseMetadata } from "@/types/proto/v1/database_service";
import type { AISetting } from "@/types/proto/v1/setting_service";
import type Emittery from "emittery";
import type { Ref } from "vue";
import type { Conversation } from "./conversation";

export type AIContextEvents = Emittery<{
  "run-statement": { statement: string };
  error: string;
  "new-conversation": { input: string };
  "send-chat": { content: string; newChat?: boolean };
}>;

export type AIChatInfo = {
  list: Ref<Conversation[]>;
  ready: Ref<boolean>;
  selected: Ref<Conversation | undefined>;
};

export type AIContext = {
  aiSetting: Ref<AISetting>;
  engine: Ref<Engine | undefined>;
  databaseMetadata: Ref<DatabaseMetadata | undefined>;
  schema: Ref<string | undefined>;
  showHistoryDialog: Ref<boolean>;

  chat: Ref<AIChatInfo>;
  pendingSendChat: Ref<{ content: string } | undefined>;
  pendingPreInput: Ref<string | undefined>;

  // Events
  events: AIContextEvents;
};
