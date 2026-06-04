import type Emittery from "emittery";

// The React-shaped AI plugin context lives in `react/context.tsx`
// (`ReactAIContext`). This module now only carries the framework-agnostic
// event bus type shared between the plugin's logic and React layers.
export type AIContextEvents = Emittery<{
  "run-statement": { statement: string };
  error: string;
  "new-conversation": { input: string };
  "send-chat": { content: string; newChat?: boolean };
}>;
