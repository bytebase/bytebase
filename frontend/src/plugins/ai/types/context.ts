import type { Ref } from "vue";
import type Emittery from "emittery";
import type { EngineType } from "@/types";
import type { DatabaseMetadata } from "@/types/proto/store/database";

export type AIContextEvents = Emittery<{
  "apply-statement": { statement: string; run: boolean };
  error: string;
}>;

export type AIContext = {
  showDialog: Ref<boolean>;
  openAIKey: Ref<string>;
  engineType: Ref<EngineType | undefined>;
  databaseMetadata: Ref<DatabaseMetadata | undefined>;
  autoRun: Ref<boolean>;

  // Events
  events: AIContextEvents;
};
