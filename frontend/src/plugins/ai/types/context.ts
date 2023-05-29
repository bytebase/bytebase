import type { Ref } from "vue";
import type Emittery from "emittery";
import type { DatabaseMetadata } from "@/types/proto/store/database";
import { Conversation } from "./conversation";
import { Engine } from "@/types/proto/v1/common";

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
  engine: Ref<Engine | undefined>;
  databaseMetadata: Ref<DatabaseMetadata | undefined>;
  autoRun: Ref<boolean>;
  showHistoryDialog: Ref<boolean>;

  chat: Ref<AIChatInfo>;

  // Events
  events: AIContextEvents;
};
