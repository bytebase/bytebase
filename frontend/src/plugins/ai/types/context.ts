import type { Ref } from "vue";
import type Emittery from "emittery";
import type { EngineType } from "@/types";
import type { DatabaseMetadata } from "@/types/proto/store/database";
import { Conversation } from "./conversation";

export type AIContextEvents = Emittery<{
  "apply-statement": { statement: string; run: boolean };
  error: string;
  "new-conversation": any;
}>;

export type AIChatInfo = {
  list: Ref<Conversation[]>;
  ready: Ref<boolean>;
  selected: Ref<Conversation | undefined>;
};

export type AIContext = {
  openAIKey: Ref<string>;
  openAIEndpoint: Ref<string>;
  engineType: Ref<EngineType | undefined>;
  databaseMetadata: Ref<DatabaseMetadata | undefined>;
  autoRun: Ref<boolean>;
  showHistoryDialog: Ref<boolean>;

  chat: Ref<AIChatInfo>;

  // Events
  events: AIContextEvents;
};
