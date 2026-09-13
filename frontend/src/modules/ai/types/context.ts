import type Emittery from "emittery";

// Event bus shared by the AI plugin's logic and its React context
// (`src/modules/ai/components/context.tsx`).
export type AIContextEvents = Emittery<{
  "run-statement": { statement: string };
  error: string;
  "new-conversation": { input: string };
  "send-chat": { content: string; newChat?: boolean };
}>;
